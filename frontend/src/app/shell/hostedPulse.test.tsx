// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HostedLeague from '../commissioner/HostedLeague';
import NativeLeaguePulse, { PULSE_FOLLOW_MS, useHostedGames } from './NativeLeaguePulse';
import { readNativePulse, readPulseMatchup } from './pulseReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import * as nfl from '../now/api';
import * as hq from '../commissioner/hqReading';
import { heldSnapshot } from '../data/lineupTestData';
import { hqEvents } from './hqReads';
import * as hostedReads from '../data/hostedReads';
import type { ReadTopic } from '../data/session';
import { parseHostedGames } from '../now/liveContract';

const common = { league: 'test', season: 2026, revision: 1, name: 'Test League', words: 'Published' };
const teams = [{ franchise: 'a', name: 'Alpha', abbreviation: 'ALP' },
  { franchise: 'b', name: 'Beta', abbreviation: 'BET' }];
const game = { week: 1, home: 'a', away: 'b', homeName: 'Alpha', awayName: 'Beta',
  status: 'No result yet', words: 'Alpha vs Beta', why: [] };
const nflGame = { season: 2026, week: 1, seasonType: 2, id: '401', kickoff: '2026-11-15T18:00:00Z',
  home: { id: '12', abbreviation: 'KC', score: 14 }, away: { id: '2', abbreviation: 'BUF', score: 10 },
  status: 'in', clock: '5:12', period: 2, state: null, last: null, box: [], plays: [] };
const board = (stale = false) => ({ league: 'test', board: { stale, source: 'ESPN',
  at: '2026-11-15T18:00:00Z', failure: '', games: [nflGame] } });
function fakePort(scheduleMode: 'default' | 'empty' | 'unavailable' = 'default',
  weekMode: 'default' | 'no-current' | 'no-dates' = 'default') {
  return vi.fn<HostedPort>(async request => {
    const args = request.arguments;
    let data: unknown;
    switch (request.command) {
      case 'native.directory.read':
        data = { league: 'test', name: 'Test League', revision: 1, rosterSeason: 2026,
          franchises: teams }; break;
      case 'week.read':
        data = weekMode === 'no-current'
          ? { schedule: { league: 'test', season: 2026, revision: 1, franchise: 'a',
            message: 'The schedule is not published yet. A commissioner must choose Generate schedule, '
              + 'then Publish schedule, in Control Room > Season.', games: [] }, current: null,
            message: 'The schedule is not published yet. A commissioner must choose Generate schedule, '
              + 'then Publish schedule, in Control Room > Season.' }
          : weekMode === 'no-dates'
            ? { schedule: { league: 'test', season: 2026, revision: 1, franchise: 'a', games: [] },
              current: null,
              message: 'The current NFL week cannot be identified from this server\'s held schedule data. '
                + 'Ask the server operator to update the NFL schedule; changing the league matchup schedule '
                + 'will not fix Pulse.' }
            : { schedule: { league: 'test', season: 2026, revision: 1, franchise: 'a',
              games: [{ week: 1, venue: 'home' }] },
              current: { week: 1 }, message: '' };
        break;
      case 'standings.read':
        data = { ...common, status: 'Ready', through: 0, decided: 0, pending: 1, criteria: [], noData: [],
          computed: teams.map((t, i) => ({ franchise: t.franchise, name: t.name, rank: i + 1,
            wins: 0, losses: 0, ties: 0,
            record: '0–0', pointsFor: 0, pointsWords: 'No final points', why: [] })) }; break;
      case 'schedule.franchise.read':
        if (scheduleMode === 'unavailable') throw new Error('schedule source offline');
        data = { league: 'test', franchise: 'a', season: 2026, revision: 1,
          message: scheduleMode === 'empty' ? 'The schedule is not published yet.' : '',
          games: scheduleMode === 'empty' ? [] : [
            { week: 1, game: 'g1', opponent: 'b', venue: 'Home' },
            { week: 2, game: 'g2', opponent: 'b', venue: 'Away' },
          ] }; break;
      case 'bracket.read':
        data = { ...common, status: 'Projected', started: false, fields: [], projected: [],
          rules: { startWeek: 14, finalWeek: 17, perConference: 2, byeSeeds: 0, shape: [] } }; break;
      case 'odds.read':
        data = { ...common, status: 'Waiting', powerWords: '', gameWords: '', oddsWords: '',
          week: 1, runs: 0, seed: 0, rows: [], games: [] }; break;
      case 'results.read':
        data = { league: 'test', season: 2026, revision: 1, words: 'Published', week: 1,
          published: true, games: [game], missed: [],
          scores: teams.map((t, i) => ({ franchise: t.franchise, name: t.name, status: 'Provisional',
            provisional: (i + 1) * 100000,
            words: 'Provisional', adjustment: 0,
            lineup: { status: 'Valid', words: 'Valid', flagged: false },
            starters: [{ player: `000${i + 1}`, name: `Starter ${i + 1}`, position: 'QB',
              words: 'Provisional', provisional: (i + 1) * 100000 }] })) }; break;
      case 'schedule.read':
        data = { league: 'test', season: 2026, revision: 1,
          weeks: [{ week: 1, games: [{ home: 'a', away: 'b' }] }] }; break;
      case 'lineup.read':
        data = { ...common, franchise: 'franchise' in args ? args.franchise : '', week: 1,
          starters: [{ player: '0001', team: 'DAL', locked: true }] }; break;
      case 'nfl.games.read': data = board(); break;
      case 'league.activity.read': data = { league: 'test', revision: 2, events: [{ id: 'notice.1',
        kind: 'notice', when: '2026-11-18T12:00:00Z', summary: 'Public league notice' }] }; break;
      case 'nfl.game.read': data = { league: 'test', game: nflGame,
        at: '2026-11-15T18:00:00Z', stale: false, words: '' }; break;
      case 'projections.read': data = { league: 'test', season: 2026, week: 1, players: [] }; break;
      default: throw new Error(`Unexpected read ${request.command}`);
    }
    return { data, football: '', writable: false, label: 'Server' };
  });
}
let root: Root;
let host: HTMLDivElement;
let local: ReturnType<typeof vi.spyOn>;
let events: ReturnType<typeof vi.spyOn>;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  local = vi.spyOn(nfl, 'targetLiveGame').mockRejectedValue(new Error('Local binding called'));
  events = vi.spyOn(hqEvents, 'onChange');
});
afterEach(async () => {
  await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks();
  hostedReads.hostedScreens.forget();
});
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
async function click(label: string) {
  const button = [...document.querySelectorAll('button')].find(b => b.textContent === label);
  expect(button, label).toBeDefined();
  await act(async () => button!.click()); await settle();
}
async function openNode(label: string) {
  const button = [...document.querySelectorAll('.thin-league-shell .node>.nb')]
    .find(node => node.textContent?.includes(label));
  expect(button, label).toBeDefined();
  await act(async () => (button as HTMLButtonElement).click()); await settle();
}
async function openLeaf(label: string) {
  const button = [...document.querySelectorAll('.thin-league-shell .node.open>.leaf')]
    .find(node => node.textContent?.includes(label));
  expect(button, label).toBeDefined();
  await act(async () => (button as HTMLButtonElement).click()); await settle();
}
const count = (port: ReturnType<typeof fakePort>, command: string) =>
  port.mock.calls.filter(([r]) => r.command === command).length;

it('opens hosted Now, refreshes active reads except odds, shows Race and returns', async () => {
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  expect(host.textContent).toContain('Starter 2');
  expect(host.textContent).toContain('20.0');
  expect(host.textContent).not.toContain('coming next');
  expect(host.textContent).toContain('NFL live');
  expect(host.textContent).toContain('Scores follow the league server as it publishes them');
  expect(host.textContent).not.toContain('unavailable');
  expect(count(port, 'lineup.read')).toBe(1);
  const commands = ['native.directory.read', 'week.read', 'schedule.franchise.read', 'standings.read',
    'bracket.read', 'odds.read', 'results.read', 'lineup.read', 'projections.read', 'nfl.games.read',
    'nfl.game.read'];
  const before = commands.map(c => count(port, c));
  await click('Refresh');
  expect(commands.map(c => count(port, c)))
    .toEqual(before.map((n, i) => n + (commands[i] === 'odds.read' ? 0 : 1)));
  for (const [r] of port.mock.calls) if (r.command === 'week.read') expect(r.arguments.at).toBe('');
  await click('NFL GAMES');
  expect(host.textContent).not.toContain('Reading the NFL scoreboard');
  expect(host.textContent).toContain('BUF');
  expect(host.textContent).toContain('KC');
  expect(host.querySelector('[aria-label="Follow BUF at KC"]')).not.toBeNull();
  expect(host.querySelector('.field')).not.toBeNull();
  await openLeaf('Race');
  expect(host.textContent).toContain('STANDINGS');
  expect(host.textContent).toContain('MY ROAD · WKS 1–2');
  expect(host.textContent).toContain('Beta');
  expect(host.textContent).toContain('0–0');
  const team = host.querySelector('button[aria-label="Open Beta franchise card"]');
  expect(team).not.toBeNull();
  await act(async () => { (team as HTMLButtonElement).click(); });
  expect(host.querySelector('.insp[aria-label="Franchise"] .nm')?.textContent).toBe('Beta');
  await openNode('Home');
  expect(host.textContent).toContain('League Pulse');
  expect(host.querySelector('.lp-v14')).toBeNull();
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});

it('opens the League phone entry on the standings table', async () => {
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} entry="league" />));
  await settle();
  expect(host.textContent).toContain('STANDINGS');
  expect(host.textContent).toContain('MY ROAD · WKS 1–2');
  expect(host.querySelector('.lp-v14')).not.toBeNull();
  expect([...host.querySelectorAll('.pulse-phone-tabs button')].map(button => button.textContent))
    .toEqual(['Table', 'Schedule', 'Activity', 'Live']);
  expect(count(port, 'league.activity.read')).toBe(0);
});

it('shows server-projected public events and schedule on the League phone screen',
  async () => {
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} entry="league" />));
  await settle();
  await click('Schedule');
  expect(host.textContent).toContain('LEAGUE SCHEDULE');
  expect(host.textContent).toContain('Beta');
  expect(count(port, 'schedule.read')).toBe(1);
  await click('MY TEAM');
  expect(host.textContent).toContain('OPPONENT');
  expect(host.textContent).toContain('Beta');
  await click('Activity');
  expect(host.textContent).toContain('Public league notice');
  expect(host.textContent).toContain('Private bids and commissioner details are excluded.');
  expect(count(port, 'league.activity.read')).toBe(1);
  await click('Player status');
  expect(host.textContent).toContain('Player status is unavailable');
  expect(count(port, 'league.activity.read')).toBe(1);
  expect(port.mock.calls.some(([request]) => String(request.command) === 'player.status.read')).toBe(false);
});

it('keeps league activity visibly unavailable when its member-safe read fails', async () => {
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  const fallback = fakePort();
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'league.activity.read') throw new Error('activity source offline');
    return fallback(request);
  });
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} entry="league" />));
  await settle();
  await click('Activity');
  expect(host.textContent).toContain('League activity is unavailable · activity source offline');
  expect(host.textContent).not.toContain('Public league notice');
});

it('shows the commissioner action when Pulse has no published schedule week', async () => {
  const port = fakePort('default', 'no-current');
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  const action = 'A commissioner must choose Generate schedule, then Publish schedule, '
    + 'in Control Room > Season.';
  expect(host.textContent).toContain(action);
  expect(host.textContent).not.toContain('The league has not said which week it is playing.');
  expect(count(port, 'results.read')).toBe(0);
});

it('shows the server-operator action when held NFL dates cannot identify Pulse week', async () => {
  const port = fakePort('default', 'no-dates');
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  const action = 'The current NFL week cannot be identified from this server\'s held schedule data. '
    + 'Ask the server operator to update the NFL schedule; changing the league matchup schedule '
    + 'will not fix Pulse.';
  expect(host.textContent).toContain(action);
  expect(host.textContent).not.toContain('The league has not said which week it is playing.');
  expect(count(port, 'results.read')).toBe(0);
});

it('routes the rail Home item to its parent instead of rendering a second league Home', async () => {
  const port = fakePort();
  const onHome = vi.fn();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} onHome={onHome} />));
  await openNode('League Pulse');
  await openNode('Home');
  expect(onHome).toHaveBeenCalledOnce();
  expect(host.querySelector('.node.open .nb')?.textContent).toContain('League Pulse');
});

it.each(['Commissioner', 'Co-commissioner'])('reads both lineups for %s', async role => {
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role={role} port={port} />));
  await openNode('League Pulse');
  expect(count(port, 'lineup.read')).toBe(2);
  expect(host.textContent).toContain('Starter 2');
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});

it.each([
  ['empty', 'The schedule is not published yet.'],
  ['unavailable', 'The schedule is unavailable · schedule source offline'],
] as const)('shows the schedule read’s own %s words in My Road', async (mode, words) => {
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  const port = fakePort(mode);
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  await openLeaf('Race');
  expect(host.textContent).toContain(words);
  expect(count(port, 'schedule.franchise.read')).toBe(1);
  expect(local).not.toHaveBeenCalled();
});

it('derives every franchise form from the same bounded results.read weeks', async () => {
  const fallback = fakePort();
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'standings.read') return { data: { ...common, status: 'Ready', through: 2,
      decided: 2, pending: 0, criteria: [], noData: [], computed: teams.map((t, i) => ({
        franchise: t.franchise, name: t.name, rank: i + 1, wins: 1, losses: 1, ties: 0,
        record: '1–1', pointsFor: 0, pointsWords: '', why: [],
      })) }, football: '', writable: false, label: 'Server' };
    if (request.command === 'results.read') {
      const week = Number(request.arguments.week);
      return { data: { league: 'test', season: 2026, revision: 1, week, published: true,
        words: `Week ${week}`, games: [{ week, home: 'a', away: 'b', homeName: 'Alpha', awayName: 'Beta',
          status: 'Final', winner: week === 1 ? 'a' : 'b', words: '', why: [] }], missed: [], scores: [] },
        football: '', writable: false, label: 'Server' };
    }
    return fallback(request);
  });
  const reading = await readNativePulse('test', 'a', port, undefined, true, false);
  expect(reading.forms.a).toEqual([{ week: 1, result: 'W' }, { week: 2, result: 'L' }]);
  expect(reading.forms.b).toEqual([{ week: 1, result: 'L' }, { week: 2, result: 'W' }]);
  expect(port.mock.calls.flatMap(([r]) => r.command === 'results.read' ? [r.arguments.week] : []))
    .toEqual([1, 2]);
});

it('uses server time only when explicitly requested; local retains an ISO timestamp', async () => {
  const port = fakePort();
  await readNativePulse('test', 'a', port, undefined, true);
  await readNativePulse('test', 'a', port);
  const weeks = port.mock.calls.map(([r]) => r).filter(r => r.command === 'week.read');
  expect(weeks[0].arguments).toMatchObject({ at: '' });
  expect(weeks[1].arguments).toMatchObject({ at: expect.stringMatching(/^\d{4}-\d\d-\d\dT.*Z$/) });
});

it('deduplicates allowed lineups, excludes opponents and still projects public starters', async () => {
  const port = fakePort();
  const reading = await readPulseMatchup('test', 2026, 1, ['a', 'b', 'a'],
    { a: ['0001'], b: ['0002'] }, port, undefined, { mine: 'a', commissioner: false });
  expect(reading.errors).toEqual([]);
  expect(Object.keys(reading.lineups)).toEqual(['a']);
  expect(count(port, 'lineup.read')).toBe(1);
  const projection = port.mock.calls.map(([r]) => r).find(r => r.command === 'projections.read');
  expect(projection?.arguments).toMatchObject({ players: '0001,0002' });
});

it('does not call a local binding or subscribe while hosted Pulse is unavailable', async () => {
  const port = vi.fn<HostedPort>().mockRejectedValue(new Error('Server offline'));
  await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
    port={port} leave={() => {}} />));
  await settle();
  expect(host.textContent).toContain('League Pulse is unavailable');
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});

it('retries a failed hosted directory read after five seconds without Refresh', async () => {
  vi.useFakeTimers();
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  const fallback = fakePort();
  let failed = false;
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'native.directory.read' && !failed) {
      failed = true;
      throw new Error('Server offline');
    }
    return fallback(request);
  });
  try {
    await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
      port={port} leave={() => {}} />));
    expect(host.textContent).toContain('League Pulse is unavailable');
    expect(host.textContent).toContain('retrying');
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(host.textContent).not.toContain('League Pulse is unavailable');
    expect(host.textContent).toContain('Starter 2');
    expect(host.textContent).not.toContain('retrying');
  } finally { vi.useRealTimers(); }
});

it('keeps opponent NFL team and lock absent while retaining public starter points', async () => {
  const port = fakePort();
  await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
    port={port} leave={() => {}} />));
  await settle();
  const starters = [...host.querySelectorAll('.mr')];
  const opponent = starters.find(row => row.textContent?.includes('Starter 2'));
  expect(opponent?.textContent).toContain('20.0');
  expect(opponent?.querySelector('.p.r small')).toBeNull();
  expect(opponent?.querySelector('.chipc.r')?.textContent).toBe('');
  expect(host.textContent).not.toContain('private');
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});


it('uses the thin rail to move from HQ to Pulse and back', async () => {
  const port = fakePort();
  vi.spyOn(hq, 'readNativeHQ').mockResolvedValue({ revision: 1, snapshot: heldSnapshot(),
    searchPlayers: [], settings: { positions: [] }, errors: [] });
  vi.spyOn(hqEvents, 'onChange').mockImplementation(() => () => {});
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} hq />));
  await settle();
  for (let i = 0; i < 20 && !document.querySelector('.thin-league-shell .rail.mock'); i++) await settle();
  await openNode('League Pulse');
  expect(host.textContent).toContain('Starter 2');
  await openNode('Franchise HQ');
  for (let i = 0; i < 20 && !host.querySelector('.hq-v10'); i++) await settle();
  expect(host.querySelector('.hq-v10')).not.toBeNull();
  expect(host.querySelector('.lp-v14')).toBeNull();
});


it('opens the existing player inspector from a hosted Pulse starter name', async () => {
  const port = fakePort(), onPlayer = vi.fn();
  await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
    port={port} leave={() => {}} onPlayer={onPlayer} />));
  await settle();
  const player = host.querySelector('button[aria-label="Open Starter 1 player card"]');
  expect(player).not.toBeNull();
  await act(async () => { (player as HTMLButtonElement).click(); });
  expect(onPlayer).toHaveBeenCalledWith({ id: '0001', name: 'Starter 1', position: 'QB' });
});

it('refreshes the open schedule through the same hosted change source', async () => {
  const port = fakePort();
  await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
    port={port} leave={() => {}} />));
  await settle();
  await click('SCHEDULE ▴');
  expect(count(port, 'schedule.read')).toBe(1);
  await click('Refresh');
  expect(count(port, 'schedule.read')).toBe(2);
  expect(host.textContent).not.toContain('schedule is unavailable');
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});

it('retains local NFL polling, local change subscriptions and client-time week reads', async () => {
  const port = fakePort();
  await act(async () => root.render(<NativeLeaguePulse league="test" mine="a"
    port={port} leave={() => {}} />));
  await settle();
  expect(local).toHaveBeenCalled(); expect(events).toHaveBeenCalled();
  const week = port.mock.calls.map(([r]) => r).find(r => r.command === 'week.read');
  expect(week?.arguments).toMatchObject({ at: expect.stringMatching(/^\d{4}-\d\d-\d\dT.*Z$/) });
  expect(host.textContent).not.toContain('Live NFL games are coming next.');
  expect(count(port, 'nfl.games.read')).toBe(0);
});

it('reads the hosted NFL board for its own league only and labels held boards', () => {
  const view = parseHostedGames(board(), 'test');
  expect(view.games.map(g => g.id)).toEqual(['401']);
  expect(view.followed).toBeNull();
  expect(view.source).toBe('ESPN · provisional');
  expect(parseHostedGames(board(true), 'test').source).toBe('Held ESPN · provisional');
  expect(() => parseHostedGames(board(), 'other')).toThrow(/another league/);
  const empty = parseHostedGames({ league: 'test', board: { stale: false, source: '',
    at: '0001-01-01T00:00:00Z', failure: '', games: [] } }, 'test');
  expect(empty.games).toEqual([]);
  expect(empty.source).toMatch(/not read the NFL scoreboard yet/);
});

it('keeps source-held NFL records on each team and leaves absent records absent', () => {
  const source = { ...board(), board: { ...board().board, games: [{ ...nflGame,
    home: { ...nflGame.home, record: '5-4' }, away: { ...nflGame.away, record: '3-6' } }] } };
  const parsed = parseHostedGames(source, 'test').games[0];
  expect(parsed.home.record).toBe('5-4');
  expect(parsed.away.record).toBe('3-6');
  expect(parseHostedGames(board(), 'test').games[0].home.record).toBeUndefined();
  expect(() => parseHostedGames({ ...source, board: { ...source.board,
    games: [{ ...source.board.games[0],
      home: { ...source.board.games[0].home, record: '5 wins' } }] } }, 'test'))
    .toThrow(/invalid team record/);
});

it('rereads open hosted Pulse once per burst of its league topics and ignores other leagues', async () => {
  let hear: ((topics: ReadTopic[]) => void) | undefined;
  const off = vi.fn();
  vi.spyOn(hostedReads, 'onHostedTopics').mockImplementation(listener => { hear = listener; return off; });
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  expect(hear).toBeDefined();
  const before = count(port, 'results.read');
  const scores: ReadTopic = { kind: 'scores', league: 'test', season: 2026, week: 1, version: '2' };
  vi.useFakeTimers();
  act(() => { hear!([{ ...scores, league: 'other' }]); });
  await act(async () => { vi.advanceTimersByTime(PULSE_FOLLOW_MS * 2); });
  expect(count(port, 'results.read')).toBe(before);
  act(() => { for (let i = 0; i < 5; i++) hear!([{ ...scores, version: String(3 + i) }]); });
  await act(async () => { vi.advanceTimersByTime(PULSE_FOLLOW_MS - 1); });
  expect(count(port, 'results.read')).toBe(before);
  await act(async () => { vi.advanceTimersByTime(1); });
  vi.useRealTimers();
  await settle();
  expect(count(port, 'results.read')).toBe(before + 1);
  await openNode('Home');
  expect(off).toHaveBeenCalled();
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});

it('rereads the league-wide Pulse reads for league changes, not for each live score', async () => {
  let hear: ((topics: ReadTopic[]) => void) | undefined;
  vi.spyOn(hostedReads, 'onHostedTopics')
    .mockImplementation(listener => { hear = listener; return vi.fn(); });
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  const results = count(port, 'results.read'), odds = count(port, 'odds.read');
  const burst = async (kind: ReadTopic['kind']) => {
    vi.useFakeTimers();
    const topic: ReadTopic = { kind, league: 'test', season: 2026, week: 1, version: String(Math.random()) };
    act(() => { hear!([topic]); });
    await act(async () => { vi.advanceTimersByTime(PULSE_FOLLOW_MS); });
    vi.useRealTimers();
    await settle();
  };
  await burst('scores');
  expect(count(port, 'results.read')).toBe(results + 1);
  expect(count(port, 'odds.read')).toBe(odds);
  await burst('league');
  expect(count(port, 'odds.read')).toBe(odds + 1);
  await click('Refresh');
  expect(count(port, 'odds.read')).toBe(odds + 1);
});

it('never logs a read that is only waiting for its week or matchup as a failure', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await openNode('League Pulse');
  await click('Refresh');
  expect(host.textContent).toContain('Starter 2');
  expect(warn.mock.calls.filter(([words]) => String(words).includes('League Pulse read failed'))).toEqual([]);
});

it.each(['Commissioner', 'Co-commissioner'])(
  'opens a teamless %s Pulse on league week 10, not the first scheduled week', async role => {
    const fallback = fakePort();
    const port = vi.fn<HostedPort>(async request => {
      if (request.command === 'commish.rules.read') throw new Error('Rules unavailable');
      if (request.command === 'week.read') return { data: {
        schedule: { league: 'test', franchise: '', season: 2026, revision: 1,
          games: Array.from({ length: 11 }, (_, i) => ({ week: i + 1, venue: '' })) },
        current: { week: 10, venue: '', kickoff: '2026-11-15T18:00:00Z' }, nextWeek: 11,
      }, football: '', writable: false, label: 'Server' };
      if (request.command === 'results.read') return { data: {
        league: 'test', season: 2026, revision: 1, words: 'Published', week: request.arguments.week,
        published: true, games: [{ ...game, week: request.arguments.week }], missed: [], scores: [],
      }, football: '', writable: false, label: 'Server' };
      return fallback(request);
    });
    await act(async () => root.render(<HostedLeague league="test" franchise="" name="" role={role}
      port={port} />));
    await settle();
    expect(host.textContent).toContain('current league week');
    expect(host.textContent).not.toContain('first scheduled week');
    const reads = port.mock.calls.map(([r]) => r);
    expect(reads.find(r => r.command === 'week.read')?.arguments).toMatchObject({
      franchise: '', season: 2026, at: '',
    });
    expect(reads.flatMap(r => r.command === 'results.read' ? [r.arguments.week] : [])).toEqual([10]);
    expect(host.textContent).toContain('WEEK 10');
    expect(reads.some(r => r.command === 'lineup.read')).toBe(false);
    expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
  });

it('follows hosted live field and plays, switches chips and rereads detail on scores', async () => {
  let hear: ((topics: ReadTopic[]) => void) | undefined;
  vi.spyOn(hostedReads, 'onHostedTopics').mockImplementation(listener => {
    hear = listener; return () => {};
  });
  const fallback = fakePort();
  const state = { possession: '12', down: 2, distance: 7, yardLine: 60, toEndzone: 40 };
  const play = { id: 'p1', sequence: 1, text: 'First live pass', clock: '5:12', period: 2,
    at: '2026-11-15T18:00:00Z', homeScore: 14, awayScore: 10, start: state, end: state, participants: [] };
  const first = { ...nflGame, state, plays: [play], last: play };
  const second = { ...first, id: '402', away: { id: '6', abbreviation: 'DAL', score: 3 },
    home: { id: '12', abbreviation: 'NYG', score: 7 },
    plays: [{ ...play, text: 'Second live rush' }], last: { ...play, text: 'Second live rush' } };
  let refreshed = false;
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'nfl.games.read') return { data: { ...board(),
      board: { ...board().board, games: [first, second] } }, football: '', writable: false, label: 'Server' };
    if (request.command === 'nfl.game.read') return { data: { league: 'test',
      game: request.arguments.game === '401' ? first : { ...second,
        plays: [{ ...play, text: refreshed ? 'Updated live rush' : 'Second live rush' }] },
      at: '2026-11-15T18:00:00Z', stale: false, words: '' }, football: '', writable: false, label: 'Server' };
    return fallback(request);
  });
  await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
    port={port} leave={() => {}} />));
  await settle();
  // The owner's DAL starter makes the second game the default, not the first scoreboard row.
  expect(host.querySelector('.field')?.getAttribute('aria-label')).toBe('NYG ball, 40 yards to the end zone');
  expect(host.querySelector('.sp-row')?.textContent).toContain('Second live rush');
  const chip = host.querySelector<HTMLButtonElement>('.fchip[aria-label="Follow BUF at KC"]');
  expect(chip).not.toBeNull();
  await act(async () => chip!.click()); await settle();
  expect(host.querySelector('.field')?.getAttribute('aria-label')).toBe('KC ball, 40 yards to the end zone');
  expect(host.querySelector('.sp-row')?.textContent).toContain('First live pass');
  const secondChip = host.querySelector<HTMLButtonElement>('.fchip[aria-label="Follow DAL at NYG"]');
  await act(async () => secondChip!.click()); await settle();
  refreshed = true;
  vi.useFakeTimers();
  try {
    act(() => hear!([{ kind: 'scores', league: 'test', season: 2026, week: 1, version: '2' }]));
    await act(async () => { await vi.advanceTimersByTimeAsync(PULSE_FOLLOW_MS); });
    expect(host.querySelector('.sp-row')?.textContent).toContain('Updated live rush');
  } finally { vi.useRealTimers(); }
  expect(local).not.toHaveBeenCalled();
});

it.each(['now', 'race'] as const)('keeps refused odds unavailable in %s until a league change', async tab => {
  vi.useFakeTimers();
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  let hear: ((topics: ReadTopic[]) => void) | undefined;
  vi.spyOn(hostedReads, 'onHostedTopics').mockImplementation(listener => {
    hear = listener; return () => {};
  });
  const fallback = fakePort();
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'odds.read') {
      throw new Error('native odds: league results are not available on this host');
    }
    return fallback(request);
  });
  try {
    await act(async () => root.render(<NativeLeaguePulse league="test" mine="a" serverTime
      tab={tab} port={port} leave={() => {}} />));
    const oddsCount = () => port.mock.calls.filter(([r]) => r.command === 'odds.read').length;
    expect(oddsCount()).toBe(1);
    const unavailable = () => {
      expect(host.textContent).toContain('Playoff odds: unavailable.');
      expect(host.textContent).not.toContain('Reading the playoff');
      expect(host.textContent).not.toContain('retrying');
    };
    unavailable();
    for (const kind of ['scores', 'team', 'values', 'clock'] as const) {
      act(() => hear!([{ kind, league: 'test', season: 2026, week: 1, version: '2' }]));
      await act(async () => { await vi.advanceTimersByTimeAsync(10000); });
      expect(oddsCount()).toBe(1);
      unavailable();
    }
    const refresh = [...host.querySelectorAll('button')].find(b => b.textContent === 'Refresh');
    await act(async () => refresh!.click());
    await act(async () => { await vi.advanceTimersByTimeAsync(60000); });
    expect(oddsCount()).toBe(1);
    unavailable();
    act(() => hear!([{ kind: 'league', league: 'other', version: '3' }]));
    await act(async () => { await vi.advanceTimersByTimeAsync(PULSE_FOLLOW_MS); });
    expect(oddsCount()).toBe(1);
    act(() => hear!([{ kind: 'league', league: 'test', version: '3' }]));
    await act(async () => { await vi.advanceTimersByTimeAsync(PULSE_FOLLOW_MS); });
    expect(oddsCount()).toBe(2);
    unavailable();
  } finally { vi.useRealTimers(); }
});

it('logs a failed pre-game read at debug level and keeps rendering the preview game', async () => {
  const debug = vi.spyOn(console, 'debug').mockImplementation(() => {});
  const preGame = { ...nflGame, status: 'pre', clock: '', period: 0,
    home: { id: '12', abbreviation: 'KC', score: 0 }, away: { id: '2', abbreviation: 'BUF', score: 0 } };
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'nfl.games.read') {
      return { label: 'Live', writable: false, football: 'sha', appRelease: 1,
        data: { league: 'test', board: { stale: false, source: 'ESPN', at: '2026-11-15T18:00:00Z',
          failure: '', games: [preGame] } } };
    }
    if (request.command === 'nfl.game.read') {
      return { label: 'Live', writable: false, football: 'sha', appRelease: 1,
        data: { league: 'test', game: preGame, at: '2026-11-15T18:00:00Z', stale: false, words: '' } };
    }
    if (request.command === 'nfl.game.pregame.read') throw new Error('pre-game cache unavailable');
    throw new Error(`unexpected ${request.command}`);
  });
  const noChange = () => () => {};
  let followGame: ((key: string) => void) | undefined;
  function Probe() {
    const live = useHostedGames('test', port, noChange);
    followGame = live.follow;
    return <div>{live.view?.followed?.away.abbreviation ?? 'none'}|{live.error}</div>;
  }
  await act(async () => root.render(<Probe />));
  await act(async () => followGame!('BUF-KC'));
  expect(debug).toHaveBeenCalledWith('Pre-game read failed:', expect.any(Error));
  expect(host.textContent).toBe('BUF|');
});
