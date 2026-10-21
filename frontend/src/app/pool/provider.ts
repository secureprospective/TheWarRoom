import { range } from '../reports/contract';
import { parseRailReceipt } from '../shell/railReceipt';
import { poolWire } from '../offers/api';
export { rawBid } from '../offers/api';
import { provenance } from '../data/parse';
import { array, choice, integer, object, text } from '../data/parseValues';
import type {
  Amount,
  BidFormState,
  DropOption,
  Board,
  BoardReading,
  Plan,
  Pool,
  PoolReading,
  PoolRow,
  PoolScope,
  Preview,
  Reading,
  Rule,
} from './contract';

type Parser<T> = (value: unknown, path: string) => T;
const flag = (v: unknown, p: string): boolean => {
  if (typeof v !== 'boolean') throw new Error(`${p}: expected boolean`);
  return v;
};
const amount: Parser<Amount> = (v, p) => {
  const r = object(v, p, ['cents', 'text']);
  return { cents: integer(r.cents, p, -Number.MAX_SAFE_INTEGER), text: text(r.text, p) };
};
function optionalNumber(v: unknown, p: string): number | null {
  if (v === null) return null;
  if (typeof v !== 'number' || !Number.isFinite(v)) throw new Error(`${p}: expected finite number`);
  return v;
}
const poolRow: Parser<PoolRow> = (v, p) => {
  const r = object(v, p, [
    'id', 'name', 'position', 'team', 'age', 'experience', 'minimum', 'plan', 'planId',
    'market', 'marketLines', 'engine', 'ppg', 'lab', 'engineText', 'ppgText', 'labText',
  ]);
  return {
    engineText: text(r.engineText, p), ppgText: text(r.ppgText, p), labText: text(r.labText, p),
    engine: optionalNumber(r.engine, p), ppg: optionalNumber(r.ppg, p), lab: optionalNumber(r.lab, p),
    marketLines: array(r.marketLines, p, text),
    market: array(r.market, p, (v, p) => {
      const fact = object(v, p, ['label', 'text', 'reason', 'range']);
      return { label: text(fact.label, p), text: text(fact.text, p),
        reason: text(fact.reason, p), range: range(fact.range, p) };
    }),
    id: text(r.id, p), name: text(r.name, p), position: text(r.position, p), team: text(r.team, p),
    age: text(r.age, p), experience: text(r.experience, p), minimum: amount(r.minimum, p),
    plan: text(r.plan, p), planId: text(r.planId, p),
  };
};
const pool: Parser<Pool> = (v, p) => {
  const r = object(v, p, ['rows', 'positions', 'summary', 'sorts', 'engineNote', 'ppgNote']);
  return {
    engineNote: text(r.engineNote, p), ppgNote: text(r.ppgNote, p),
    sorts: array(r.sorts, p, (v, p) => {
      const s = object(v, p, ['key', 'label']);
      return { key: text(s.key, p), label: text(s.label, p) };
    }),
    rows: array(r.rows, p, poolRow), positions: array(r.positions, p, text), summary: text(r.summary, p),
  };
};
const rule: Parser<Rule> = (v, p) => {
  const r = object(v, p, ['rule', 'label', 'verdict', 'reason', 'mark']);
  return {
    rule: text(r.rule, p), label: text(r.label, p),
    verdict: choice(r.verdict, p, ['pass', 'fail', 'unknown', 'info']),
    reason: text(r.reason, p), mark: text(r.mark, p),
  };
};
const dropOption: Parser<DropOption> = (v, p) => {
  const r = object(v, p, ['id', 'name', 'label', 'salary', 'deadCap', 'freed', 'covers', 'available']);
  return {
    id: text(r.id, p), name: text(r.name, p), label: text(r.label, p),
    salary: integer(r.salary, p), deadCap: integer(r.deadCap, p),
    freed: integer(r.freed, p, -Number.MAX_SAFE_INTEGER),
    covers: flag(r.covers, p), available: flag(r.available, p),
  };
};
export function parsePreview(v: unknown): Preview {
  const p = 'Bid preview';
  const r = object(v, p, [
    'problem', 'player', 'salary', 'snapped', 'points', 'rules', 'blocked', 'note', 'post',
    'ruleSummary', 'closesNote', 'conditionalDrops',
  ]);
  return {
    conditionalDrops: r.conditionalDrops === null ? [] : array(r.conditionalDrops, p, dropOption),
    problem: text(r.problem, p), player: text(r.player, p), salary: amount(r.salary, p),
    snapped: text(r.snapped, p), points: text(r.points, p), rules: array(r.rules, p, rule),
    blocked: flag(r.blocked, p), note: text(r.note, p), post: text(r.post, p),
    ruleSummary: text(r.ruleSummary, p), closesNote: text(r.closesNote, p),
  };
}
const action: Parser<Plan['actions'][number]> = (v, p) => {
  const r = object(v, p, ['id', 'label', 'disabled', 'reason']);
  return {
    id: text(r.id, p), label: text(r.label, p), disabled: flag(r.disabled, p), reason: text(r.reason, p),
  };
};
export function parsePlan(v: unknown, path = 'Bid plan'): Plan {
  const r = object(v, path, [
    'receipt', 'facts', 'player', 'salary', 'points', 'contract', 'beatText', 'form', 'state',
    'status', 'rules', 'ruleSummary', 'post', 'actions',
  ]);
  const facts = object(r.facts, path, [
    'player', 'salary', 'years', 'points', 'snipe', 'beat', 'evidence', 'closes',
    'conditionalDrop', 'conditionalName',
  ]);
  const form = object(r.form, path, ['salary', 'beat']);
  const head = parseRailReceipt(r.receipt, path);
  return {
    conditionalDrop: text(facts.conditionalDrop, path),
    id: head.id, stateCode: head.state, franchiseId: head.franchiseId, history: head.history,
    receipt: head.rail,
    playerId: text(facts.player, path), years: integer(facts.years, path, 1), snipe: flag(facts.snipe, path),
    evidence: text(facts.evidence, path), closes: text(facts.closes, path),
    form: { salary: text(form.salary, path), beat: text(form.beat, path) },
    player: text(r.player, path), salary: amount(r.salary, path), points: text(r.points, path),
    contract: text(r.contract, path), beatText: text(r.beatText, path),
    state: text(r.state, path), status: text(r.status, path),
    rules: array(r.rules, path, rule), ruleSummary: text(r.ruleSummary, path), post: text(r.post, path),
    actions: array(r.actions, path, action),
  };
}
function parseReading<T>(v: unknown, parse: Parser<T>, label: string): Reading<T> {
  const r = object(v, label, [
    'value', 'provenance', 'status', 'absenceReason', 'count', 'threshold', 'baseline', 'cutoff',
    'method', 'inputSHA', 'buildStamp', 'howCounted',
  ]);
  const status = choice(r.status, label, ['ok', 'thin', 'failed', 'not_wired']);
  const absenceReason = r.absenceReason === null ? null : text(r.absenceReason, label);
  if (status !== 'ok' && !absenceReason) throw new Error(`${label}: missing absence reason`);
  return {
    status, absenceReason, value: parse(r.value, label), provenance: provenance(r.provenance, label),
    count: integer(r.count, label), threshold: integer(r.threshold, label), baseline: text(r.baseline, label),
    cutoff: text(r.cutoff, label), method: text(r.method, label), inputSHA: text(r.inputSHA, label),
    buildStamp: text(r.buildStamp, label), howCounted: text(r.howCounted, label),
  };
}
export const parsePool = (v: unknown): PoolReading => parseReading(v, pool, 'Pool');
export const parseBoard = (v: unknown): BoardReading =>
  parseReading(v, (b, p): Board => {
    const r = object(b, p, ['plans', 'summary']);
    return { plans: array(r.plans, p, (plan, pp) => parsePlan(plan, pp)), summary: text(r.summary, p) };
  }, 'Bid plans');

export type PoolProvider = {
  refresh(force: boolean): Promise<void>;
  pool(franchiseId: string, scope: PoolScope): Promise<PoolReading>;
  plans(franchiseId: string): Promise<BoardReading>;
  preview(franchiseId: string, form: BidFormState): Promise<Preview>;
  draft(franchiseId: string, form: BidFormState): Promise<Plan>;
  revise(id: string, franchiseId: string, form: BidFormState): Promise<Plan>;
  handOff(id: string): Promise<Plan>;
  posted(id: string): Promise<Plan>;
  withdraw(id: string): Promise<Plan>;
};
export const poolProvider: PoolProvider = {
  refresh: async force => { await poolWire.refresh(force); },
  pool: async (mine, scope) => parsePool(await poolWire.pool(mine, scope)),
  plans: async mine => parseBoard(await poolWire.plans(mine)),
  preview: async (mine, form) => parsePreview(await poolWire.preview(mine, form)),
  draft: async (mine, form) => parsePlan(await poolWire.draft(mine, form)),
  revise: async (id, mine, form) => parsePlan(await poolWire.revise(id, mine, form)),
  handOff: async id => parsePlan(await poolWire.handOff(id)),
  posted: async id => parsePlan(await poolWire.posted(id)),
  withdraw: async id => parsePlan(await poolWire.withdraw(id)),
};
