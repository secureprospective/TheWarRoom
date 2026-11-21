import { describe, expect, it } from 'vitest';
import { fixActorName, fixExceptions, undoableEntries } from './CtrlFixV3';
import type { HostedFixReadings } from '../commissioner/hostedControlRoomData';

const as = (x: unknown) => x as HostedFixReadings;

describe('fixExceptions', () => {
  it('is empty without a read', () => expect(fixExceptions(undefined)).toEqual([]));
  it('keeps corrections with an end or rule and drops calendar exceptions they cover', () => {
    const r = as({
      corrections: { items: [
        { entry: 'e1', what: 'cap relief', franchise: '0001', until: '', rule: 'calendar.trade', when: 'w1' },
        { entry: 'e2', what: 'plain', franchise: '0002' }] },
      calendarExceptions: [
        { action: 'trade', franchise: '0001', start: 'w1', permanent: true, until: '' },
        { action: 'waiver', franchise: '0003', start: 'w2', permanent: true, until: '' }] });
    expect(fixExceptions(r).map(e => e.key)).toEqual(['e1', 'waiver:0003:w2']);
    expect(fixExceptions(r)[0].ends).toBe('until cleared');
  });
});

describe('undoableEntries', () => {
  it('keeps entries with an undo that are not replaced, up to the limit', () => {
    const r = as({ ledger: { entries: [
      { id: 'a', undo: 'x', replacedAt: '' }, { id: 'b', undo: '' }, { id: 'c', undo: 'x', replacedAt: 't' },
      { id: 'd', undo: 'y' }] } });
    expect(undoableEntries(r).map(e => e.id)).toEqual(['a', 'd']);
    expect(undoableEntries(r, 1).map(e => e.id)).toEqual(['a']);
  });
});

describe('fixActorName', () => {
  it('resolves ledger actor ids to member names with id fallback', () => {
    const people = { seats: [{ franchise: 'f25', name: 'Arizona Cardinals', abbreviation: 'ARI',
      owner: 'acct.az' }] };
    expect(fixActorName('acct.az', people)).toBe('Arizona Cardinals');
    expect(fixActorName('acct.other', people)).toBe('acct.other');
  });
});
