import {
  PHASES,
  URGENCIES,
  WINDOW_KINDS,
  CONTRACT_STATUSES,
  FRESHNESS_STATES,
  POSITIONS,
  PROVENANCE_KINDS,
  ROSTER_STATUSES,
} from './contract';
import type {
  ClockReading,
  Deadline,
  Franchise,
  League,
  Player,
  Provenance,
  Roster,
  RosterPlayer,
  Snapshot,
  Sourced,
} from './contract';

import {
  object, text, integer, choice, timestamp, id, array, requiredText, utc,
} from './parseValues';
export { parseReceipt, parseEnvelopeDemo } from './parseEnvelope';

function provenance(value: unknown, path: string): Provenance {
  const row = object(value, path, ['source', 'freshness', 'kind']);
  const fresh = object(row.freshness, `${path}.freshness`, [
    'state',
    'fetchedAt',
    'note',
  ]);
  return {
    source: text(row.source, `${path}.source`),
    kind: choice(row.kind, `${path}.kind`, PROVENANCE_KINDS),
    freshness: {
      state: choice(fresh.state, `${path}.freshness.state`, FRESHNESS_STATES),
      fetchedAt: timestamp(fresh.fetchedAt, `${path}.freshness.fetchedAt`),
      note: text(fresh.note, `${path}.freshness.note`),
    },
  };
}

function sourced<T>(
  value: unknown,
  path: string,
  parse: (v: unknown, p: string) => T,
): Sourced<T> {
  const row = object(value, path, ['value', 'provenance']);
  return {
    value: parse(row.value, `${path}.value`),
    provenance: provenance(row.provenance, `${path}.provenance`),
  };
}

function league(value: unknown, path: string): League {
  const row = object(value, path, ['season', 'franchiseCount', 'salaryCap']);
  return {
    season: integer(row.season, `${path}.season`),
    franchiseCount: integer(row.franchiseCount, `${path}.franchiseCount`),
    ...(row.salaryCap === undefined
      ? {}
      : { salaryCap: integer(row.salaryCap, `${path}.salaryCap`) }),
  };
}

function franchise(value: unknown, path: string): Franchise {
  const row = object(value, path, ['id', 'name', 'capUsed', 'capRoom']);
  const money = (key: string) =>
    row[key] === undefined
      ? {}
      : { [key]: integer(row[key], `${path}.${key}`, Number.MIN_SAFE_INTEGER) };
  return {
    id: id(row.id, `${path}.id`),
    ...(row.name === undefined ? {} : { name: text(row.name, `${path}.name`) }),
    ...money('capUsed'),
    ...money('capRoom'),
  };
}

function rosterPlayer(value: unknown, path: string): RosterPlayer {
  const row = object(value, path, [
    'id',
    'salary',
    'yearsRemaining',
    'contractStatus',
    'rosterStatus',
  ]);
  return {
    id: id(row.id, `${path}.id`, true),
    salary: integer(row.salary, `${path}.salary`),
    ...(row.yearsRemaining === undefined
      ? {}
      : { yearsRemaining: integer(row.yearsRemaining, `${path}.yearsRemaining`) }),
    contractStatus: choice(
      row.contractStatus,
      `${path}.contractStatus`,
      CONTRACT_STATUSES,
    ),
    rosterStatus: choice(row.rosterStatus, `${path}.rosterStatus`, ROSTER_STATUSES),
  };
}

function roster(value: unknown, path: string): Roster {
  const row = object(value, path, ['franchiseId', 'players']);
  return {
    franchiseId: id(row.franchiseId, `${path}.franchiseId`),
    players: array(row.players, `${path}.players`, rosterPlayer),
  };
}

function player(value: unknown, path: string): Player {
  const row = object(value, path, [
    'id',
    'name',
    'position',
    'team',
    'birthdate',
    'isRookie',
  ]);
  if (row.isRookie !== undefined && typeof row.isRookie !== 'boolean')
    throw new Error(`${path}.isRookie: expected boolean`);
  return {
    id: id(row.id, `${path}.id`, true),
    ...(row.name === undefined ? {} : { name: text(row.name, `${path}.name`) }),
    ...(row.team === undefined ? {} : { team: text(row.team, `${path}.team`) }),
    ...(row.position === undefined
      ? {}
      : { position: choice(row.position, `${path}.position`, POSITIONS) }),
    ...(row.birthdate === undefined
      ? {}
      : {
          birthdate: integer(row.birthdate, `${path}.birthdate`, Number.MIN_SAFE_INTEGER),
        }),
    ...(row.isRookie === undefined ? {} : { isRookie: row.isRookie as boolean }),
  };
}

export function parseSnapshot(value: unknown): Snapshot {
  const row = object(value, 'snapshot', ['league', 'franchises', 'rosters', 'players']);
  return {
    league: sourced(row.league, 'snapshot.league', league),
    franchises: sourced(row.franchises, 'snapshot.franchises', (v, p) =>
      array(v, p, franchise),
    ),
    rosters: sourced(row.rosters, 'snapshot.rosters', (v, p) => array(v, p, roster)),
    players: sourced(row.players, 'snapshot.players', (v, p) => array(v, p, player)),
  };
}

function boolean(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path}: expected boolean`);
  return value;
}
function deadline(value: unknown, path: string): Deadline {
  const r = object(value, path, ['id', 'label', 'at', 'urgency', 'pinned', 'promoted']);
  const urgency = choice(r.urgency, `${path}.urgency`, URGENCIES);
  if ((r.at === undefined) !== (urgency === 'U0')) {
    throw new Error(`${path}.at: date required exactly when urgency is dated`);
  }
  return {
    id: requiredText(r.id, `${path}.id`), label: requiredText(r.label, `${path}.label`),
    ...(r.at === undefined ? {} : { at: utc(r.at, `${path}.at`) }),
    urgency,
    pinned: boolean(r.pinned, `${path}.pinned`), promoted: boolean(r.promoted, `${path}.promoted`),
  };
}
function clockReading(value: unknown, path: string): ClockReading {
  const r = object(value, path, ['season', 'phase', 'week', 'deadlines', 'windows']);
  const week = r.week === undefined ? undefined : integer(r.week, `${path}.week`, 1);
  if (week !== undefined && week > 18) throw new Error(`${path}.week: expected integer 1–18`);
  const windows = array(r.windows, `${path}.windows`, (v, p) => {
    const w = object(v, p, ['kind', 'status', 'reason']);
    return {
      kind: choice(w.kind, `${p}.kind`, WINDOW_KINDS),
      status: choice(w.status, `${p}.status`, ['unknown'] as const),
      reason: requiredText(w.reason, `${p}.reason`),
    };
  });
  const uniqueWindows = new Set(windows.map((w) => w.kind)).size;
  if (windows.length !== WINDOW_KINDS.length || uniqueWindows !== windows.length) {
    throw new Error(`${path}.windows: expected each known window exactly once`);
  }
  return {
    season: integer(r.season, `${path}.season`), phase: choice(r.phase, `${path}.phase`, PHASES),
    ...(week === undefined ? {} : { week }),
    deadlines: array(r.deadlines, `${path}.deadlines`, deadline), windows,
  };
}
export function parseClock(value: unknown): Sourced<ClockReading> {
  return sourced(value, 'clock', clockReading);
}
