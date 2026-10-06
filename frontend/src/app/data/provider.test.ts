import { afterEach, describe, expect, it, vi } from 'vitest';
import fixture from './fixtures/snapshot.json';
import { FixtureProvider, LiveProvider, selectProvider } from './provider';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('provider selection', () => {
  it('chooses fixtures without Wails and returns parsed fixture provenance', async () => {
    vi.stubGlobal('window', {});
    const provider = selectProvider();
    expect(provider).toBeInstanceOf(FixtureProvider);
    const parsed = await provider.snapshot();
    for (const section of Object.values(parsed)) expect(section.provenance.kind).toBe('fixture');
  });

  it('chooses live when window.go exists and calls the binding', async () => {
    const payload = JSON.parse(JSON.stringify(fixture));
    for (const section of Object.values(payload) as {provenance: {kind: string}}[]) section.provenance.kind = 'live';
    const call = vi.fn().mockResolvedValue(payload);
    vi.stubGlobal('window', { go: { main: { App: { TargetSnapshot: call } } } });
    const provider = selectProvider();
    expect(provider).toBeInstanceOf(LiveProvider);
    expect((await provider.snapshot()).players.provenance.kind).toBe('live');
    expect(call).toHaveBeenCalledTimes(1);
  });

  it.each(['outage', 'corrupt payload'])('never substitutes fixtures for a live %s', async cause => {
    const call = cause === 'outage'
      ? vi.fn().mockRejectedValue(new Error('MFL offline'))
      : vi.fn().mockResolvedValue({ players: [] });
    vi.stubGlobal('window', { go: { main: { App: { TargetSnapshot: call } } } });
    const parsed = await selectProvider().snapshot();
    for (const section of Object.values(parsed)) {
      expect(section.provenance.kind).toBe('live');
      expect(section.provenance.freshness.state).toBe('fail');
    }
    expect(parsed.players.value).toEqual([]);
    expect(parsed.franchises.value).toEqual([]);
    expect(parsed.league.value).toEqual({season: 0, franchiseCount: 0});
    expect(call).toHaveBeenCalledTimes(1);
  });
});
