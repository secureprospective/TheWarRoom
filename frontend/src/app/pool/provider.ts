import type * as Bindings from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { provenance } from '../data/parse';
import { selectProvider } from '../data/provider';
import { array, choice, integer, object, text } from '../data/parseValues';
import { railStates } from '../shell/moveLabels';
import type {
  Amount,
  BidFormState,
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
const poolRow: Parser<PoolRow> = (v, p) => {
  const r = object(v, p, [
    'id', 'name', 'position', 'team', 'age', 'experience', 'minimum', 'plan', 'planId',
  ]);
  return {
    id: text(r.id, p), name: text(r.name, p), position: text(r.position, p), team: text(r.team, p),
    age: text(r.age, p), experience: text(r.experience, p), minimum: amount(r.minimum, p),
    plan: text(r.plan, p), planId: text(r.planId, p),
  };
};
const pool: Parser<Pool> = (v, p) => {
  const r = object(v, p, ['rows', 'positions', 'summary']);
  return {
    rows: array(r.rows, p, poolRow), positions: array(r.positions, p, text), summary: text(r.summary, p),
  };
};
export const rule: Parser<Rule> = (v, p) => {
  const r = object(v, p, ['rule', 'label', 'verdict', 'reason', 'mark']);
  return {
    rule: text(r.rule, p), label: text(r.label, p),
    verdict: choice(r.verdict, p, ['pass', 'fail', 'unknown', 'info']),
    reason: text(r.reason, p), mark: text(r.mark, p),
  };
};
export function parsePreview(v: unknown): Preview {
  const p = 'Bid preview';
  const r = object(v, p, [
    'problem', 'player', 'salary', 'snapped', 'points', 'rules', 'blocked', 'note', 'post',
    'ruleSummary', 'closesNote',
  ]);
  return {
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
// The receipt and facts are read for what the screens show; the envelope's own shape is
// checked by the move rail's parser, and this wire never reaches the rail.
function receipt(v: unknown, p: string) {
  const r = object(v, p, ['correlationId', 'spec', 'state', 'audit']);
  const spec = object(r.spec, p, [
    'intent', 'leagueId', 'franchiseId', 'subject', 'expected', 'gravity', 'undo', 'target', 'deadline',
  ]);
  const state = choice(r.state, p, railStates);
  const audit = array(r.audit, p, (e, ep) => {
    const entry = object(e, ep, ['at', 'from', 'event', 'to', 'note']);
    return {
      at: text(entry.at, ep), from: choice(entry.from, ep, railStates),
      to: choice(entry.to, ep, railStates), note: text(entry.note, ep),
    };
  });
  return {
    id: text(r.correlationId, p), state, franchiseId: text(spec.franchiseId, p),
    history: audit.map((entry) => ({ at: entry.at, state: entry.to, note: entry.note })),
    rail: { state, spec: { intent: text(spec.intent, p) }, audit },
  };
}
export function parsePlan(v: unknown, path = 'Bid plan'): Plan {
  const r = object(v, path, [
    'receipt', 'facts', 'player', 'salary', 'points', 'contract', 'beatText', 'form', 'state',
    'status', 'rules', 'ruleSummary', 'post', 'actions',
  ]);
  const facts = object(r.facts, path, [
    'player', 'salary', 'years', 'points', 'snipe', 'beat', 'evidence', 'closes',
  ]);
  const form = object(r.form, path, ['salary', 'beat']);
  const head = receipt(r.receipt, path);
  return {
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

// The pool calls Go through window.go, typed by the generated declarations, instead of importing
// the generated wrappers: an imported wrapper is placed in the entry chunk even when only a lazy
// chunk uses it, and nine of them would break the entry budget.
type Go = typeof Bindings;
function go(): Go {
  if (selectProvider().kind !== 'live') throw new Error('The free-agent pool needs the desktop app');
  return (window as unknown as { go: { main: { App: Go } } }).go.main.App;
}
// The form's figures as Go takes them; the window close is local time on screen, UTC on the wire.
// A date without a time (or the reverse) is no closing time yet, and Go says so.
export function rawBid(franchiseId: string, form: BidFormState): main.RawBidInput {
  const closes = form.closesDate && form.closesTime
    ? new Date(`${form.closesDate}T${form.closesTime}`) : undefined;
  return main.RawBidInput.createFrom({
    franchiseId, player: form.playerId, salary: form.salary, years: form.years, snipe: form.snipe,
    beat: form.beat, evidence: form.evidence,
    closes: closes && !Number.isNaN(closes.getTime()) ? closes.toISOString() : '',
  });
}
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
  refresh: async (force) => {
    await go().TargetPoolRefresh(force);
  },
  pool: async (franchiseId, scope) =>
    parsePool(
      await go().TargetPool(
        main.RawPoolScope.createFrom({
          franchiseId, position: scope.position, sort: scope.sort, desc: scope.desc,
        }),
      ),
    ),
  plans: async (franchiseId) => parseBoard(await go().TargetBidPlans(franchiseId)),
  preview: async (franchiseId, form) =>
    parsePreview(await go().TargetBidPreview(rawBid(franchiseId, form))),
  draft: async (franchiseId, form) =>
    parsePlan(await go().TargetBidDraft(rawBid(franchiseId, form))),
  revise: async (id, franchiseId, form) =>
    parsePlan(await go().TargetBidRevise(id, rawBid(franchiseId, form))),
  handOff: async (id) => parsePlan(await go().TargetBidHandOff(id)),
  posted: async (id) => parsePlan(await go().TargetBidPosted(id)),
  withdraw: async (id) => parsePlan(await go().TargetBidWithdraw(id)),
};
