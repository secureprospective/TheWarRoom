// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HomeBrief from './HomeBrief';
import { commands } from '../commands/registry';
import type { HostedPort } from '../commissioner/hostedRoom';
import { layoutKey } from './homeLayout';
import pulse from './pulse.wire.testdata.json';

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
async function mount(p: HostedPort) {
  await act(async () => { root.render(<HomeBrief league={LEAGUE} mine={MINE} port={p} />); });
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
  const spy = vi.spyOn(commands, 'dispatch');
  await click(tile('mu').querySelector('.lnk'));
  expect(spy).toHaveBeenCalledWith('nav.open', { node: 'pulse', workspace: 'now' });
  const cap = Array.from(tile('sc').querySelectorAll('.scb')).find(b => b.textContent?.startsWith('Cap'))!;
  await click(cap);
  expect(spy).toHaveBeenCalledWith('nav.open', { node: 'hq', workspace: 'contracts-and-cap' });
});

it('customize moves a tile to overflow, saves it for this owner and league, and restores it', async () => {
  await mount(port());
  await click(q('.queue'));
  await click(tile('race').querySelector('.off'));
  expect(tile('race')).toBeNull();
  const saved = JSON.parse(localStorage.getItem(layoutKey(LEAGUE, MINE))!);
  expect(saved).toMatchObject({ mode: 'custom', over: ['race'] });
  await click(Array.from(container.querySelectorAll('.pull')).find(b => b.textContent?.startsWith('OVERFLOW'))!);
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
