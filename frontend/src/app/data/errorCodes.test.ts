import { afterEach, describe, expect, it } from 'vitest';
import {
  attachErrorCode,
  createFailureError,
  errorCodeFor,
  formatCentralTime,
  formatFailureDetails,
  lookupErrorCode,
  newClientErrorCode,
  recordAppRelease,
  registerLeagueProvider,
  resetFailureStateForTest,
  setFailureScreen,
  validErrorCode,
} from './errorCodes';
import { plainFailure } from './plainFailure';
import { createReadQuery } from './query';

afterEach(() => {
  resetFailureStateForTest();
});

describe('errorCodes', () => {
  it('mints unambiguous TWR-CXXX-XXXX client codes without 0/O/1/I', () => {
    const seen = new Set<string>();
    for (let i = 0; i < 100; i++) {
      const code = newClientErrorCode();
      expect(code).toMatch(
        /^TWR-C[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{3}-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}$/,
      );
      expect(code.slice(4)).not.toMatch(/[0O1I]/);
      expect(validErrorCode(code)).toBe(true);
      seen.add(code);
    }
    expect(seen.size).toBeGreaterThan(95);
    expect(validErrorCode('TWR-7K9M-2P4X')).toBe(true);
    expect(validErrorCode('TWR-0000-0000')).toBe(false);
    expect(validErrorCode('TWR-O234-5678')).toBe(false);
    expect(validErrorCode('TWR-1234-5678')).toBe(false);
    expect(validErrorCode('TWR-I234-5678')).toBe(false);
  });

  it('preserves server codes and mints client codes on createFailureError', () => {
    const serverErr = createFailureError('The league refused this action.', 'TWR-7K9M-2P4X');
    expect(serverErr.message).toBe('The league refused this action.');
    expect(errorCodeFor(serverErr)).toBe('TWR-7K9M-2P4X');
    expect(lookupErrorCode('The league refused this action.')).toBe('TWR-7K9M-2P4X');

    const clientErr = createFailureError('Network down.');
    const clientCode = errorCodeFor(clientErr);
    expect(clientCode).toMatch(/^TWR-C/);
    expect(lookupErrorCode('Network down.')).toBe(clientCode);

    const existing = new Error('Local error');
    attachErrorCode(existing, 'TWR-8N3Q-4V6W');
    expect(errorCodeFor(existing)).toBe('TWR-8N3Q-4V6W');
  });

  it('formats Central time and full copy/report details', () => {
    const stamp = formatCentralTime(new Date('2026-11-20T20:15:30Z'));
    expect(stamp).toMatch(/2:15:30 PM C[SD]T/);

    setFailureScreen('League', 'phone/league');
    recordAppRelease(42);
    registerLeagueProvider(() => ({ league: 'native.hosted', name: 'Test League' }));

    const text = formatFailureDetails({
      code: 'TWR-7K9M-2P4X',
      message: 'Standings failed',
      at: new Date('2026-11-20T20:15:30Z'),
      screen: 'League (phone/league)',
      appVersion: '0.5.0 (release 42)',
      league: 'Test League (native.hosted)',
    });
    expect(text).toContain('Error code: TWR-7K9M-2P4X');
    expect(text).toContain('Time (Central): ');
    expect(text).toContain('Screen: League (phone/league)');
    expect(text).toContain('App version: 0.5.0 (release 42)');
    expect(text).toContain('League: Test League (native.hosted)');
    expect(text).toContain('Message: Standings failed');
  });

  it('strips Error: prefix in plainFailure and preserves server code through createReadQuery', async () => {
    const serverErr = createFailureError('The league refused this action.', 'TWR-7K9M-2P4X');
    expect(plainFailure(String(serverErr))).toBe('The league refused this action.');
    expect(lookupErrorCode(String(serverErr))).toBe('TWR-7K9M-2P4X');

    const q1 = createReadQuery('draft.TargetDraftReports', {
      initial: () => ({}),
      read: async () => { throw createFailureError('Shared fallback', 'TWR-8N3Q-4V6W'); },
      accept: () => ({}),
      failure: () => 'Custom screen wording',
    });
    const q2 = createReadQuery('market.TargetAssetBoard', {
      initial: () => ({}),
      read: async () => { throw createFailureError('Shared fallback', 'TWR-9P4R-5W7X'); },
      accept: () => ({}),
    });
    await Promise.all([q1.load([]), q2.load([])]);
    expect(q1.peek().error).toBe('Custom screen wording');
    expect(q1.peek().errorCode).toBe('TWR-8N3Q-4V6W');
    expect(q2.peek().error).toBe('Shared fallback');
    expect(q2.peek().errorCode).toBe('TWR-9P4R-5W7X');
  });
});
