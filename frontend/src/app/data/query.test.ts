import { expect, it, vi } from 'vitest';
import { createReadQuery, readKey } from './query';
import { readCommands } from '../commands/reads.gen';

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (cause: unknown) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
function raceQuery(
  read: (season: number, team: string, other: string) => Promise<string> = async () => 'held',
) {
  return createReadQuery('race.TargetRace', {
    initial: (): { value?: string; selection: number } => ({ selection: 0 }), read,
    accept: value => ({ value }),
  });
}

it('uses registry command identity and validated, canonical complete scopes', () => {
  const first = readKey('comms.TargetComms', [{ team: '0001', kind: 'trade' }]);
  expect(first).toContain(readCommands['comms.TargetComms'].command);
  expect(first).toBe(readKey('comms.TargetComms', [{ kind: 'trade', team: '0001' }]));
  expect(first).not.toBe(readKey('comms.TargetComms', [{ team: '0002', kind: 'trade' }]));
  expect(readKey('race.TargetRace', [2026, '0001', '0002']))
    .not.toBe(readKey('race.TargetRace', [2025, '0001', '0002']));
  expect(readKey('race.TargetRace', [1, '0001', '0001']))
    .not.toBe(readKey('race.TargetRace', [1, '0001', '0002']));
  expect(() => readKey('race.TargetRace', [NaN, '', ''])).toThrow('expected integer');
  expect(() => readKey('draft.TargetDraftReports', [1] as never)).toThrow('argument count');
  expect(() => readKey('comms.TargetComms', [{ team: '0001', kind: '', injected: true }] as never))
    .toThrow('unknown argument');
});

it('rejects malformed arguments before calling a provider and finishes pending', async () => {
  const read = vi.fn(async () => 'held');
  const query = raceQuery(read);
  await query.load(['2026', '', ''] as never);
  expect(read).not.toHaveBeenCalled();
  expect(query.peek()).toMatchObject({ pending: false, error: expect.stringContaining('expected integer') });
});

it('retains last-good data on failure, clears errors on retry, and revalidates by default', async () => {
  const read = vi.fn(async () => 'first');
  const query = raceQuery(read);
  await query.load([2026, '', '']);
  read.mockRejectedValueOnce(new Error('offline'));
  await query.load([2025, '', '']);
  expect(query.peek()).toMatchObject({ value: 'first', pending: false, error: 'Error: offline' });
  read.mockResolvedValueOnce('retried');
  await query.load([2026, '', '']);
  expect(query.peek()).toMatchObject({ value: 'retried', error: '', pending: false });
  expect(read).toHaveBeenCalledTimes(3);
});

it('allows explicit cache-first reads, isolates reader instances and bounds the cache', async () => {
  const read = vi.fn(async (season: number) => String(season));
  const query = raceQuery(read);
  await query.load([2026, '', '']);
  await query.load([2026, '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(1);
  const another = raceQuery(read);
  await another.load([2026, '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(2);
  for (let i = 0; i < 33; i++) await query.load([2000 + i, '', '']);
  const count = read.mock.calls.length;
  await query.load([2000, '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(count + 1);
});

it('keeps action admission explicit, invalidates before actions and samples scope afterward', async () => {
  let team = '0001';
  const read = vi.fn(async (_season: number, scope: string) => scope);
  const query = raceQuery(read);
  await query.load([2026, team, '']);
  await query.load(() => [2026, team, ''], {
    policy: 'cache-first', before: async () => { team = '0002'; },
  });
  expect(read).toHaveBeenLastCalledWith(2026, '0002', '');
  expect(query.peek().value).toBe('0002');
  const count = read.mock.calls.length;
  await query.load([2026, team, ''], { before: async () => { throw new Error('refresh refused'); } });
  expect(read).toHaveBeenCalledTimes(count);
  expect(query.peek()).toMatchObject({ value: '0002', error: 'Error: refresh refused', pending: false });
});

it.each(['success', 'failure'] as const)('rejects a late older %s after the newest result', async outcome => {
  const old = deferred<string>(), newer = deferred<string>();
  const read = vi.fn().mockReturnValueOnce(old.promise).mockReturnValueOnce(newer.promise);
  const query = raceQuery(read);
  const first = query.load([2025, '', '']);
  const second = query.load([2026, '', '']);
  newer.resolve('current');
  await second;
  if (outcome === 'success') old.resolve('stale'); else old.reject(new Error('late error'));
  await first;
  expect(query.peek()).toMatchObject({ value: 'current', error: '', pending: false });
});

it('invalidates pending work, resets private data and cannot repopulate after reset', async () => {
  const old = deferred<string>();
  const read = vi.fn(async () => 'held').mockReturnValueOnce(old.promise);
  const query = raceQuery(read);
  const pending = query.load([2026, '', '']);
  query.invalidate();
  old.resolve('invalidated');
  await pending;
  expect(query.peek().value).toBeUndefined();
  await query.load([2026, '', ''], { policy: 'cache-first' });
  expect(query.peek().value).toBe('held');
  query.reset();
  expect(query.peek()).toEqual({ selection: 0, pending: false, error: '' });
  await query.load([2026, '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(3);
});

it('does not leak another pending request through an older finally block', async () => {
  const old = deferred<string>(), newer = deferred<string>();
  const query = raceQuery(vi.fn().mockReturnValueOnce(old.promise).mockReturnValueOnce(newer.promise));
  const first = query.load([2025, '', '']);
  const second = query.load([2026, '', '']);
  old.resolve('old');
  await first;
  expect(query.peek().pending).toBe(true);
  newer.resolve('new');
  await second;
  expect(query.peek()).toMatchObject({ pending: false, value: 'new' });
});

// These expressions must remain compile-time errors, without executing a transport.
function typeChecks() {
  const query = raceQuery();
  // @ts-expect-error query loads retain the registry's complete positional tuple
  void query.load([2026]);
  // @ts-expect-error query arguments cannot substitute numbers for string franchise ids
  void query.load([2026, 1, '']);
  // @ts-expect-error writes are not query commands
  readKey('draft.TargetOpenMFLDraftRoom', []);
  // @ts-expect-error a read's argument tuple comes from the registry binding
  readKey('race.TargetRace', ['one', '', '']);
  // @ts-expect-error a read requires all its arguments
  readKey('race.TargetRace', [2026]);
}
void typeChecks;

it('validates static inputs before actions and detaches them across an awaited refresh', async () => {
  const before = vi.fn(async () => {});
  const read = vi.fn(async (_season: number, team: string) => team);
  const query = raceQuery(read);
  await query.load(['bad', '', ''] as never, { before });
  expect(before).not.toHaveBeenCalled();
  expect(read).not.toHaveBeenCalled();
  const gate = deferred<void>();
  const args: [number, string, string] = [2026, '0001', ''];
  const request = query.load(args, { before: () => gate.promise });
  args[1] = '0002';
  gate.resolve();
  await request;
  expect(read).toHaveBeenLastCalledWith(2026, '0001', '');
});

it('does not erase unexpected undefined fields before runtime validation', async () => {
  const read = vi.fn(async (_scope: { team: string; kind: string }) => 'held');
  const query = createReadQuery('comms.TargetComms', {
    initial: (): { value?: string } => ({}), read, accept: value => ({ value }),
  });
  await query.load([{ team: '', kind: '', injected: undefined }] as never);
  expect(read).not.toHaveBeenCalled();
  expect(query.peek().error).toContain('unknown argument injected');
});

it('keeps the latest failure instead of replacing it with an older success', async () => {
  const old = deferred<string>(), newer = deferred<string>();
  const query = raceQuery(vi.fn().mockReturnValueOnce(old.promise).mockReturnValueOnce(newer.promise));
  const first = query.load([2025, '', '']);
  const second = query.load([2026, '', '']);
  newer.reject(new Error('current failure'));
  await second;
  old.resolve('stale success');
  await first;
  expect(query.peek()).toEqual({ selection: 0, pending: false, error: 'Error: current failure' });
});

it('does not cache a failed publication adapter and fences an awaited action after reset', async () => {
  const read = vi.fn(async () => 'held');
  let reject = true;
  const query = createReadQuery('draft.TargetDraftReports', {
    initial: (): { value?: string } => ({}), read,
    accept: value => { if (reject) throw new Error('bad parsed view'); return { value }; },
  });
  await query.load([]);
  reject = false;
  await query.load([], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(2);
  const gate = deferred<void>();
  const waiting = query.load([], { before: () => gate.promise });
  query.reset();
  gate.resolve();
  await waiting;
  expect(read).toHaveBeenCalledTimes(2);
  expect(query.peek()).toEqual({ pending: false, error: '' });
});

it('does not reuse the same-scope cache after an explicit refresh action', async () => {
  const read = vi.fn(async () => 'before refresh');
  const query = raceQuery(read);
  await query.load([2026, '0001', '']);
  read.mockResolvedValueOnce('after refresh');
  const before = vi.fn(async () => {});
  await query.load([2026, '0001', ''], { before, policy: 'cache-first' });
  expect(before).toHaveBeenCalledOnce();
  expect(read).toHaveBeenCalledTimes(2);
  expect(query.peek().value).toBe('after refresh');
});

it('fences pending detail publication while keeping the single last-scope cache', async () => {
  const delayed = deferred<string>();
  const read = vi.fn().mockResolvedValueOnce('held').mockReturnValueOnce(delayed.promise)
    .mockResolvedValueOnce('another');
  const accepted = vi.fn(value => ({ value }));
  const query = createReadQuery('race.TargetRace', {
    initial: (): { value?: string } => ({}), cacheSize: 1, read, accept: accepted,
  });
  await query.load([2025, '', '']);
  const pending = query.load([2026, '', '']);
  query.fence();
  delayed.resolve('late');
  await pending;
  expect(query.peek()).toMatchObject({ value: 'held', pending: false });
  expect(accepted).toHaveBeenCalledTimes(1);
  await query.load([2025, '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(2);
  await query.load([2024, '', ''], { policy: 'cache-first' });
  await query.load([2025, '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(4);
});

it('validates every same-scope companion before reading a compound view', async () => {
  const read = vi.fn(async () => 'never');
  const query = createReadQuery('draft.TargetDraftReports', {
    initial: () => ({}), companions: ['race.TargetRace'], read, accept: () => ({}),
  });
  await query.load([]);
  expect(read).not.toHaveBeenCalled();
  expect(query.peek().error).toContain('race.TargetRace: wrong argument count');
});

it('shares a wrapper-owned view without losing independently accepted fields', async () => {
  const league = createReadQuery('control.TargetAccounting', {
    initial: (): { accounting?: string; news?: string } => ({}),
    read: async () => 'ledger', accept: accounting => ({ accounting }),
  });
  const news = createReadQuery('control.TargetSiteNews', {
    initial: (): { accounting?: string; news?: string } => ({}), store: league.store,
    read: async () => 'news', accept: value => ({ news: value }),
  });
  await Promise.all([league.load([]), news.load([])]);
  expect(league.peek()).toEqual({ accounting: 'ledger', news: 'news', pending: false, error: '' });
});

