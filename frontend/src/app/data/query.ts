import { createStore } from 'zustand/vanilla';
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

// Feature wrappers retain selection and actions; this is the sole read/cache lifecycle.
export function createReadQuery<K extends ReadCommandId, T, S extends object>(command: K, options: {
  initial: () => S;
  read: (...args: ReadArguments<K>) => Promise<T>;
  accept: (value: T) => Partial<S>;
}) {
  const initial = () => ({ ...options.initial(), pending: false, error: '' });
  const store = createStore<S & ReadStatus>(initial);
  const cache = new Map<string, T>();
  let revision = 0;

  async function load(input: ReadArguments<K> | (() => ReadArguments<K>), config: LoadOptions = {}) {
    const request = ++revision;
    store.setState({ pending: true, error: '' } as Partial<S & ReadStatus>);
    try {
      let captured: ReadArguments<K> | undefined;
      if (typeof input !== 'function') {
        readKey(command, input);
        captured = canonical(input) as ReadArguments<K>;
      }
      if (config.before) { cache.clear(); await config.before(); }
      if (request !== revision) return;
      const requested = typeof input === 'function' ? input() : captured!;
      const key = readKey(command, requested);
      const args = canonical(requested) as ReadArguments<K>;
      const value = config.policy === 'cache-first' && cache.has(key)
        ? cache.get(key)! : await options.read(...args);
      if (request !== revision) return;
      const accepted = options.accept(value);
      cache.delete(key);
      cache.set(key, value);
      if (cache.size > 32) cache.delete(cache.keys().next().value!);
      store.setState(accepted as Partial<S & ReadStatus>);
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) } as Partial<S & ReadStatus>);
    } finally {
      if (request === revision) store.setState({ pending: false } as Partial<S & ReadStatus>);
    }
  }

  function reset() {
    revision++;
    cache.clear();
    store.setState(initial(), true);
  }
  return {
    use: () => useStore(store), peek: store.getState, set: store.setState, load,
    invalidate: () => { revision++; cache.clear(); store.setState({ pending: false } as Partial<S & ReadStatus>); }, reset,
  };
}
