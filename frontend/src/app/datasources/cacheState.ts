import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { sourcesAPI } from './api';
import { array, object, text } from '../data/parseValues';
import { parseOutcome, type Outcome, type Scope } from './provider';

// The held copies Go can read afresh (TargetCacheScopes) and the answer to asking for one.
type CacheProvider = {
  scopes: () => Promise<Scope[]>;
  clear: (scope: string) => Promise<Outcome>;
};
export function parseCacheScopes(v: unknown): Scope[] {
  return array(v, 'Held copies', (s, p) => {
    const o = object(s, p, ['id', 'label', 'note', 'action', 'progress']);
    return { id: text(o.id, p), label: text(o.label, p), note: text(o.note, p), action: text(o.action, p),
      progress: text(o.progress, p) };
  });
}
function desktop() {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Reading data again needs the desktop app. No sample data is substituted.');
}
const cacheProvider: CacheProvider = {
  scopes: async () => { desktop(); return parseCacheScopes(await sourcesAPI.cacheScopes()); },
  clear: async scope => { desktop(); return parseOutcome(await sourcesAPI.cacheClear(scope)); },
};

function createCacheState(provider: CacheProvider) {
  type Held = { scopes: Scope[]; busy: string; error: string; outcomes: Record<string, string> };
  const store = createStore<Held>(() => ({ scopes: [], busy: '', error: '', outcomes: {} }));
  async function load() {
    try {
      store.setState({ scopes: await provider.scopes(), error: '' });
    } catch (cause) {
      store.setState({ error: String(cause) });
    }
  }
  async function run(scope: string) {
    if (store.getState().busy) return;
    store.setState(s => ({ busy: scope, error: '', outcomes: { ...s.outcomes, [scope]: '' } }));
    try {
      const result = await provider.clear(scope);
      const said = result.ok ? result.summary : result.error;
      store.setState(s => ({ outcomes: { ...s.outcomes, [scope]: said } }));
    } catch (cause) {
      store.setState(s => ({ outcomes: { ...s.outcomes, [scope]: String(cause) } }));
    } finally {
      store.setState({ busy: '' });
    }
  }
  const forget = () => store.setState({ scopes: [], busy: '', error: '', outcomes: {} });
  return { use: () => useStore(store), peek: store.getState, load, run, forget };
}
export const cacheState = createCacheState(cacheProvider);
