import {
  CONTRACT_STATUSES,
  FRESHNESS_STATES,
  POSITIONS,
  PROVENANCE_KINDS,
  ROSTER_STATUSES,
} from './contract';
import type {
  Franchise,
  League,
  Player,
  Provenance,
  Roster,
  RosterPlayer,
  Snapshot,
  Sourced,
} from './contract';

type Row = Record<string, unknown>;

function object(value: unknown, path: string, keys: string[]): Row {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error(`${path}: expected object`);
  }
  const row = value as Row;
  for (const key of Object.keys(row)) {
    if (!keys.includes(key)) throw new Error(`${path}.${key}: unexpected field`);
  }
  return row;
}

function text(value: unknown, path: string): string {
  if (typeof value !== 'string') throw new Error(`${path}: expected string`);
  return value;
}

function integer(value: unknown, path: string, minimum = 0): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < minimum) {
    throw new Error(`${path}: expected safe integer >= ${minimum}`);
  }
  return value;
}

function choice<T extends string>(
  value: unknown,
  path: string,
  allowed: readonly T[],
): T {
  const parsed = text(value, path);
  if (!allowed.includes(parsed as T))
    throw new Error(`${path}: expected ${allowed.join(' | ')}`);
  return parsed as T;
}

function timestamp(value: unknown, path: string): string {
  const parsed = text(value, path);
  if (
    parsed !== '' &&
    (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(
      parsed,
    ) ||
      Number.isNaN(Date.parse(parsed)))
  ) {
    throw new Error(`${path}: expected RFC3339 timestamp or empty string`);
  }
  return parsed;
}

function id(value: unknown, path: string, player = false): string {
  const parsed = text(value, path);
  if (!(player ? /^(?:\d{4}|[1-9]\d{4,})$/ : /^\d+$/).test(parsed)) {
    throw new Error(
      `${path}: expected ${player ? 'canonical player' : 'numeric string'} id`,
    );
  }
  return parsed;
}

function array<T>(
  value: unknown,
  path: string,
  parse: (v: unknown, p: string) => T,
): T[] {
  if (!Array.isArray(value)) throw new Error(`${path}: expected array`);
  return value.map((entry, index) => parse(entry, `${path}[${index}]`));
}

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
