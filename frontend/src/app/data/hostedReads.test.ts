import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { IDBFactory } from 'fake-indexeddb';
import type { NativeView } from '../control/nativeApi';
import type { TaggedReply } from './session';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(yes => { resolve = yes; });
  return { promise, resolve };
}
const view = (generation = '1', league = 'native.test'): NativeView => ({
  authGeneration: generation, server: 'https://league.example', league, name: 'League',
  message: '', role: 'Owner', franchise: 'one', franchiseName: 'One', seed: '',
  connection: '', helperId: '', confirmation: '', seats: [],
});
const reply = (data: string): TaggedReply<string> => ({ data,
  tags: { serverEpoch: '', league: {}, football: '', live: '', values: '' } });
const response = (value: unknown) => ({ ok: true, status: 200,
  headers: new Headers({ 'Content-Type': 'application/json' }), json: async () => value });
let stream = deferred<ReturnType<typeof response>>();
let status = deferred<ReturnType<typeof response>>();
let statuses = 0;
let seq = 0;
let owner: typeof import('./hostedReads');
let api: typeof import('../control/nativeApi');
let createQuery: typeof import('./query').createReadQuery;
let next = view();
beforeEach(async () => {
  vi.resetModules();
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
  stream = deferred(); status = deferred(); statuses = 0; seq = 0; next = view();
  vi.stubGlobal('indexedDB', new IDBFactory());
  vi.stubGlobal('fetch', vi.fn((url: string, init?: RequestInit) => {
    if (url === '/desktop/changes') return stream.promise;
    if (init?.body && JSON.parse(String(init.body)).action === 'status') {
      statuses++;
      return status.promise;
    }
    return Promise.resolve(response(next));
  }));
  owner = await import('./hostedReads');
  api = await import('../control/nativeApi');
  createQuery = (await import('./query')).createReadQuery;
});
afterEach(() => { vi.clearAllTimers(); vi.useRealTimers(); vi.unstubAllGlobals(); });
async function signIn(generation = '1', league = 'native.test') {
  next = view(generation, league);
  await api.nativeAction('login');
}
function query(read: () => Promise<TaggedReply<string>> = async () => reply('A'), publicRead = false) {
  return createQuery('schedule.read', {
    initial: (): { value?: string } => ({}),
    hosted: { session: owner.hostedReads, schemaVersion: '1', dependencies: [{ kind: 'league' }],
      policy: publicRead ? 'persist' : 'memory', public: publicRead,
      decode: value => { if (typeof value !== 'string') throw new Error('bad reading'); return value; } },
    read, accept: value => ({ value }),
  });
}
async function change(state: string, generation = '1', team = false) {
  const previous = stream;
  stream = deferred();
  previous.resolve(response({ seq: ++seq, authGeneration: generation, state, league: 'native.test',
    topics: team ? [{ kind: 'team', franchise: 'one', version: String(seq) }] : [] }));
  await vi.advanceTimersByTimeAsync(0);
}
it('account B never inherits a rendered private value from A', async () => {
  await signIn();
  const held = query();
  await held.load([]);
  expect(held.peek().value).toBe('A');
  next = { ...view('2'), name: '', role: '', franchise: '' };
  await api.nativeAction('forget');
  expect(held.peek().value).toBeUndefined();
  await signIn('3');
  expect(held.peek().value).toBeUndefined();
  expect(owner.hostedSnapshot().key).toContain('3');
});
it('drops a late private A reply after B signs in, without persisting it', async () => {
  await signIn();
  const flight = deferred<TaggedReply<string>>();
  const held = query(() => flight.promise);
  const loading = held.load([]);
  await signIn('2');
  flight.resolve(reply('A secret'));
  await loading;
  expect(held.peek().value).toBeUndefined();
  const storage = (await import('./clientCache')).createPublicReadStorage();
  expect(await storage.get(owner.hostedReads.key('schedule.read', '1', []))).toBeUndefined();
});
it.each(['refused', 'signed-out'])('%s purges held and in-flight private values', async state => {
  await signIn();
  const held = query();
  await held.load([]);
  const flight = deferred<TaggedReply<string>>();
  const pending = query(() => flight.promise);
  const loading = pending.load(['pending']);
  await change(state, '2');
  expect(held.peek().value).toBeUndefined();
  expect(owner.hostedSnapshot().blocked).toBe(true);
  flight.resolve(reply('late secret'));
  await loading;
  expect(pending.peek().value).toBeUndefined();
});
it('hides an old role before the shared admission re-read lands', async () => {
  await signIn();
  const held = query();
  await held.load([]);
  await change('connected', '1', true);
  expect(held.peek().value).toBeUndefined();
  expect(owner.hostedSnapshot().blocked).toBe(true);
  status.resolve(response({ ...view(), role: 'Observer', franchise: '' }));
  await vi.advanceTimersByTimeAsync(0);
  expect(owner.hostedSnapshot().view?.franchise).toBe('');
  expect(owner.hostedSnapshot().key).toContain('Observer/');
});
it('preserves public persisted readings on account switch but drops them on league change', async () => {
  await signIn();
  const held = query(async () => reply('directory'), true);
  await held.load([]);
  const oldKey = owner.hostedReads.key('schedule.read', '1', []);
  const storage = (await import('./clientCache')).createPublicReadStorage();
  expect(await storage.get(oldKey)).toContain('directory');
  await signIn('2');
  expect(await storage.get(oldKey)).toContain('directory');
  const online = vi.fn(async () => reply('new directory'));
  const shared = query(online, true);
  await shared.load([]);
  expect(shared.peek().value).toBe('directory');
  expect(online).not.toHaveBeenCalled();
  await signIn('3', 'native.other');
  expect(await storage.get(oldKey)).toBeUndefined();
  await shared.load([]);
  expect(shared.peek().value).toBe('new directory');
});
it('coalesces five team hints into one in-flight plus one trailing status read', async () => {
  await signIn();
  for (let index = 0; index < 5; index++) await change('connected', '1', true);
  expect(statuses).toBe(1);
  const first = status;
  status = deferred();
  first.resolve(response(view()));
  await vi.advanceTimersByTimeAsync(0);
  expect(statuses).toBe(2);
  expect(owner.hostedSnapshot().blocked).toBe(true);
  status.resolve(response(view()));
  await vi.advanceTimersByTimeAsync(0);
  expect(statuses).toBe(2);
  expect(owner.hostedSnapshot().blocked).toBe(false);
});
it('does not subscribe in local mode', async () => {
  vi.mocked(fetch).mockResolvedValueOnce({ ok: false, status: 404 } as Response);
  expect(await api.nativeStartup()).toBeNull();
  expect(vi.mocked(fetch).mock.calls.map(call => call[0])).toEqual(['/desktop/native']);
});
