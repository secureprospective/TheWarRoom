import { afterEach, describe, expect, it, vi } from 'vitest';
import { createMovesPrefetch, MovesMount, PlayerActMount } from './MovesMount';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { createCommands } from '../commands/registry';
import { parseSnapshot } from '../data/parse';
import fixture from '../data/fixtures/snapshot.json';

const idle = Object.getOwnPropertyDescriptor(globalThis, 'requestIdleCallback');
afterEach(() => {
  if (idle) Object.defineProperty(globalThis, 'requestIdleCallback', idle);
  else Reflect.deleteProperty(globalThis, 'requestIdleCallback');
  vi.useRealTimers();
});

describe('moves idle prefetch', () => {
  it('schedules once after the snapshot loads, not before', () => {
    const schedule = vi.fn();
    vi.stubGlobal('requestIdleCallback', schedule);
    const executor = createCommands(() => undefined);
    expect(schedule).not.toHaveBeenCalled();
    executor.loadSnapshot(parseSnapshot(fixture));
    executor.loadSnapshot(parseSnapshot(fixture));
    expect(schedule).toHaveBeenCalledTimes(1);
  });
  it('loads both view chunks on idle, once', async () => {
    let callback!: () => void;
    vi.stubGlobal('requestIdleCallback', vi.fn((run: () => void) => { callback = run; }));
    const load = vi.fn().mockResolvedValue([]);
    const prefetch = createMovesPrefetch(load);
    prefetch();
    prefetch();
    expect(load).not.toHaveBeenCalled();
    callback();
    await Promise.resolve();
    expect(load).toHaveBeenCalledTimes(1);
  });
  it('renders both prefetched mounts without a Suspense fallback', async () => {
    let callback!: () => Promise<unknown>;
    vi.stubGlobal('requestIdleCallback', (run: typeof callback) => { callback = run; });
    createMovesPrefetch()();
    await callback();
    const snapshot = parseSnapshot(fixture);
    const moves = renderToStaticMarkup(createElement(MovesMount, { snapshot }));
    expect(moves).toContain('My moves');
    expect(moves).not.toContain('Opening My moves…');
    const player = renderToStaticMarkup(createElement(PlayerActMount, {
      subject: { kind: 'player', id: '11675', franchiseId: '0001' },
    }));
    expect(player).toContain('Draft IR placement');
    expect(player).not.toContain('Act loading…');
  });
  it('uses a timer when idle callbacks are unavailable', () => {
    vi.stubGlobal('requestIdleCallback', undefined);
    vi.useFakeTimers();
    const load = vi.fn().mockResolvedValue([]);
    const prefetch = createMovesPrefetch(load);
    prefetch();
    prefetch();
    expect(load).not.toHaveBeenCalled();
    vi.runAllTimers();
    expect(load).toHaveBeenCalledTimes(1);
  });
});
