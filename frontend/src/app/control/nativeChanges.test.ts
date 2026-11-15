import { afterEach, expect, it, vi } from 'vitest';
import { subscribeChanged } from './nativeApi';
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });
it('delivers topics, advances the cursor and cancels on unsubscribe', async () => {
  vi.useFakeTimers();
  const change = { seq: 2, state: 'connected', serverEpoch: 'boot', league: 'native.test',
    topics: [{ kind: 'scores', version: '1', season: 2026, week: 10 }] };
  const fetcher = vi.fn().mockResolvedValueOnce({ ok: true, json: async () => change })
    .mockImplementation(() => new Promise(() => {}));
  vi.stubGlobal('fetch', fetcher);
  const listener = vi.fn();
  const stop = subscribeChanged(listener);
  await vi.advanceTimersByTimeAsync(0);
  expect(listener).toHaveBeenCalledWith(change);
  expect(JSON.parse(fetcher.mock.calls[1][1].body)).toEqual({ after: 2 });
  stop();
  expect(fetcher.mock.calls[1][1].signal.aborted).toBe(true);
  await vi.advanceTimersByTimeAsync(60_000);
  expect(fetcher).toHaveBeenCalledTimes(2);
});
it('backs off after failures and rejects malformed topics', async () => {
  vi.useFakeTimers();
  const fetcher = vi.fn().mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValue({ ok: true, json: async () => ({ seq: 1, state: 'connected', topics: [{}] }) });
  vi.stubGlobal('fetch', fetcher);
  const listener = vi.fn();
  const stop = subscribeChanged(listener);
  await vi.advanceTimersByTimeAsync(249);
  expect(fetcher).toHaveBeenCalledTimes(1);
  await vi.advanceTimersByTimeAsync(1);
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(listener).not.toHaveBeenCalled();
  stop();
});
