import { expect, it, vi } from 'vitest';
import { createReadQuery } from './query';
import { createReadSession } from './session';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(yes => { resolve = yes; });
  return { promise, resolve };
}
function query(read: (season: number) => Promise<string> = async season => `held ${season}`) {
  return createReadQuery('race.TargetRace', {
    initial: (): { value?: string } => ({}),
    read: (season: number) => read(season), accept: value => ({ value }),
  });
}

it('keeps a shared view but drops its cache and fences an old-session reply', async () => {
  const session = createReadSession();
  const read = vi.fn(async (season: number) => `held ${season}`);
  const shared = query(read);
  session.join(shared, 'shared');
  await shared.load([2026, '', '', '']);
  const late = deferred<string>();
  read.mockImplementationOnce(() => late.promise);
  const pending = shared.load([2025, '', '', '']);
  session.end();
  late.resolve('old account');
  await pending;
  expect(shared.peek()).toMatchObject({ value: 'held 2026', pending: false, error: '' });
  await shared.load([2026, '', '', ''], { policy: 'cache-first' });
  expect(read).toHaveBeenCalledTimes(3);
});

it('clears a private view back to its initial state', async () => {
  const session = createReadSession();
  const mine = query();
  session.join(mine, 'private');
  await mine.load([2026, '', '', '']);
  session.end();
  expect(mine.peek()).toEqual({ pending: false, error: '' });
});

it('leaves a query that left, or joined while ending, to its own lifetime', async () => {
  const session = createReadSession();
  const left = query();
  const leave = session.join(left, 'private');
  leave();
  const later = query();
  const joiner = { invalidate: vi.fn(), reset: vi.fn(() => { session.join(later, 'private'); }) };
  session.join(joiner, 'private');
  await left.load([2026, '', '', '']);
  await later.load([2026, '', '', '']);
  session.end();
  expect(left.peek().value).toBe('held 2026');
  expect(later.peek().value).toBe('held 2026');
  expect(joiner.reset).toHaveBeenCalledTimes(1);
  expect(joiner.invalidate).not.toHaveBeenCalled();
  session.end();
  expect(later.peek().value).toBeUndefined();
});
