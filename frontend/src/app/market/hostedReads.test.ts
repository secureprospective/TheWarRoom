import { afterEach, describe, expect, it, vi } from 'vitest';
import { readAgeArc } from './api';
import { readPlayerUsage } from '../research/api';
import { hostedWorkflow } from '../control/nativeApi';

vi.mock('../control/nativeApi', () => ({
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

  it('does not invoke Wails when hosted facts are unavailable', async () => {
    const ageArc = vi.fn();
    const usage = vi.fn();
    vi.stubGlobal('window', { go: { market: { API: { TargetAgeArc: ageArc } },
      research: { API: { TargetPlayerUsage: usage } } } });
    vi.mocked(hostedWorkflow).mockRejectedValue(new Error(
      'player age arc: hosted player reading is unavailable'));
    await expect(readAgeArc('0099', 'native.hosted')).rejects.toThrow(
      'hosted player reading is unavailable');
    await expect(readPlayerUsage('0099', 'native.hosted')).rejects.toThrow(
      'hosted player reading is unavailable');
    expect(ageArc).not.toHaveBeenCalled();
    expect(usage).not.toHaveBeenCalled();
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
