import { afterEach, expect, it, vi } from 'vitest';
import * as api from './api';

const methods = Object.entries(api).filter(([name]) => name.startsWith('Target'));
const heldWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (heldWindow) Object.defineProperty(globalThis, 'window', heldWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

it('routes every moved method through the generated Commissioner module', async () => {
  const calls = Object.fromEntries(methods.map(([name]) => [name, vi.fn(async () => name)]));
  vi.stubGlobal('window', { go: { commissioner: { API: calls } } });
  expect(methods).toHaveLength(47);
  for (const [name, method] of methods) {
    const call = method as (...args: unknown[]) => Promise<unknown>;
    const args = ['scope', { request: 'retry-key' }, 2026, 1, 'UTC', 'viewer', 'extra'];
    expect(await call(...args)).toBe(name);
    // The generated signature forwards exactly its declared arguments, not a rest tuple.
    expect(calls[name]).toHaveBeenCalledWith(...args.slice(0, method.length));
  }
  expect(api.ledgerBindings().TargetCommissionerLedger).toBe(api.TargetCommissionerLedger);
  expect(api.nativeBindings().TargetNativeAssets).toBe(api.TargetNativeAssets);
  expect(api.draftBindings().TargetNativeDraft).toBe(api.TargetNativeDraft);
});

it('preserves each caller’s no-desktop refusal without contacting a binding', () => {
  for (const window of [undefined, {}]) {
    vi.stubGlobal('window', window);
    expect(api.ledgerBindings).toThrow('The league ledger requires the desktop app.');
    expect(api.nativeBindings).toThrow('League changes require the desktop app.');
    expect(api.draftBindings).toThrow('The draft room requires the desktop app.');
  }
});
