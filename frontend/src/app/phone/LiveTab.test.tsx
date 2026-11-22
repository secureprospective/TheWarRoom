// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import { LiveTab } from './LiveTab';

const league: LeagueChoice = { id: 'league', name: 'Test League', app: true, franchise: 'mine',
  franchiseName: 'My Team', role: 'Owner' };
const state = { possession: '12', down: 2, distance: 7, yardLine: 57, toEndzone: 57 };
const play = { id: 'p1', sequence: 1, text: 'Mahomes pass short right', clock: '8:42', period: 2,
  at: '2026-11-15T18:00:00Z', homeScore: 7, awayScore: 3, start: state, end: state, participants: [] };
const nflGame = { season: 2026, week: 11, seasonType: 2, id: '401', kickoff: '2026-11-15T18:00:00Z',
  home: { id: '12', abbreviation: 'KC', score: 7, record: '8-2' },
  away: { id: '2', abbreviation: 'BUF', score: 3 },
  status: 'in', clock: '8:42', period: 2, state, last: play, plays: [play],
  box: [{ team: 'KC', category: 'passing', labels: ['C/ATT', 'YDS'], rows: [{ name: 'P. Mahomes',
    stats: ['6/10', '95'] }] }] };
const reply = (data: unknown) => ({ data, football: '', writable: false, label: 'Server' });
function fakePort(nfl: 'ok' | 'fail' = 'ok') {
  return vi.fn<HostedPort>(async request => {
    if (nfl === 'fail') throw new Error('scoreboard down');
    if (request.command === 'nfl.games.read') return reply({ league: 'league', board: { stale: false,
      source: 'ESPN', at: '2026-11-15T18:00:00Z', failure: '', games: [nflGame] } });
    if (request.command === 'nfl.game.read') return reply({ league: 'league', game: nflGame,
      at: '2026-11-15T18:00:00Z', stale: false, words: '' });
    throw new Error(`Unexpected read ${request.command}`);
  });
}
const starter = (player: string, name: string, position: string, provisional: number) =>
  ({ player, name, position, words: '', provisional });
const brief = (games: unknown[]): HomeBrief => ({ pulse: { league: 'league', name: 'Test League',
  season: 2026, week: 11, lastWeek: 18, weekNote: '', seats: [], fixtures: [], forms: {}, errors: [],
  odds: { games: [{ week: 11, home: 'mine', away: 'other', homeWin: 0.44 }], rows: [] } },
results: { league: 'league', season: 2026, week: 11, revision: 1, published: true, words: '',
  games, missed: [], scores: [
    { franchise: 'mine', name: 'Iron Curtain', status: '', provisional: 24.2, words: '', adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: false,
      starters: [starter('m1', 'Mahomes, Patrick', 'QB', 18.1)] },
    { franchise: 'other', name: 'Gridlock', status: '', provisional: 25.8, words: '', adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: false,
      starters: [starter('o1', 'Allen, Josh', 'QB', 20.4)] }] },
matchup: { lineups: { mine: [{ player: 'm1', team: 'KC', locked: true }],
  other: [{ player: 'o1', team: 'BUF', locked: true }] }, projections: {}, errors: [] },
at: {}, errors: {} } as unknown as HomeBrief);
const myGameRow = { week: 11, home: 'mine', away: 'other', homeName: 'Iron Curtain', awayName: 'Gridlock',
  status: 'in', homeProvisional: 24.2, awayProvisional: 25.8, words: '', why: [] };

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  vi.useRealTimers();
  vi.restoreAllMocks();
});
async function show(readBrief: () => Promise<HomeBrief>, port: HostedPort = fakePort()) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  HTMLDialogElement.prototype.showModal ??= function (this: HTMLDialogElement) { this.open = true; };
  HTMLDialogElement.prototype.close ??= function (this: HTMLDialogElement) { this.open = false; };
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<LiveTab league={league} port={port} readBrief={readBrief}
    subscribe={() => () => {}} />));
  await act(async () => { await new Promise(r => setTimeout(r, 0)); });
  return host;
}
const tap = async (label: string) => {
  const key = [...host!.querySelectorAll<HTMLElement>('button')]
    .find(b => b.getAttribute('aria-label') === label || b.textContent === label);
  if (!key) throw new Error(`No control ${label}`);
  await act(async () => key.click());
};
const region = (name: string) =>
  host!.querySelector(`:is(section,nav)[aria-label="${name}"]`)?.textContent ?? '';

describe('LiveTab', () => {
  it('lays out the mock Live page from the held reads when loaded', async () => {
    await show(async () => brief([myGameRow]));
    expect(region('My matchup')).toContain('My matchup · wk 11');
    expect(region('My matchup')).toContain('Iron Curtain · YOU');
    expect(region('My matchup')).toContain('44% WIN');
    expect(region('My matchup')).not.toContain('Mahomes');
    expect(region('NFL games')).toContain('BUF3KC7Q2 8:42');
    expect(region('League matchups')).toBe('');
    expect(region('NFL live')).toContain('2ND & 7');
    expect(region('NFL live')).toContain('KC 43');
    expect(region('NFL live')).toContain('57 YD');
    expect(host!.querySelector('.live6__fresh')?.textContent).toContain('updated');
    expect(region('NFL box score')).toContain('P. Mahomes');
    expect(region('Play by play')).toContain('Mahomes pass short right');
    await tap('Matchup box score');
    expect(region('Matchup box score')).toContain('Patrick Mahomes');
    expect(region('Matchup box score')).toContain('Josh Allen');
  });
  it('opens and closes the all-matchups sheet', async () => {
    await show(async () => brief([myGameRow]));
    await tap('All matchups');
    const sheet = host!.querySelector('dialog');
    expect(sheet?.textContent).toContain('Week 11 matchups');
    expect(sheet?.textContent).toContain('YOUR MATCHUP');
    await tap('Close Week 11 matchups');
    expect(host!.querySelector('dialog')).toBeNull();
  });
  it('follows the matchup picked in All matchups and returns to mine from the header', async () => {
    Element.prototype.scrollIntoView ??= () => {};
    const otherRow = { week: 11, home: 'h2', away: 'a2', homeName: 'Hammers', awayName: 'Arrows',
      status: 'in', homeProvisional: 40.5, awayProvisional: 31.25, words: '', why: [] };
    const b = brief([myGameRow, otherRow]);
    b.pulse!.odds!.games.push({ week: 11, home: 'h2', away: 'a2', homeName: 'Hammers',
      awayName: 'Arrows', homeWin: 0.7 });
    await show(async () => b);
    const score = () => host!.querySelector('[aria-label="My matchup"] .mys')?.textContent ?? '';
    expect(score()).toContain('YOU');
    await tap('All matchups');
    await tap('Arrows 31.3, Hammers 40.5');
    expect(score()).toContain('Arrows');
    expect(score()).toContain('Hammers');
    expect(score()).toContain('30% WIN');
    expect(score()).not.toContain('YOU');
    await tap('Show my matchup');
    expect(score()).toContain('Iron Curtain · YOU');
    expect(score()).toContain('44% WIN');
  });
  it('says there is no game when the week holds none for me', async () => {
    await show(async () => brief([]));
    expect(region('My matchup')).toContain('No game this week');
    expect(region('My matchup')).not.toContain('WIN');
  });
  it('shows each read\'s own failure words when the reads fail', async () => {
    await show(async () => { throw new Error('results.read refused'); }, fakePort('fail'));
    expect(host!.querySelector('[aria-label="My matchup"] [role="alert"]')?.textContent)
      .toContain('results.read refused');
    expect(region('League matchups')).toBe('');
  });
  it('keeps the matchup header as one row: title, then both controls', async () => {
    await show(async () => brief([myGameRow]));
    const row = host!.querySelector<HTMLElement>('[aria-label="My matchup"] .live6__head')!;
    expect(row.children[0].getAttribute('aria-label')).toBe('Show my matchup');
    expect(row.querySelectorAll(':scope > span > .live6__link')).toHaveLength(2);
  });
  it('shows how old the held scores are, refreshed every 30 seconds', async () => {
    vi.useFakeTimers({ toNotFake: ['setTimeout', 'queueMicrotask', 'nextTick'] });
    vi.setSystemTime(new Date('2026-11-15T18:00:00Z'));
    await show(async () => brief([myGameRow]));
    const fresh = () => region('NFL live');
    expect(fresh()).toContain('Updated just now');
    await act(async () => { vi.advanceTimersByTime(3 * 60_000); });
    expect(fresh()).toContain('Updated 3 min ago');
  });
  it('draws the field graphic only when the field position is known', async () => {
    await show(async () => brief([myGameRow]));
    expect(host!.querySelector('[aria-label="NFL live"] .field:not([hidden])')).not.toBeNull();
  });
  it('shows no field graphic when the game holds no field position', async () => {
    const noPosition = { ...nflGame, state: null };
    const port = vi.fn<HostedPort>(async request => request.command === 'nfl.games.read'
      ? reply({ league: 'league',
        board: { stale: false, source: 'ESPN', at: '', failure: '', games: [noPosition] } })
      : reply({ league: 'league', game: noPosition, at: '', stale: false, words: '' }));
    await show(async () => brief([myGameRow]), port);
    expect(host!.querySelector('[aria-label="NFL live"] .field:not([hidden])')).toBeNull();
    expect(region('NFL live')).toContain('Field position not supplied');
  });
  it('shows the mock score header, drive bar and last play from the followed game', async () => {
    await show(async () => brief([myGameRow]));
    const live = host!.querySelector('[aria-label="NFL live"]')!;
    expect(live.querySelector('.tm-R .tm-ab')?.textContent).toBe('KC');
    expect(live.querySelector('[aria-label="Score"]')?.getAttribute('data-poss')).toBe('R');
    const bar = host!.querySelector<HTMLElement>('[data-testid="live-drive-bar"]')!;
    expect(bar.dataset.dir).toBe('l');
    expect(bar.querySelector<HTMLElement>('.db-ball')!.style.left).toBe(`${(100 / 120) * 67}%`);
    expect(bar.querySelector<HTMLElement>('.db-first')!.style.left).toBe(`${(100 / 120) * 60}%`);
    expect(bar.textContent).toBe('BUF205020KC');
    expect(region('Latest play')).toContain('Q2 8:42');
    expect(region('Latest play')).toContain('Mahomes pass short right');
  });
  it('renders the matchup scores and win-chance bar with no starter tiles', async () => {
    const richBrief = brief([myGameRow]);
    richBrief.pulse!.seats = [
      { franchise: 'mine', name: 'Iron Curtain', abbreviation: 'IRN', division: '', owner: 'me' },
      { franchise: 'other', name: 'Gridlock', abbreviation: 'GRD', division: '', owner: 'opp' },
    ];
    richBrief.results!.scores[0].starters = [
      starter('m1', 'Mahomes, Patrick', 'QB', 18.1),
      starter('m2', 'Bass, Tyler', 'K', 2.5),
      starter('m3', 'Allen, Josh', 'QB', 5.3),
    ];
    richBrief.matchup!.lineups.mine = [
      { player: 'm1', team: 'KC', locked: true },
      { player: 'm2', team: 'KC', locked: true },
      { player: 'm3', team: 'BUF', locked: true },
    ];
    await show(async () => richBrief);
    const mym = host!.querySelector('[aria-label="My matchup"]')!;
    expect(mym.classList.contains('mym')).toBe(true);
    expect(mym.querySelector('.mys .t.me')?.textContent).toContain('IRN · YOU');
    expect(mym.querySelector('.mys .t.r')?.textContent).toContain('GRD');
    expect(mym.querySelector<HTMLElement>('.wp-bar i')?.style.width).toBe('44%');
    expect(mym.querySelector('.myp, .live6__pl')).toBeNull();
  });
  it('animates held score and win-chance updates between reads, snapping under reduced motion', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    vi.useFakeTimers({ toNotFake: ['queueMicrotask', 'nextTick'] });
    let notify: (() => void) | undefined;
    let current = brief([myGameRow]);
    const read = async () => current;
    const port = fakePort();
    const subscribe = (load: () => void) => { notify = load; return () => {}; };
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    await act(async () => root!.render(<LiveTab league={league} port={port} readBrief={read}
      subscribe={subscribe} reduced={false} />));
    await act(async () => { vi.advanceTimersByTime(10); });

    const mym = () => host!.querySelector('[aria-label="My matchup"]')!;
    expect(mym().querySelector('.mys .t.me b')?.textContent).toBe('24.2');
    expect(mym().querySelector<HTMLElement>('.wp-bar i')?.style.width).toBe('44%');

    const updated = brief([{ ...myGameRow, homeProvisional: 34.5, awayProvisional: 26.8 }]);
    updated.pulse!.odds!.games[0].homeWin = 0.74;
    current = updated;
    await act(async () => { notify?.(); });
    await act(async () => { await Promise.resolve(); });

    const midScore = Number(mym().querySelector('.mys .t.me b')?.textContent);
    expect(midScore).toBeGreaterThan(24.2);
    expect(mym().querySelector('.mys .t.me b')?.classList.contains('chg')).toBe(true);
    expect(mym().querySelector<HTMLElement>('.wp-bar i')?.style.width).toBe('74%');

    await act(async () => { vi.advanceTimersByTime(600); });
    expect(mym().querySelector('.mys .t.me b')?.textContent).toBe('34.5');
    expect(mym().querySelector('.wp-l b')?.textContent).toBe('74% WIN');

    // Reduced motion snaps instantly with .rm and no .chg highlight.
    await act(async () => root!.render(<LiveTab league={league} port={port} readBrief={read}
      subscribe={subscribe} reduced={true} />));
    const third = brief([{ ...myGameRow, homeProvisional: 38.9, awayProvisional: 26.8 }]);
    third.pulse!.odds!.games[0].homeWin = 0.82;
    current = third;
    await act(async () => { notify?.(); });
    await act(async () => { await Promise.resolve(); });
    expect(host!.querySelector('.live6.rm')).not.toBeNull();
    expect(mym().querySelector('.mys .t.me b')?.textContent).toBe('38.9');
    expect(mym().querySelector('.mys .t.me b')?.classList.contains('chg')).toBe(false);
    expect(mym().querySelector('.wp-l b')?.textContent).toBe('82% WIN');
  });
  it('does not refetch another matchup on 30s freshness ticks or tab switches', async () => {
    vi.useFakeTimers({ toNotFake: ['setTimeout', 'queueMicrotask', 'nextTick'] });
    vi.setSystemTime(new Date('2026-11-15T18:00:00Z'));
    const otherGameRow = { week: 11, home: 't3', away: 't4', homeName: 'North Stars', awayName: 'Bayou',
      status: 'in', homeProvisional: 19.0, awayProvisional: 14.5, words: '', why: [] };
    const port = vi.fn<HostedPort>(async request => {
      if (request.command === 'nfl.games.read') return reply({ league: 'league', board: { stale: false,
        source: 'ESPN', at: '2026-11-15T18:00:00Z', failure: '', games: [nflGame] } });
      if (request.command === 'nfl.game.read') return reply({ league: 'league', game: nflGame,
        at: '2026-11-15T18:00:00Z', stale: false, words: '' });
      if (request.command === 'lineup.read') return reply({ league: 'league',
        franchise: request.arguments?.franchise, week: 11, starters: [], bench: [] });
      if (request.command === 'projections.read') return reply({ league: 'league', season: 2026,
        week: 11, projections: [] });
      throw new Error(`Unexpected read ${request.command}`);
    });
    const lineupReads = () => port.mock.calls.filter(([req]) => req.command === 'lineup.read').length;

    await show(async () => brief([myGameRow, otherGameRow]), port);
    await tap('All matchups');
    await tap('Bayou 14.5, North Stars 19.0');
    await act(async () => { await new Promise(r => setTimeout(r, 0)); });
    const afterPick = lineupReads();
    expect(afterPick).toBe(2);

    await act(async () => { vi.advanceTimersByTime(3 * 60_000); });
    await tap('NFL box score');
    await tap('Matchup box score');
    await act(async () => { await new Promise(r => setTimeout(r, 0)); });
    expect(lineupReads()).toBe(afterPick);
  });

  it('shows the Broadcast pre-game preview for pre games and keeps LiveField for live and final games',
    async () => {
    vi.useFakeTimers({ toNotFake: ['setTimeout', 'queueMicrotask', 'nextTick'] });
    vi.setSystemTime(new Date('2026-11-22T16:00:00Z'));
    const preGame = {
      season: 2026, week: 11, seasonType: 2, id: '405', kickoff: '2026-11-22T18:00:00Z',
      home: { id: '2', abbreviation: 'BUF', score: 0, record: '8-2' },
      away: { id: '12', abbreviation: 'KC', score: 0, record: '9-1' },
      status: 'pre', clock: '', period: 0, state: null, last: null, plays: [], box: [],
      predictor: { away: 45, home: 55 },
      venue: { name: 'Highmark Stadium', city: 'Orchard Park', state: 'NY', indoor: false },
      weather: { text: 'Mostly Cloudy', tempF: 66, wind: '7 mph S', precipPct: 0, source: 'NWS' },
    };
    const finalGame = { ...nflGame, id: '406', status: 'post', clock: '0:00', period: 4 };
    const port = vi.fn<HostedPort>(async request => {
      if (request.command === 'nfl.games.read') return reply({ league: 'league', board: { stale: false,
        source: 'ESPN', at: '2026-11-22T16:00:00Z', failure: '', games: [nflGame, preGame, finalGame] } });
      if (request.command === 'nfl.game.read') {
        const id = (request.arguments as { game: string }).game;
        const found = [nflGame, preGame, finalGame].find(g => g.id === id) ?? null;
        return reply({ league: 'league', game: found, at: '2026-11-22T16:00:00Z',
          stale: false, words: '' });
      }
      throw new Error(`Unexpected read ${request.command}`);
    });

    await show(async () => brief([myGameRow]), port);
    // Live game: LiveField shows, preview never shows.
    expect(host!.querySelector('[data-testid="live-field"]')).not.toBeNull();
    expect(host!.querySelector('[data-testid="live-pregame"]')).toBeNull();

    // Switch to the pre-game: Broadcast preview shows with both matchup sides' players in the crawl.
    await tap('Follow KC at BUF');
    await act(async () => { await new Promise(r => setTimeout(r, 0)); });
    expect(host!.querySelector('[data-testid="live-pregame"]')).not.toBeNull();
    expect(host!.querySelector('[data-testid="live-field"]')).toBeNull();
    expect(host!.querySelector('[data-testid="pregame-winbar"]')?.textContent).toContain('KC 45%');
    expect(host!.querySelector('[data-testid="pregame-crawl"]')?.textContent).toContain('Patrick Mahomes');
    expect(host!.querySelector('[data-testid="pregame-crawl"]')?.textContent).toContain('Josh Allen');

    // Final game: preview never shows; LiveField shows as today.
    const buttons = [...host!.querySelectorAll<HTMLElement>('button')]
      .filter(b => b.getAttribute('aria-label') === 'Follow BUF at KC');
    await act(async () => buttons[buttons.length - 1].click());
    await act(async () => { await new Promise(r => setTimeout(r, 0)); });
    expect(host!.querySelector('[data-testid="live-pregame"]')).toBeNull();
    expect(host!.querySelector('[data-testid="live-field"]')).not.toBeNull();
  });
});
