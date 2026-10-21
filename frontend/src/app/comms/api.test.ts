import { afterEach, describe, expect, it, vi } from 'vitest';
import { commsAPI, subscribeComms } from './api';
import { createCommsState } from './state';
import wire from './wire.testdata.json';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

const flush = () => new Promise<void>(resolve => setTimeout(resolve, 0));
// The adapter loads the Wails runtime with a dynamic import; one macrotask is not always enough
// under load, and a late resolution would leak into the next test. Wait for it to settle.
const runtimeLoaded = async () => {
  await vi.dynamicImportSettled();
  await flush();
};

describe('Comms transport adapter', () => {
  it('uses the feature API namespace for all three operations', async () => {
    const TargetComms = vi.fn(async () => wire);
    const TargetCommsRefresh = vi.fn(async () => {});
    const TargetOpenComms = vi.fn(async () => {});
    vi.stubGlobal('window', { go: { comms: { API: { TargetComms, TargetCommsRefresh, TargetOpenComms } } } });
    expect((await commsAPI.read('0001', 'trade')).feed.id).toBe('C-05');
    expect(TargetComms).toHaveBeenCalledWith({ team: '0001', kind: 'trade' });
    await commsAPI.refresh(true);
    expect(TargetCommsRefresh).toHaveBeenCalledWith(true);
    await commsAPI.open('league-board');
    expect(TargetOpenComms).toHaveBeenCalledWith('league-board');
  });

  it('keeps plain-browser reads honest without accessing missing globals', async () => {
    for (const window of [undefined, {}]) {
      vi.stubGlobal('window', window);
      const state = createCommsState(commsAPI);
      await state.load();
      expect(state.peek().error)
        .toContain('Comms requires the desktop app; no browser fixture is substituted.');
      expect(state.peek().pending).toBe(false);
      const stop = subscribeComms(() => { throw new Error('browser event fired'); });
      await flush();
      stop();
    }
  });

  it('subscribes once and calls the returned unsubscribe on unmount', async () => {
    const unsubscribe = vi.fn();
    let callback: (() => void) | undefined;
    const EventsOnMultiple = vi.fn((_name: string, changed: () => void) => {
      callback = changed;
      return unsubscribe;
    });
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const changed = vi.fn();
    const stop = subscribeComms(changed);
    await runtimeLoaded();
    expect(EventsOnMultiple).toHaveBeenCalledWith('target:season', changed, -1);
    callback?.();
    expect(changed).toHaveBeenCalledTimes(1);
    stop();
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('does not subscribe when unmounted before the runtime import resolves', async () => {
    const EventsOnMultiple = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const stop = subscribeComms(vi.fn());
    stop();
    await runtimeLoaded();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });
});
