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
