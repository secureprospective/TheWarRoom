import { parseRace, type Race } from './contract';
import type * as Bindings from '../../../wailsjs/go/race/API';

export type RaceReader = {
  read(week: number, team: string, mine: string): Promise<Race>; refresh(force: boolean): Promise<void>;
};
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League race reports require the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { race: { API: typeof Bindings } } }).go.race.API;
}
export const raceAPI: RaceReader = {
  read: async (week, team, mine) => parseRace(await bindings().TargetRace(week, team, mine)),
  refresh: async force => bindings().TargetRaceRefresh(force),
};

export function subscribeRace(changed: () => void): () => void {
  let disposed = false;
  let stop: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(m => {
    if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
      stop = m.EventsOn('target:season', changed);
  });
  return () => { disposed = true; stop?.(); };
}
