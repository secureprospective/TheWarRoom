import { describe, expect, it } from 'vitest';
import { kickWords } from './HQLineupV3';

// Sunday 22 November 2026, 10:00 CT.
const now = Date.parse('2026-11-22T16:00:00Z');

describe('kickWords', () => {
  it('names a bye when the player has no game this week', () => {
    expect(kickWords(undefined, now)).toEqual({ w: 'BYE', c: 'd' });
  });
  it('says LIVE for a game under way and DONE once it is long over', () => {
    expect(kickWords('2026-11-22T15:00:00Z', now).w).toBe('LIVE');
    expect(kickWords('2026-11-20T01:15:00Z', now).w).toBe('DONE');
  });
  it('gives the Sunday kickoff time in Central, and the night games by name', () => {
    expect(kickWords('2026-11-22T21:25:00Z', now).w).toBe('3:25');
    expect(kickWords('2026-11-23T01:20:00Z', now).w).toBe('SNF');
    expect(kickWords('2026-11-24T01:15:00Z', now).w).toBe('MNF');
    expect(kickWords('2026-11-27T01:15:00Z', now).w).toBe('TNF');
  });
});
