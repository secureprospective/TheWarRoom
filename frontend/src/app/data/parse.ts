import {
  ENVELOPE_EVENTS,
  ENVELOPE_STATES,
  GRAVITIES,
  PHASES,
  UNDO_CLASSES,
  URGENCIES,
  WINDOW_KINDS,
  CONTRACT_STATUSES,
  FRESHNESS_STATES,
  POSITIONS,
  PROVENANCE_KINDS,
  ROSTER_STATUSES,
} from './contract';
import type {
  AuditEntry,
  ClockReading,
  Deadline,
  EnvelopeDemo,
  EnvelopeSpec,
  Franchise,
  League,
  Player,
  Provenance,
  Receipt,
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

function boolean(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path}: expected boolean`);
  return value;
}
function requiredText(value: unknown, path: string): string {
  const parsed = text(value, path);
  if (!parsed.trim()) throw new Error(`${path}: expected nonempty string`);
  return parsed;
}
function utc(value: unknown, path: string): string {
  const parsed = timestamp(value, path);
  if (!parsed.endsWith('Z')) throw new Error(`${path}: expected UTC timestamp`);
  return parsed;
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
  const r = object(value, path, ['season', 'phase', 'deadlines', 'windows']);
  const windows = array(r.windows, `${path}.windows`, (v, p) => {
    const w = object(v, p, ['kind', 'status', 'reason']);
    return {
      kind: choice(w.kind, `${p}.kind`, WINDOW_KINDS),
      status: choice(w.status, `${p}.status`, ['unknown'] as const),
      reason: requiredText(w.reason, `${p}.reason`),
    };
  });
  if (windows.length !== WINDOW_KINDS.length || new Set(windows.map((w) => w.kind)).size !== windows.length) {
    throw new Error(`${path}.windows: expected each known window exactly once`);
  }
  return {
    season: integer(r.season, `${path}.season`), phase: choice(r.phase, `${path}.phase`, PHASES),
    deadlines: array(r.deadlines, `${path}.deadlines`, deadline), windows,
  };
}
export function parseClock(value: unknown): Sourced<ClockReading> {
  return sourced(value, 'clock', clockReading);
}
function target(value: unknown, path: string): EnvelopeSpec['target'] {
  const r = object(value, path, ['kind', 'url']);
  const kind = choice(r.kind, `${path}.kind`, ['mapped', 'unmapped'] as const);
  if (kind === 'unmapped') {
    if (r.url !== undefined) throw new Error(`${path}.url: unmapped target cannot have URL`);
    return { kind };
  }
  const url = requiredText(r.url, `${path}.url`);
  let parsed: URL;
  try { parsed = new URL(url); } catch { throw new Error(`${path}.url: expected HTTPS URL`); }
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password) {
    throw new Error(`${path}.url: expected HTTPS URL`);
  }
  return { kind, url };
}
function envelopeSpec(value: unknown, path: string): EnvelopeSpec {
  const r = object(value, path, [
    'intent', 'leagueId', 'franchiseId', 'subject', 'expected', 'gravity', 'undo', 'target', 'deadline',
  ]);
  const subject = object(r.subject, `${path}.subject`, ['players', 'picks']);
  const expected = object(r.expected, `${path}.expected`, ['player', 'rosterStatus']);
  const players = array(subject.players, `${path}.subject.players`, (v, p) => id(v, p, true));
  const player = id(expected.player, `${path}.expected.player`, true);
  if (!players.includes(player)) throw new Error(`${path}.expected.player: must be a subject`);
  return {
    intent: requiredText(r.intent, `${path}.intent`), leagueId: id(r.leagueId, `${path}.leagueId`),
    franchiseId: id(r.franchiseId, `${path}.franchiseId`),
    subject: { players, picks: array(subject.picks, `${path}.subject.picks`, requiredText) },
    expected: {
      player,
      rosterStatus: choice(expected.rosterStatus, `${path}.expected.rosterStatus`, ROSTER_STATUSES),
    },
    gravity: choice(r.gravity, `${path}.gravity`, GRAVITIES),
    undo: choice(r.undo, `${path}.undo`, UNDO_CLASSES),
    target: target(r.target, `${path}.target`),
    ...(r.deadline === undefined ? {} : { deadline: utc(r.deadline, `${path}.deadline`) }),
  };
}
function auditEntry(value: unknown, path: string): AuditEntry {
  const r = object(value, path, ['at', 'from', 'event', 'to', 'note']);
  return {
    at: utc(r.at, `${path}.at`), from: choice(r.from, `${path}.from`, ENVELOPE_STATES),
    event: choice(r.event, `${path}.event`, ENVELOPE_EVENTS),
    to: choice(r.to, `${path}.to`, ENVELOPE_STATES), note: text(r.note, `${path}.note`),
  };
}
function receipt(value: unknown, path: string): Receipt {
  const r = object(value, path, ['correlationId', 'spec', 'state', 'audit']);
  const spec = envelopeSpec(r.spec, `${path}.spec`);
  const state = choice(r.state, `${path}.state`, ENVELOPE_STATES);
  if (spec.target.kind === 'unmapped' && state !== 'draft' && state !== 'blocked') {
    throw new Error(`${path}.state: unmapped target must stay draft or blocked`);
  }
  const audit = array(r.audit, `${path}.audit`, auditEntry);
  let previous = 'draft';
  let at = -Infinity;
  for (const entry of audit) {
    if (entry.from !== previous || Date.parse(entry.at) < at) {
      throw new Error(`${path}.audit: discontinuous trail`);
    }
    previous = entry.to; at = Date.parse(entry.at);
  }
  if (previous !== state) throw new Error(`${path}.state: must match audit head`);
  return { correlationId: requiredText(r.correlationId, `${path}.correlationId`), spec, state, audit };
}
export function parseReceipt(value: unknown): Receipt {
  return receipt(value, 'receipt');
}
export function parseEnvelopeDemo(value: unknown): EnvelopeDemo {
  const r = object(value, 'envelope', ['kind', 'receipt']);
  return {
    kind: choice(r.kind, 'envelope.kind', ['fixture'] as const),
    receipt: receipt(r.receipt, 'envelope.receipt'),
  };
}
