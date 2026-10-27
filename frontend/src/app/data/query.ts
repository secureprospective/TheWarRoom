import { createStore, type StoreApi } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { readCommands, type ReadArguments, type ReadCommandId } from '../commands/reads.gen';
import { validateShape } from '../commands/shape';

export type ReadStatus = { pending: boolean; error: string };
type LoadOptions = { before?: () => Promise<void>; policy?: 'revalidate' | 'cache-first' };

// Object order is not scope; array order is. Never share a cache between reader instances.
function canonical(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(canonical);
  if (value !== null && typeof value === 'object') return Object.fromEntries(
    Object.entries(value).filter(([, item]) => item !== undefined)
      .sort(([a], [b]) => a.localeCompare(b)).map(([key, item]) => [key, canonical(item)]),
  );
  return value;
}

export function readKey<K extends ReadCommandId>(command: K, args: ReadArguments<K>): string {
  const definition = readCommands[command];
  if (!definition) throw new Error('Unknown held read command');
  const fields = definition.arguments.Fields;
  if (args.length !== fields.length) throw new Error(`${command}: wrong argument count`);
  fields.forEach((field, index) => validateShape(args[index], field.Schema, `${command}.${field.Name}`));
  return JSON.stringify([definition.command, command, canonical(args)]);
}

type QueryOptions<T, S extends object, A extends unknown[]> = {
  initial: () => S;
  store?: StoreApi<S & ReadStatus>;
  companions?: ReadCommandId[];
  cacheSize?: 1 | 32;
  retainErrorWhilePending?: boolean;
  read: (...args: A) => Promise<T>;
  accept: (value: T, args: A) => Partial<S>;
};
type ReadQuery<A extends unknown[], S extends object> = {
  store: StoreApi<S & ReadStatus>;
  use: () => S & ReadStatus;
  peek: StoreApi<S & ReadStatus>['getState'];
  set: StoreApi<S & ReadStatus>['setState'];
  load: (input: A | (() => A), config?: LoadOptions) => Promise<void>;
  fence: () => void;
  invalidate: () => void;
  reset: () => void;
};

// A wrapper may project its parsed scope to the existing binding arguments, never a new schema.
export function createReadQuery<K extends ReadCommandId, T, S extends object>(command: K,
  options: QueryOptions<T, S, ReadArguments<K>> & { arguments?: never }): ReadQuery<ReadArguments<K>, S>;
export function createReadQuery<K extends ReadCommandId, T, S extends object, A extends unknown[]>(command: K,
  options: QueryOptions<T, S, A> & { arguments: (args: A) => ReadArguments<K> }): ReadQuery<A, S>;
export function createReadQuery<K extends ReadCommandId, T, S extends object, A extends unknown[]>(command: K,
  options: QueryOptions<T, S, A> & { arguments?: (args: A) => ReadArguments<K> }): ReadQuery<A, S> {
  const initial = () => ({ ...options.initial(), pending: false, error: '' });
  const store = options.store ?? createStore<S & ReadStatus>(initial);
  const cache = new Map<string, T>();
  let revision = 0;

  function key(input: A) {
    const args = options.arguments
      ? options.arguments(input) : input as unknown as ReadArguments<K>;
    const companions = (options.companions ?? []).map(id => readKey(id, args as ReadArguments<typeof id>));
    return JSON.stringify([readKey(command, args), ...companions, canonical(input)]);
  }

  async function load(input: A | (() => A), config: LoadOptions = {}) {
    const request = ++revision;
    store.setState({ pending: true,
      error: options.retainErrorWhilePending ? store.getState().error : '',
    } as Partial<S & ReadStatus>);
    try {
      let captured: A | undefined;
      if (typeof input !== 'function') {
        key(input);
        captured = canonical(input) as A;
      }
      if (config.before) { cache.clear(); await config.before(); }
      if (request !== revision) return;
      const requested = typeof input === 'function' ? input() : captured!;
      const cacheKey = key(requested);
      const args = canonical(requested) as A;
      const value = config.policy === 'cache-first' && cache.has(cacheKey)
        ? cache.get(cacheKey)! : await options.read(...args);
      if (request !== revision) return;
      const accepted = options.accept(value, args);
      cache.delete(cacheKey);
      cache.set(cacheKey, value);
      if (cache.size > (options.cacheSize ?? 32)) cache.delete(cache.keys().next().value!);
      store.setState({ ...accepted, error: '' } as Partial<S & ReadStatus>);
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) } as Partial<S & ReadStatus>);
    } finally {
      if (request === revision) store.setState({ pending: false } as Partial<S & ReadStatus>);
    }
  }

  function fence() {
    revision++;
    store.setState({ pending: false } as Partial<S & ReadStatus>);
  }
  function reset() {
    revision++;
    cache.clear();
    store.setState(initial(), true);
  }
  return {
    store, fence,
    use: () => useStore(store), peek: store.getState, set: store.setState, load,
    invalidate: () => {
      fence();
      cache.clear();
    }, reset,
  };
}
