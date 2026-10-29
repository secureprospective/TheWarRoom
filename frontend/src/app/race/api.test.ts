import { afterEach, describe, expect, it, vi } from 'vitest';
import { raceAPI, subscribeRace } from './api';
import { createRaceState } from './state';
import wire from './wire.testdata.json';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const flush = () => new Promise<void>(resolve => setTimeout(resolve, 5));

describe('Race transport adapter', () => {
  it('routes both methods only through race.API and parses the response', async () => {
    const TargetRace = vi.fn(async () => wire.race);
    const TargetRaceRefresh = vi.fn(async () => {});
    vi.stubGlobal('window', { go: { race: { API: { TargetRace, TargetRaceRefresh } } } });
    expect((await raceAPI.read(3, '0001', '0001')).standings.id).toBe('M-002');
    expect(TargetRace).toHaveBeenCalledWith(3, '0001', '0001', '');
    await raceAPI.refresh(true);
    expect(TargetRaceRefresh).toHaveBeenCalledWith(true);
    TargetRace.mockResolvedValueOnce({ ...wire.race, week: 99 });
    await expect(raceAPI.read(99, '', '')).rejects.toThrow('week outside');
  });

  it.each([undefined, {}])('does not substitute a fixture without desktop bindings: %s', async window => {
    vi.stubGlobal('window', window);
    const state = createRaceState(raceAPI, () => '');
    await state.load(false);
    expect(state.peek().error).toContain(
      'Power rankings could not be updated. Try refreshing again.',
    );
    expect(state.peek().pending).toBe(false);
    await expect(raceAPI.refresh(false)).rejects.toThrow('require the desktop app');
    const stop = subscribeRace(() => { throw new Error('unexpected browser event'); });
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
    const stop = subscribeRace(changed);
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
    subscribeRace(vi.fn())();
    await flush();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });
});
