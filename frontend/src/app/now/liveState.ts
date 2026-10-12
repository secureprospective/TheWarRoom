import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { nowState } from './state';
import { parseLive, type LiveView } from './liveContract';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type LiveReader = (open: boolean, follow: string, mine: string) => Promise<LiveView>;
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Live games need the desktop app; no fixture is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
export function createLiveState(reader: LiveReader) {
  const store = createStore<{ view?: LiveView; follow: string; error: string; pending: boolean }>(() =>
    ({ follow: '', error: '', pending: false }));
  let revision = 0;
  async function load(open: boolean, mine: string) {
    if (store.getState().pending) return;
    const request = ++revision; store.setState({ pending: true });
    try {
      const view = await reader(open, store.getState().follow, mine);
      if (request === revision) store.setState({ view, error: '' });
    } catch (cause) { if (request === revision) store.setState({ error: String(cause) }); }
    finally { if (request === revision) store.setState({ pending: false }); }
  }
  function follow(id: string) {
    if (!/^(?:\d{1,20}|[A-Z]{2,3}-[A-Z]{2,3}|nfl\.g\.\d+)?$/.test(id))
      throw new Error('game.follow: invalid id');
    revision++; store.setState({ follow: id, pending: false });
  }
  return { use: () => useStore(store), peek: store.getState, load, follow };
}
export const liveState = createLiveState(async (open, follow, mine) =>
  parseLive(await bindings().TargetLiveGame(open, follow, mine)));

export function startLivePolling(mine: string,
  load = liveState.load, update = () => nowState.load(false)) {
  let disposed = false;
  async function poll() {
    if (document.hidden || disposed) return;
    await load(true, mine);
    if (!disposed) await update();
  }
  void poll();
  const timer = setInterval(() => { void poll(); }, 15000);
  document.addEventListener('visibilitychange', poll);
  return () => {
    disposed = true;
    clearInterval(timer);
    document.removeEventListener('visibilitychange', poll);
  };
}
