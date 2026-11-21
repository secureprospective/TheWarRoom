import { useEffect, useRef, useState } from 'react';
import type { Game, Play } from '../now/liveContract';
import { club, clubColours } from '../phone/liveClubs';
import { fieldPlay, freshPlays, latestSequence, PlayQueue, type FieldPlay } from '../phone/livePlays';
import { isSpotlightPlay, spotlightClub, spotlightTitle } from './wallPlays';

export const SPOTLIGHT_MS = 8_000;

export type SpotlightItem = {
  id: string;
  gameId: string;
  title: string;
  subtitle: string;
  text: string;
  tag: string;
  teamAb: string;
  color: string;
  fill: string;
  ink: string;
  kind: 'score' | 'turnover';
};

export function toSpotlightItem(play: Play, fp: FieldPlay, game: Game): SpotlightItem {
  const teamAb = spotlightClub(fp, game);
  const c = clubColours(club(teamAb));
  const q = play.period > 4 ? 'OT' : `Q${play.period}`;
  const score = `${game.away.abbreviation} ${play.awayScore} – ${play.homeScore} ${game.home.abbreviation}`;
  return {
    id: `${game.id}:${play.id}`,
    gameId: game.id,
    title: spotlightTitle(fp, game),
    subtitle: `${score} · ${q} ${play.clock}`,
    text: play.text,
    tag: fp.scoring === 'td' ? 'TOUCHDOWN' : fp.scoring === 'fg' ? 'FIELD GOAL'
      : fp.type === 'int' ? 'INTERCEPTION' : 'TURNOVER',
    teamAb,
    color: c.color,
    fill: c.fillc,
    ink: c.onc,
    kind: fp.scoring ? 'score' : 'turnover',
  };
}

export type SpotlightQueue = { push: (item: SpotlightItem) => void; close: () => void };

/** createSpotlightQueue runs wall spotlights strictly one at a time for `durationMs` each. */
export function createSpotlightQueue(
  onActive: (item: SpotlightItem | null) => void,
  durationMs = SPOTLIGHT_MS,
  limit = 8,
): SpotlightQueue {
  let closed = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let release: (() => void) | undefined;
  const q = new PlayQueue<SpotlightItem>(async item => {
    if (closed) return;
    onActive(item);
    await new Promise<void>(resolve => {
      release = resolve;
      timer = setTimeout(() => { timer = undefined; release = undefined; resolve(); }, durationMs);
    });
    if (!closed) onActive(null);
  }, limit);
  return {
    push(item: SpotlightItem) { if (!closed) q.push(item); },
    close() {
      closed = true;
      q.close();
      if (timer !== undefined) { clearTimeout(timer); timer = undefined; }
      release?.();
      release = undefined;
    },
  };
}

/** useWallSpotlight watches live games for newly arrived scoring plays and turnovers, queuing one
 *  spotlight at a time without replaying historical plays on first load. */
export function useWallSpotlight(games: Game[], durationMs = SPOTLIGHT_MS): SpotlightItem | null {
  const [active, setActive] = useState<SpotlightItem | null>(null);
  const seen = useRef<Record<string, number>>({});
  const queue = useRef<SpotlightQueue>();

  useEffect(() => {
    const q = createSpotlightQueue(setActive, durationMs);
    queue.current = q;
    return () => { q.close(); queue.current = undefined; };
  }, [durationMs]);

  useEffect(() => {
    for (const game of games) {
      const prior = seen.current[game.id];
      const latest = latestSequence(game);
      if (prior === undefined) {
        if (latest >= 0) seen.current[game.id] = latest;
        continue;
      }
      const arriving = freshPlays(prior, game, 6);
      if (!arriving.length) continue;
      seen.current[game.id] = latest;
      for (const play of arriving) {
        const fp = fieldPlay(play, game);
        if (isSpotlightPlay(fp)) queue.current?.push(toSpotlightItem(play, fp, game));
      }
    }
  }, [games]);

  return active;
}
