import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import { parseNow, type Now } from './contract';
import { targetNow, targetNowRefresh } from './api';

export type NowReader = {
  read(week: number, reportWeek: number, mine: string): Promise<Now>;
  refresh(week: number, force: boolean): Promise<void>;
};
const live: NowReader = {
  read: async (week, reportWeek, mine) => parseNow(await targetNow(week, reportWeek, mine)),
  refresh: async (week, force) => targetNowRefresh(week, force),
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
      if (refresh) {
        await reader.refresh(week, force);
        if (reportWeek !== 0 && reportWeek !== week) await reader.refresh(reportWeek, force);
      }
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
