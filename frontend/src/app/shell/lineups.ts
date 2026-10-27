import { useEffect, useSyncExternalStore } from 'react';
import type { LineupReading, Snapshot } from '../data/contract';
import type { Provider } from '../data/provider';
import { createReadQuery } from '../data/query';

type ReadingState<T> = { reading?: T; error?: string };
type HeldCommand = 'contracts.TargetLineup' | 'trades.TargetTrades' | 'control.TargetAlerts';
export function createReadings<T>(
  provider: Provider, read: (id: string) => Promise<T>, command: HeldCommand,
) {
  const entries = new Map<string, ReadingState<T>>();
  const listeners = new Set<() => void>();
  const pending: ReadingState<T> = {};
  function publish(id: string, value: ReadingState<T>) {
    if (JSON.stringify(entries.get(id)) === JSON.stringify(value)) return;
    entries.set(id, value);
    for (const listener of listeners) listener();
  }
  function createReading(id: string) {
    const query = createReadQuery(command, {
      initial: (): { reading?: T } => ({}), read,
      accept: reading => ({ reading }),
      failure: cause => cause instanceof Error ? cause.message : String(cause),
    });
    query.store.subscribe(state => {
      if (!state.pending) publish(id, { reading: state.reading,
        ...(state.error ? { error: state.error } : {}) });
    });
    return query;
  }
  const queries = new Map<string, ReturnType<typeof createReading>>();
  function refresh(id: string) {
    let query = queries.get(id);
    if (!query) { query = createReading(id); queries.set(id, query); }
    return query.load([id]);
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
  return createReadings(provider, (id) => provider.lineup(id), 'contracts.TargetLineup');
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
