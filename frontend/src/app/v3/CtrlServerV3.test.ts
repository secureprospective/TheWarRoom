import { describe, expect, it } from 'vitest';
import { ledgerDetail } from './CtrlServerV3';
import type { LedgerEntry } from '../commissioner/proxy';

const entry = (why: string): LedgerEntry => ({
  id: 'e1', who: 'Chris', what: 'Set reception to 1.0', why,
  when: '2026-11-20T12:00:00Z', gravity: 'league', undo: '',
});

describe('ledgerDetail', () => {
  it('formats actor, reason and safeguard', () => {
    expect(ledgerDetail(entry('Vote passed')))
      .toBe('By Chris · Reason: Vote passed · Safeguard: league');
  });
  it('falls back when no reason was recorded', () => {
    expect(ledgerDetail(entry('')))
      .toBe('By Chris · Reason: No reason recorded · Safeguard: league');
  });
});
