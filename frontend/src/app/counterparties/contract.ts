import type { Provenance } from '../data/contract';

export type CounterScope = {
  from: number;
  to: number;
  tiers: string[];
  windows: string[];
  postures: string[];
  team: string;
};
// The filters travel on the existing market command; Market.set routes counterpart-* here.
export type CounterFilter = {
  field:
    | 'counterpart-tier'
    | 'counterpart-window'
    | 'counterpart-posture'
    | 'counterpart-team'
    | 'counterpart-reset'
    | 'counterpart-retry'
    | 'counterpart-sort';
  value: string;
};
export type TeamEdge = {
  n: number;
  mean: number | null;
  lo: number | null;
  hi: number | null;
  total?: number;
  tone: string;
};
export type TeamShare = {
  countText?: string;
  k: number;
  n: number;
  p: number | null;
  lo?: number;
  hi?: number;
  base?: number;
  tone?: string;
};
export type TierShift = {
  early: number;
  late: number;
  from: [number, number];
  to: [number, number];
  clear: boolean;
} | null;
export type TierPick = {
  with: string;
  tradingAs: string;
  id: string;
  label: string;
  hint: string;
  count: number;
};
export type TeamOption = { fid: string; name: string };
export type PostureCell = {
  id: string;
  label: string;
  p: number | null;
  tone: string;
  title: string;
};
export type CardRow = { label: string; text: string; tone: string; range?: TeamEdge };
export type TierCard = {
  id: string;
  label: string;
  hint: string;
  thin: boolean;
  perSeason: number;
  perSeasonText: string;
  posture: PostureCell[];
  rows: CardRow[];
};
export type TrendRow = {
  nText: string;
  showShares: boolean;
  season: number;
  n: number;
  now: TeamShare;
  later: TeamShare;
  net: number | null;
};
export type TierTrend = {
  tier: string;
  rows: TrendRow[];
  shift: { now: TierShift; later: TierShift };
};
export type TeamTrends = { seasons: number[]; tiers: TierTrend[] };
export type PairView = {
  other: string;
  n: number;
  now: TeamEdge;
  real: TeamEdge;
  trades: string;
  title: string;
};
export type CounterRow = {
  dayText: string;
  sinceText: string;
  nText: string;
  partnersText: string;
  fid: string;
  name: string;
  n: number;
  perSeason: number;
  tiers: Record<string, number>;
  forNow: TeamShare;
  forLater: TeamShare;
  day: TeamEdge;
  since: TeamEdge;
  picks: TeamEdge;
  age: TeamEdge;
  partner: { fid: string; n: number } | null;
  partnerName: string;
  partnersCount: number;
  standingTitle: string;
  partnerTrades: string;
};
export type CounterWords = {
  team: string;
  sides: string;
  perSeason: string;
  now: string;
  later: string;
  picks: string;
  age: string;
  partners: string;
  partner: string;
  day: string;
  since: string;
  legend: string;
  up: string;
  down: string;
};
export type Counterparties = {
  words: CounterWords;
  countText: string;
  summary: string;
  asOf: string;
  headline: string;
  note: string;
  picks: TierPick[];
  teams: TeamOption[];
  cards: TierCard[];
  trendSay: string[];
  trends: TeamTrends;
  pairs: PairView[];
  rows: CounterRow[];
};
export type SeasonCellView = {
  nowText: string;
  laterText: string;
  swapText: string;
  againstText: string;
  season: number;
  n: number;
  now: number;
  later: number;
  swap: number;
  against: number;
  flagged: boolean;
  mostlyText: string;
  net: number | null;
  finish: number | null;
  label: string;
  title: string;
};
export type KindFlowView = {
  kind: string;
  name: string;
  inN: number;
  outN: number;
  inW: number;
  outW: number;
  text: string;
};
export type PartnerView = {
  fid: string;
  name: string;
  n: number;
  net: TeamEdge;
  tradesText: string;
  netText: string;
};
export type AsTierView = {
  tier: string;
  n: number;
  edge: TeamEdge;
  label: string;
  text: string;
};
export type TeamCounterpart = {
  words: CounterWords;
  countText: string;
  flowScale: number;
  fid: string;
  name: string;
  asOf: string;
  sides: number;
  empty: boolean;
  note: string;
  seasons: SeasonCellView[];
  seasonSay: string;
  posture: PostureCell[];
  kinds: KindFlowView[];
  kindSay: string;
  partners: PartnerView[];
  partnersSay: string;
  partnersText: string;
  behaviour: {
    n: number;
    perSeason: number;
    posture: Record<string, TeamShare>;
    windows: Record<string, TeamShare>;
    now: TeamEdge;
    real: TeamEdge;
    picks: TeamEdge;
    age: TeamEdge;
  };
  asTier: AsTierView[];
  day: TeamEdge;
  since: TeamEdge;
  dayText: string;
  sinceText: string;
};
type ReadingShell = {
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
  provenance?: Omit<Provenance, 'kind'> & { kind: 'archive' };
};
export type CounterpartiesReading = ReadingShell & { value?: Counterparties };
export type TeamPanelReading = ReadingShell & { value?: TeamCounterpart };
