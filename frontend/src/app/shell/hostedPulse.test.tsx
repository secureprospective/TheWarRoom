// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HostedLeague from '../commissioner/HostedLeague';
import NativeLeaguePulse from './NativeLeaguePulse';
import { readNativePulse, readPulseMatchup } from './pulseReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import * as nfl from '../now/api';
import * as hq from '../commissioner/hqReading';
import { heldSnapshot } from '../data/lineupTestData';
import { hqEvents } from './hqReads';

const common = { league: 'test', season: 2026, revision: 1, name: 'Test League', words: 'Published' };
const teams = [{ franchise: 'a', name: 'Alpha', abbreviation: 'ALP' },
  { franchise: 'b', name: 'Beta', abbreviation: 'BET' }];
const game = { week: 1, home: 'a', away: 'b', homeName: 'Alpha', awayName: 'Beta',
  status: 'No result yet', words: 'Alpha vs Beta', why: [] };
function fakePort() {
  return vi.fn<HostedPort>(async request => {
    const args = request.arguments;
    let data: unknown;
    switch (request.command) {
      case 'native.directory.read':
        data = { league: 'test', name: 'Test League', revision: 1, rosterSeason: 2026,
          franchises: teams }; break;
      case 'week.read':
        data = { schedule: { league: 'test', season: 2026, revision: 1, franchise: 'a',
          games: [{ week: 1, venue: 'home' }] },
          current: { week: 1 }, message: '' }; break;
      case 'standings.read':
        data = { ...common, status: 'Ready', through: 0, decided: 0, pending: 1, criteria: [], noData: [],
          computed: teams.map((t, i) => ({ franchise: t.franchise, name: t.name, rank: i + 1,
            wins: 0, losses: 0, ties: 0,
            record: '0–0', pointsFor: 0, pointsWords: 'No final points', why: [] })) }; break;
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
});
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
async function click(label: string) {
  const button = [...document.querySelectorAll('button')].find(b => b.textContent === label);
  expect(button, label).toBeDefined();
  await act(async () => button!.click()); await settle();
}
const count = (port: ReturnType<typeof fakePort>, command: string) =>
  port.mock.calls.filter(([r]) => r.command === command).length;

it('opens hosted Now, refreshes each active read once, shows Race and returns', async () => {
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} />));
  await click('League Pulse');
  expect(host.textContent).toContain('Starter 2');
  expect(host.textContent).toContain('20.0');
  expect(host.textContent).toContain('Live NFL games are coming next.');
  expect(host.textContent).toContain('Scores update when the league server publishes them');
  expect(host.textContent).not.toContain('unavailable');
  expect(count(port, 'lineup.read')).toBe(1);
  const commands = ['native.directory.read', 'week.read', 'standings.read', 'bracket.read',
    'odds.read', 'results.read', 'lineup.read', 'projections.read'];
  const before = commands.map(c => count(port, c));
  await click('Refresh');
  expect(commands.map(c => count(port, c))).toEqual(before.map(n => n + 1));
  for (const [r] of port.mock.calls) if (r.command === 'week.read') expect(r.arguments.at).toBe('');
  await click('NFL GAMES');
  expect(host.textContent).not.toContain('Reading the NFL scoreboard');
  await click('Race');
  expect(host.textContent).toContain('STANDINGS');
  expect(host.textContent).toContain('0–0');
  await click('Back');
  expect(host.textContent).toContain('League Pulse');
  expect(host.querySelector('.lp-v14')).toBeNull();
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
});

it.each(['Commissioner', 'Co-commissioner'])('reads both lineups for %s', async role => {
  const port = fakePort();
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role={role} port={port} />));
  await click('League Pulse');
  expect(count(port, 'lineup.read')).toBe(2);
  expect(host.textContent).toContain('Starter 2');
  expect(local).not.toHaveBeenCalled(); expect(events).not.toHaveBeenCalled();
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


it('opens from the HQ rail and Back returns to HQ', async () => {
  const port = fakePort();
  vi.spyOn(hq, 'readNativeHQ').mockResolvedValue({ revision: 1, snapshot: heldSnapshot(),
    settings: { positions: [] }, errors: [] });
  vi.spyOn(hqEvents, 'onChange').mockImplementation(() => () => {});
  await act(async () => root.render(<HostedLeague league="test" franchise="a" name="Alpha"
    role="Owner" port={port} hq />));
  await settle();
  for (let i = 0; i < 20 && !document.querySelector('.hq-v10 .rail'); i++) await settle();
  const pulse = [...document.querySelectorAll('button')].find(b => b.textContent?.includes('League Pulse'));
  expect(pulse, document.body.textContent ?? '').toBeDefined();
  await act(async () => pulse!.click()); await settle();
  expect(host.textContent).toContain('Starter 2');
  await click('Back');
  expect(document.querySelector('.hq-v10')).not.toBeNull();
  expect(host.querySelector('.lp-v14')).toBeNull();
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
});
