import { useSyncExternalStore } from 'react';
import type { Provider } from '../data/provider';
import type { WindowKind } from '../data/contract';

// editing: the owner asked to set a date on a window that has none, so its date box is shown.
type Result = { error?: string; pending?: boolean; editing?: boolean };
let results: Partial<Record<WindowKind, Result>> = {};
const listeners = new Set<() => void>();
const read = () => results;
const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
};
function write(kind: WindowKind, result: Result) {
  results = { ...results, [kind]: result };
  listeners.forEach(listener => listener());
}
export const useWindowResults = () => useSyncExternalStore(subscribe, read, read);
export async function setWindow(provider: Provider, kind: WindowKind, at: string) {
  if (results[kind]?.pending) return;
  write(kind, { pending: true });
  try {
    await provider.setWindow(kind, at);
    write(kind, {});
  } catch (cause) {
    write(kind, { error: cause instanceof Error ? cause.message : String(cause) });
  }
}
export function editWindow(kind: WindowKind) {
  write(kind, { ...results[kind], editing: true });
}
