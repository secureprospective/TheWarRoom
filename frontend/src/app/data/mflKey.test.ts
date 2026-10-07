import { afterEach, describe, expect, it, vi } from 'vitest';
import { parseMFLKeyStatus } from './parse';
import { FixtureProvider, LiveProvider } from './provider';

const absent = { state: 'absent', league: '14432', season: 2026 };
const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('MFL key boundary (§M5)', () => {
  it.each(['absent', 'connected', 'rejected', 'unreachable', 'unavailable'])(
    'accepts the Go %s shape', (state) => {
      const status = { ...absent, state, verifiedAt: '2026-10-07T08:00:00-05:00', detail: 'detail' };
      expect(parseMFLKeyStatus(status)).toEqual(status);
      expect(parseMFLKeyStatus({ ...absent, state })).toEqual({ ...absent, state });
    },
  );
  it.each([
    null, [], {}, { ...absent, state: 'guess' }, { ...absent, key: 'not allowed' },
    { ...absent, league: 14432 }, { ...absent, season: '2026' }, { ...absent, season: 2026.5 },
    { ...absent, verifiedAt: '' }, { ...absent, verifiedAt: null },
    { ...absent, verifiedAt: '2026-10-07' }, { ...absent, verifiedAt: '2026-10-07T00:00:00' },
    { ...absent, verifiedAt: '2026-99-07T00:00:00Z' }, { ...absent, detail: null },
    { ...absent, detail: 7 },
  ])('rejects malformed or extra fields', (value) => {
    expect(() => parseMFLKeyStatus(value)).toThrow('mflKey');
  });
  it('uses all three generated bindings and parses their results', async () => {
    const read = vi.fn().mockResolvedValue(absent);
    const connect = vi.fn().mockResolvedValue({ ...absent, state: 'connected' });
    const forget = vi.fn().mockResolvedValue(absent);
    vi.stubGlobal('window', { go: { main: { App: {
      MFLKeyStatus: read, SetMFLKey: connect, DeleteMFLKey: forget,
    } } } });
    const provider = new LiveProvider();
    expect(await provider.mflKey()).toEqual(absent);
    expect((await provider.connectMFL('candidate')).state).toBe('connected');
    expect(await provider.forgetMFL()).toEqual(absent);
    expect(read).toHaveBeenCalledTimes(1);
    expect(connect).toHaveBeenCalledWith('candidate');
    expect(forget).toHaveBeenCalledTimes(1);
    for (const binding of [read, connect, forget]) binding.mockResolvedValue({ ...absent, state: 'bad' });
    await expect(provider.mflKey()).rejects.toThrow('mflKey.state');
    await expect(provider.connectMFL('candidate')).rejects.toThrow('mflKey.state');
    await expect(provider.forgetMFL()).rejects.toThrow('mflKey.state');
    connect.mockRejectedValue(new Error('app not started'));
    await expect(provider.connectMFL('candidate')).rejects.toThrow('app not started');
  });
  it('fixtures give an honest absent reading and reject both mutations', async () => {
    const provider = new FixtureProvider();
    expect(await provider.mflKey()).toEqual({
      state: 'absent', league: '', season: 0, detail: 'Connecting MFL needs the desktop app',
    });
    await expect(provider.connectMFL('candidate')).rejects.toThrow('Connecting MFL needs the desktop app');
    await expect(provider.forgetMFL()).rejects.toThrow('Connecting MFL needs the desktop app');
  });
});
