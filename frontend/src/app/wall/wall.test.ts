import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Game, Play } from '../now/liveContract';
import { fieldPlay } from '../phone/livePlays';
import { createDemoFeed, demoBoard, wallGame } from './fixtures';
import { chooseWallLayout, TICKER_H, WALL_PAD } from './layout';
import { createSpotlightQueue, SPOTLIGHT_MS, toSpotlightItem } from './spotlight';
import { buildTicker, isSpotlightPlay, selectBigPlays, selectFirstDowns, spotlightTitle } from './wallPlays';

const asGame = (g: ReturnType<typeof wallGame>) => g as unknown as Game;

function makePlay(game: Game, overrides: Partial<Play> & { id: string; text: string }): Play {
  const start = overrides.start === undefined
    ? { possession: game.home.id, down: 2, distance: 7, yardLine: 30, toEndzone: 30 }
    : overrides.start;
  const end = overrides.end === undefined
    ? { possession: game.home.id, down: 1, distance: 10, yardLine: 10, toEndzone: 10 }
    : overrides.end;
  return {
    sequence: 10, clock: '6:00', period: 2, at: '2026-11-22T19:10:00Z',
    homeScore: game.home.score, awayScore: game.away.score,
    participants: [], ...overrides, start, end,
  };
}

describe('chooseWallLayout', () => {
  it('picks 1 row of 6 at 2560x1440 and 1920x1080, 3x2 at half-screen, 2x3 on narrow portrait', () => {
    const qhd = chooseWallLayout(6, 2560, 1440);
    expect(qhd).toMatchObject({ cols: 6, rows: 1, rail: 'bottom' });
    expect(qhd.scale).toBeGreaterThan(1);

    const fhd = chooseWallLayout(6, 1920, 1080);
    expect(fhd).toMatchObject({ cols: 6, rows: 1, rail: 'bottom' });
    expect(fhd.scale).toBeGreaterThan(0.7);

    const half = chooseWallLayout(6, 1280, 1400);
    expect(half).toMatchObject({ cols: 3, rows: 2, rail: 'right' });
    expect(half.railW).toBeGreaterThanOrEqual(240);

    const portrait = chooseWallLayout(6, 580, 1400);
    expect(portrait).toMatchObject({ cols: 2, rows: 3 });
  });

  it('grows phone size to fill larger viewports and never overflows viewport bounds', () => {
    const sizes: [number, number][] = [[1280, 720], [1920, 1080], [2560, 1440]];
    const layouts = sizes.map(([w, h]) => chooseWallLayout(6, w, h));
    expect(layouts[1].phoneW).toBeGreaterThan(layouts[0].phoneW);
    expect(layouts[2].phoneW).toBeGreaterThan(layouts[1].phoneW);
    for (const n of [1, 2, 3, 4, 6, 9]) {
      for (const [w, h] of [...sizes, [1280, 1400] as [number, number], [1000, 1080] as [number, number]]) {
        const l = chooseWallLayout(n, w, h);
        expect(l.gridW).toBeLessThanOrEqual(w - 2 * WALL_PAD + 0.5);
        expect(l.gridH).toBeLessThanOrEqual(h - TICKER_H - 2 * WALL_PAD + 0.5);
      }
    }
  });

  it('omits the rail when phones already fill the viewport tightly', () => {
    const four = chooseWallLayout(4, 1920, 1080);
    expect(four).toMatchObject({ cols: 4, rows: 1, rail: 'none' });
    expect(four.scale).toBeGreaterThan(chooseWallLayout(6, 1920, 1080).scale);
  });

  it('keeps the split bottom rail visible on S25 phone width (412x915)', () => {
    const phone = chooseWallLayout(6, 412, 915);
    expect(phone.rail).toBe('bottom');
    expect(phone.railH).toBeGreaterThanOrEqual(150);
  });
});

describe('spotlight formatting and queue', () => {
  afterEach(() => { vi.useRealTimers(); });

  it('identifies scoring plays and turnovers and formats headlines from held play text', () => {
    const g0 = asGame(wallGame(0));
    const td = makePlay(g0, {
      id: 'td1', text: 'KC pass short right to T. Kelce for 8 yards, TOUCHDOWN.',
      start: { possession: g0.home.id, down: 2, distance: 6, yardLine: 8, toEndzone: 8 },
      end: { possession: g0.away.id, down: 1, distance: 10, yardLine: 75, toEndzone: 75 },
      homeScore: 14, awayScore: 3,
    });
    const fpTd = fieldPlay(td, g0)!;
    expect(isSpotlightPlay(fpTd)).toBe(true);
    expect(spotlightTitle(fpTd, g0)).toBe('TOUCHDOWN KC — Kelce 8 yd');

    const pick = makePlay(g0, {
      id: 'int1', text: 'KC pass deep left intended for R. Rice INTERCEPTED by M. Milano at BUF 20.',
      start: { possession: g0.home.id, down: 3, distance: 8, yardLine: 45, toEndzone: 45 },
      end: { possession: g0.away.id, down: 1, distance: 10, yardLine: 20, toEndzone: 80 },
    });
    const fpInt = fieldPlay(pick, g0)!;
    expect(isSpotlightPlay(fpInt)).toBe(true);
    expect(spotlightTitle(fpInt, g0)).toBe('INTERCEPTION BUF — Milano');

    const punt = makePlay(g0, {
      id: 'p1', text: 'KC punts 48 yards to BUF 15.',
      start: { possession: g0.home.id, down: 4, distance: 7, yardLine: 63, toEndzone: 63 },
      end: { possession: g0.away.id, down: 1, distance: 10, yardLine: 15, toEndzone: 85 },
    });
    expect(isSpotlightPlay(fieldPlay(punt, g0))).toBe(false);

    const shortRun = makePlay(g0, {
      id: 'r1', text: 'KC rush up the middle for 4 yards.',
      start: { possession: g0.home.id, down: 1, distance: 10, yardLine: 50, toEndzone: 50 },
      end: { possession: g0.home.id, down: 2, distance: 6, yardLine: 46, toEndzone: 46 },
    });
    expect(isSpotlightPlay(fieldPlay(shortRun, g0))).toBe(false);
  });

  it('runs queued spotlights strictly one at a time for SPOTLIGHT_MS each', async () => {
    vi.useFakeTimers();
    const g0 = asGame(wallGame(0));
    const g1 = asGame(wallGame(1));
    const p1 = makePlay(g0, {
      id: 's1', text: 'KC pass short right to T. Kelce for 8 yards, TOUCHDOWN.',
      start: { possession: g0.home.id, down: 2, distance: 6, yardLine: 8, toEndzone: 8 },
      homeScore: 14, awayScore: 3,
    });
    const p2 = makePlay(g1, {
      id: 's2', text: 'PHI 52 yard field goal is GOOD.',
      start: { possession: g1.home.id, down: 4, distance: 5, yardLine: 34, toEndzone: 34 },
      homeScore: 13, awayScore: 6,
    });
    const seen: (string | null)[] = [];
    const q = createSpotlightQueue(item => seen.push(item ? item.title : null), SPOTLIGHT_MS);
    q.push(toSpotlightItem(p1, fieldPlay(p1, g0)!, g0));
    q.push(toSpotlightItem(p2, fieldPlay(p2, g1)!, g1));

    await Promise.resolve();
    expect(seen).toEqual(['TOUCHDOWN KC — Kelce 8 yd']);

    await vi.advanceTimersByTimeAsync(SPOTLIGHT_MS - 1);
    expect(seen).toEqual(['TOUCHDOWN KC — Kelce 8 yd']);

    await vi.advanceTimersByTimeAsync(1);
    expect(seen).toEqual(['TOUCHDOWN KC — Kelce 8 yd', null, 'FIELD GOAL PHI — 52 yd']);

    await vi.advanceTimersByTimeAsync(SPOTLIGHT_MS);
    expect(seen).toEqual(['TOUCHDOWN KC — Kelce 8 yd', null, 'FIELD GOAL PHI — 52 yd', null]);
    q.close();
  });
});

describe('selectBigPlays', () => {
  it('picks newest 8 scoring, turnover, and 20+ yd plays with club colour edge', () => {
    const g0 = asGame(wallGame(0));
    const g1 = asGame(wallGame(1));
    const base = Date.parse('2026-11-22T20:00:00Z');
    const at = (min: number) => new Date(base + min * 60_000).toISOString();
    const plays0: Play[] = [
      makePlay(g0, { id: 'short', sequence: 1, at: at(1), text: 'KC rush up the middle for 5 yards',
        start: { possession: g0.home.id, down: 1, distance: 10, yardLine: 60, toEndzone: 60 },
        end: { possession: g0.home.id, down: 2, distance: 5, yardLine: 55, toEndzone: 55 } }),
      makePlay(g0, { id: 'pen25', sequence: 2, at: at(2),
        text: 'PENALTY on BUF-J. Allen, Defensive Pass Interference, 25 yards, enforced at KC 40.',
        start: { possession: g0.home.id, down: 2, distance: 5, yardLine: 60, toEndzone: 60 },
        end: { possession: g0.home.id, down: 1, distance: 10, yardLine: 35, toEndzone: 35 } }),
      makePlay(g0, { id: 'g24', sequence: 3, at: at(3), text: 'KC pass deep middle to R. Rice for 24 yards',
        start: { possession: g0.home.id, down: 1, distance: 10, yardLine: 60, toEndzone: 60 },
        end: { possession: g0.home.id, down: 1, distance: 10, yardLine: 36, toEndzone: 36 } }),
      makePlay(g0, { id: 'td8', sequence: 4, at: at(10.5),
        text: 'KC pass short right to T. Kelce for 8 yards, TOUCHDOWN.',
        start: { possession: g0.home.id, down: 2, distance: 4, yardLine: 8, toEndzone: 8 },
        end: { possession: g0.away.id, down: 1, distance: 10, yardLine: 75, toEndzone: 75 } }),
    ];
    const plays1: Play[] = Array.from({ length: 8 }, (_, idx) => makePlay(g1, {
      id: `p-${idx}`, sequence: idx + 1, at: at(4 + idx),
      text: idx === 7
        ? 'DAL pass short right intended for C. Lamb INTERCEPTED by D. Slay at PHI 22.'
        : `PHI pass deep left to A. Brown for ${21 + idx} yards`,
      start: { possession: idx === 7 ? g1.away.id : g1.home.id, down: 2, distance: 8,
        yardLine: 50, toEndzone: 50 },
      end: { possession: g1.home.id, down: 1, distance: 10, yardLine: 25, toEndzone: 25 },
    }));
    const list = selectBigPlays([
      { ...g0, plays: plays0, last: plays0[plays0.length - 1] },
      { ...g1, plays: plays1, last: plays1[plays1.length - 1] },
    ], 8);
    expect(list).toHaveLength(8);
    expect(list[0].playId).toBe('p-7');
    expect(list[0].kind).toBe('turnover');
    expect(list[1].playId).toBe('td8');
    expect(list[1].kind).toBe('td');
    expect(list.some(p => p.playId === 'short' || p.playId === 'pen25')).toBe(false);
    expect(list.every(p => p.color.startsWith('rgb('))).toBe(true);
  });

  it('splits a mix of plays into big plays and normal first downs, newest first', () => {
    const g0 = asGame(wallGame(0));
    const base = Date.parse('2026-11-22T20:00:00Z');
    const at = (min: number) => new Date(base + min * 60_000).toISOString();
    const plays: Play[] = [
      makePlay(g0, { id: 'fd-old', sequence: 1, at: at(1),
        text: 'KC rush up the middle I. Pacheco for 6 yards',
        start: { possession: g0.home.id, down: 3, distance: 4, yardLine: 60, toEndzone: 60 },
        end: { possession: g0.home.id, down: 1, distance: 10, yardLine: 54, toEndzone: 54 } }),
      makePlay(g0, { id: 'short-no-fd', sequence: 2, at: at(2),
        text: 'KC rush left guard I. Pacheco for 3 yards',
        start: { possession: g0.home.id, down: 1, distance: 10, yardLine: 54, toEndzone: 54 },
        end: { possession: g0.home.id, down: 2, distance: 7, yardLine: 51, toEndzone: 51 } }),
      makePlay(g0, { id: 'big-old', sequence: 3, at: at(3),
        text: 'KC pass deep middle to R. Rice for 24 yards',
        start: { possession: g0.home.id, down: 2, distance: 7, yardLine: 51, toEndzone: 51 },
        end: { possession: g0.home.id, down: 1, distance: 10, yardLine: 27, toEndzone: 27 } }),
      makePlay(g0, { id: 'fd-new', sequence: 4, at: at(4),
        text: 'KC pass short right to T. Kelce for 11 yards',
        start: { possession: g0.home.id, down: 1, distance: 10, yardLine: 27, toEndzone: 27 },
        end: { possession: g0.home.id, down: 1, distance: 10, yardLine: 16, toEndzone: 16 } }),
      makePlay(g0, { id: 'big-new', sequence: 5, at: at(5),
        text: 'KC pass short right to T. Kelce for 16 yards, TOUCHDOWN.',
        start: { possession: g0.home.id, down: 1, distance: 10, yardLine: 16, toEndzone: 16 },
        end: { possession: g0.away.id, down: 1, distance: 10, yardLine: 75, toEndzone: 75 } }),
    ];
    const game: Game = { ...g0, plays, last: plays[plays.length - 1] };
    const big = selectBigPlays([game], 8);
    const firsts = selectFirstDowns([game], 16);
    expect(big.map(p => p.playId)).toEqual(['big-new', 'big-old']);
    expect(firsts.map(p => p.playId)).toEqual(['fd-new', 'fd-old']);
    expect(firsts.every(p => p.tag === '1ST DOWN')).toBe(true);
  });
});

describe('buildTicker and demo feed', () => {
  it('includes every game on the board with score, clock and possession, live first', () => {
    const board = demoBoard().map(asGame);
    const items = buildTicker(board);
    expect(items).toHaveLength(12);
    expect(items.slice(0, 8).every(i => i.status === 'in')).toBe(true);
    expect(items.slice(8, 10).every(i => i.status === 'pre')).toBe(true);
    expect(items.slice(10, 12).every(i => i.status === 'post')).toBe(true);
    expect(items[0]).toMatchObject({ awayAb: 'BUF', homeAb: 'KC', poss: 'R', clock: 'Q2 8:42' });
    expect(items[8].poss).toBeUndefined();
    expect(items[10].clock).toBe('FINAL');
    expect(items[10].poss).toBeUndefined();
  });

  it('demo feed emits scripted scoring and turnover plays on advance', () => {
    const feed = createDemoFeed(0);
    const pings = vi.fn();
    const off = feed.subscribe(pings);
    const before = feed.games[0].home.score;
    feed.advance();
    expect(pings).toHaveBeenCalledTimes(1);
    expect(feed.games[0].home.score).toBe(before + 7);
    expect(feed.games[0].last?.text).toContain('Kelce');
    expect(feed.games[0].last?.text).toContain('TOUCHDOWN');
    const newest = selectBigPlays(feed.games.map(asGame), 8);
    expect(newest[0].headline).toBe('TOUCHDOWN KC — Kelce 8 yd');
    off();
  });
});
