import { describe, expect, it, vi } from 'vitest';
import { IDBFactory } from 'fake-indexeddb';
import { createReadQuery } from './query';
import { createPublicReadStorage } from './clientCache';
import { createReadSession, topicKey, type Dependency, type ReadIdentity, type ReadTags,
  type ReadTopic, type Retention, type TaggedReply } from './session';

const identity: ReadIdentity = {
  serverId: 'server', serverEpoch: 'epoch', accountId: 'owner', authGeneration: '1',
  league: 'league', view: '0001:owner', season: 2026,
};
const scores: Dependency = { kind: 'scores', season: 2026, week: 3 };
function tags(versions: Record<string, string> = {}): ReadTags {
  return { serverEpoch: 'epoch', league: versions, football: 'pin', live: 'live', values: 'run' };
}
function reply(data = 'held', versions: Record<string, string> = {}): TaggedReply<string> {
  return { data, tags: tags(versions) };
}
function ping(dependency: Dependency, version = '1'): ReadTopic {
  return { ...dependency, league: 'league', version };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(yes => { resolve = yes; });
  return { promise, resolve };
}
function decode(value: unknown): string {
  if (typeof value !== 'string') throw new Error('Invalid cached reading');
  return value;
}
function query(session: ReturnType<typeof createReadSession>, config: {
  command?: string; dependencies?: Dependency[]; retention?: Retention;
  read?: (args: [Record<string, unknown>], known?: ReadTags) => Promise<TaggedReply<string>>;
} = {}) {
  return createReadQuery(config.command ?? 'results.read', {
    initial: (): { value?: string } => ({}),
    hosted: { session, schemaVersion: '1', dependencies: config.dependencies ?? [scores],
      ...config.retention ?? { policy: 'memory' }, decode },
    read: config.read ?? (async () => reply()), accept: value => ({ value }),
  });
}
function session(options: Parameters<typeof createReadSession>[0] = {}) {
  return createReadSession({ identity, ...options });
}
const input: [Record<string, unknown>] = [{ season: 2026, week: 3 }];

it('keys every identity component, command/schema and normalized args, without a clock key', () => {
  const owner = session();
  const key = owner.key('week.read', '1', [{ b: 2, a: 1 }]);
  expect(key).toBe(owner.key('week.read', '1', [{ a: 1, b: 2 }]));
  for (const field of Object.keys(identity) as (keyof ReadIdentity)[]) {
    const changed = { ...identity, [field]: field === 'season' ? 2025 : 'different' };
    expect(session({ identity: changed }).key('week.read', '1', [{ a: 1, b: 2 }])).not.toBe(key);
  }
  expect(owner.key('week.read', '2', [{ a: 1, b: 2 }])).not.toBe(key);
  expect(owner.key('schedule.read', '1', [{ a: 1, b: 2 }])).not.toBe(key);
  expect(owner.key('week.read', '1', [{ a: 1, b: 3 }])).not.toBe(key);
  expect(owner.key('week.read', '1', ['0001'])).not.toBe(owner.key('week.read', '1', ['1']));
});

const cases: [string, Retention, Dependency[]][] = [
  ['native.directory.read', { policy: 'persist', public: true }, [{ kind: 'league' }]],
  ['schedule.read', { policy: 'persist', public: true }, [{ kind: 'league' }]],
  ['bracket.read', { policy: 'memory' },
    [{ kind: 'league' }, { kind: 'scores', season: 2026 }, { kind: 'clock' }]],
  ['results.read', { policy: 'memory' }, [scores, { kind: 'league' }]],
  ['standings.read', { policy: 'memory' }, [{ kind: 'scores', season: 2026 }, { kind: 'league' }]],
  ['odds.read', { policy: 'memory' },
    [{ kind: 'scores', season: 2026 }, { kind: 'league' }, { kind: 'values' }]],
  ['lineup.read', { policy: 'memory' },
    [{ kind: 'team', franchise: '0001' }, { kind: 'league' }, { kind: 'clock' }]],
  ['projections.read', { policy: 'memory' }, [{ kind: 'values' }, scores]],
  ['week.read', { policy: 'memory' }, [{ kind: 'clock' }, { kind: 'league' }]],
];
describe.each(cases)('%s retention', (command, retention, dependencies) => {
  it.each(dependencies)('drops only on its scoped dependency $kind', async dependency => {
    const owner = session();
    const read = vi.fn(async () => reply('fresh', { [topicKey(dependency)]: '2' }));
    const held = query(owner, { command, retention, dependencies, read });
    await held.load(input);
    await held.load(input);
    expect(read).toHaveBeenCalledTimes(1);
    owner.applyTopics([{ ...ping(dependency, '3'), league: 'other' }]);
    await held.load(input);
    expect(read).toHaveBeenCalledTimes(1);
    read.mockResolvedValue(reply('updated', { [topicKey(dependency)]: '3' }));
    owner.applyTopics([ping(dependency, '3')]);
    expect(read).toHaveBeenCalledTimes(1); // Inactive entries do not reread.
    await held.load(input);
    expect(held.peek().value).toBe('updated');
    expect(read).toHaveBeenCalledTimes(2);
  });
});

it('deduplicates identical reads across screens and keeps one trailing read for active entries', async () => {
  const owner = session();
  const late = deferred<TaggedReply<string>>();
  const read = vi.fn().mockReturnValueOnce(late.promise)
    .mockResolvedValue(reply('new', { 'scores:2026:3': '9' }));
  const first = query(owner, { read });
  const second = query(owner, { read });
  const stop = first.activate();
  const one = first.load(input);
  const two = second.load(input);
  expect(read).toHaveBeenCalledTimes(1);
  owner.applyTopics([ping(scores, '2')]);
  owner.applyTopics([ping(scores, '9')]);
  owner.applyTopics([ping(scores, '4')]); // Crossed/older ping cannot lower the rejection fence.
  late.resolve(reply('obsolete', { 'scores:2026:3': '1' }));
  await Promise.all([one, two]);
  await vi.waitFor(() => expect(first.peek().value).toBe('new'));
  expect(second.peek().value).toBeUndefined();
  expect(read).toHaveBeenCalledTimes(2);
  stop();
  owner.applyTopics([ping(scores, '10')]);
  await Promise.resolve();
  expect(read).toHaveBeenCalledTimes(2);
});

it('marks all matching entries stale before the first notification', async () => {
  const owner = session();
  const first = query(owner);
  const second = query(owner, { command: 'other.read' });
  await first.load(input);
  await second.load(input);
  const read = vi.fn(async () => reply('new', { 'scores:2026:3': '2' }));
  const key = owner.key('results.read', '1', input);
  let refreshed: Promise<string> | undefined;
  owner.watch(key, () => {
    refreshed = owner.read(owner.key('other.read', '1', input), {
      policy: 'memory', dependencies: [scores], read,
    });
  });
  owner.applyTopics([ping(scores, '2')]);
  expect(await refreshed).toBe('new');
  expect(read).toHaveBeenCalledTimes(1);
});

it('rejects a late reply after a newer topic without losing the last good view', async () => {
  const owner = session();
  const late = deferred<TaggedReply<string>>();
  const read = vi.fn().mockResolvedValueOnce(reply('good')).mockReturnValueOnce(late.promise);
  const held = query(owner, { read });
  await held.load(input);
  owner.applyTopics([ping(scores, '1')]);
  const pending = held.load(input);
  owner.applyTopics([ping(scores, '3')]);
  late.resolve(reply('old', { 'scores:2026:3': '2' }));
  await pending;
  expect(held.peek()).toEqual({ value: 'good', error: '', pending: false });
});

it('uses newer reply tags as a fence as well as change messages', async () => {
  const owner = session();
  const late = deferred<TaggedReply<string>>();
  const old = query(owner, { read: () => late.promise });
  const pending = old.load(input);
  const newer = query(owner, { command: 'standings.read',
    read: async () => reply('new', { 'scores:2026:3': '9' }) });
  await newer.load(input);
  late.resolve(reply('old', { 'scores:2026:3': '8' }));
  await pending;
  expect(old.peek().value).toBeUndefined();
  expect(newer.peek().value).toBe('new');
});

it.each(['accountId', 'authGeneration', 'serverId', 'serverEpoch', 'league', 'view'] as const)(
  'clears private data and fences in-flight replies on %s switch', async field => {
    const owner = session();
    const late = deferred<TaggedReply<string>>();
    const read = vi.fn().mockResolvedValueOnce(reply('private')).mockReturnValueOnce(late.promise);
    const held = query(owner, { read });
    await held.load(input);
    owner.applyTopics([ping(scores)]);
    const pending = held.load(input);
    owner.setIdentity({ ...identity, [field]: 'next' });
    expect(held.peek()).toEqual({ error: '', pending: false });
    late.resolve(reply('old account'));
    await pending;
    expect(held.peek()).toEqual({ error: '', pending: false });
  },
);

it('ends a session explicitly, rejects queued reads, and requires a new identity', async () => {
  const owner = session();
  const late = deferred<TaggedReply<string>>();
  const read = vi.fn(() => late.promise);
  const held = query(owner, { read });
  const stop = held.activate();
  const pending = held.load(input);
  owner.applyTopics([ping(scores)]);
  owner.endSession('revoked');
  late.resolve(reply('private'));
  await pending;
  await Promise.resolve();
  await Promise.resolve();
  expect(read).toHaveBeenCalledTimes(1);
  expect(held.peek().value).toBeUndefined();
  expect(() => owner.key('results.read', '1', input)).toThrow('superseded');
  stop();
});

it('sends known tags and handles unchanged by retaining data and refreshing tags', async () => {
  const owner = session();
  const read = vi.fn().mockResolvedValueOnce(reply('held', { 'scores:2026:3': '1' }))
    .mockResolvedValueOnce({ unchanged: true, tags: tags({ 'scores:2026:3': '2' }) })
    .mockResolvedValueOnce({ unchanged: true, tags: tags({ 'scores:2026:3': '3' }) });
  const held = query(owner, { read });
  await held.load(input);
  owner.applyTopics([ping(scores, '2')]);
  await held.load(input);
  expect(held.peek().value).toBe('held');
  expect(read.mock.calls[1][1]).toEqual(tags({ 'scores:2026:3': '1' }));
  owner.applyTopics([ping(scores, '3')]);
  await held.load(input);
  expect(read.mock.calls[2][1]).toEqual(tags({ 'scores:2026:3': '2' }));
  expect(held.peek().value).toBe('held');
});

it('rejects unchanged without a held body', async () => {
  const held = query(session(), { read: async () => ({ unchanged: true, tags: tags() }) });
  await held.load(input);
  expect(held.peek().value).toBeUndefined();
  expect(held.peek().error).toContain('no held data');
});

it('keeps dependency scope isolated, including leading-zero franchises', async () => {
  const owner = session();
  const read = vi.fn(async () => reply());
  const held = query(owner, { dependencies: [{ kind: 'team', franchise: '0001' }, scores], read });
  await held.load(input);
  owner.applyTopics([ping({ kind: 'team', franchise: '1' }),
    ping({ kind: 'scores', season: 2025, week: 3 }), ping({ kind: 'scores', season: 2026, week: 4 })]);
  await held.load(input);
  expect(read).toHaveBeenCalledTimes(1);
});

it('bounds memory entries with LRU reuse and falls back online for oversized data', async () => {
  const owner = session({ maxEntries: 2 });
  const read = vi.fn(async () => reply());
  const held = query(owner, { read });
  await held.load([{ id: 1 }]);
  await held.load([{ id: 2 }]);
  await held.load([{ id: 1 }]);
  await held.load([{ id: 3 }]);
  await held.load([{ id: 2 }]);
  expect(read).toHaveBeenCalledTimes(4);
  const tiny = query(session({ maxBytes: 1 }), { read });
  await tiny.load(input);
  await tiny.load(input);
  expect(tiny.peek().value).toBe('held');
  expect(read).toHaveBeenCalledTimes(6);
});

it('explicit invalidate and reset do not reuse stale session data', async () => {
  const owner = session();
  const read = vi.fn(async () => reply());
  const held = query(owner, { read });
  await held.load(input);
  held.invalidate();
  await held.load(input);
  held.reset();
  await held.load(input);
  expect(read).toHaveBeenCalledTimes(3);
});

it('persists public data across a fake IndexedDB restart, but never private data or requests', async () => {
  const indexedDB = new IDBFactory();
  const storage = createPublicReadStorage({ indexedDB });
  const publicRead = vi.fn(async () => reply('public'));
  const privateRead = vi.fn(async () => reply('private'));
  const retention: Retention = { policy: 'persist', public: true };
  const first = session({ storage });
  await query(first, { command: 'schedule.read', retention, read: publicRead }).load(input);
  await query(first, { command: 'lineup.read', retention: { policy: 'persist' },
    read: privateRead }).load(input);
  const restarted = session({ storage: createPublicReadStorage({ indexedDB }) });
  const publicView = query(restarted, { command: 'schedule.read', retention, read: publicRead });
  const privateView = query(restarted, { command: 'lineup.read', retention: { policy: 'persist' },
    read: privateRead });
  await publicView.load(input);
  await privateView.load(input);
  expect(publicView.peek().value).toBe('public');
  expect(publicRead).toHaveBeenCalledTimes(1);
  expect(privateRead).toHaveBeenCalledTimes(2);
  const raw = await storage.get(first.key('schedule.read', '1', input));
  expect(JSON.parse(raw!)).toEqual({ schema: 1, tags: tags(), data: 'public' });
  expect(await storage.get(first.key('lineup.read', '1', input))).toBeUndefined();
});

it.each(['corrupt', 'mismatched', 'invalid body', 'quota'])(
  'ignores %s persistence safely', async failure => {
  const storage = createPublicReadStorage({ indexedDB: new IDBFactory() });
  const owner = session({ storage });
  const key = owner.key('results.read', '1', input);
  if (failure === 'quota') {
    vi.spyOn(storage, 'put').mockRejectedValue(new DOMException('Full', 'QuotaExceededError'));
  }
  else await storage.put(key, failure === 'corrupt' ? '{broken' : JSON.stringify({
    schema: failure === 'mismatched' ? 2 : 1, data: 123, tags: tags(),
  }));
  const read = vi.fn(async () => reply('online'));
  const held = query(owner, { retention: { policy: 'persist', public: true }, read });
  await held.load(input);
  expect(held.peek()).toEqual({ value: 'online', error: '', pending: false });
  expect(read).toHaveBeenCalledTimes(1);
});

it('ignores unavailable IndexedDB and bounds disk by entries and bytes', async () => {
  const unavailable = createPublicReadStorage({
    indexedDB: { open() { throw new Error('Denied'); } } as never,
  });
  await expect(unavailable.get('key')).resolves.toBeUndefined();
  await expect(unavailable.put('key', 'value')).resolves.toBeUndefined();
  const storage = createPublicReadStorage({ indexedDB: new IDBFactory(), maxEntries: 2, maxBytes: 20 });
  await storage.put('a', '1234');
  await storage.put('b', '1234');
  await storage.get('a');
  await storage.put('c', '1234');
  expect(await storage.get('a')).toBe('1234');
  expect(await storage.get('b')).toBeUndefined();
  await storage.put('huge', 'x'.repeat(30));
  expect(await storage.get('huge')).toBeUndefined();
  const bytes = createPublicReadStorage({ indexedDB: new IDBFactory(), maxBytes: 6 });
  await bytes.put('a', '1234');
  await bytes.put('b', '1234');
  expect(await bytes.get('a')).toBeUndefined();
  expect(await bytes.get('b')).toBe('1234');
});

it('rejects a topic that arrives while optional persistence is completing', async () => {
  const pendingWrite = deferred<void>();
  const storage = { get: async () => undefined, put: vi.fn(() => pendingWrite.promise),
    remove: vi.fn(async () => undefined) };
  const owner = session({ storage });
  const held = query(owner, { retention: { policy: 'persist', public: true },
    read: async () => reply('old', { 'scores:2026:3': '1' }) });
  const pending = held.load(input);
  await vi.waitFor(() => expect(storage.put).toHaveBeenCalledTimes(1));
  owner.applyTopics([ping(scores, '2')]);
  pendingWrite.resolve();
  await pending;
  expect(held.peek().value).toBeUndefined();
  expect(storage.remove).toHaveBeenCalled();
});

it('handles an IndexedDB quota exception inside the transaction adapter', async () => {
  const { IDBObjectStore } = await import('fake-indexeddb');
  const storage = createPublicReadStorage({ indexedDB: new IDBFactory() });
  const put = vi.spyOn(IDBObjectStore.prototype, 'put').mockImplementation(() => {
    throw new DOMException('Full', 'QuotaExceededError');
  });
  try {
    const held = query(session({ storage }), { retention: { policy: 'persist', public: true } });
    await held.load(input);
    expect(held.peek()).toEqual({ value: 'held', error: '', pending: false });
  } finally { put.mockRestore(); }
});
