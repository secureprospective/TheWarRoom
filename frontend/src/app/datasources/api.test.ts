import { afterEach, describe, expect, it, vi } from 'vitest';
import { sourcesAPI, subscribeSources } from './api';
import { dataProvider } from './provider';
import wire from './wire.testdata.json';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const flush = () => new Promise<void>(resolve => setTimeout(resolve, 0));

describe('Sources transport adapter', () => {
  it('routes every bound method through sources.API and keeps parsed readings', async () => {
    const api = {
      TargetDataSources: vi.fn(async () => wire.data),
      TargetCrosswalk: vi.fn(async () => wire.crosswalk),
      GetCrosswalkReport: vi.fn(async () => ({ ok: true })),
      LoadCrosswalk: vi.fn(async () => ({ ok: true })),
      TargetDataRefresh: vi.fn(async () => wire.result),
      SetLeagueArchiveDir: vi.fn(async () => wire.result),
    };
    vi.stubGlobal('window', { go: { sources: { API: api } } });
    expect((await dataProvider.read()).archiveDir).toBe('/fixture/raw');
    expect((await dataProvider.crosswalk()).id).toBe('C-08');
    expect((await sourcesAPI.crosswalkReport()).ok).toBe(true);
    expect((await sourcesAPI.loadCrosswalk()).ok).toBe(true);
    expect((await dataProvider.refresh('archive')).ok).toBe(true);
    expect(api.TargetDataRefresh).toHaveBeenCalledWith('archive');
    await dataProvider.folder('/fixture/raw');
    expect(api.SetLeagueArchiveDir).toHaveBeenCalledWith('/fixture/raw');
    for (const method of Object.values(api)) expect(method).toHaveBeenCalledTimes(1);
  });

  it('preserves the honest no-window/no-go path for every operation', async () => {
    for (const window of [undefined, {}]) {
      vi.stubGlobal('window', window);
      for (const call of [sourcesAPI.read, sourcesAPI.crosswalk, sourcesAPI.crosswalkReport,
        sourcesAPI.loadCrosswalk, () => sourcesAPI.refresh('league'), () => sourcesAPI.folder('/raw')]) {
        await expect(call()).rejects.toThrow(
          'Data and sources needs the desktop app. No sample data is substituted.',
        );
      }
      const stop = subscribeSources(() => { throw new Error('fixture event fired'); });
      await flush();
      stop();
    }
  });

  it('preserves event catch-up, delivery and cleanup through the shared provider', async () => {
    const unsubscribe = vi.fn();
    let subscribed!: () => void;
    const ready = new Promise<void>(resolve => { subscribed = resolve; });
    let callback: (() => void) | undefined;
    const EventsOnMultiple = vi.fn((_name: string, changed: () => void) => {
      callback = changed;
      subscribed();
      return unsubscribe;
    });
    vi.stubGlobal('window', { go: {}, runtime: { EventsOnMultiple } });
    const changed = vi.fn();
    const stop = subscribeSources(changed);
    await ready;
    expect(EventsOnMultiple).toHaveBeenCalledWith('target:season', changed, -1);
    expect(changed).toHaveBeenCalledTimes(1);
    callback?.();
    expect(changed).toHaveBeenCalledTimes(2);
    stop();
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('does not subscribe after disposal while the runtime import is pending', async () => {
    const EventsOnMultiple = vi.fn();
    vi.stubGlobal('window', { go: {}, runtime: { EventsOnMultiple } });
    const stop = subscribeSources(vi.fn());
    stop();
    await flush();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });
});
