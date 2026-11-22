import { afterEach, describe, expect, it, vi } from 'vitest';
import { readAgeArc } from './api';
import { readPlayerUsage } from '../research/api';
import { hostedWorkflow } from '../control/nativeApi';

vi.mock('../control/nativeApi', async importOriginal => ({
  ...await importOriginal<typeof import('../control/nativeApi')>(),
  hostedWorkflow: vi.fn(async () => ({ data: { source: 'hosted' } })),
}));

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  vi.clearAllMocks();
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('player hosted read adapters', () => {
  it('uses hosted workflow for age arc and usage when a hosted league is supplied', async () => {
    vi.stubGlobal('window', {});
    expect(await readAgeArc('0099', 'native.hosted')).toEqual({ source: 'hosted' });
    expect(await readPlayerUsage('0099', 'native.hosted')).toEqual({ source: 'hosted' });
    expect(hostedWorkflow).toHaveBeenNthCalledWith(1, { command: 'player.agearc.read', mode: 'read',
      arguments: { league: 'native.hosted', player: '0099' } });
    expect(hostedWorkflow).toHaveBeenNthCalledWith(2, { command: 'player.usage.read', mode: 'read',
      arguments: { league: 'native.hosted', player: '0099' } });
  });

  it('returns a typed unavailable result for hosted facts the server does not hold', async () => {
    vi.stubGlobal('window', {});
    vi.mocked(hostedWorkflow).mockResolvedValue({ data: { unavailable: true, reason: 'input-not-held' },
      label: 'Hosted', writable: false, football: '' });
    expect(await readAgeArc('0099', 'native.hosted')).toEqual({ unavailable: true,
      reason: 'input-not-held' });
    expect(await readPlayerUsage('0099', 'native.hosted')).toEqual({ unavailable: true,
      reason: 'input-not-held' });
  });

  it('keeps local reads on their Wails bindings without hosted league context', async () => {
    const ageArc = vi.fn(async () => ({ source: 'desktop-arc' }));
    const usage = vi.fn(async () => ({ source: 'desktop-usage' }));
    vi.stubGlobal('window', { go: { market: { API: { TargetAgeArc: ageArc } },
      research: { API: { TargetPlayerUsage: usage } } } });
    expect(await readAgeArc('0099')).toEqual({ source: 'desktop-arc' });
    expect(await readPlayerUsage('0099')).toEqual({ source: 'desktop-usage' });
    expect(ageArc).toHaveBeenCalledWith('0099');
    expect(usage).toHaveBeenCalledWith('0099');
    expect(hostedWorkflow).not.toHaveBeenCalled();
  });
});
