// @vitest-environment jsdom
import { stubFrames } from './frameStub';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Game, Play } from '../now/liveContract';
import { createFieldView } from './fieldView';
import { LiveField } from './LiveField';
import { club } from './liveClubs';
import { fieldPlay } from './livePlays';

const spotA = { possession: '12', down: 2, distance: 7, yardLine: 57, toEndzone: 57 };
const spotA2 = { possession: '12', down: 1, distance: 10, yardLine: 45, toEndzone: 45 };
const spotB = { possession: '20', down: 1, distance: 10, yardLine: 75, toEndzone: 75 };

const playA1: Play = {
  id: 'a1', sequence: 1, text: 'P. Mahomes pass short right to T. Kelce for 8 yards',
  clock: '8:42', period: 2, at: '2026-11-15T18:00:00Z', homeScore: 7, awayScore: 3,
  start: spotA, end: spotA, participants: [],
};
const playA2: Play = {
  id: 'a2', sequence: 2, text: 'I. Pacheco up the middle for 12 yards',
  clock: '8:10', period: 2, at: '2026-11-15T18:01:00Z', homeScore: 7, awayScore: 3,
  start: spotA, end: spotA2, participants: [],
};
const playB1: Play = {
  id: 'b1', sequence: 1, text: 'J. Hurts scrambles for 6 yards',
  clock: '12:00', period: 1, at: '2026-11-15T18:00:00Z', homeScore: 0, awayScore: 0,
  start: spotB, end: spotB, participants: [],
};

const gameA: Game = {
  id: '401', kickoff: '2026-11-15T18:00:00Z',
  away: { id: '2', abbreviation: 'BUF', score: 3 },
  home: { id: '12', abbreviation: 'KC', score: 7 },
  status: 'in', clock: '8:42', period: 2,
  state: spotA, last: playA1, plays: [playA1], box: [],
};
const gameB: Game = {
  id: '402', kickoff: '2026-11-15T18:00:00Z',
  away: { id: '21', abbreviation: 'DAL', score: 0 },
  home: { id: '20', abbreviation: 'PHI', score: 0 },
  status: 'in', clock: '12:00', period: 1,
  state: spotB, last: playB1, plays: [playB1], box: [],
};

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  vi.useRealTimers();
  vi.restoreAllMocks();
});

async function mount(node: React.ReactElement) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
  await act(async () => root!.render(node));
}

describe('LiveField', () => {
  it('resets card and reduced-motion banner when switching games without a key', async () => {
    vi.useFakeTimers();
    await mount(<LiveField game={gameA} reduced={true} />);
    expect(host!.querySelector('.c-text')?.textContent).toContain('Kelce');

    const gameAUpdated: Game = {
      ...gameA, state: spotA2, last: playA2, plays: [playA1, playA2],
    };
    await act(async () => root!.render(<LiveField game={gameAUpdated} reduced={true} />));
    await act(async () => { await Promise.resolve(); });
    expect(host!.querySelector('.c-text')?.textContent).toContain('Pacheco');
    expect(host!.querySelector('.rmb.on')).not.toBeNull();

    await act(async () => root!.render(<LiveField game={gameB} reduced={true} />));
    expect(host!.querySelector('.c-text')?.textContent).toContain('Hurts');
    expect(host!.querySelector('.rmb.on')).toBeNull();
  });

  it('clears the told card timer on game switch so the old play cannot overwrite the new card', async () => {
    vi.useFakeTimers();
    await mount(<LiveField game={gameA} reduced={false} />);
    expect(host!.querySelector('.c-text')?.textContent).toContain('Kelce');

    const gameAUpdated: Game = {
      ...gameA, state: spotA2, last: playA2, plays: [playA1, playA2],
    };
    await act(async () => root!.render(<LiveField game={gameAUpdated} reduced={false} />));
    await act(async () => { vi.advanceTimersByTime(100); });

    await act(async () => root!.render(<LiveField game={gameB} reduced={false} />));
    expect(host!.querySelector('.c-text')?.textContent).toContain('Hurts');

    await act(async () => { vi.advanceTimersByTime(2000); });
    expect(host!.querySelector('.c-text')?.textContent).toContain('Hurts');
    expect(host!.querySelector('.c-text')?.textContent).not.toContain('Pacheco');
  });

  it('recalculates --k when spot becomes defined after mounting hidden', async () => {
    const preGame: Game = { ...gameA, state: null };
    await mount(<LiveField game={preGame} reduced={true} />);
    const stage = host!.querySelector<HTMLElement>('#v2stage')!;
    expect(stage.style.getPropertyValue('--k')).toBe('1.000');

    Object.defineProperty(stage, 'clientHeight', { configurable: true, value: 552 });
    await act(async () => root!.render(<LiveField game={gameA} reduced={true} />));
    expect(stage.style.getPropertyValue('--k')).toBe('2.000');
  });
});

describe('createFieldView', () => {
  it('shows no down or distance once the game is over', async () => {
    const box = document.createElement('div');
    document.body.append(box);
    const view = createFieldView(box, { L: club('BUF'), R: club('KC') }, () => true);
    const game = { poss: 'R' as const, x: 57, lat: 0, down: 1, dist: 11, driveStart: 75, over: true };
    await view.sync(game, { instant: true });
    expect(box.querySelector('#hDown')?.textContent).toBe('—');
    expect(box.querySelector('#hDist')?.textContent).toBe('—');
    expect(box.querySelector('#v2stk')?.classList.contains('away')).toBe(true);
    view.destroy();
    box.remove();
  });

  it('clears all pending timers when destroy() is called', async () => {
    vi.useFakeTimers();
    const box = document.createElement('div');
    document.body.append(box);
    const view = createFieldView(box, { L: club('BUF'), R: club('KC') }, () => false);
    void view.sync({ poss: 'R', x: 57, lat: 0, down: 1, dist: 10, driveStart: 75 }, { first: true });

    const fgPlay = fieldPlay({
      ...playA1, id: 'fg1', text: 'H. Butker 45 yard field goal is GOOD',
      start: { possession: '12', down: 4, distance: 3, yardLine: 27, toEndzone: 27 },
      end: null,
    }, gameA)!;
    void view.play(fgPlay);
    await Promise.resolve();
    const tdPlay = fieldPlay({
      ...playA1, id: 'td1', text: 'I. Pacheco up the middle for 25 yards, TOUCHDOWN',
      start: { possession: '12', down: 1, distance: 10, yardLine: 25, toEndzone: 25 },
      end: null,
    }, gameA)!;
    void view.banner(tdPlay, 'I. Pacheco');
    await Promise.resolve();

    expect(vi.getTimerCount()).toBeGreaterThan(0);
    view.destroy();
    expect(vi.getTimerCount()).toBe(0);
    box.remove();
  });

  it('scales bigPlay streak via transform without querying .yl per frame', async () => {
    vi.useFakeTimers();
    const box = document.createElement('div');
    document.body.append(box);
    const view = createFieldView(box, { L: club('BUF'), R: club('KC') }, () => false);
    const world = box.querySelector<HTMLElement>('#v2world')!;
    const qsa = vi.spyOn(world, 'querySelectorAll');

    let now = 0;
    vi.spyOn(performance, 'now').mockImplementation(() => now);
    const frames: FrameRequestCallback[] = [];
    stubFrames(cb => {
      frames.push(cb);
      return frames.length;
    });

    const bigRun = fieldPlay({
      ...playA1, id: 'big1', text: 'I. Pacheco up the middle for 25 yards',
      start: { possession: '12', down: 1, distance: 10, yardLine: 60, toEndzone: 60 },
      end: { possession: '12', down: 1, distance: 10, yardLine: 35, toEndzone: 35 },
    }, gameA)!;
    void view.play(bigRun);

    const streak = box.querySelector<HTMLElement>('.bp-streak')!;
    const initialWidth = streak.style.width;
    expect(initialWidth).not.toBe('');

    now = 500;
    const cb = frames.shift()!;
    cb(now);

    expect(streak.style.width).toBe(initialWidth);
    expect(streak.style.transform).toContain('scaleX(');
    expect(qsa).not.toHaveBeenCalledWith('.yl');

    view.destroy();
    box.remove();
  });
});
