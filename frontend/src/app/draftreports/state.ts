import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { provider, type Board, type ReportProvider } from './provider';

export function createReportState(reader: ReportProvider) {
  const store = createStore<{ board?: Board; pending: boolean; error: string }>(() => ({
    pending: false, error: '',
  }));
  let revision = 0;
  async function refresh(force = false) {
    const request = ++revision;
    store.setState({ pending: true, error: '' });
    try {
      await reader.refresh(force);
      const board = await reader.read();
      if (request === revision) store.setState({ board });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
  async function action(action: 'refresh' | 'mfl') {
    if (action === 'refresh') return refresh(true);
    store.setState({ error: '' });
    try { await reader.openMFL(); }
    catch (cause) { store.setState({ error: String(cause) }); }
  }
  return { use: () => useStore(store), peek: store.getState, refresh, action };
}
export const reportState = createReportState(provider);
