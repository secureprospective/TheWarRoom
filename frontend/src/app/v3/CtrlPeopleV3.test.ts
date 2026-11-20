import { describe, expect, it } from 'vitest';
import { deputyRows, filterSeats, seatHolder, seatRoles, seatState, vacantSeats } from './CtrlPeopleV3';
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
  it('lists deputies by seat name with their permissions', () => {
    expect(deputyRows(people)).toEqual([
      { person: 'ann', team: 'Team a', roles: 'co-commissioner', permissions: 'edit-lineups' },
      { person: 'zed', team: 'zed', roles: '—', permissions: 'No deputy permissions' }]);
  });
});
