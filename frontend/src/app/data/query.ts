import { createStore, type StoreApi } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { readCommands, type ReadArguments, type ReadCommandId } from '../commands/reads.gen';
import { validateShape } from '../commands/shape';

export type ReadStatus = { pending: boolean; error: string };

// Reads of the same reader and scope never overlap: a slow reading must not stack up copies of itself
// behind every poll and change event. A request while one runs queues at most one read after it, so a
// change that lands mid-read is still read; further requests join that queued read. 'join' asks only for
// the newest result already coming. Shared across query instances; nothing is kept after it settles.
type Flight = { running: Promise<unknown>; queued?: Promise<unknown> };
const inflight = new WeakMap<object, Map<string, Flight>>();
const settle = () => undefined;
export function joinRead<T>(reader: object, key: string, read: () => Promise<T>,
  mode: 'after' | 'join' = 'after'): Promise<T> {
  let flights = inflight.get(reader);
  if (!flights) { flights = new Map(); inflight.set(reader, flights); }
  const map = flights;
  const held = map.get(key);
  const done = (flight: Flight, p: Promise<unknown>) => {
    if (flight.running === p && !flight.queued && map.get(key) === flight) map.delete(key);
  };
  if (!held) {
    // The caller gets read's own promise, so joining adds no turn of the event loop to a lone read.
    const running = read();
    const flight: Flight = { running };
    map.set(key, flight);
    running.then(() => done(flight, running), () => done(flight, running));
    return running;
  }
  if (held.queued || mode === 'join') return (held.queued ?? held.running) as Promise<T>;
  const queued: Promise<T> = held.running.then(settle, settle).then(() => {
    held.queued = undefined;
    held.running = queued;
    return read();
  });
  held.queued = queued;
  queued.then(() => done(held, queued), () => done(held, queued));
  return queued;
}

// poll runs read now and again `every` ms after each read completes, so reads never overlap. The
// returned stop cancels the next read; a read already running is left to settle and is ignored.
export function poll(read: () => Promise<unknown> | void, every: number): () => void {
  let stopped = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const run = async () => {
    timer = undefined;
    if (stopped) return;
    try { await read(); } catch { /* the reader reports its own failure */ }
    if (!stopped) timer = setTimeout(() => void run(), every);
  };
  void run();
  return () => { stopped = true; if (timer !== undefined) clearTimeout(timer); };
}
// overlap: 'after' (default) reads again once a read of the same scope in flight has finished, for a
// change that landed mid-read; 'join' takes the read already coming, for a mount or a poll.
type LoadOptions = { before?: () => Promise<void>; policy?: 'revalidate' | 'cache-first';
  overlap?: 'after' | 'join' };

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
  failure?: (cause: unknown) => string;
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
        ? cache.get(cacheKey)! : config.before ? await options.read(...args)
          : await joinRead(options.read, cacheKey, () => options.read(...args), config.overlap);
      if (request !== revision) return;
      const accepted = options.accept(value, args);
      cache.delete(cacheKey);
      cache.set(cacheKey, value);
      if (cache.size > (options.cacheSize ?? 32)) cache.delete(cache.keys().next().value!);
      store.setState({ error: '', ...accepted } as Partial<S & ReadStatus>);
    } catch (cause) {
      if (request === revision) store.setState({
        error: options.failure ? options.failure(cause) : String(cause),
      } as Partial<S & ReadStatus>);
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
