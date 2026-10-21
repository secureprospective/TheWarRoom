import { targetContracts, targetPlayerContract } from './api';
import { freshness } from '../data/parse';
import { object, text, integer, choice, array, id } from '../data/parseValues';
import { selectProvider } from '../data/provider';
import type {
  Amount,
  Year,
  Row,
  Table,
  Cap,
  Adjustment,
  PlayerContract,
  Reading,
  Readings,
  ContractsProvider,
} from './contract';

type Parser<T> = (value: unknown, path: string) => T;
const cents = (v: unknown, p: string) => integer(v, p, -Number.MAX_SAFE_INTEGER);
function boolean(v: unknown, p: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${p}: expected boolean`);
  return v;
}
const amount: Parser<Amount> = (v, p) => {
  const r = object(v, p, ['cents', 'text']);
  return { cents: cents(r.cents, p), text: text(r.text, p) };
};
const year: Parser<Year> = (v, p) => {
  const r = object(v, p, ['season', 'cost', 'inferred', 'tag', 'reason']);
  return {
    season: integer(r.season, p),
    cost: amount(r.cost, p),
    inferred: boolean(r.inferred, p),
    tag: choice(r.tag, p, ['Observed', 'Inferred']),
    reason: text(r.reason, p),
  };
};
const row: Parser<Row> = (v, p) => {
  const r = object(v, p, [
    'id',
    'franchiseId',
    'team',
    'name',
    'position',
    'salary',
    'expiration',
    'status',
    'info',
  ]);
  return {
    id: id(r.id, p, true),
    franchiseId: id(r.franchiseId, p),
    team: text(r.team, p),
    name: text(r.name, p),
    position: text(r.position, p),
    salary: amount(r.salary, p),
    expiration: integer(r.expiration, p),
    status: text(r.status, p),
    info: text(r.info, p),
  };
};
const table: Parser<Table> = (v, p) => {
  const r = object(v, p, ['directorySource', 'directoryAsOf', 'rows', 'teams', 'positions']);
  return {
    directorySource: text(r.directorySource, p),
    directoryAsOf: text(r.directoryAsOf, p),
    rows: array(r.rows, p, row),
    positions: array(r.positions, p, text),
    teams: array(r.teams, p, (v, p) => {
      const t = object(v, p, ['id', 'name']);
      return { id: id(t.id, p), name: text(t.name, p) };
    }),
  };
};
const cap: Parser<Cap> = (v, p) => {
  const r = object(v, p, ['team', 'season', 'cap', 'used', 'room', 'capSource', 'future']);
  return {
    team: text(r.team, p),
    season: integer(r.season, p),
    cap: r.cap === null ? null : amount(r.cap, p),
    used: amount(r.used, p),
    room: r.room === null ? null : amount(r.room, p),
    capSource: text(r.capSource, p),
    future: array(r.future, p, year),
  };
};
const adjustment: Parser<Adjustment> = (v, p) => {
  const r = object(v, p, ['id', 'team', 'franchiseId', 'amount', 'description']);
  return {
    id: text(r.id, p),
    team: text(r.team, p),
    franchiseId: id(r.franchiseId, p),
    amount: amount(r.amount, p),
    description: text(r.description, p),
  };
};
const player: Parser<PlayerContract> = (v, p) => {
  const r = object(v, p, ['player', 'years']);
  return { player: row(r.player, p), years: array(r.years, p, year) };
};
export function parseReading<T>(v: unknown, parse: Parser<T>): Reading<T> {
  const p = 'Contracts reading';
  const r = object(v, p, [
    'value',
    'provenance',
    'status',
    'absenceReason',
    'count',
    'threshold',
    'baseline',
    'cutoff',
    'method',
    'inputSHA',
    'buildStamp',
    'howCounted',
  ]);
  const source = object(r.provenance, p, ['source', 'kind', 'freshness']);
  const status = choice(r.status, p, ['ok', 'thin', 'failed', 'not_wired']);
  const absenceReason = r.absenceReason === null ? null : text(r.absenceReason, p);
  if (status !== 'ok' && !absenceReason) throw new Error(`${p}: missing absence reason`);
  return {
    status,
    absenceReason,
    value: status === 'ok' ? parse(r.value, p) : undefined,
    provenance: {
      source: text(source.source, p),
      kind: choice(source.kind, p, ['live', 'fixture']),
      freshness: freshness(source.freshness, p),
    },
    count: integer(r.count, p),
    threshold: integer(r.threshold, p),
    baseline: text(r.baseline, p),
    cutoff: text(r.cutoff, p),
    method: text(r.method, p),
    inputSHA: text(r.inputSHA, p),
    buildStamp: text(r.buildStamp, p),
    howCounted: text(r.howCounted, p),
  };
}
export function parseContracts(v: unknown): Readings {
  const r = object(v, 'Contracts', ['table', 'cap', 'adjustments']);
  return {
    table: parseReading(r.table, table),
    cap: parseReading(r.cap, cap),
    adjustments: parseReading(r.adjustments, (v, p) => {
      const r = object(v, p, ['rows']);
      return { rows: array(r.rows, p, adjustment) };
    }),
  };
}
export function parsePlayerContract(v: unknown) {
  return parseReading(v, player);
}
export function contractsProvider(): ContractsProvider {
  const selected = selectProvider();
  const desktop = () => {
    if (selected.kind !== 'live')
      throw new Error('Contracts and cap need the desktop app and its MFL mirror');
  };
  return {
    contracts: async (scope) => {
      desktop();
      return parseContracts(await targetContracts(scope));
    },
    player: async (id) => {
      desktop();
      return parsePlayerContract(await targetPlayerContract(id));
    },
    onChange: selected.onSeasonChange.bind(selected),
  };
}
