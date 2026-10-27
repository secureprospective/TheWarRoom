import { expect, it, vi } from 'vitest';
import { FixtureProvider } from '../data/provider';
import { heldLineup } from '../data/lineupTestData';
import type { LineupReading } from '../data/contract';
import { createLineups } from './lineups';

it('ignores the first slow response after the second fast response', async () => {
  let resolve!: (r: LineupReading) => void;
  const slow = new Promise<LineupReading>((done) => { resolve = done; });
  const provider = new FixtureProvider();
  const r = heldLineup();
  const read = vi.fn().mockReturnValueOnce(slow).mockResolvedValueOnce({ ...r, week: 6 });
  provider.lineup = read;
  const cache = createLineups(provider);
  const changed = vi.fn();
  const stop = cache.subscribe(changed);
  const first = cache.refresh('0025');
  await cache.refresh('0025');
  expect(cache.peek('0025').reading?.week).toBe(6);
  resolve(r);
  await first;
  expect(cache.peek('0025').reading?.week).toBe(6);
  expect(changed).toHaveBeenCalledTimes(1);
  stop();
});
it('keeps franchise caches independent and suppresses equal readings by value', async () => {
  const provider = new FixtureProvider();
  const r = heldLineup();
  const read = vi.fn((id: string) => Promise.resolve({ ...structuredClone(r), franchise: id }));
  provider.lineup = read;
  const cache = createLineups(provider);
  const changed = vi.fn();
  cache.subscribe(changed);
  for (const id of ['0025', '0001', '0025']) {
    await cache.refresh(id);
  }
  expect(cache.peek('0025').reading?.franchise).toBe('0025');
  expect(cache.peek('0001').reading?.franchise).toBe('0001');
  expect(changed).toHaveBeenCalledTimes(2);
});

it('retains held lineup on failure and exposes no extra pending notifications on retry', async () => {
  const provider = new FixtureProvider();
  const held = heldLineup();
  provider.lineup = async () => held;
  const cache = createLineups(provider);
  const changed = vi.fn();
  cache.subscribe(changed);
  await cache.refresh('0001');
  provider.lineup = async () => { throw new Error('lineup unavailable'); };
  await cache.refresh('0001');
  expect(cache.peek('0001')).toEqual({ reading: held, error: 'lineup unavailable' });
  expect(changed).toHaveBeenCalledTimes(2);
  let finish!: (value: LineupReading) => void;
  provider.lineup = () => new Promise(resolve => { finish = resolve; });
  const retry = cache.refresh('0001');
  expect(cache.peek('0001').error).toBe('lineup unavailable');
  expect(changed).toHaveBeenCalledTimes(2);
  finish(held);
  await retry;
  expect(cache.peek('0001')).toEqual({ reading: held });
  expect(changed).toHaveBeenCalledTimes(3);
});
