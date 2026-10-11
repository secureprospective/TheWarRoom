import type { MarketRange } from '../market/contract';
import type { Provenance } from '../data/contract';
import type { RailReceipt } from '../shell/EnvelopeRail';

export type Amount = { cents: number; text: string };
export type Reading<T> = {
  status: 'ok' | 'thin' | 'failed' | 'not_wired';
  absenceReason: string | null;
  count: number;
  threshold: number;
  baseline: string;
  cutoff: string;
  method: string;
  inputSHA: string;
  buildStamp: string;
  howCounted: string;
  provenance: Provenance;
  value: T;
};
export type MarketFact = { label: string; text: string; reason: string; range: MarketRange | null };
export type PoolRow = {
  market?: MarketFact[];
  marketLines: string[];
  id: string;
  name: string;
  position: string;
  team: string;
  age: string;
  experience: string;
  minimum: Amount;
  plan: string;
  planId: string;
};
export type Pool = {
  rows: PoolRow[]; positions: string[]; summary: string; sorts: { key: string; label: string }[];
};
export type PoolScope = {
  position: string;
  sort: 'name' | 'position' | 'minimum' | 'projected' | 'rank' | 'adp' | 'aav';
  desc: boolean;
};
export type Rule = {
  rule: string;
  label: string;
  verdict: 'pass' | 'fail' | 'unknown' | 'info';
  reason: string;
  mark: string;
};
export type Preview = {
  problem: string;
  player: string;
  salary: Amount;
  snapped: string;
  points: string;
  rules: Rule[];
  blocked: boolean;
  note: string;
  post: string;
  ruleSummary: string;
  closesNote: string;
};
export type BidAction = { id: string; label: string; disabled: boolean; reason: string };
export type Plan = {
  id: string;
  stateCode: string;
  franchiseId: string;
  playerId: string;
  years: number;
  snipe: boolean;
  evidence: string;
  closes: string;
  form: { salary: string; beat: string };
  player: string;
  salary: Amount;
  points: string;
  contract: string;
  beatText: string;
  state: string;
  status: string;
  rules: Rule[];
  ruleSummary: string;
  post: string;
  actions: BidAction[];
  history: { at: string; state: string; note: string }[];
  receipt: RailReceipt;
};
const over = ['landed', 'awarded_off_app', 'withdrawn', 'stale', 'failed'];
// A plan the owner can still act on or that is still on the board.
export const planOpen = (stateCode: string) => !over.includes(stateCode);
export type Board = { plans: Plan[]; summary: string };
export type PoolReading = Reading<Pool>;
export type BoardReading = Reading<Board>;

// What the bid form holds while it is open: the owner's figures as typed.
export type BidFormState = {
  planId: string;
  playerId: string;
  playerName: string;
  salary: string;
  years: number;
  snipe: boolean;
  beat: string;
  evidence: string;
  // The closing time stays out until the owner asks for it: an empty date box looks filled in.
  closesOpen: boolean;
  closesDate: string;
  closesTime: string;
};
export type FormField =
  'salary' | 'years' | 'snipe' | 'beat' | 'evidence' | 'closesDate' | 'closesTime';

export type PoolFilter = {
  field:
    | 'pool-position'
    | 'pool-sort'
    | 'pool-refresh'
    | 'pool-plan'
    | 'pool-cancel'
    | 'pool-save'
    | 'pool-handoff'
    | 'pool-posted'
    | 'pool-withdraw'
    | 'pool-revise'
    | 'pool-copy'
    | 'pool-open'
    | 'pool-toggle'
    | 'pool-unposted'
    | 'pool-closes-set'
    | 'pool-closes-clear';
  value: string;
};
