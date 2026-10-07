import { useEffect, useSyncExternalStore } from 'react';
import type { LineupReading, Snapshot } from '../data/contract';
import type { Provider } from '../data/provider';

export type LineupState = { reading?: LineupReading; error?: string };
export function createLineups(provider: Provider) {
  const entries = new Map<string, LineupState>();
  const requests = new Map<string, number>();
  const listeners = new Set<() => void>();
  const pending: LineupState = {};
  function publish(id: string, value: LineupState) {
    if (JSON.stringify(entries.get(id)) === JSON.stringify(value)) return;
    entries.set(id, value);
    for (const listener of listeners) listener();
  }
  function refresh(id: string) {
    const request = (requests.get(id) ?? 0) + 1;
    requests.set(id, request);
    return provider.lineup(id).then((reading) => {
      if (requests.get(id) === request) publish(id, { reading });
    }).catch((cause) => {
      if (requests.get(id) === request) {
        publish(id, {
          reading: entries.get(id)?.reading,
          error: cause instanceof Error ? cause.message : String(cause),
        });
      }
    });
  }
  return {
    refresh,
    peek: (id: string) => entries.get(id) ?? pending,
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
    onSeasonChange: provider.onSeasonChange.bind(provider),
  };
}
export function useLineup(store: ReturnType<typeof createLineups>, id: string, snapshot: Snapshot) {
  const state = useSyncExternalStore(store.subscribe, () => store.peek(id));
  useEffect(() => {
    const read = () => store.refresh(id);
    const stop = store.onSeasonChange(read);
    read();
    return stop;
  }, [store, id, snapshot]);
  return state;
}
