import { afterEach, expect, it, vi } from 'vitest';
import { lineupReceipt } from '../data/lineupReceiptTestData';
import { targetDraftLineup } from './api';
import { parseReceipt } from '../data/parseEnvelope';

afterEach(() => vi.unstubAllGlobals());

it('routes the namespaced local lineup plan to the existing desktop binding', async () => {
  const receipt = lineupReceipt();
  const TargetDraftLineup = vi.fn().mockResolvedValue(receipt);
  vi.stubGlobal('window', { go: { contracts: { API: { TargetDraftLineup } } } });
  const starters = receipt.spec.expected.lineup?.starters ?? [];
  expect(parseReceipt(await targetDraftLineup('0025', starters)).spec.intent).toBe('mfl.lineup.set');
  expect(TargetDraftLineup).toHaveBeenCalledExactlyOnceWith('0025', starters);
  TargetDraftLineup.mockRejectedValueOnce(new Error('local plan refused'));
  await expect(targetDraftLineup('0025', starters)).rejects.toThrow('local plan refused');
});

it('refuses a missing desktop binding and malformed arguments before dispatch', async () => {
  vi.stubGlobal('window', {});
  await expect(targetDraftLineup('0025', ['0042'])).rejects.toThrow('need the desktop app');
  const TargetDraftLineup = vi.fn();
  vi.stubGlobal('window', { go: { contracts: { API: { TargetDraftLineup } } } });
  await expect(targetDraftLineup('0025', [42] as unknown as string[])).rejects.toThrow();
  expect(TargetDraftLineup).not.toHaveBeenCalled();
});
