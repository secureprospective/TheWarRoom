// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HomeBrief from './HomeBrief';
import { commands } from '../commands/registry';
import type { HostedPort } from '../commissioner/hostedRoom';
import { layoutKey } from './homeLayout';
import pulse from './pulse.wire.testdata.json';
import assets from '../../../../testdata/wire/native/assets.json';
import jobs from '../../../../testdata/wire/commissioner/jobs.json';
import type { ResearchReader } from '../research/api';
import type { PlayerReading, Research } from '../research/contract';

const LEAGUE = 'native.home';
const MINE = 'franchise.01';
const ledger = { league: LEAGUE, name: 'Held league', revision: 8, entries: [
  { id: 'e1', who: 'Held team 2', what: 'Trade accepted with Held team 3', why: 'two players',
    when: '2026-09-08T10:00:00Z', gravity: 'G2', undo: '' },
  { id: 'e2', who: 'Held team 4', what: 'Released a player', why: '', when: '2026-09-09T10:00:00Z',
    gravity: 'G1', undo: '' }] };
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

function port(over: Record<string, unknown> = {}) {
  return vi.fn<HostedPort>(async request => {
    const league = request.arguments.league;
    const franchise = 'franchise' in request.arguments ? String(request.arguments.franchise) : '';
    const data: Record<string, unknown> = {
      'native.directory.read': { ...pulse.directory, league },
      'week.read': { ...pulse.week, schedule: { ...pulse.week.schedule, league, franchise } },
      'standings.read': { ...pulse.standings, league },
      'bracket.read': { ...pulse.bracket, league },
      'results.read': { ...pulse.results, league },
      'commish.ledger.read': ledger,
      ...over,
    };
    if (data[request.command] === undefined) throw new Error(`No fixture for ${request.command}`);
    return { label: 'Fake local port', writable: false, football: '', data: data[request.command] };
  });
}
async function mount(p: HostedPort, research?: ResearchReader) {
  await act(async () => { root.render(<HomeBrief league={LEAGUE} mine={MINE} port={p} research={research} />);
    });
  for (let i = 0; i < 20; i++)
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 5)); });
}
const q = (s: string) => container.querySelector(s);
const tile = (id: string) => q(`article.t[data-id="${id}"]`) as HTMLElement;
const click = async (el: Element | null) => { await act(async () => { (el as HTMLElement).click(); }); };

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  localStorage.clear();
  container = document.createElement('div'); document.body.append(container); root = createRoot(container);
});
afterEach(async () => {
  await act(async () => { root.unmount(); }); container.remove(); localStorage.clear();
  vi.restoreAllMocks(); vi.unstubAllGlobals();
});

it('reads each region from the existing league reads with this league and franchise', async () => {
  const p = port();
  await mount(p);
  const calls = p.mock.calls.map(([r]) => [r.command, r.arguments]);
  expect(calls).toContainEqual(['standings.read', { league: LEAGUE, season: 2026 }]);
  expect(calls).toContainEqual(['results.read', { league: LEAGUE, season: 2026, week: 2 }]);
  expect(calls).toContainEqual(['commish.ledger.read', { league: LEAGUE }]);
  expect(calls).toContainEqual(['bids.read', { league: LEAGUE, franchise: MINE }]);
  expect(calls).toContainEqual(['draft.read', expect.objectContaining({ league: LEAGUE, franchise: MINE })]);
});

it('renders the now strip, this week and the race from standings and results', async () => {
  await mount(port());
  const mine = pulse.standings.computed.find(r => r.franchise === MINE)!;
  const now = q('section.now')!.textContent!;
  expect(now).toContain(mine.record);
  expect(now).toContain('SEED');
  expect(tile('mu').textContent).toContain('Held team 1');
  expect(tile('race').textContent).toContain(mine.name);
  expect(q('.tick')!.textContent).toContain('franchise.01 starter');
});

it('lists the ledger newest first and opens a transaction card', async () => {
  await mount(port());
  const rows = tile('wire').querySelectorAll('.r.wire');
  expect(rows[0].textContent).toContain('Released a player');
  await click(rows[1]);
  expect(q('.card.open h2')!.textContent).toBe('Held team 2');
  expect(q('.card.open')!.textContent).toContain('two players');
});

it('says plainly what is not available instead of showing numbers', async () => {
  await mount(port());
  expect(tile('news').textContent).toContain('Not available yet');
  expect(q('.aband')!.textContent).toContain('this is not an all-clear');
});

it('opens a tile link and a shortcut through nav.open', async () => {
  await mount(port());
  await click(tile('mu').querySelector('.lnk'));
  expect(commands.history().at(-1)).toMatchObject({
    id: 'nav.open', args: { node: 'pulse', workspace: 'now' }, status: 'done' });
  const cap = Array.from(tile('sc').querySelectorAll('.scb')).find(b => b.textContent?.startsWith('Cap'))!;
  await click(cap);
  expect(commands.history().at(-1)).toMatchObject({
    id: 'nav.open', args: { node: 'hq', workspace: 'contracts-and-cap' }, status: 'done' });
});

it('customize moves a tile to overflow, saves it for this owner and league, and restores it', async () => {
  await mount(port());
  await click(q('.queue'));
  await click(tile('race').querySelector('.off'));
  expect(tile('race')).toBeNull();
  const saved = JSON.parse(localStorage.getItem(layoutKey(LEAGUE, MINE))!);
  expect(saved).toMatchObject({ mode: 'custom', over: ['race'] });
  await click(Array.from(container.querySelectorAll('.pull')).find(b =>
    b.textContent?.startsWith('OVERFLOW'))!);
  await click(q('.bdrawer .at2.ov'));
  expect(tile('race')).not.toBeNull();
});

it('draws the playoff cut from the league bracket setting, not a fixed number', async () => {
  const cutWith = async (seeds: number) => {
    const seed = pulse.bracket.fields[0].seeds[0];
    const fields = [{ ...pulse.bracket.fields[0],
      seeds: Array.from({ length: seeds }, (_, i) => ({ ...seed, seed: i + 1 })) }];
    await mount(port({ 'bracket.read': { ...pulse.bracket, league: LEAGUE, fields } }));
    const cut = tile('race').querySelector('.r.mini.cut .m')?.textContent;
    await act(async () => { root.unmount(); });
    root = createRoot(container);
    return cut;
  };
  const rank = pulse.standings.computed.find(r => r.franchise === MINE)!.rank;
  expect(await cutWith(rank - 1)).toBe(String(rank - 1));
  expect(await cutWith(rank - 2)).toBe(String(rank - 2));
});

// ---- APP-HOME-2: the reads HQ, the War Room, Pulse and Research already make ----
const report = (id: string, columns: string[], rows: string[][]) => ({ id, title: id, note: '', status: 'ok',
  source: '', asOf: '', how: '', columns, rows: rows.map(cells => ({ cells, range: null })), summary: [],
    link: null });
function reader(over: Partial<Research> = {}, form: Record<string, number[]> = {}) {
  const research: Research = { board: { reports: [report('M-048', ['Player', 'Position', 'Depth'],
    [['Player, 1001', 'LB', '1'], ['Somebody Else', 'QB', '2']])] } as unknown as Research['board'],
  news: report('M-058', [], []) as unknown as Research['news'], players: [], teams: [], positions: [],
  articles: [{ id: '1001', article: 'a1', name: 'Player, 1001', headline:
    'Player 1001 ruled out with a hamstring injury' },
    { id: '9999', article: 'a2', name: 'Nobody Held', headline: 'Signs a practice squad deal' }],
  scope: { mine: '', search: '', team: '', position: '', group: 'QB', trend: 'topAdds', depth: 'team',
    injuries: 'all' },
  ...over };
  const player = vi.fn(async (id: string) => ({ id, name: id, position: '', team: '', franchiseId: '', owner:
    '',
    articles: [], howLabel: '', report: report('R', [], (form[id] ?? []).map((v, i) => [`Form · Wk ${i + 1}`,
      String(v)])),
  }) as unknown as PlayerReading);
  return { read: vi.fn(async () => research), refresh: vi.fn(), news: vi.fn(), open: vi.fn(), player } as
    unknown as ResearchReader & { read: ReturnType<typeof vi.fn>; player: ReturnType<typeof vi.fn> };
}
// My roster in the assets fixture: player 1001 is mine, ends this season, and 1002 ends 2027 as an RFA.
const myAssets = { ...assets, league: LEAGUE, identities: [...assets.identities,
  { player: '1001', franchise: MINE, name: 'Player, 1001', position: 'LB' },
  { player: '1003', franchise: MINE, name: 'Player, 1003', position: 'WR' }],
  snapshot: { ...assets.snapshot, players: [...assets.snapshot.players,
    { id: '1001', franchise: MINE, status: 'ROSTER', salaryCents: 100, expirationYear: 2026, contractStatus:
      'UFA', contractInfo: '' },
    { id: '1003', franchise: MINE, status: 'ROSTER', salaryCents: 100, expirationYear: 2027, contractStatus:
      'RFA', contractInfo: '' }] } };
const lineups = pulse.lineups as Record<string, unknown>;
const full = (over: Record<string, unknown> = {}) => {
  const base = port(over);
  return vi.fn<HostedPort>(async request => {
    const f = 'franchise' in request.arguments ? String(request.arguments.franchise) : '';
    const extra: Record<string, unknown> = {
      'lineup.read': lineups[f] && { ...(lineups[f] as object), league: LEAGUE },
      'projections.read': { ...pulse.projections, league: LEAGUE, players:
        pulse.projections.players.filter(x =>
        String((request.arguments as { players?: string }).players ?? '').split(',').includes(x.player)) },
      'native.assets.read': myAssets,
      'commish.jobs.read': { ...jobs, league: LEAGUE },
      ...over,
    };
    if (extra[request.command] !== undefined)
      return { label: 'Fake local port', writable: false, football: '', data: extra[request.command] };
    return base(request);
  });
};

it('THIS WEEK projects each side from League Pulse\'s matchup read and counts locked starters', async () => {
  const p = full();
  await mount(p);
  const calls = p.mock.calls.map(([r]) => [r.command, r.arguments]);
  expect(calls).toContainEqual(['lineup.read', { league: LEAGUE, franchise: MINE, week: 2, at: '' }]);
  expect(calls).toContainEqual(['projections.read', expect.objectContaining({ league: LEAGUE, season: 2026,
    week: 2 })]);
  const mu = tile('mu').textContent!;
  expect(mu).toMatch(/proj \d+\.\d/);
  expect(mu).toContain('WIN % · Not available yet');
});

it('FORM · LAST 5 reads each of my starters through the research player read HQ\'s form chart uses',
  async () => {
  const r = reader({}, { '1001': [4, 8, 12, 16, 20, 24] });
  localStorage.setItem(layoutKey(LEAGUE, MINE), JSON.stringify({ mode: 'custom', over: [], pins: {},
    tiles: [['mu', '1x1'], ['news', '1x2'], ['wire', '1x1'], ['form', '1x1'], ['sc', '1x1']] }));
  await mount(full(), r);
  expect(r.player).toHaveBeenCalledWith('1001');
  const form = tile('form');
  expect(form.querySelector('.r.pw .v')!.textContent).toBe('16'); // last five: 8..24, mean 16
  expect(form.querySelectorAll('.form i')).toHaveLength(5);
});

it('NFL NEWS lists the research articles and filters to my players by this league\'s rosters', async () => {
  await mount(full(), reader());
  const news = tile('news');
  expect(news.querySelectorAll('.r.nw')).toHaveLength(2);
  await click(Array.from(news.querySelectorAll('.nfb button')).find(b => b.textContent === 'MY PLAYERS')!);
  expect(news.querySelectorAll('.r.nw')).toHaveLength(1);
  expect(news.textContent).toContain('Yours');
  await click(Array.from(news.querySelectorAll('.nfb button')).find(b => b.textContent === 'ON THE WIRE')!);
  expect(news.textContent).toContain('practice squad');
});

it('DEPTH CHARTS shows the research depth report, tagging names held in this league', async () => {
  await mount(full(), reader());
  await click(Array.from(container.querySelectorAll('[role=tab]')).find(t => t.textContent ===
    'DEPTH CHARTS')!);
  const rows = container.querySelectorAll('.wsx.on tbody tr');
  expect(rows).toHaveLength(2);
  expect(rows[0].className).toBe('me');
  expect(rows[0].textContent).toContain('Yours');
  expect(q('.tray .msg')!.textContent).toContain('nflverse');
});

it('CAP RUNWAY notes contracts ending each season from the league assets', async () => {
  await mount(full({ 'native.cap.check': undefined }));
  // cap.check has no fixture in the base port, so the tile is unavailable; the note logic is checked direct.
  const { capNote } = await import('./homeTiles');
  const { nativeHQAssets } = await import('../commissioner/hqReading');
  const b = { at: {}, errors: {}, assets: nativeHQAssets(myAssets, LEAGUE) };
  expect(capNote(b, MINE, 2026)).toBe('1 expiring');
  expect(capNote(b, MINE, 2027)).toBe('RFA: Player, 1003');
  expect(capNote(b, MINE, 2028)).toBe('');
});

it('ALERTS come from the commissioner jobs board: blocked jobs and live jobs with a deadline', async () => {
  const closes = new Date(Date.now() + 86_400_000).toISOString();
  const board = { ...jobs, league: LEAGUE, jobs: jobs.jobs.map(j => j.id === 'waivers' ? { ...j, closes } : j)
    };
  await mount(full({ 'commish.jobs.read': board }));
  const blocked = jobs.jobs.filter(j => j.state === 'blocked').length;
  expect(q('.aband .more')!.textContent).toBe(`+${blocked}ALL ALERTS ›`);
  expect(q('.aband .ak')!.textContent).toBe(`▲ ${jobs.jobs.find(j => j.state ===
    'blocked')!.name.toUpperCase()}`);
  expect(q('.aband')!.className).toContain('crit');
  const pull = Array.from(container.querySelectorAll('.pull')).find(b =>
    b.textContent?.startsWith('ALERTS'))!;
  expect(pull.textContent).toBe(`ALERTS ${blocked + 1}`);
  await click(pull);
  expect(q('.sdrawer.open')!.textContent).toContain('Waiver processing');
  expect(q('.sdrawer.open')!.textContent).not.toContain('League setup'); // live without a deadline
});

it('binds the shortcut keys, ignores disabled ones, and adds pinned tiles to MY SHORTCUTS', async () => {
  await mount(full());
  const spy = vi.spyOn(commands, 'dispatch');
  await act(async () => { document.dispatchEvent(new KeyboardEvent('keydown', { key: 'c' })); });
  expect(spy).toHaveBeenCalledWith('nav.open', { node: 'hq', workspace: 'contracts-and-cap' });
  spy.mockClear();
  await act(async () => { document.dispatchEvent(new KeyboardEvent('keydown', { key: 'p' })); });
  expect(spy).not.toHaveBeenCalled();
  const plans = Array.from(tile('sc').querySelectorAll('.scb')).find(b =>
    b.textContent?.startsWith('Plans'))!;
  expect((plans as HTMLButtonElement).disabled).toBe(true);
  await click(q('.queue'));
  await click(tile('race').querySelector('.pin'));
  await click(q('.queue'));
  const pinned = Array.from(tile('sc').querySelectorAll('.scb')).find(b =>
    b.textContent?.startsWith('RACE'))!;
  await click(pinned);
  expect(commands.history().at(-1)).toMatchObject({
    id: 'nav.open', args: { node: 'pulse', workspace: 'race' }, status: 'done' });
});

it('footers show the Central clock time each tile was read, and the tray says Home only reads', async () => {
  await mount(full());
  expect(tile('wire').querySelector('.tf span')!.textContent).toMatch(/^as of \d{2}:\d{2}$/);
  expect(tile('mu').querySelector('.tf span')!.textContent).toMatch(/^as of \d{2}:\d{2}:\d{2} · live$/);
  expect(q('.tray .msg')!.textContent).toBe('Home reads and links only · nothing here changes the league');
});

it('marks a tile loading until its reads answer, and stale once they are old', async () => {
  const { tileState, STALE_MS } = await import('./homeTiles');
  const now = Date.parse('2026-11-02T12:00:00Z');
  expect(tileState('wire', { at: {}, errors: {} }, now)).toBe('loading');
  expect(tileState('wire', { at: { ledger: new Date(now - 1000).toISOString() }, errors: {} },
    now)).toBe('fresh');
  expect(tileState('wire', { at: { ledger: new Date(now - STALE_MS - 1).toISOString() }, errors: {} },
    now)).toBe('stale');
});

it('hides rows that do not fit whole and says how many more live in the source screen', async () => {
  // jsdom lays nothing out: each tile body ends at 100px and every ledger row is 60px tall.
  const rect = (bottom: number) => ({ bottom, top: bottom - 60, left: 0, right: 0, width: 0, height: 60, x: 0,
    y: 0,
    toJSON: () => ({}) }) as DOMRect;
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    if (this.classList.contains('tb')) return rect(100);
    if (this.classList.contains('r')) return rect(60 * (Array.from(this.parentElement!.children).indexOf(this)
      + 1));
    return rect(0);
  });
  await mount(full());
  const rows = Array.from(tile('wire').querySelectorAll<HTMLElement>('.r.wire'));
  expect(rows.map(r => r.style.display)).toEqual(['', 'none']);
  expect(tile('wire').querySelector('.tf .more')!.textContent).toBe('+1 more in LEDGER');
});

// ---- APP-HOME-3: what the live walk on Claude-OS showed ----
it('THIS WEEK and the TOP 5 ticker name players through HQ\'s identity block, not the read\'s stand-in',
  async () => {
  const stand = { ...pulse.results, league: LEAGUE, scores: pulse.results.scores.map(s => ({ ...s,
    starters: s.starters.map(x => ({ ...x, name: `player ${x.player}` })) })) };
  await mount(full({ 'results.read': stand }));
  expect(tile('mu').textContent).toContain('Player, 1001');
  expect(tile('mu').textContent).not.toContain('player 1001');
  expect(container.querySelector('.tick')!.textContent).toContain('Player, 1001');
});

it('NFL NEWS second line is the held or researched name, never the directory-miss words', async () => {
  const miss = 'A player MFL’s directory does not list';
  const r = reader({ articles: [{ id: '9999', article: 'a2', name: miss, headline:
    'Signs a practice squad deal' },
    { id: '1001', article: 'a1', name: miss, headline: 'Player 1001 ruled out' }],
  players: [{ id: '9999', name: miss, position: 'Unknown', team: 'Unknown', franchiseId: '', owner: '' }] });
  await mount(full(), r);
  const news = tile('news');
  expect(news.textContent).not.toContain('directory does not list');
  expect(Array.from(news.querySelectorAll('.r.nw small')).map(x => x.textContent)).toEqual(['',
    'Player, 1001']);
});

it('RECORD, SEED and the RACE records say "—" while the standings have no decided game, not 0-0', async () =>
  {
  const none = { ...pulse.standings, league: LEAGUE, decided: 0, pending: 48,
    words: 'Standings through week 7: 0 games have a result, 48 do not yet',
    computed: pulse.standings.computed.map(r => ({ ...r, wins: 0, losses: 0, ties: 0, record: '0-0' })) };
  await mount(full({ 'standings.read': none }));
  expect(container.textContent).not.toContain('0-0');
  expect(tile('race').querySelector('.seedl')!.getAttribute('title')).toContain('0 games have a result');
  await act(async () => { root.unmount(); }); root = createRoot(container);
  await mount(full());
  expect(tile('race').textContent).toContain('1-2');
});


it('uses sibling shared buttons for tile selection, with no nested interactive controls', async () => {
  await mount(full());
  expect(container.querySelector('button button')).toBeNull();
  const select = tile('race').querySelector('button.tile-select')!;
  expect(select.classList.contains('act-surface')).toBe(true);
  expect(select.getAttribute('aria-label')).toBe('Select RACE');
  await click(select);
  expect(tile('race').classList.contains('sel')).toBe(true);
  expect(q('aside.insp')!.textContent).toContain('RACE');
  await click(q('.aband'));
  expect(q('.sdrawer.open')).not.toBeNull();
  await act(async () => { document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' })); });
  expect(q('.sdrawer.open')).toBeNull();
});
