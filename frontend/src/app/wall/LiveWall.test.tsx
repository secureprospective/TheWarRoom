// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Game } from '../now/liveContract';
import { createDemoFeed, demoBoard, fakeWallPort, wallGame } from './fixtures';
import { LiveWall } from './LiveWall';
import { nextKickoff, pickGames, wallCount, wallGrid } from './pickGames';

const NOW = Date.parse('2026-11-22T17:00:00Z');
const g = (i: number, status: 'in' | 'pre' | 'post', kickoff: string) =>
  wallGame(i, status, kickoff) as unknown as Game;

describe('pickGames', () => {
  it('takes live games first, then games about to start, soonest first', () => {
    const games = [g(0, 'pre', '2026-11-22T20:00:00Z'), g(1, 'in', '2026-11-22T16:00:00Z'),
      g(2, 'pre', '2026-11-22T18:00:00Z'), g(3, 'post', '2026-11-22T10:00:00Z')];
    expect(pickGames(games, 6, NOW).map(x => x.id)).toEqual(['401', '402', '400']);
  });
  it('skips games that are days away and caps the count', () => {
    const games = [g(0, 'pre', '2026-11-29T20:00:00Z'),
      ...[1, 2, 3, 4].map(i => g(i, 'in', '2026-11-22T16:00:00Z'))];
    expect(pickGames(games, 6, NOW)).toHaveLength(4);
    expect(pickGames(games, 2, NOW)).toHaveLength(2);
  });
  it('reads ?n= from 1 to 9 and falls back to six', () => {
    expect([wallCount('?n=3'), wallCount('?n=9'), wallCount('?n=0'), wallCount('?n=10'), wallCount('?n=x'),
      wallCount('')]).toEqual([3, 9, 6, 6, 6, 6]);
  });
  it('picks fluid grid dimensions for viewport and names next kickoff in Central', () => {
    expect(wallGrid(6, 1280, 1400)).toMatchObject({ cols: 3, rows: 2, rail: 'right' });
    expect(wallGrid(6, 2560, 1440)).toMatchObject({ cols: 6, rows: 1, rail: 'bottom' });
    expect(wallGrid(4, 1920, 1080).scale).toBeGreaterThan(wallGrid(6, 1920, 1080).scale);
    expect(nextKickoff([g(0, 'pre', '2026-11-22T19:00:00Z')], NOW))
      .toBe('Next kickoff Sun, Nov 22, 1:00 PM CT');
  });
});

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined; vi.restoreAllMocks(); vi.useRealTimers();
});

async function show(games: ReturnType<typeof wallGame>[], search = '') {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<LiveWall league="league" port={fakeWallPort(games)}
    subscribe={() => () => {}} search={search} />));
  await act(async () => { await new Promise(r => setTimeout(r, 0)); });
  return host;
}

describe('LiveWall', () => {
  it('says so, with the next kickoff in Central, when no game is live', async () => {
    vi.spyOn(Date, 'now').mockReturnValue(NOW);
    const el = await show([wallGame(0, 'pre', '2026-11-29T19:00:00Z')]);
    expect(el.textContent).toContain('No NFL games live right now');
    expect(el.textContent).toContain('Next kickoff Sun, Nov 29, 1:00 PM CT');
    expect(el.querySelectorAll('.wall__tile')).toHaveLength(0);
  });
  it('draws one phone per live game, up to ?n=, with up to 6 plays newest first', async () => {
    const games = [0, 1, 2, 3, 4, 5].map(i => wallGame(i));
    const el = await show(games);
    expect(el.querySelectorAll('.wall__tile')).toHaveLength(6);
    const firstPlays = el.querySelector('.wall__tile')?.querySelectorAll('.live6__plays li');
    expect(firstPlays).toHaveLength(6);
    expect(firstPlays?.[0]?.textContent).toContain('N. Gray');
    await act(async () => root!.unmount()); host!.remove(); root = undefined;
    expect((await show(games, '?n=2')).querySelectorAll('.wall__tile')).toHaveLength(2);
  });
  it('shows no-plays note when a live game holds no plays', async () => {
    const empty = { ...wallGame(0), plays: [], last: null } as unknown as ReturnType<typeof wallGame>;
    const el = await show([empty]);
    expect(el.querySelector('.live6__plays')).toBeNull();
    expect(el.textContent).toContain('No plays are held for this game yet.');
  });
  it('shows every board game in the ticker even when phones are capped, plus Big Plays rail', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 1920, configurable: true });
    Object.defineProperty(window, 'innerHeight', { value: 1080, configurable: true });
    const el = await show(demoBoard(), '?n=4');
    expect(el.querySelectorAll('.wall__tile')).toHaveLength(4);
    const tickerItems = el.querySelectorAll('[data-testid="wall-ticker-list"] > li');
    expect(tickerItems).toHaveLength(12);
    expect(el.querySelector('.wall__mark')?.textContent).toBe('THE WAR ROOM');

    await act(async () => root!.unmount()); host!.remove(); root = undefined;
    const withRail = await show(demoBoard());
    const rail = withRail.querySelector('[data-testid="wall-big-plays"]');
    expect(rail).not.toBeNull();
    expect(rail!.querySelectorAll('.wall__bp').length).toBeGreaterThan(0);
    expect(rail!.querySelectorAll('.wall__bp').length).toBeLessThanOrEqual(8);
  });
  it('spotlights a scoring play when a new demo play arrives and highlights that phone tile', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    const feed = createDemoFeed(0);
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    await act(async () => root!.render(<LiveWall league="league" port={feed.port}
      subscribe={feed.subscribe} search="" />));
    await act(async () => { await new Promise(r => setTimeout(r, 0)); });
    expect(host.querySelector('[data-testid="wall-spotlight"]')).toBeNull();

    await act(async () => {
      feed.advance();
      await new Promise(r => setTimeout(r, 20));
    });
    const banner = host.querySelector('[data-testid="wall-spotlight"]');
    expect(banner?.textContent).toContain('TOUCHDOWN KC — Kelce 8 yd');
    const spotTile = host.querySelector('.wall__tile--spot');
    expect(spotTile?.getAttribute('data-game-id')).toBe('400');
  });

  it('renders a 50/50 split rail with big plays on top and first downs on bottom, newest first', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 412, configurable: true });
    Object.defineProperty(window, 'innerHeight', { value: 915, configurable: true });
    const base = Date.parse('2026-11-22T20:00:00Z');
    const at = (min: number) => new Date(base + min * 60_000).toISOString();
    const games = demoBoard().slice(0, 6);
    const g0 = games[0];
    const h = g0.home.id, a = g0.away.id;
    const st = (p: string, down: number, distance: number, toEndzone: number) =>
      ({ possession: p, down, distance, yardLine: toEndzone, toEndzone });
    g0.plays = [
      { id: 'fd-1', sequence: 1, text: 'KC rush up the middle I. Pacheco for 6 yards', clock: '9:00',
        period: 2, at: at(10), homeScore: 7, awayScore: 3,
        start: st(h, 3, 4, 60), end: st(h, 1, 10, 54), participants: [] },
      { id: 'short-1', sequence: 2, text: 'KC rush left guard I. Pacheco for 2 yards', clock: '8:30',
        period: 2, at: at(11), homeScore: 7, awayScore: 3,
        start: st(h, 1, 10, 54), end: st(h, 2, 8, 52), participants: [] },
      { id: 'big-1', sequence: 3, text: 'KC pass deep middle to R. Rice for 25 yards', clock: '8:00',
        period: 2, at: at(12), homeScore: 7, awayScore: 3,
        start: st(h, 2, 8, 52), end: st(h, 1, 10, 27), participants: [] },
      { id: 'fd-2', sequence: 4, text: 'KC pass short right to T. Kelce for 12 yards', clock: '7:30',
        period: 2, at: at(13), homeScore: 7, awayScore: 3,
        start: st(h, 1, 10, 27), end: st(h, 1, 10, 15), participants: [] },
      { id: 'big-2', sequence: 5, text: 'KC pass short left to N. Gray for 15 yards, TOUCHDOWN.',
        clock: '7:00', period: 2, at: at(14), homeScore: 14, awayScore: 3,
        start: st(h, 1, 10, 15), end: st(a, 1, 10, 75), participants: [] },
    ];
    g0.last = g0.plays[g0.plays.length - 1];
    for (const gm of games.slice(1)) { gm.plays = []; gm.last = null; }

    const el = await show(games);
    const bigHalf = el.querySelector('[data-testid="wall-rail-big"]');
    const firstHalf = el.querySelector('[data-testid="wall-rail-first"]');
    expect(bigHalf).not.toBeNull();
    expect(firstHalf).not.toBeNull();

    const bigTitles = [...bigHalf!.querySelectorAll('.wall__bp-title')].map(n => n.textContent);
    const firstTitles = [...firstHalf!.querySelectorAll('.wall__bp-title')].map(n => n.textContent);
    expect(bigTitles).toEqual(['TOUCHDOWN KC — Gray 15 yd', 'KC · Rice · 25 yd catch']);
    expect(firstTitles).toEqual(['KC · Kelce · 12 yd catch', 'KC · Pacheco · 6 yd run']);
    expect(el.querySelector('[data-testid="wall-big-plays"]')?.textContent).not.toContain('for 2 yards');
  });
});
