import { describe, expect, it } from 'vitest';
import { deputyRows, filterSeats, roleRows, seatHolder, seatRoles, seatState, vacantSeats }
  from './CtrlPeopleV3';
import type { PeopleSeat } from '../commissioner/people';

const seat = (franchise: string, owner: string, caretaker?: string): PeopleSeat =>
  ({ franchise, name: `Team ${franchise}`, abbreviation: franchise.toUpperCase(), owner,
    ...(caretaker ? { caretaker } : {}) });
const seats = [seat('a', 'ann'), seat('b', ''), seat('c', '', 'cal')];

describe('seats', () => {
  it('names the holder and the state', () => {
    expect(seats.map(seatHolder)).toEqual(['ann', '', 'cal']);
    expect(seats.map(seatState)).toEqual(['Owner seated', 'Vacant', 'Caretaker seated']);
  });
  it('finds vacant seats and filters by search', () => {
    expect(vacantSeats(seats).map(s => s.franchise)).toEqual(['b']);
    expect(filterSeats(seats, 'vacant', '').map(s => s.franchise)).toEqual(['b']);
    expect(filterSeats(seats, 'all', ' CAL ').map(s => s.franchise)).toEqual(['c']);
    expect(filterSeats(seats, 'all', '').length).toBe(3);
  });
});

describe('roles and deputies', () => {
  const people = { seats, roles: [{ person: 'ann', role: 'co-commissioner' }],
    actors: [{ person: 'ann', effectivePermissions: ['edit-lineups' as const] },
      { person: 'zed', effectivePermissions: [] }] };
  it('joins an owner’s roles, dash when none', () => {
    expect(seatRoles(people, seats[0])).toBe('co-commissioner');
    expect(seatRoles(people, seats[1])).toBe('—');
  });
  it('lists deputies by seat name with permissions; resolves caretakers and local principals', () => {
    expect(deputyRows(people)).toEqual([
      { person: 'ann', team: 'Team a', roles: 'co-commissioner', permissions: 'edit-lineups' },
      { person: 'zed', team: 'zed', roles: '—', permissions: 'No deputy permissions' }]);
    const withCaretaker = {
      seats,
      roles: [{ person: 'cal', role: 'Deputy' }],
      actors: [
        { person: 'cal', effectivePermissions: ['edit-lineups' as const] },
        { person: 'local.admin', effectivePermissions: [] },
      ],
    };
    expect(deputyRows(withCaretaker).map(row => row.team)).toEqual(['Team c', 'Local operator']);
  });
  it('maps role account ids to seat display names with id fallback', () => {
    expect(roleRows({
      seats,
      roles: [{ person: 'ann', role: 'Commissioner' }, { person: 'acct.unseated', role: 'DOT member' }],
    })).toEqual([
      { person: 'ann', name: 'Team a', role: 'Commissioner' },
      { person: 'acct.unseated', name: 'acct.unseated', role: 'DOT member' },
    ]);
  });
});
