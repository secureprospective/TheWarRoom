// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Game } from '../now/liveContract';
import { fakeWallPort, wallGame } from './fixtures';
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
  it('lays six phones as 3x2 and names the next kickoff in Central', () => {
    expect(wallGrid(6)).toMatchObject({ cols: 3, rows: 2 });
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
  it('draws one phone per live game, up to ?n=', async () => {
    const games = [0, 1, 2, 3, 4, 5].map(i => wallGame(i));
    expect((await show(games)).querySelectorAll('.wall__tile')).toHaveLength(6);
    await act(async () => root!.unmount()); host!.remove(); root = undefined;
    expect((await show(games, '?n=2')).querySelectorAll('.wall__tile')).toHaveLength(2);
  });
});
