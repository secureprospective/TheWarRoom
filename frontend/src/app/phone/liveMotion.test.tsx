// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Game } from '../now/liveContract';
import {
  countUpSteps, detectChange, onFieldFromPossession, scoreRollSteps,
  useAnimatedValue, useScoreRoll,
} from './liveMotion';

const gameIn = (possession: string, status: Game['status'] = 'in'): Game => ({
  id: '401', kickoff: '2026-11-15T18:00:00Z',
  away: { id: '2', abbreviation: 'BUF', score: 3 },
  home: { id: '12', abbreviation: 'KC', score: 7 },
  status, clock: '8:42', period: 2,
  state: status === 'in' ? { possession, down: 2, distance: 7, yardLine: 43, toEndzone: 57 } : null,
  last: null, box: [], plays: [],
});

describe('detectChange', () => {
  it('never triggers on the first read or missing values', () => {
    expect(detectChange(undefined, 24.2)).toEqual({ changed: false });
    expect(detectChange(24.2, undefined)).toEqual({ changed: false });
    expect(detectChange(undefined, undefined)).toEqual({ changed: false });
  });
  it('ignores unchanged values and context switches', () => {
    expect(detectChange(24.2, 24.2)).toEqual({ changed: false });
    expect(detectChange(24.2, 24.2000001)).toEqual({ changed: false });
    expect(detectChange(24.2, 34.5, false)).toEqual({ changed: false });
  });
  it('reports real increases and decreases within the same context', () => {
    expect(detectChange(24.2, 34.5, true)).toEqual({ changed: true, from: 24.2, to: 34.5 });
    expect(detectChange(0.44, 0.74, true)).toEqual({ changed: true, from: 0.44, to: 0.74 });
    expect(detectChange(34.5, 31.2, true)).toEqual({ changed: true, from: 34.5, to: 31.2 });
  });
});

describe('countUpSteps', () => {
  it('snaps directly to the target under reduced motion', () => {
    expect(countUpSteps(24.2, 34.5, { reduced: true, decimals: 1 })).toEqual([34.5]);
    expect(countUpSteps(44, 74, { reduced: true, decimals: 0 })).toEqual([74]);
  });
  it('produces monotonic rounded steps ending on the exact target', () => {
    const up = countUpSteps(24.2, 34.5, { reduced: false, decimals: 1 });
    expect(up.length).toBeGreaterThan(2);
    expect(up[0]).toBeGreaterThan(24.2);
    expect(up[up.length - 1]).toBe(34.5);
    for (let i = 1; i < up.length; i++) expect(up[i]).toBeGreaterThan(up[i - 1]);

    const down = countUpSteps(34.5, 31.0, { reduced: false, decimals: 1 });
    expect(down.length).toBeGreaterThan(2);
    expect(down[0]).toBeLessThan(34.5);
    expect(down[down.length - 1]).toBe(31.0);
    for (let i = 1; i < down.length; i++) expect(down[i]).toBeLessThan(down[i - 1]);
  });
  it('steps integer win-chance percentages without duplicate frames', () => {
    const pct = countUpSteps(44, 74, { reduced: false, decimals: 0 });
    expect(pct.length).toBeGreaterThan(2);
    expect(pct[pct.length - 1]).toBe(74);
    expect(new Set(pct).size).toBe(pct.length);
  });
});

describe('scoreRollSteps', () => {
  it('rolls one point at a time like the mock odometer when score rises', () => {
    expect(scoreRollSteps(7, 14, false)).toEqual([8, 9, 10, 11, 12, 13, 14]);
    expect(scoreRollSteps(3, 6, false)).toEqual([4, 5, 6]);
  });
  it('snaps instantly under reduced motion or when score does not rise', () => {
    expect(scoreRollSteps(7, 14, true)).toEqual([14]);
    expect(scoreRollSteps(14, 14, false)).toEqual([14]);
    expect(scoreRollSteps(14, 7, false)).toEqual([7]);
  });
  it('caps large catch-up jumps while ending on the exact target', () => {
    const big = scoreRollSteps(0, 28, false);
    expect(big.length).toBeLessThanOrEqual(14);
    expect(big[big.length - 1]).toBe(28);
    for (let i = 1; i < big.length; i++) expect(big[i]).toBeGreaterThan(big[i - 1]);
  });
});

describe('onFieldFromPossession', () => {
  it('approximates on-field state from offensive possession only', () => {
    const kcBall = gameIn('12');
    expect(onFieldFromPossession({ slot: 'RB', team: 'KC' }, kcBall)).toBe(true);
    expect(onFieldFromPossession({ slot: 'WR', team: 'KCC' }, kcBall)).toBe(true);
    expect(onFieldFromPossession({ slot: 'QB', team: 'BUF' }, kcBall)).toBe(false);
    expect(onFieldFromPossession({ slot: 'K', team: 'KC' }, kcBall)).toBe(false);
    expect(onFieldFromPossession({ slot: 'DEF', team: 'KC' }, kcBall)).toBe(false);
    expect(onFieldFromPossession({ slot: 'RB', team: 'KC' }, gameIn('12', 'post'))).toBe(false);
    expect(onFieldFromPossession({ slot: 'RB', team: 'SF' }, kcBall)).toBe(false);
  });
});

describe('useAnimatedValue and useScoreRoll hooks', () => {
  let root: Root | undefined;
  let host: HTMLDivElement | undefined;
  afterEach(async () => {
    if (root) await act(async () => root!.unmount());
    host?.remove(); root = undefined; host = undefined;
    vi.useRealTimers();
  });

  function ValueProbe({ value, ctx, reduced }: { value?: number; ctx: string; reduced: boolean }) {
    const { display, changing } = useAnimatedValue(value, ctx, { reduced, decimals: 1, stepMs: 40 });
    return <div data-val={display ?? 'none'} data-chg={String(changing)} />;
  }

  function RollProbe({ score, gameId, reduced }: { score: number; gameId: string; reduced: boolean }) {
    const { display, scoring } = useScoreRoll(score, gameId, reduced);
    return <div data-val={display} data-scoring={String(scoring)} />;
  }

  it('animates a held score change step by step and settles on the exact target', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    vi.useFakeTimers();
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    await act(async () => root!.render(<ValueProbe value={24.2} ctx="m1" reduced={false} />));
    const el = () => host!.firstElementChild as HTMLElement;
    expect(el().dataset.val).toBe('24.2');
    expect(el().dataset.chg).toBe('false');

    await act(async () => root!.render(<ValueProbe value={34.5} ctx="m1" reduced={false} />));
    expect(el().dataset.chg).toBe('true');
    const mid = Number(el().dataset.val);
    expect(mid).toBeGreaterThan(24.2);
    expect(mid).toBeLessThan(34.5);

    await act(async () => { vi.advanceTimersByTime(600); });
    expect(el().dataset.val).toBe('34.5');
    await act(async () => { vi.advanceTimersByTime(1000); });
    expect(el().dataset.chg).toBe('false');
  });

  it('snaps immediately under reduced motion and on context switch', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    vi.useFakeTimers();
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    await act(async () => root!.render(<ValueProbe value={24.2} ctx="m1" reduced={true} />));
    const el = () => host!.firstElementChild as HTMLElement;

    await act(async () => root!.render(<ValueProbe value={34.5} ctx="m1" reduced={true} />));
    expect(el().dataset.val).toBe('34.5');
    expect(el().dataset.chg).toBe('false');

    await act(async () => root!.render(<ValueProbe value={10.0} ctx="m2" reduced={false} />));
    expect(el().dataset.val).toBe('10');
    expect(el().dataset.chg).toBe('false');
  });

  it('rolls an NFL score point by point and highlights scoring, snapping under reduced motion', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    vi.useFakeTimers();
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    await act(async () => root!.render(<RollProbe score={7} gameId="g1" reduced={false} />));
    const el = () => host!.firstElementChild as HTMLElement;
    expect(el().dataset.val).toBe('7');
    expect(el().dataset.scoring).toBe('false');

    await act(async () => root!.render(<RollProbe score={14} gameId="g1" reduced={false} />));
    expect(el().dataset.val).toBe('8');
    expect(el().dataset.scoring).toBe('true');

    await act(async () => { vi.advanceTimersByTime(130 * 6); });
    expect(el().dataset.val).toBe('14');
    await act(async () => { vi.advanceTimersByTime(5300); });
    expect(el().dataset.scoring).toBe('false');

    await act(async () => root!.render(<RollProbe score={21} gameId="g1" reduced={true} />));
    expect(el().dataset.val).toBe('21');
    expect(el().dataset.scoring).toBe('false');
  });
});
