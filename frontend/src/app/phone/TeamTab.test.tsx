// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import type { NativeHQ } from '../commissioner/hqReading';
import type { Draft } from '../draftroom/draft';
import type { Game } from '../now/liveContract';
import { benchStarter, canMoveTo, gameWords, moveStarters, myPicks, teamView } from './teamModel';
import { HOLD_MS } from './rowDrag';

const read = vi.hoisted(() => ({ value: undefined as unknown, fail: false }));
const lineupMock = vi.hoisted(() => ({ canEdit: true, assign: vi.fn() }));
vi.mock('../commissioner/hqReading', () => ({
  readNativeHQ: async (_l: string, _m: string, _p: unknown, publish?: (r: unknown) => void) => {
    if (read.fail) throw new Error('lineup.read: server unavailable');
    publish?.(read.value); return read.value;
  },
}));
vi.mock('../shell/hqNativeLineup', () => ({
  useNativeLineup: (_l: string, _m: string, lineup: unknown) => ({
    edited: lineup, canEdit: lineupMock.canEdit, dirty: false,
    changed: [], message: '', assign: lineupMock.assign, discard: vi.fn(), save: vi.fn(), state: {} }),
  NativeLineupReview: () => null,
}));
vi.mock('../commissioner/hostedReadings', async original => ({
  ...(await original<object>()),
  parseHostedRoster: (data: { open?: number }) => ({ openBenchSlots: { mine: data.open } }),
}));
const { TeamTab, emptyBench, placeWords } = await import('./TeamTab');

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

it('keeps an owed pick in My picks and opens the shared hosted draft room from its sheet', async () => {
  read.value = hq();
  const draftRead = { league: 'L', season: 2026, revision: 2, rules: [], owners: [], order: [
    { round: 1, words: '', picks: [{ round: 1, original: 'mine', owner: 'mine', slot: 1 }] },
  ], words: 'All turns have ended; missed picks may still be made until the draft closes.',
  nextOwnedPick: null, closed: null, players: {},
  available: [{ id: '1001', name: 'Rookie One', position: 'RB', team: 'KC' }], checkpoints: [],
  draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: true, onClock: null,
    picks: [{ round: 1, original: 'mine', owner: 'mine', slot: 1, extra: 0, player: '', skipped: true,
      forfeited: false, how: 'expired', contract: { salaryCents: 1234500, years: 3, rfaAfter: false } }],
  } };
  const port = vi.fn(async (request: { command: string; arguments?: Record<string, unknown> }) => {
    if (request.command === 'draft.read') return { label: '', writable: true, football: '', data: draftRead };
    if (request.command === 'draft.pick') return { label: '', writable: true, football: '', data: {
      entry: 'entry-1', league: 'L', revision: 3, recordedAt: '2026-05-01T00:02:00Z',
      message: 'Rookie One was picked.',
    } };
    throw new Error('offline');
  });
  const page = await render(port as never);
  await act(async () => (page.querySelector('.team-head__picks') as HTMLElement).click());
  await act(async () => { await new Promise(r => setTimeout(r, 0)); });
  const picksSheet = document.querySelector('dialog')!;
  expect(picksSheet.textContent).toContain('2026 round 1');
  await act(async () => (picksSheet.querySelector('[aria-label="Open draft room"]') as HTMLElement).click());
  await act(async () => { await new Promise(r => setTimeout(r, 0)); });
  const draftSheet = document.querySelector('dialog')!;
  expect(draftSheet.textContent).toContain('You missed round 1 pick 1. Pick any time.');
  expect(port.mock.calls.filter(([request]) => request.command === 'draft.read')).toHaveLength(2);
  await act(async () => {
    const player = draftSheet.querySelectorAll('select')[1] as HTMLSelectElement;
    player.value = '1001'; player.dispatchEvent(new Event('change', { bubbles: true }));
  });
  await act(async () => (draftSheet.querySelector('[aria-label="Pick Rookie One"]') as HTMLElement).click());
  const pick = port.mock.calls.find(([request]) => request.command === 'draft.pick')?.[0];
  expect(pick?.arguments).toMatchObject({ franchise: 'mine', player: '1001' });
});

let root: Root | undefined; let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount()); host?.remove(); read.fail = false;
  lineupMock.canEdit = true; lineupMock.assign.mockReset(); vi.useRealTimers(); vi.restoreAllMocks();
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

it('header place reads 11th of 12 from standings, else the division standing or unavailable', () => {
  expect(placeWords({ place: { rank: 11, of: 12 }, standing: '1st East' })).toBe('11th of 12');
  expect(placeWords({ standing: '1st East' })).toBe('1st East');
  expect(placeWords({})).toBe('Not available yet');
});

it('open bench spots draw as one counted row that are not swap targets', async () => {
  const reading = hq();
  const view = teamView(reading, reading.lineup, 'mine', games);
  const [open] = emptyBench(1, 2);
  expect(emptyBench(1, 2).map(r => [r.key, r.index, r.name]))
    .toEqual([['bench-open', 1, '2 empty bench spots']]);
  expect(emptyBench(1, 1)[0].name).toBe('Empty bench spot');
  expect(emptyBench(1, 0)).toEqual([]);
  expect(canMoveTo(view.starters[1], open, reading)).toBe(false);
  read.value = hq();
  const port = vi.fn(async (request: { command: string }) => {
    if (request.command === 'native.assets.read') return { label: '', writable: true, football: '',
      data: { open: 2 } };
    throw new Error('offline');
  });
  const page = await render(port as never);
  await act(async () => { await new Promise(r => setTimeout(r, 10)); });
  const row = page.querySelector('[data-row="bench-open"]');
  expect(row?.textContent).toContain('2 empty bench spotsAdd a player from Moves');
  expect(row?.querySelector('.team-row__mv')?.textContent).toBe('—');
  expect(page.querySelector('[data-row="bench-0"] .team-row__pt')?.getAttribute('style')).toBeNull();
  expect(page.querySelector('[data-row="starters-0"] .team-row__pt')?.getAttribute('style'))
    .toContain('--lo: 0.333');
});

function pointerEvent(type: string, init: { pointerId?: number; clientX?: number; clientY?: number } = {}) {
  const e = new MouseEvent(type, {
    bubbles: true, cancelable: true, button: 0,
    clientX: init.clientX ?? 0, clientY: init.clientY ?? 0,
  });
  Object.defineProperty(e, 'pointerId', { value: init.pointerId ?? 1 });
  return e;
}

it('MOVE on a starter benches him straight away', async () => {
  const reading = hq();
  expect(benchStarter(reading.lineup!, '0102')).toEqual(['0101']);
  read.value = reading;
  const page = await render();
  const btn = page.querySelector<HTMLButtonElement>('[data-row="starters-1"] .team-row__mv')!;
  await act(async () => btn.click());
  expect(lineupMock.assign).toHaveBeenCalledWith(['0101']);
  expect(page.querySelector('.team-movebar')).toBeNull();
});

it('MOVE on a bench player fills a fitting open starting slot', async () => {
  const base = hq();
  read.value = hq({
    lineup: {
      ...base.lineup!,
      starters: [{ id: '0101', position: 'QB' }],
      bench: [{ id: '0103', position: 'RB' }],
    },
  });
  const page = await render();
  const btn = page.querySelector<HTMLButtonElement>('[data-row="bench-0"] .team-row__mv')!;
  await act(async () => btn.click());
  expect(lineupMock.assign).toHaveBeenCalledWith(['0101', '0103']);
  expect(page.querySelector('.team-movebar')).toBeNull();
});

it('MOVE with no fitting open slot enters lit-row mode and swaps on tap', async () => {
  read.value = hq();
  const page = await render();
  const btn = page.querySelector<HTMLButtonElement>('[data-row="bench-0"] .team-row__mv')!;
  await act(async () => btn.click());
  expect(lineupMock.assign).not.toHaveBeenCalled();
  expect(page.querySelector('.team-movebar')?.textContent).toContain('Moving Rb Bench. Tap a lit row.');
  expect(page.querySelector('[data-row="starters-1"]')?.classList.contains('can')).toBe(true);
  expect(page.querySelector('[data-row="starters-0"]')?.classList.contains('can')).toBe(false);
  const target = page.querySelector<HTMLButtonElement>('[data-row="starters-1"] .team-row__mv')!;
  await act(async () => target.click());
  expect(lineupMock.assign).toHaveBeenCalledWith(['0101', '0103']);
});

it('the card sheet shows the shared full card with news, stats, contract, status, note and actions',
  async () => {
    const base = hq();
    read.value = hq({
      contracts: new Map([['0103', [{ season: 2026, cents: 400_000_000 }]]]),
      snapshot: {
        ...base.snapshot,
        rosters: {
          value: [{
            franchiseId: 'mine',
            players: [
              { id: '0101', rosterStatus: 'ROSTER' }, { id: '0102', rosterStatus: 'ROSTER' },
              { id: '0103', rosterStatus: 'ROSTER', salary: 400_000_000, contractStatus: 'RFA' },
              { id: '0104', rosterStatus: 'IR' },
            ],
          }],
        },
      } as never,
    });
    const port = vi.fn(async (request: { command: string }) => {
      if (request.command === 'player.usage.read') {
        return {
          label: 'L', writable: false, football: '',
          data: {
            snapShare: 0.65, targetShare: 0.14, carryShare: 0.58, weeksUsed: 4,
            opponent: 'MIA', allowedPoints: 17.2, allowedRank: 20, teams: 32,
            label: 'Neutral', reason: 'Last 4 weeks',
          },
        };
      }
      if (request.command === 'player.note.read') {
        return {
          label: 'L', writable: true, football: '',
          data: { league: 'L', player: '0103', count: 1, watched: false,
            notes: [{ kind: 'note', entity: 'player', id: '0103', text: 'Watch snaps',
              at: '2026-11-19T10:00:00Z', order: 1 }] },
        };
      }
      throw new Error('offline');
    });
    const page = await render(port as never);
    await act(async () => (page.querySelector('[aria-label="Rb Bench player card"]') as HTMLElement).click());
    await act(async () => { await new Promise(r => setTimeout(r, 10)); });
    const dialog = document.querySelector('dialog')!;
    expect(dialog.querySelector('[aria-label="Player news"]')).toBeTruthy();
    expect(dialog.textContent).toContain('Player news');
    expect(dialog.textContent).toContain('MIA');
    expect(dialog.textContent).toContain('58%');
    expect(dialog.textContent).toContain('Contract years');
    expect(dialog.textContent).toContain('$4.0M');
    expect(dialog.textContent).toContain('Status Active roster');
    expect(dialog.querySelector<HTMLTextAreaElement>('#team-note')?.value).toBe('Watch snaps');
    const buttons = [...dialog.querySelectorAll('.team-card__a button')].map(b => b.textContent);
    expect(buttons).toEqual(['Move', 'Trade', 'Drop']);
    const statsBtn = [...dialog.querySelectorAll('button')].find(b => b.textContent === '2026 Stats')!;
    await act(async () => statsBtn.click());
    expect(dialog.textContent).toContain('Last five weeks');
  });

it('press-and-hold drag moves onto a lit row, cancels elsewhere, and cancels on early scroll', async () => {
  read.value = hq();
  const page = await render();
  vi.useFakeTimers();
  const tab = page.querySelector<HTMLElement>('.team-tab')!;
  const benchRow = page.querySelector<HTMLElement>('[data-row="bench-0"]')!;
  const targetRow = page.querySelector<HTMLElement>('[data-row="starters-1"]')!;

  // Moving beyond threshold before HOLD_MS cancels so scrolling is never blocked.
  await act(async () => {
    benchRow.dispatchEvent(pointerEvent('pointerdown', { pointerId: 1, clientX: 50, clientY: 200 }));
    tab.dispatchEvent(pointerEvent('pointermove', { pointerId: 1, clientX: 50, clientY: 220 }));
    vi.advanceTimersByTime(HOLD_MS + 20);
  });
  expect(tab.classList.contains('dragging')).toBe(false);

  // Hold past HOLD_MS lights legal rows; drop on a lit row assigns the move.
  const origFromPoint = document.elementFromPoint;
  document.elementFromPoint = vi.fn(() => targetRow);
  try {
    await act(async () => {
      benchRow.dispatchEvent(pointerEvent('pointerdown', { pointerId: 2, clientX: 50, clientY: 200 }));
      vi.advanceTimersByTime(HOLD_MS);
    });
    expect(tab.classList.contains('dragging')).toBe(true);
    expect(tab.classList.contains('moving')).toBe(true);
    expect(targetRow.classList.contains('can')).toBe(true);
    await act(async () => {
      tab.dispatchEvent(pointerEvent('pointermove', { pointerId: 2, clientX: 50, clientY: 120 }));
      tab.dispatchEvent(pointerEvent('pointerup', { pointerId: 2, clientX: 50, clientY: 120 }));
    });
    expect(lineupMock.assign).toHaveBeenCalledWith(['0101', '0103']);
    expect(tab.classList.contains('dragging')).toBe(false);

    // Releasing off a lit row cancels without assigning.
    lineupMock.assign.mockReset();
    document.elementFromPoint = vi.fn(() => null);
    await act(async () => {
      benchRow.dispatchEvent(pointerEvent('pointerdown', { pointerId: 3, clientX: 50, clientY: 200 }));
      vi.advanceTimersByTime(HOLD_MS);
      tab.dispatchEvent(pointerEvent('pointerup', { pointerId: 3, clientX: 50, clientY: 10 }));
    });
    expect(lineupMock.assign).not.toHaveBeenCalled();
    expect(tab.classList.contains('dragging')).toBe(false);
  } finally {
    document.elementFromPoint = origFromPoint;
  }
});

it('keeps rows read-only when editor.canEdit is false', async () => {
  lineupMock.canEdit = false;
  read.value = hq();
  const page = await render();
  vi.useFakeTimers();
  const tab = page.querySelector<HTMLElement>('.team-tab')!;
  const benchRow = page.querySelector<HTMLElement>('[data-row="bench-0"]')!;
  expect(benchRow.querySelector<HTMLButtonElement>('.team-row__mv')?.disabled).toBe(true);
  await act(async () => {
    benchRow.dispatchEvent(pointerEvent('pointerdown', { pointerId: 4, clientX: 50, clientY: 200 }));
    vi.advanceTimersByTime(HOLD_MS + 20);
  });
  expect(tab.classList.contains('dragging')).toBe(false);
  expect(lineupMock.assign).not.toHaveBeenCalled();
});
