import type { Report } from '../reports/contract';
import { personLabel } from './forms';
import type { People } from './people';

const base = { note: '', status: 'ok' as const, asOf: '', link: null, summary: [] as Report['summary'] };
const table = (id: string, title: string, source: string, columns: string[], rows: string[][]): Report => ({
  ...base, id, title, source, columns, how: 'Read from this league’s ledger.',
  rows: rows.map(cells => ({ cells, range: null })),
});

export const seatName = (people: People, franchise: string) => {
  const seat = people.seats.find(s => s.franchise === franchise);
  return seat ? `${seat.name} (${seat.abbreviation})` : franchise;
};
export function seatsReport(people: People): Report {
  return table('league-seats', 'Franchises and owners', people.name,
    ['Franchise', 'Owner'], people.seats.map(s => [
      `${s.name} (${s.abbreviation})`,
      s.caretaker ? `Vacant — caretaker: ${personLabel(s.caretaker)}` : personLabel(s.owner),
    ]));
}
// Roles and invitations come from Go only for the commissioner and co-commissioner; an owner gets none.
export function rolesReport(people: People): Report {
  return table('league-roles', 'Roles', 'Private: owners do not see these rows', ['Person', 'Role'],
    people.roles.map(r => [personLabel(r.person), r.role]));
}
export function invitesReport(people: People): Report {
  return table('league-invites', 'Owner invitations', 'Private: owners do not see these rows',
    ['Franchise', 'Person', 'Status'],
    people.invites.map(i => [seatName(people, i.franchise), personLabel(i.person), i.status]));
}
