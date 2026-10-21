import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import type { Comms } from './contract';
import { commsAPI, type CommsReader } from './api';

export function createCommsState(reader: CommsReader) {
  type State = { comms?: Comms; pending: boolean; error: string; team: string; kind: string; limit: number };
  const store = createStore<State>(() => ({ pending: false, error: '', team: '', kind: '', limit: 50 }));
  let revision = 0;
  async function load(refresh = false) {
    const request = ++revision;
    store.setState({ pending: true, error: '' });
    try {
      if (refresh) await reader.refresh(true);
      const { team, kind } = store.getState();
      const comms = await reader.read(team, kind);
      if (request === revision) store.setState({ comms, team: comms.team, kind: comms.kind });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
  async function choose(action: 'refresh' | 'team' | 'kind' | 'more' | 'open', value: string) {
    if (action === 'open') { await reader.open(value); return; }
    if (action === 'more') { store.setState(s => ({ limit: s.limit + 50 })); return; }
    if (action === 'refresh') { await load(true); return; }
    store.setState({ [action]: value, limit: 50 });
    await load();
  }
  return { use: () => useStore(store), peek: store.getState, load, choose };
}
export const commsState = createCommsState(commsAPI);
