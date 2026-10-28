import { seatsReport } from './peopleReport';
import { describe, expect, it } from 'vitest';
import { parsePeople } from './people';
import wire from './people.wire.testdata.json';

describe('Commissioner people wire', () => {
  it('reads the real Go reading for the commissioner, with private roles and who voted', () => {
    const people = parsePeople(wire.commissioner);
    expect(people.revision).toBe(7);
    expect(people.seats.map(s => s.owner)).toEqual(['owner-one', 'owner-two', 'owner-three', 'Vacant']);
    expect(people.roles).toEqual([{ person: 'creator', role: 'Commissioner' },
      { person: 'private-deputy', role: 'Co-commissioner' }]);
    expect(people).toMatchObject({ dotCount: 0,
      dotStatus: 'Needs a ruling: trade review needs at least one seat',
      yourRole: 'Commissioner', mayChangeRules: true, mayGrantRoles: true });
    expect(people.votes[0])
      .toMatchObject({ ballotsCast: 1, votesRequired: 3, votedFranchises: ['franchise.02'] });
  });
  it('reads the owner’s reading with no roles, no invitations and no list of who voted', () => {
    const people = parsePeople(wire.owner);
    expect(people.roles).toEqual([]);
    expect(people.invites).toEqual([]);
    expect(people.votes[0].votedFranchises).toEqual([]);
    expect(people.mayChangeRules).toBe(false);
    expect(people.yourFranchise).toBe('franchise.01');
    expect(parsePeople(wire.commissioner).yourFranchise).toBe('');
    expect(JSON.stringify(wire.owner)).not.toContain('private-deputy');
  });
  it('refuses a reading Go did not send', () => {
    expect(() => parsePeople({ ...wire.owner, confidence: 0.4 })).toThrow();
    expect(() => parsePeople({ ...wire.owner, mayChangeRules: 'yes' })).toThrow();
    expect(() => parsePeople({ ...wire.owner, seats: [{ ...wire.owner.seats[0], pin: '1234' }] })).toThrow();
    const vote = { ...wire.owner.votes[0], endsAt: 'soon' };
    expect(() => parsePeople({ ...wire.owner, votes: [vote] })).toThrow();
  });
});

it('shows the caretaker without presenting them as the owner', () => {
  const seat = { ...wire.owner.seats[3], caretaker: 'commissioner' };
  const reading = parsePeople({ ...wire.owner, seats: [seat] });
  expect(reading.seats[0].owner).toBe('Vacant');
  expect(seatsReport(reading).rows[0].cells[1]).toBe('Vacant — caretaker: commissioner');
  expect(() => parsePeople({ ...wire.owner, seats: [{ ...seat, caretaker: true }] })).toThrow();
});
