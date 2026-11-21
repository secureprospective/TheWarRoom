import { describe, expect, it } from 'vitest';
import { memberDisplayName, parsePeople } from './people';

const reading = {
  league: 'native.people', name: 'People Test', revision: 7, seats: [], roles: [], invites: [],
  dotCount: 0, dotStatus: 'No DOT seats', votes: [], succession: 'Commissioner active',
  actors: [{ person: 'acct.owner', effectivePermissions: ['approve-stat-fixes', 'rule-on-trades'] }],
  yourPermissions: ['approve-stat-fixes', 'rule-on-trades'], yourRole: 'Commissioner',
  yourFranchise: '', mayChangeRules: true, mayGrantRoles: true, mayPostNotices: true,
};

describe('parsePeople', () => {
  it('reads a seat the server calls Vacant as a seat with no owner', () => {
    const seat = { franchise: 'franchise.01', name: 'Buffalo Bills', abbreviation: 'BUF', owner: 'Vacant' };
    expect(parsePeople({ ...reading, seats: [seat] }).seats[0].owner).toBe('');
  });
  it('parses the catalog People actors and effective-permission contract', () => {
    expect(parsePeople(reading)).toMatchObject({
      actors: [{ person: 'acct.owner', effectivePermissions: ['approve-stat-fixes', 'rule-on-trades'] }],
      yourPermissions: ['approve-stat-fixes', 'rule-on-trades'],
    });
  });

  it('rejects unknown People fields and permissions instead of silently accepting them', () => {
    expect(() => parsePeople({ ...reading, futureField: true })).toThrow('unexpected field');
    expect(() => parsePeople({ ...reading, actors: [{ person: 'acct.owner',
      effectivePermissions: ['delete-league'] }] })).toThrow('expected approve-stat-fixes');
    expect(() => parsePeople({ ...reading, actors: [{ person: 'acct.owner',
      effectivePermissions: [], extra: true }] })).toThrow('unexpected field');
  });
});

describe('memberDisplayName', () => {
  const people = {
    seats: [
      { franchise: 'f1', name: 'Arizona Cardinals', abbreviation: 'ARI', owner: 'acct.111' },
      { franchise: 'f2', name: 'Detroit Lions', abbreviation: 'DET', owner: '', caretaker: 'acct.222' },
    ],
  };

  it('maps owner, caretaker, AI/CLI delegate, and local operator principals to display names', () => {
    expect(memberDisplayName('acct.111', people)).toBe('Arizona Cardinals');
    expect(memberDisplayName('acct.222', people)).toBe('Detroit Lions');
    expect(memberDisplayName('via AI/CLI, acting for acct.111', people))
      .toBe('via AI/CLI, acting for Arizona Cardinals');
    expect(memberDisplayName('local.operator', people)).toBe('Local operator');
  });

  it('prefers the owned seat over an earlier caretaker seat for the same person', () => {
    const seats = [
      { franchise: 'f2', name: 'Detroit Lions', abbreviation: 'DET', owner: '', caretaker: 'acct.111' },
      { franchise: 'f1', name: 'Arizona Cardinals', abbreviation: 'ARI', owner: 'acct.111' },
    ];
    expect(memberDisplayName('acct.111', { seats })).toBe('Arizona Cardinals');
  });

  it('falls back to the raw id when no seat name is known', () => {
    expect(memberDisplayName('acct.unknown', people)).toBe('acct.unknown');
    expect(memberDisplayName('acct.111', undefined)).toBe('acct.111');
    expect(memberDisplayName('', people)).toBe('');
  });
});
