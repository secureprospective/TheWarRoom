// Shared and private legacy views retain their existing end-of-session behavior.
type Member = { invalidate: () => void; reset: () => void };
export type SessionEnd = 'shared' | 'private';
export type ReadIdentity = {
  serverId: string; serverEpoch: string; accountId: string; authGeneration: string;
  league: string; view: string; season?: number;
};
export type Dependency = {
  kind: 'league' | 'team' | 'scores' | 'values' | 'clock';
  franchise?: string; season?: number; week?: number; run?: string;
};
export type ReadTopic = Dependency & { league: string; version: string };
export type ReadTags = {
  serverEpoch: string; league: Record<string, string>; football: string; live: string; values: string;
};
export type TaggedReply<T> = { tags: ReadTags } & ({ data: T; unchanged?: false } | { unchanged: true });
export type Retention = { policy: 'memory' | 'persist'; public?: boolean };
export type PublicStorage = {
  get: (key: string) => Promise<string | undefined>;
  put: (key: string, value: string) => Promise<void>;
  remove: (key: string) => Promise<void>;
  retainScope?: (serverId: string, league: string) => Promise<void>;
};
type ReadSpec<T> = Retention & {
  dependencies: Dependency[];
  read: (known?: ReadTags) => Promise<TaggedReply<T>>;
  decode?: (value: unknown) => T;
};
type Entry = Retention & {
  key: string; context: string; dependencies: Dependency[]; stale: boolean; serial: number; bytes: number;
  hasValue: boolean; value?: unknown; tags?: ReadTags; flight?: Promise<unknown>;
};

export function normalized(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(normalized);
  if (value !== null && typeof value === 'object') return Object.fromEntries(Object.entries(value)
    .filter(([, item]) => item !== undefined).sort(([a], [b]) => a.localeCompare(b))
    .map(([key, item]) => [key, normalized(item)]));
  return value;
}

// The wire scope spelling is kind[:franchise|season[:week]|run]. Omitted fields are wildcards
// for dependency matching, not additional version counters.
export function topicKey(topic: Dependency): string {
  return [topic.kind, topic.franchise, topic.season, topic.week, topic.run]
    .filter(value => value !== undefined).join(':');
}
function matches(dependency: Dependency, topic: Dependency): boolean {
  return dependency.kind === topic.kind && (['franchise', 'season', 'week', 'run'] as const)
    .every(field => dependency[field] === undefined || topic[field] === undefined
      || dependency[field] === topic[field]);
}
function validTags(value: unknown): value is ReadTags {
  if (!value || typeof value !== 'object') return false;
  const tags = value as ReadTags;
  return ['serverEpoch', 'football', 'live', 'values'].every(key =>
    typeof tags[key as keyof ReadTags] === 'string') && !!tags.league
    && typeof tags.league === 'object' && !Array.isArray(tags.league)
    && Object.values(tags.league).every(version => typeof version === 'string' && /^\d+$/.test(version));
}
export class RejectedRead extends Error {
  constructor() { super('Read superseded by session or topic change'); }
}

export function createReadSession(options: {
  identity?: ReadIdentity; storage?: PublicStorage; maxEntries?: number; maxBytes?: number;
} = {}) {
  const members = new Map<Member, SessionEnd>();
  const entries = new Map<string, Entry>();
  const watchers = new Map<string, Set<(ended?: boolean) => void>>();
  const versions = new Map<string, ReadTopic>();
  let identity = options.identity ? { ...options.identity } : undefined;
  let generation = 0;
  const maxEntries = options.maxEntries ?? 32;
  const maxBytes = options.maxBytes ?? 4 * 1024 * 1024;
  const ignore = () => undefined;
  function remove(entry: Entry) {
    entries.delete(entry.key);
    if (entry.policy === 'persist' && entry.public) void options.storage?.remove(entry.key).catch(ignore);
  }
  function trim() {
    let bytes = [...entries.values()].reduce((sum, entry) => sum + entry.bytes, 0);
    for (const entry of entries.values()) {
      if (entries.size <= maxEntries && bytes <= maxBytes) break;
      bytes -= entry.bytes;
      remove(entry);
    }
  }
  function current(entry: Entry, tags: ReadTags) {
    return tags.serverEpoch === identity?.serverEpoch && [...versions.values()].every(topic =>
      !entry.dependencies.some(dependency => matches(dependency, topic))
      || BigInt(tags.league[topicKey(topic)] ?? '-1') >= BigInt(topic.version));
  }
  function markStale(entry: Entry) {
    entry.stale = true;
    entry.serial++;
    if (entry.policy === 'persist' && entry.public) void options.storage?.remove(entry.key).catch(ignore);
  }
  function observe(tags: ReadTags) {
    for (const [key, version] of Object.entries(tags.league)) {
      const [kind, scope, week] = key.split(':');
      if (!['league', 'team', 'scores', 'values', 'clock'].includes(kind)) continue;
      const topic: ReadTopic = { kind: kind as Dependency['kind'], league: identity!.league, version };
      if (kind === 'team') topic.franchise = scope;
      else if (kind === 'values') topic.run = scope;
      else if (scope !== undefined) {
        if (!/^\d+$/.test(scope) || (week !== undefined && !/^\d+$/.test(week))) continue;
        topic.season = Number(scope);
        if (week !== undefined) topic.week = Number(week);
      }
      if (BigInt(version) > BigInt(versions.get(key)?.version ?? '-1')) versions.set(key, topic);
    }
  }
  function endSession(_reason: string = 'session ended') {
    generation++;
    identity = undefined;
    versions.clear();
    for (const entry of entries.values()) {
      entry.serial++;
      if (!entry.public) remove(entry);
    }
    for (const listeners of watchers.values()) for (const notify of [...listeners]) notify(true);
    for (const [query, end] of [...members]) {
      if (end === 'private') query.reset();
      else query.invalidate();
    }
  }
  return {
    join(query: Member, end: SessionEnd) {
      members.set(query, end);
      return () => { members.delete(query); };
    },
    end: endSession, endSession,
    setIdentity(next: ReadIdentity) {
      if (JSON.stringify(normalized(identity)) === JSON.stringify(normalized(next))) return;
      for (const entry of entries.values()) {
        const scope = JSON.parse(entry.context) as ReadIdentity;
        if (entry.public && (scope.serverId !== next.serverId || scope.league !== next.league
          || scope.serverEpoch !== next.serverEpoch)) remove(entry);
      }
      void options.storage?.retainScope?.(next.serverId, next.league).catch(ignore);
      endSession('identity changed');
      identity = { ...next };
    },
    key(command: string, schemaVersion: string, args: unknown) {
      if (!identity) throw new RejectedRead();
      const id = identity;
      return JSON.stringify([id.serverId, id.serverEpoch, id.accountId, id.authGeneration,
        id.league, id.view, id.season ?? null, `${command}@${schemaVersion}`, normalized(args)]);
    },
    watch(key: string, notify: (ended?: boolean) => void) {
      let held = watchers.get(key);
      if (!held) { held = new Set(); watchers.set(key, held); }
      held.add(notify);
      const listeners = held;
      return () => { listeners.delete(notify); if (!listeners.size) watchers.delete(key); };
    },
    invalidate(key: string) { const entry = entries.get(key); if (entry) markStale(entry); },
    drop(key: string) { const entry = entries.get(key); if (entry) remove(entry); },
    applyTopics(topics: ReadTopic[]) {
      const changed = topics.filter(topic => topic.league === identity?.league
        && /^\d+$/.test(topic.version)
        && BigInt(topic.version) > BigInt(versions.get(topicKey(topic))?.version ?? '-1'));
      for (const topic of changed) {
        const previous = versions.get(topicKey(topic));
        if (!previous || BigInt(topic.version) > BigInt(previous.version)) {
          versions.set(topicKey(topic), topic);
        }
      }
      const stale = [...entries.values()].filter(entry =>
        entry.context === JSON.stringify(normalized(identity)) && changed.some(topic =>
          entry.dependencies.some(dependency => matches(dependency, topic))));
      for (const entry of stale) markStale(entry);
      for (const entry of stale) for (const notify of [...(watchers.get(entry.key) ?? [])]) notify();
    },
    async read<T>(key: string, spec: ReadSpec<T>): Promise<T> {
      if (!identity) throw new RejectedRead();
      let entry = entries.get(key);
      if (!entry) {
        entry = { key, ...spec, context: JSON.stringify(normalized(identity)),
          dependencies: spec.dependencies.map(item => ({ ...item })),
          stale: true, serial: 0, bytes: 0, hasValue: false };
        entries.set(key, entry);
        trim();
      }
      if (spec.public && !entry.hasValue) {
        const requested = JSON.parse(key) as unknown[];
        const reusable = [...entries.values()].find(other => {
          if (!other.public || other.stale || !other.hasValue) return false;
          const previous = JSON.parse(other.key) as unknown[];
          return [0, 1, 4, 6, 7, 8].every(index =>
            JSON.stringify(previous[index]) === JSON.stringify(requested[index]));
        });
        if (reusable) Object.assign(entry, { value: reusable.value, tags: reusable.tags,
          hasValue: true, stale: false, bytes: reusable.bytes });
      }
      const held = entry;
      entries.delete(key);
      entries.set(key, held);
      if (held.flight) return held.flight as Promise<T>;
      if (!held.stale && held.hasValue) return held.value as T;
      const started = generation;
      const serial = held.serial;
      const allowed = () => started === generation;
      const run = async () => {
        if (!held.hasValue && spec.policy === 'persist' && spec.public && spec.decode) {
          try {
            const raw = await options.storage?.get(key);
            if (raw) {
              const saved = JSON.parse(raw);
              if (saved.schema === 1 && validTags(saved.tags) && current(held, saved.tags)
                && allowed() && serial === held.serial) {
                held.value = spec.decode(saved.data);
                held.tags = saved.tags;
                held.hasValue = true;
                held.stale = false;
                held.bytes = new TextEncoder().encode(key + raw).byteLength;
                observe(saved.tags);
                trim();
              }
            }
          } catch { /* Optional public storage never prevents an online read. */ }
        }
        if (!allowed()) throw new RejectedRead();
        if (!held.stale && held.hasValue) return held.value as T;
        const reply = await spec.read(held.tags);
        if (!allowed() || !validTags(reply.tags) || !current(held, reply.tags)) throw new RejectedRead();
        if (reply.unchanged && !held.hasValue) throw new Error('Unchanged read has no held data');
        const value = reply.unchanged ? held.value as T : reply.data;
        const saved = JSON.stringify({ schema: 1, tags: reply.tags, data: value });
        held.value = value;
        held.tags = reply.tags;
        observe(reply.tags);
        held.hasValue = true;
        held.stale = false;
        held.bytes = new TextEncoder().encode(key + saved).byteLength;
        trim();
        if (allowed() && entries.get(key) === held && held.policy === 'persist' && held.public) {
          await options.storage?.put(key, saved).catch(ignore);
          if (!allowed()) await options.storage?.remove(key).catch(ignore);
        }
        if (!allowed() || !current(held, reply.tags)) throw new RejectedRead();
        return value;
      };
      const flight = run();
      held.flight = flight;
      try { return await flight; } finally { if (held.flight === flight) held.flight = undefined; }
    },
  };
}
export type ReadSession = ReturnType<typeof createReadSession>;
