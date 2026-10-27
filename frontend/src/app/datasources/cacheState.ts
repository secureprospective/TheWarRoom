import { createReadQuery } from '../data/query';
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
  const query = createReadQuery('sources.TargetCacheScopes', {
    initial: (): Held => ({ scopes: [], busy: '', error: '', outcomes: {} }),
    read: provider.scopes, accept: scopes => ({ scopes, error: '' }),
  });
  const store = query.store;
  const load = () => query.load([]);
  async function run(scope: string) {
    if (store.getState().busy) return;
    store.setState(s => ({ busy: scope, error: '', outcomes: { ...s.outcomes, [scope]: '' } }));
    try {
      const result = await provider.clear(scope);
      const said = result.ok ? result.summary : result.error;
      store.setState(s => ({ outcomes: { ...s.outcomes, [scope]: said },
        error: result.ok ? '' : result.error }));
    } catch (cause) {
      store.setState(s => ({ outcomes: { ...s.outcomes, [scope]: String(cause) }, error: String(cause) }));
    } finally {
      store.setState({ busy: '' });
    }
  }
  const forget = query.reset;
  return { use: query.use, peek: (): Held => store.getState(), load, run, forget };
}
export const cacheState = createCacheState(cacheProvider);
