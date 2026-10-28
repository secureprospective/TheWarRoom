import { afterEach, describe, expect, it, vi } from 'vitest';
import { marketAPI, subscribeSeason } from './api';
import { createAssetState } from '../assets/state';
import { marketReportState } from '../reports/marketState';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const flush = () => new Promise<void>(resolve => setTimeout(resolve, 5));
describe('Market transport adapter', () => {
  it('routes every method only through market.API', async () => {
    const api = {
      TargetAssetBoard: vi.fn(async () => 'board'), TargetMarketReports: vi.fn(async () => 'reports'),
      TargetMarketReportsRefresh: vi.fn(async () => {}), TargetMarket: vi.fn(),
    };
    vi.stubGlobal('window', { go: { market: { API: api } } });
    expect(await marketAPI.assetBoard()).toBe('board');
    expect(await marketAPI.reports()).toBe('reports');
    await marketAPI.refreshReports(true);
    expect(api.TargetMarketReportsRefresh).toHaveBeenCalledWith(true);

  });

  it.each([undefined, {}])('does not substitute a fixture without desktop bindings: %s', async window => {
    vi.stubGlobal('window', window);
    await expect(marketAPI.assetBoard()).rejects.toThrow(
      'Player values require the desktop app; no browser fixture is substituted.',
    );
    await expect(marketAPI.reports()).rejects.toThrow(
      'Player market reports require the desktop app; no browser fixture is substituted.',
    );
    await expect(marketAPI.refreshReports(false)).rejects.toThrow('require the desktop app');
    await marketReportState.refresh();
    expect(marketReportState.peek().error).toContain('Player market reports require the desktop app');
    const assets = createAssetState(
      async () => { await marketAPI.assetBoard(); throw new Error('unreachable'); },
      () => () => {},
    );
    await assets.load();
    expect(assets.peek().error).toContain('Player values require the desktop app');
    const stop = subscribeSeason(() => { throw new Error('unexpected browser event'); });
    await flush();
    stop();
  });

  it('delivers season events and removes the listener', async () => {
    const unsubscribe = vi.fn();
    let callback: (() => void) | undefined;
    const EventsOnMultiple = vi.fn((_name: string, changed: () => void) => {
      callback = changed;
      return unsubscribe;
    });
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const changed = vi.fn();
    const stop = subscribeSeason(changed);
    for (let i = 0; i < 100 && EventsOnMultiple.mock.calls.length === 0; i++) await flush();
    expect(EventsOnMultiple).toHaveBeenCalledWith('target:season', changed, -1);
    callback?.();
    expect(changed).toHaveBeenCalledTimes(1);
    stop();
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('does not attach after disposal before runtime import resolves', async () => {
    const EventsOnMultiple = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    subscribeSeason(vi.fn())();
    await flush();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });
});
