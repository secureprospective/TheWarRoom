import { describe, expect, it } from 'vitest';
import { parsePeople } from './people';

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
