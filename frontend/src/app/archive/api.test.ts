import { afterEach, describe, expect, it, vi } from 'vitest';
import { archiveAPI, subscribeArchive } from './api';
import { createArchiveState } from './state';
import wire from '../../../../testdata/wire/archive/unimported.json';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const flush = async () => {
  await vi.dynamicImportSettled();
  await new Promise<void>(resolve => setTimeout(resolve, 0));
};

describe('Archive transport adapter', () => {
  it('reads through its feature namespace and passes each choice unchanged', async () => {
    const TargetArchive = vi.fn(async () => wire);
    vi.stubGlobal('window', { go: { archive: { API: { TargetArchive } } } });
    expect((await archiveAPI.read(2015, '0001', '0002')).board.title).toBe('Archive');
    expect(TargetArchive).toHaveBeenCalledWith(2015, '0001', '0002');
  });

  it('keeps the browser path honest and finishes pending reads', async () => {
    for (const window of [undefined, {}]) {
      vi.stubGlobal('window', window);
      const state = createArchiveState(archiveAPI.read, () => '0001');
      await state.load();
      expect(state.peek().error)
        .toBe('Error: League archive reports require the desktop app; no browser fixture is substituted.');
      expect(state.peek().pending).toBe(false);
      const stop = subscribeArchive(() => { throw new Error('browser event fired'); });
      await flush();
      stop();
    }
  });

  it('delivers season events and cleans up on disposal', async () => {
    const unsubscribe = vi.fn();
    let callback: (() => void) | undefined;
    const EventsOnMultiple = vi.fn((_name: string, changed: () => void) => {
      callback = changed;
      return unsubscribe;
    });
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const changed = vi.fn();
    const stop = subscribeArchive(changed);
    await flush();
    expect(EventsOnMultiple).toHaveBeenCalledWith('target:season', changed, -1);
    callback?.();
    expect(changed).toHaveBeenCalledTimes(1);
    stop();
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('does not subscribe after early disposal', async () => {
    const EventsOnMultiple = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const stop = subscribeArchive(vi.fn());
    stop();
    await flush();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });

  it('retains last-good data on read failure and rejects malformed wire data', async () => {
    const TargetArchive = vi.fn().mockResolvedValueOnce(wire).mockResolvedValueOnce({});
    vi.stubGlobal('window', { go: { archive: { API: { TargetArchive } } } });
    const state = createArchiveState(archiveAPI.read, () => '0001');
    await state.load();
    const held = state.peek().archive;
    await state.choose('refresh', '');
    expect(state.peek().archive).toBe(held);
    expect(state.peek().error).toBe('Error: Draft reports: expected object');
    expect(state.peek().pending).toBe(false);
  });
});
