import { useEffect, useState } from 'react';
import type { HostedPort } from '../commissioner/hostedRoom';
import { parseHostedGame, parseHostedGames, type Game } from '../now/liveContract';
import { useNow } from '../shell/hostedGames';
import { pickGames } from './pickGames';

export type WallRead = {
  board?: Game[];
  detail: Record<string, Game>;
  error: string;
  picked: Game[];
  allGames: Game[];
};

/** Board re-read cadence when no change topic arrives (LiveTab reloads only on topics; the wall adds a
 *  slow fallback because nobody taps it). */
export const wallPollMs = 20_000;

/** useWallGames reads the NFL board, then the detail of each picked game, the way LiveTab does
 *  (`nfl.games.read`, then `nfl.game.read` per game), reloading on every change topic and on a timer. */
export function useWallGames(league: string, port: HostedPort, cap: number,
  onChange: (load: () => void) => () => void, pollMs = wallPollMs): WallRead {
  const [held, setHeld] = useState<Omit<WallRead, 'picked' | 'allGames'>>({ detail: {}, error: '' });
  const now = useNow(30_000);
  useEffect(() => {
    let live = true;
    let generation = 0;
    const load = async () => {
      const request = ++generation;
      const current = () => live && request === generation;
      try {
        const reply = await port({ command: 'nfl.games.read', mode: 'read', arguments: { league } });
        const board = parseHostedGames(reply.data, league).games;
        if (current()) setHeld(prior => ({ ...prior, board, error: '' }));
        const chosen = pickGames(board, cap, Date.now());
        const found = await Promise.all(chosen.map(async g => {
          const r = await port({ command: 'nfl.game.read', mode: 'read', arguments: { league, game: g.id } });
          return parseHostedGame(r.data, league, g.id) ?? g;
        }));
        if (current()) setHeld({ board, error: '', detail: Object.fromEntries(found.map(g => [g.id, g])) });
      } catch (e: unknown) {
        const error = e instanceof Error ? e.message : String(e);
        if (current()) setHeld(prior => ({ ...prior, error }));
      }
    };
    void load();
    const off = onChange(() => { void load(); });
    const timer = setInterval(() => { void load(); }, pollMs);
    return () => { live = false; off(); clearInterval(timer); };
  }, [league, port, cap, onChange, pollMs]);
  const allGames = (held.board ?? []).map(g => held.detail[g.id] ?? g);
  const picked = pickGames(held.board ?? [], cap, now).map(g => held.detail[g.id] ?? g);
  return { ...held, picked, allGames };
}
