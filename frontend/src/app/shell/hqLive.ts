import { useEffect, useMemo } from 'react';
import { createLiveState, startLivePolling } from '../now/liveState';
import { parseLive, type LiveView } from '../now/liveContract';
import { targetLiveGame } from '../now/api';

export function useHQLive(mine: string) {
  const store = useMemo(() => createLiveState(async (open, follow, id) =>
    parseLive(await targetLiveGame(open, follow, id))), [mine]);
  const state = store.use();
  useEffect(() => startLivePolling(mine, store.load, async () => {}), [mine, store]);
  return state.error || state.view?.failure ? undefined : state.view;
}
export function liveTeam(team?: string, view?: LiveView): boolean {
  return Boolean(team && view?.games.some(g => g.status === 'in' &&
    [g.home.abbreviation, g.away.abbreviation].includes(team)));
}

/** usePulseLive is the same held NFL read, with a game the viewer can follow. Following asks for that
 *  game's plays at once instead of waiting for the next poll. */
export function usePulseLive(mine: string) {
  const store = useMemo(() => createLiveState(async (open, follow, id) =>
    parseLive(await targetLiveGame(open, follow, id))), [mine]);
  const state = store.use();
  useEffect(() => startLivePolling(mine, store.load, async () => {}), [mine, store, state.follow]);
  const view = state.view;
  useEffect(() => {
    // Nothing followed yet: follow the first game being played, so the field is never empty mid-game.
    const first = view && !state.follow && !view.followed && view.games.find(g => g.status === 'in');
    if (first) store.follow(`${first.away.abbreviation}-${first.home.abbreviation}`);
  }, [view, state.follow, store]);
  return { view, error: state.error || view?.failure || '',
    follow: (id: string) => store.follow(id) };
}
