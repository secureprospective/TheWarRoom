import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import { personLabel } from './forms';

// The Go people reading (TargetCommissionerPeople). Owners get no role or invite rows from Go.
export type PeopleSeat = {
  franchise: string; name: string; abbreviation: string; owner: string; caretaker?: string;
};
// Go names an empty seat's owner "Vacant" (people_reading.go readSeats); here an empty seat has no owner.
const ownerOf = (owner: string) => owner === 'Vacant' ? '' : owner;
export type PeopleVote = {
  id: string; status: string; ballotsCast: number; totalFranchises: number; votesRequired: number;
  requirement: string; endsAt: string; candidates: string[]; counts: { choice: string; ballots: number }[];
  votedFranchises: string[];
};
export const DEPUTY_PERMISSIONS = ['approve-stat-fixes', 'rule-on-trades', 'edit-lineups', 'move-picks',
  'change-rules', 'run-rollover'] as const;
export type DeputyPermission = (typeof DEPUTY_PERMISSIONS)[number];
export type PeopleActor = { person: string; effectivePermissions: DeputyPermission[] };
export type People = {
  league: string; name: string; revision: number; seats: PeopleSeat[];
  roles: { person: string; role: string }[];
  invites: { id: string; franchise: string; person: string; status: string }[];
  dotCount: number; dotStatus: string; votes: PeopleVote[]; succession: string;
  actors: PeopleActor[]; yourPermissions: DeputyPermission[];
  yourRole: string; yourFranchise: string; mayChangeRules: boolean; mayGrantRoles: boolean;
  mayPostNotices: boolean;
};

function permissions(value: unknown, path: string): DeputyPermission[] {
  return array(value, path, (permission, p) => choice(permission, p, DEPUTY_PERMISSIONS));
}
function flag(v: unknown, path: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${path}: expected true or false`);
  return v;
}
function parseVote(v: unknown, p: string): PeopleVote {
  const keys = ['id', 'status', 'ballotsCast', 'totalFranchises', 'votesRequired', 'requirement', 'endsAt',
    'candidates', 'counts', 'votedFranchises'];
  const r = object(v, p, keys);
  return {
    id: text(r.id, p), status: text(r.status, p), ballotsCast: integer(r.ballotsCast, p),
    totalFranchises: integer(r.totalFranchises, p), votesRequired: integer(r.votesRequired, p),
    requirement: text(r.requirement, p), endsAt: timestamp(r.endsAt, p),
    candidates: array(r.candidates, p, text),
    counts: array(r.counts, p, (c, q) => {
      const o = object(c, q, ['choice', 'ballots']);
      return { choice: text(o.choice, q), ballots: integer(o.ballots, q) };
    }),
    // Go leaves the list out for owners: who voted is for the commissioners only.
    votedFranchises: r.votedFranchises === undefined ? [] : array(r.votedFranchises, p, text),
  };
}
export function parsePeople(v: unknown): People {
  const keys = ['league', 'name', 'revision', 'seats', 'roles', 'invites', 'dotCount', 'dotStatus', 'votes',
    'succession', 'actors', 'yourPermissions', 'yourRole', 'yourFranchise', 'mayChangeRules', 'mayGrantRoles',
    'mayPostNotices'];
  const r = object(v, 'League people', keys);
  return {
    league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'League version'),
    seats: array(r.seats, 'Seats', (s, p) => {
      const o = object(s, p, ['franchise', 'name', 'abbreviation', 'owner', 'caretaker']);
      return { franchise: text(o.franchise, p), name: text(o.name, p),
        abbreviation: text(o.abbreviation, p), owner: ownerOf(text(o.owner, p)),
        ...(o.caretaker === undefined ? {} : { caretaker: text(o.caretaker, p) }) };
    }),
    roles: array(r.roles, 'Roles', (s, p) => {
      const o = object(s, p, ['person', 'role']);
      return { person: text(o.person, p), role: text(o.role, p) };
    }),
    invites: array(r.invites, 'Invitations', (s, p) => {
      const o = object(s, p, ['id', 'franchise', 'person', 'status']);
      return { id: text(o.id, p), franchise: text(o.franchise, p), person: text(o.person, p),
        status: text(o.status, p) };
    }),
    dotCount: integer(r.dotCount, 'DOT seats'), dotStatus: text(r.dotStatus, 'DOT status'),
    votes: array(r.votes, 'Votes', parseVote), succession: text(r.succession, 'Succession'),
    actors: array(r.actors, 'People actors', (actor, p) => {
      const entry = object(actor, p, ['person', 'effectivePermissions']);
      return { person: text(entry.person, p),
        effectivePermissions: permissions(entry.effectivePermissions, p) };
    }),
    yourPermissions: permissions(r.yourPermissions, 'Your permissions'),
    yourRole: text(r.yourRole, 'Your role'), yourFranchise: text(r.yourFranchise, 'Your franchise'),
    mayChangeRules: flag(r.mayChangeRules, 'May change rules'),
    mayGrantRoles: flag(r.mayGrantRoles, 'May give roles'),
    mayPostNotices: flag(r.mayPostNotices, 'May post notices'),
  };
}

const VIA_AI_PREFIX = 'via AI/CLI, acting for ';

/** Resolves an account id (or "via AI/CLI, acting for <id>") to the member's seat name,
 *  falling back to "Local operator" for local.* principals or the original id when unknown. */
export function memberDisplayName(person: string, people?: Pick<People, 'seats'>): string {
  if (!person) return '';
  if (person.startsWith(VIA_AI_PREFIX))
    return `${VIA_AI_PREFIX}${memberDisplayName(person.slice(VIA_AI_PREFIX.length), people)}`;
  const seat = people?.seats.find(s => s.owner === person || s.caretaker === person);
  return seat?.name || personLabel(person);
}
