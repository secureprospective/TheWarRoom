import { useEffect, useSyncExternalStore } from 'react';
import type { LineupReading, Snapshot } from '../data/contract';
import type { Provider } from '../data/provider';

export type ReadingState<T> = { reading?: T; error?: string };
export type LineupState = ReadingState<LineupReading>;
export function createReadings<T>(provider: Provider, read: (id: string) => Promise<T>) {
  const entries = new Map<string, ReadingState<T>>();
  const requests = new Map<string, number>();
  const listeners = new Set<() => void>();
  const pending: ReadingState<T> = {};
  function publish(id: string, value: ReadingState<T>) {
    if (JSON.stringify(entries.get(id)) === JSON.stringify(value)) return;
    entries.set(id, value);
    for (const listener of listeners) listener();
  }
  function refresh(id: string) {
    const request = (requests.get(id) ?? 0) + 1;
    requests.set(id, request);
    return read(id).then((reading) => {
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
    onMovesChange: provider.onMovesChange.bind(provider),
  };
}
export function createLineups(provider: Provider) {
  return createReadings(provider, (id) => provider.lineup(id));
}
export function useReading<T>(
  store: ReturnType<typeof createReadings<T>>, id: string, snapshot: Snapshot,
) {
  const state = useSyncExternalStore(store.subscribe, () => store.peek(id));
  useEffect(() => {
    const read = () => store.refresh(id);
    const stop = store.onSeasonChange(read);
    const stopMoves = store.onMovesChange(read);
    read();
    return () => { stop(); stopMoves(); };
  }, [store, id, snapshot]);
  return state;
}

export const useLineup = useReading<LineupReading>;
