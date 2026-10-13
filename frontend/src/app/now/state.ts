import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import { parseNow, type Now } from './contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type NowReader = {
  read(week: number, reportWeek: number, mine: string): Promise<Now>;
  refresh(week: number, force: boolean): Promise<void>;
};
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League Pulse needs the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
const live: NowReader = {
  read: async (week, reportWeek, mine) => parseNow(await bindings().TargetNow(week, reportWeek, mine)),
  refresh: async (week, force) => bindings().TargetNowRefresh(week, force),
};

// The week is a request; Go validates it and the board echoes the week it shows.
export function createNowState(reader: NowReader, mine: () => string) {
  type State = { now?: Now; pending: boolean; error: string; week: number; reportWeek: number };
  const store = createStore<State>(() => ({ pending: false, error: '', week: 0, reportWeek: 0 }));
  let revision = 0;
  async function load(refresh: boolean, force = false) {
    const request = ++revision;
    store.setState({ pending: true, error: '' });
    try {
      const { week, reportWeek } = store.getState();
      if (refresh) await reader.refresh(week, force);
      const now = await reader.read(week, reportWeek, mine());
      if (request === revision) store.setState({ now, week: now.week });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
  function choose(action: 'refresh' | 'week' | 'reports-week', value: string) {
    if (action === 'refresh') return load(true, true);
    if (!/^\d{1,2}$/.test(value)) throw new Error('League Pulse: invalid week');
    store.setState(action === 'reports-week' ? { reportWeek: Number(value) } : { week: Number(value) });
    return load(true);
  }
  return { use: () => useStore(store), peek: store.getState, load, choose };
}
export const nowState = createNowState(live, () => commands.read().franchiseId ?? '');
