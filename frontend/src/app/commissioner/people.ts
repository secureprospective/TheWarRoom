import { array, integer, object, text, timestamp } from '../data/parseValues';

// The Go people reading (TargetCommissionerPeople). Owners get no role or invite rows from Go.
export type PeopleSeat = { franchise: string; name: string; abbreviation: string; owner: string };
export type PeopleVote = {
  id: string; status: string; ballotsCast: number; totalFranchises: number; votesRequired: number;
  requirement: string; endsAt: string; candidates: string[]; counts: { choice: string; ballots: number }[];
  votedFranchises: string[];
};
export type People = {
  league: string; name: string; revision: number; seats: PeopleSeat[];
  roles: { person: string; role: string }[];
  invites: { id: string; franchise: string; person: string; status: string }[];
  dotCount: number; dotStatus: string; votes: PeopleVote[]; succession: string;
  yourRole: string; yourFranchise: string; mayChangeRules: boolean; mayGrantRoles: boolean;
  mayPostNotices: boolean;
};

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
    'succession', 'yourRole', 'yourFranchise', 'mayChangeRules', 'mayGrantRoles', 'mayPostNotices'];
  const r = object(v, 'League people', keys);
  return {
    league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'League version'),
    seats: array(r.seats, 'Seats', (s, p) => {
      const o = object(s, p, ['franchise', 'name', 'abbreviation', 'owner']);
      return { franchise: text(o.franchise, p), name: text(o.name, p),
        abbreviation: text(o.abbreviation, p), owner: text(o.owner, p) };
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
    yourRole: text(r.yourRole, 'Your role'), yourFranchise: text(r.yourFranchise, 'Your franchise'),
    mayChangeRules: flag(r.mayChangeRules, 'May change rules'),
    mayGrantRoles: flag(r.mayGrantRoles, 'May give roles'),
    mayPostNotices: flag(r.mayPostNotices, 'May post notices'),
  };
}
