import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import { parseRace, type Race } from './contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type RaceReader = {
  read(week: number, team: string, mine: string): Promise<Race>; refresh(force: boolean): Promise<void>;
};
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League race reports require the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
const live: RaceReader = {
  read: async (week, team, mine) => parseRace(await bindings().TargetRace(week, team, mine)),
  refresh: async force => bindings().TargetRaceRefresh(force),
};

// Week and team are requests; Go validates them and the board echoes what it accepted.
export function createRaceState(reader: RaceReader, mine: () => string) {
  type State = { race?: Race; pending: boolean; error: string; week: number; team: string | null };
  const store = createStore<State>(() => ({ pending: false, error: '', week: 0, team: null }));
  let revision = 0;
  async function load(refresh: boolean, force = false) {
    const request = ++revision;
    store.setState({ pending: true, error: '' });
    try {
      if (refresh) await reader.refresh(force);
      const { week, team } = store.getState();
      const race = await reader.read(week, team ?? mine(), mine());
      if (request === revision) store.setState({ race, week: race.week, team: race.team });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
  function choose(action: 'refresh' | 'week' | 'team', value: string) {
    if (action === 'refresh') return load(true, true);
    if (action === 'week') {
      if (!/^\d{1,2}$/.test(value)) throw new Error('League race: invalid week');
      store.setState({ week: Number(value) });
    } else {
      if (!/^\d{4}$/.test(value)) throw new Error('League race: invalid franchise');
      store.setState({ team: value });
    }
    return load(false);
  }
  return { use: () => useStore(store), peek: store.getState, load, choose };
}
export const raceState = createRaceState(live, () => commands.read().franchiseId ?? '');
