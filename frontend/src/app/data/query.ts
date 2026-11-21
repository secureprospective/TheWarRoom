import { createStore, type StoreApi } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { useEffect } from 'react';
import { normalized, RejectedRead, type Dependency, type ReadSession, type ReadTags,
  type Retention, type TaggedReply } from './session';
import { readCommands, type ReadArguments, type ReadCommandId } from '../commands/reads.gen';
import { validateShape } from '../commands/shape';
import { errorCodeFor, errorMessage } from './errorCodes';

export type ReadStatus = { pending: boolean; error: string; errorCode?: string };

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
const canonical = normalized;

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
export type HostedQueryOptions<T, S extends object, A extends unknown[]> =
  Omit<QueryOptions<T, S, A>, 'read'> & {
    hosted: Retention & {
      session: ReadSession; schemaVersion: string; dependencies: Dependency[];
      decode?: (value: unknown) => T;
    };
    read: (args: A, known?: ReadTags) => Promise<TaggedReply<T>>;
  };

type ReadQuery<A extends unknown[], S extends object> = {
  store: StoreApi<S & ReadStatus>;
  use: () => S & ReadStatus;
  peek: StoreApi<S & ReadStatus>['getState'];
  set: StoreApi<S & ReadStatus>['setState'];
  load: (input: A | (() => A), config?: LoadOptions) => Promise<void>;
  activate: () => () => void;
  dispose: () => void;
  fence: () => void;
  invalidate: () => void;
  reset: () => void;
};

export function createReadQuery<T, S extends object, A extends unknown[]>(command: string,
  options: HostedQueryOptions<T, S, A>): ReadQuery<A, S>;
// A wrapper may project its parsed scope to the existing binding arguments, never a new schema.
export function createReadQuery<K extends ReadCommandId, T, S extends object>(command: K,
  options: QueryOptions<T, S, ReadArguments<K>> & { arguments?: never }): ReadQuery<ReadArguments<K>, S>;
export function createReadQuery<K extends ReadCommandId, T, S extends object, A extends unknown[]>(command: K,
  options: QueryOptions<T, S, A> & { arguments: (args: A) => ReadArguments<K> }): ReadQuery<A, S>;
export function createReadQuery<T, S extends object, A extends unknown[]>(command: string,
  inputOptions: (QueryOptions<T, S, A> & { arguments?: (args: A) => unknown })
    | HostedQueryOptions<T, S, A>): ReadQuery<A, S> {
  const hosted = 'hosted' in inputOptions ? inputOptions.hosted : undefined;
  const options = inputOptions as QueryOptions<T, S, A> & { arguments?: (args: A) => unknown };
  const hostedRead = inputOptions.read as HostedQueryOptions<T, S, A>['read'];
  const reader = hosted ? (...args: A) => hosted.session.read(key(args), {
    ...hosted, read: known => hostedRead(args, known),
  }) : options.read;
  const initial = () => ({ ...options.initial(), pending: false, error: '' });
  const store = options.store ?? createStore<S & ReadStatus>(initial);
  const cache = new Map<string, T>();
  let revision = 0;
  let active = 0;
  let unwatch: (() => void) | undefined;
  let lastInput: A | undefined;
  let lastKey: string | undefined;
  function activate() {
    active++;
    return () => { active--; };
  }

  function key(input: A) {
    if (hosted) return hosted.session.key(command, hosted.schemaVersion, input);
    const args = options.arguments
      ? options.arguments(input) : input as unknown as ReadArguments<ReadCommandId>;
    const companions = (options.companions ?? []).map(id => readKey(id, args as ReadArguments<typeof id>));
    return JSON.stringify([readKey(command as ReadCommandId, args as ReadArguments<ReadCommandId>),
      ...companions, canonical(input)]);
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
      if (config.before) {
        cache.clear();
        if (hosted && lastKey) hosted.session.invalidate(lastKey);
        await config.before();
      }
      if (request !== revision) return;
      const requested = typeof input === 'function' ? input() : captured!;
      const cacheKey = key(requested);
      const args = canonical(requested) as A;
      if (hosted) {
        lastInput = args;
        lastKey = cacheKey;
        unwatch?.();
        unwatch = hosted.session.watch(cacheKey, ended => {
          if (ended) { if (hosted.public) fence(); else reset(); return; }
          if (active && lastInput) void load(lastInput, { overlap: 'after' });
          else fence();
        });
      }
      const fetch = async () => {
        if (hosted && key(args) !== cacheKey) throw new RejectedRead();
        return reader(...args);
      };
      const value = !hosted && config.policy === 'cache-first' && cache.has(cacheKey)
        ? cache.get(cacheKey)! : config.before ? await fetch()
          : await joinRead(reader, cacheKey, fetch, hosted ? config.overlap ?? 'join' : config.overlap);
      if (request !== revision) return;
      const accepted = options.accept(value, args);
      if (!hosted) {
        cache.delete(cacheKey);
        cache.set(cacheKey, value);
        if (cache.size > (options.cacheSize ?? 32)) cache.delete(cache.keys().next().value!);
      }
      store.setState({ error: '', errorCode: undefined, ...accepted } as Partial<S & ReadStatus>);
    } catch (cause) {
      if (request === revision && !(cause instanceof RejectedRead)) store.setState({
        error: options.failure ? options.failure(cause) : errorMessage(cause),
        errorCode: errorCodeFor(cause),
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
    unwatch?.();
    if (hosted && lastKey) hosted.session.drop(lastKey);
    lastInput = undefined;
    lastKey = undefined;
    store.setState(initial(), true);
  }
  const leaveSession = hosted?.session.join({ reset, invalidate: () => { fence(); cache.clear(); } },
    hosted?.public ? 'shared' : 'private');
  return {
    store, fence, activate,
    dispose: () => { leaveSession?.(); unwatch?.(); fence(); },
    use: () => { useEffect(activate, []); return useStore(store); },
    peek: store.getState, set: store.setState, load,
    invalidate: () => {
      fence();
      if (hosted && lastKey) hosted.session.invalidate(lastKey);
      cache.clear();
    }, reset,
  };
}
