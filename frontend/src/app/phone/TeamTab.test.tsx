// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import type { NativeHQ } from '../commissioner/hqReading';
import type { Draft } from '../draftroom/draft';
import type { Game } from '../now/liveContract';
import { canMoveTo, gameWords, moveStarters, myPicks, teamView } from './teamModel';

const read = vi.hoisted(() => ({ value: undefined as unknown, fail: false }));
vi.mock('../commissioner/hqReading', () => ({
  readNativeHQ: async (_l: string, _m: string, _p: unknown, publish?: (r: unknown) => void) => {
    if (read.fail) throw new Error('lineup.read: server unavailable');
    publish?.(read.value); return read.value;
  },
}));
vi.mock('../shell/hqNativeLineup', () => ({
  useNativeLineup: (_l: string, _m: string, lineup: unknown) => ({
    edited: lineup, canEdit: true, dirty: false,
    changed: [], message: '', assign: vi.fn(), discard: vi.fn(), save: vi.fn(), state: {} }),
  NativeLineupReview: () => null,
}));
vi.mock('../commissioner/hostedReadings', async original => ({
  ...(await original<object>()),
  parseHostedRoster: (data: { open?: number }) => ({ openBenchSlots: { mine: data.open } }),
}));
const { TeamTab } = await import('./TeamTab');

const later = new Date(Date.now() + 86_400_000).toISOString();
const past = new Date(Date.now() - 3_600_000).toISOString();
function hq(overrides: Partial<NativeHQ> = {}): NativeHQ {
  const players = [
    { id: '0101', name: 'Qb One', position: 'QB', team: 'KC' },
    { id: '0102', name: 'Rb Bye', position: 'RB', team: 'HOU' },
    { id: '0103', name: 'Rb Bench', position: 'RB', team: 'NYJ' },
    { id: '0104', name: 'Wr Hurt', position: 'WR', team: 'BUF' },
  ];
  return {
    revision: 1, errors: [], week: 11, record: '7-3', standing: '2nd of 12', searchPlayers: [],
    settings: { positions: [{ code: 'QB', min: 1, max: 1 }, { code: 'RB', min: 1, max: 2 }] },
    snapshot: {
      players: { value: players }, franchises: { value: [{ id: 'mine', name: 'My Squad' }] },
      rosters: { value: [{ franchiseId: 'mine', players: [
        { id: '0101', rosterStatus: 'ROSTER' }, { id: '0102', rosterStatus: 'ROSTER' },
        { id: '0103', rosterStatus: 'ROSTER' }, { id: '0104', rosterStatus: 'IR' }] }] },
      league: { value: { season: 2026 } },
    },
    lineup: { franchise: 'mine', week: 11, starterCount: 2,
      starters: [{ id: '0101', position: 'QB' }, { id: '0102', position: 'RB' }],
      bench: [{ id: '0103', position: 'RB' }] },
    kickoffs: new Map([['0101', past], ['0103', later]]),
    projections: new Map([['0101', 20], ['0102', 10.5]]),
    thisWeek: { scores: [{ franchise: 'mine', points: 31.2, starters: [{ player: '0101', points: 31.2 }] }] },
    playerHistory: { '0101': [10, 25, 18, 30, 12, 22].map((points, i) => ({ week: 5 + i, points })) },
    playerForm: { '0101': [25, 18, 30, 12, 22].map((points, i) => ({ week: 6 + i, points })) },
    ...overrides,
  } as unknown as NativeHQ;
}
const games = [{ id: 'g1', kickoff: past, status: 'in', period: 2, clock: '4:10',
  home: { abbreviation: 'KC' }, away: { abbreviation: 'BUF' } },
  { id: 'g2', kickoff: later, status: 'pre', period: 0, clock: '',
    home: { abbreviation: 'NYJ' }, away: { abbreviation: 'MIA' } }] as unknown as Game[];

it('builds rows with game state, bye, lock count, IR and six-game range from the reads', () => {
  const reading = hq();
  const view = teamView(reading, reading.lineup, 'mine', games);
  expect(view.starters.map(r => [r.slot, r.name, r.state, r.locked])).toEqual([
    ['QB', 'Qb One', 'live', true], ['RB', 'Rb Bye', 'bye', false]]);
  expect(view.starters[0].words).toBe('KC · LIVE Q2 4:10');
  expect(view.starters[0].range).toEqual([10, 30]);
  expect(view.points).toBe(31.2);
  expect(view.projection).toBe(30.5);
  expect(view.locked).toBe(1);
  expect(view.bye?.name).toBe('Rb Bye');
  expect(view.ir.map(r => [r.name, r.state, r.words])).toEqual([['Wr Hurt', 'out', 'INJURED RESERVE']]);
  expect(view.irOutsideBench).toBe(true);
});

it('without a games read, uses the lineup kickoff and never claims a bye', () => {
  expect(gameWords('HOU', undefined, undefined)).toEqual({ state: '', started: false, words: 'HOU' });
  expect(gameWords('KC', [], past).started).toBe(true);
});

it('allows only fitting swaps between unlocked starters and bench players', () => {
  const reading = hq();
  const view = teamView(reading, reading.lineup, 'mine', games);
  const [qb, rb] = view.starters; const bench = view.bench[0];
  expect(canMoveTo(bench, rb, reading)).toBe(true);
  expect(canMoveTo(bench, qb, reading)).toBe(false);
  expect(canMoveTo(rb, view.ir[0], reading)).toBe(false);
  expect(canMoveTo(qb, rb, reading)).toBe(false);
  expect(view.bench[0].words).toMatch(/^NYJ · .* VS MIA$/);
  expect(moveStarters(reading.lineup!, bench, rb)).toEqual(['0101', '0103']);
});

it('lists unused owned picks with their original owner', () => {
  const draft = { season: 2027, owners: [{ id: 'other', name: 'Rivals' }], picks: [{ round: 1, slot: 1 }],
    order: [{ round: 1, picks: [{ round: 1, slot: 1, owner: 'mine', original: 'mine' },
      { round: 1, slot: 2, owner: 'mine', original: 'other' }] }] } as unknown as Draft;
  expect(myPicks(draft, 'mine')).toEqual([{ key: '1-2', label: '2027 round 1', from: 'from Rivals' }]);
});

let root: Root | undefined; let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount()); host?.remove(); read.fail = false;
});
async function render(port = vi.fn(async () => { throw new Error('offline'); })) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<TeamTab league="L" mine="mine" port={port} />));
  await act(async () => { await new Promise(r => setTimeout(r, 0)); });
  return host;
}

it('renders the loaded team: header, sections and the move controls', async () => {
  read.value = hq();
  const page = await render();
  expect(page.textContent).toContain('My Squad');
  expect(page.textContent).toContain('7-3 · 2nd of 12 · week 11');
  expect(page.textContent).toContain('PROJ 30.5');
  expect(page.textContent).toContain('1 locked · games started');
  expect(page.textContent).toContain("doesn't count toward the bench");
  expect(page.querySelector('.team-tab button, .team-tab [onclick]')).toBeTruthy();
  expect([...page.querySelectorAll('.team-row__mv')].map(b => b.textContent)).toEqual(
    ['LOCKED', 'MOVE', 'MOVE', 'IR']);
});

it('renders empty bench and IR words when the roster has none', async () => {
  const base = hq();
  read.value = hq({ lineup: { ...base.lineup!, bench: [] },
    snapshot: { ...base.snapshot, rosters: { value: [{ franchiseId: 'mine', players: [] }] } } as never });
  const page = await render();
  expect(page.textContent).toContain('No bench players.');
  expect(page.textContent).toContain('No players on injured reserve.');
});

it('shows the unavailable words when the reads fail', async () => {
  read.fail = true;
  const page = await render();
  expect(page.textContent).toContain('Not available yet');
  expect(page.textContent).not.toContain('My Squad');
});

it('bench shows N of capacity; injury facts on rows and the card', async () => {
  read.value = hq();
  const port = vi.fn(async (request: { command: string }) => {
    if (request.command === 'native.assets.read') return { label: '', writable: true, football: '',
      data: { open: 5 } };
    if (request.command === 'nfl.player-status.read') return { label: '', writable: false, football: '',
      data: { league: 'L', state: 'available', season: 2026, fetchedAt: '2026-11-18T15:00:00Z',
        players: [{ player: '0103', designation: 'Questionable', bodyPart: 'Ankle' }] } };
    throw new Error('offline');
  });
  const page = await render(port as never);
  await act(async () => { await new Promise(r => setTimeout(r, 10)); });
  expect(page.textContent).toContain('Bench 1 of 6');
  expect(page.querySelector('[data-row="bench-0"]')?.textContent).toContain('Questionable · Ankle');
  await act(async () => (page.querySelector('[aria-label="Rb Bench player card"]') as HTMLElement).click());
  expect(document.querySelector('dialog')?.textContent).toContain('Injury Questionable · Ankle');
});

it('without those reads the bench counts players and injury is unavailable', async () => {
  read.value = hq();
  const page = await render();
  expect(page.textContent).toContain('Bench 1 players');
  await act(async () => (page.querySelector('[aria-label="Rb Bench player card"]') as HTMLElement).click());
  expect(document.querySelector('dialog')?.textContent).toContain('Injury Not available yet');
});
