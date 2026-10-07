import { afterEach, describe, expect, it, vi } from 'vitest';
import { FixtureProvider, LiveProvider } from './provider';
import { heldLineup } from './lineupTestData';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
describe('lineup provider', () => {
  it('calls the binding and parses strictly, never replacing bad data with fixtures', async () => {
    const r = heldLineup();
    const call = vi.fn().mockResolvedValue(r);
    vi.stubGlobal('window', { go: { main: { App: { TargetLineup: call } } } });
    const provider = new LiveProvider();
    expect(await provider.lineup('0025')).toEqual(r);
    expect(call).toHaveBeenCalledWith('0025');
    call.mockResolvedValue({ ...r, starters: null });
    await expect(provider.lineup('0025')).rejects.toThrow('lineup.starters');
    call.mockRejectedValue(new Error('target lineup: no snapshot loaded'));
    await expect(provider.lineup('0025')).rejects.toThrow('target lineup: no snapshot loaded');
  });
  it('fixtures reject without subscribing or inventing a lineup', async () => {
    const listener = vi.fn();
    const subscribe = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple: subscribe } });
    const provider = new FixtureProvider();
    await expect(provider.lineup('0025')).rejects.toThrow('Lineups need the desktop app');
    provider.onSeasonChange(listener)();
    expect(subscribe).not.toHaveBeenCalled();
    expect(listener).not.toHaveBeenCalled();
  });
  it('subscribes to season changes, catches up once, and releases the subscription', async () => {
    const unsubscribe = vi.fn();
    const subscribe = vi.fn(() => unsubscribe);
    const listener = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple: subscribe } });
    const stop = new LiveProvider().onSeasonChange(listener);
    await import('../../../wailsjs/runtime/runtime');
    expect(subscribe).toHaveBeenCalledWith('target:season', listener, -1);
    expect(listener).toHaveBeenCalledTimes(1);
    listener();
    expect(listener).toHaveBeenCalledTimes(2);
    stop();
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });
  it('does not subscribe after cleanup during a lazy import', async () => {
    const subscribe = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple: subscribe } });
    new LiveProvider().onSeasonChange(vi.fn())();
    await import('../../../wailsjs/runtime/runtime');
    expect(subscribe).not.toHaveBeenCalled();
  });
});
