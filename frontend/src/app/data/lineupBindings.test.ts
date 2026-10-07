import { afterEach, expect, it, vi } from 'vitest';
import { LiveProvider, FixtureProvider } from './provider';
import { lineupReceipt } from './lineupReceiptTestData';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
it('checks, drafts and hands off through lazy bindings and strict parsers', async () => {
  const check = vi.fn().mockResolvedValue({ full: true, legal: true, problems: [] });
  const draft = vi.fn().mockResolvedValue(lineupReceipt());
  const handOff = vi.fn().mockResolvedValue(lineupReceipt());
  vi.stubGlobal('window', { go: { main: { App: {
    TargetCheckLineup: check, TargetDraftLineup: draft, TargetHandOff: handOff,
  } } } });
  const provider = new LiveProvider();
  const starters = lineupReceipt().spec.expected.lineup!.starters;
  expect(await provider.checkLineup('0025', starters)).toEqual({ full: true, legal: true, problems: [] });
  expect(check).toHaveBeenCalledWith('0025', starters);
  expect(await provider.draftLineup('0025', starters)).toEqual(lineupReceipt());
  expect(draft).toHaveBeenCalledWith('0025', starters);
  expect(await provider.handOff('lineup-test')).toEqual(lineupReceipt());
  expect(handOff).toHaveBeenCalledWith('lineup-test');
  check.mockResolvedValue({ full: 'yes', legal: true, problems: [] });
  await expect(provider.checkLineup('0025', starters)).rejects.toThrow('lineup.check.full');
  draft.mockResolvedValue({ ...lineupReceipt(), spec: {} });
  await expect(provider.draftLineup('0025', starters)).rejects.toThrow('receipt.spec');
  handOff.mockRejectedValue(new Error('target hand off: not ready'));
  await expect(provider.handOff('lineup-test')).rejects.toThrow('target hand off: not ready');
});
it('subscribes only live to target:moves, catches up once and cleans up', async () => {
  const unsubscribe = vi.fn();
  const subscribe = vi.fn(() => unsubscribe);
  const listener = vi.fn();
  vi.stubGlobal('window', { runtime: { EventsOnMultiple: subscribe } });
  new FixtureProvider().onMovesChange(listener)();
  expect(subscribe).not.toHaveBeenCalled();
  const stop = new LiveProvider().onMovesChange(listener);
  await import('../../../wailsjs/runtime/runtime');
  expect(subscribe).toHaveBeenCalledWith('target:moves', listener, -1);
  expect(listener).toHaveBeenCalledTimes(1);
  stop();
  expect(unsubscribe).toHaveBeenCalledTimes(1);
});
