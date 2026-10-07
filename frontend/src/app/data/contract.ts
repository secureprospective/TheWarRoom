import type { FreshnessState } from '../../components/board/freshness';

// The TypeScript half of internal/snapshot. JSON keys and enum strings must match the Go structs
// and domain constants exactly; parse.ts enforces them at the boundary.

export const FRESHNESS_STATES = [
  'live',
  'stale',
  'fail',
] as const satisfies readonly FreshnessState[];
export const PROVENANCE_KINDS = ['live', 'fixture'] as const;
export const POSITIONS = [
  'QB',
  'RB',
  'WR',
  'TE',
  'K',
  'DT',
  'DE',
  'LB',
  'CB',
  'S',
  'FLAG',
] as const;
export const CONTRACT_STATUSES = ['UFA', 'RFA', 'FT1', 'FT2', 'FLAG'] as const;
export const ROSTER_STATUSES = ['ROSTER', 'IR', 'TAXI_SQUAD'] as const;

export type Position = (typeof POSITIONS)[number];
export type Freshness = { state: FreshnessState; fetchedAt: string; note: string };
export type Provenance = {
  source: string;
  freshness: Freshness;
  kind: (typeof PROVENANCE_KINDS)[number];
};
export type Sourced<T> = { value: T; provenance: Provenance };
export type League = { season: number; franchiseCount: number; salaryCap?: number };
export type Franchise = { id: string; name?: string; capUsed?: number; capRoom?: number };
export type RosterPlayer = {
  id: string;
  salary: number;
  yearsRemaining?: number;
  contractStatus: (typeof CONTRACT_STATUSES)[number];
  rosterStatus: (typeof ROSTER_STATUSES)[number];
};
export type Roster = { franchiseId: string; players: RosterPlayer[] };
export type Player = {
  id: string;
  name?: string;
  position?: Position;
  team?: string;
  birthdate?: number;
  isRookie?: boolean;
};
export type Snapshot = {
  league: Sourced<League>;
  franchises: Sourced<Franchise[]>;
  rosters: Sourced<Roster[]>;
  players: Sourced<Player[]>;
};

export const PHASES = ['OFFSEASON', 'REGULAR_SEASON', 'PLAYOFFS'] as const;
export const URGENCIES = ['U0', 'U1', 'U2', 'U3'] as const;
export const WINDOW_KINDS = [
  'contract_options', 'rfa_tender', 'ufa_bidding', 're_sign', 'cut_day', 'trade_deadline', 'draft',
] as const;
export type Deadline = {
  id: string;
  label: string;
  at?: string;
  urgency: (typeof URGENCIES)[number];
  pinned: boolean;
  promoted: boolean;
};
export type ClockReading = {
  season: number;
  week?: number;
  phase: (typeof PHASES)[number];
  deadlines: Deadline[];
  windows: { kind: (typeof WINDOW_KINDS)[number]; status: 'unknown'; reason: string }[];
};
export const ENVELOPE_STATES = [
  'draft', 'blocked', 'ready', 'handed_off', 'not_yet_done', 'landed', 'not_verified',
  'failed', 'stale', 'dot_review', 'bid_pending', 'waiver_pending',
] as const;
export const ENVELOPE_EVENTS = [
  'checks_pass', 'checks_block', 'hand_off', 'no_change', 'match', 'partial', 'contradiction',
  'invalidate', 'rebase', 'await_dot', 'await_bid', 'await_waiver',
] as const;
export const GRAVITIES = ['G0', 'G1', 'G2', 'G3'] as const;
export const UNDO_CLASSES = ['instant', 'reversible', 'irreversible'] as const;
export type EnvelopeState = (typeof ENVELOPE_STATES)[number];
export type EnvelopeEvent = (typeof ENVELOPE_EVENTS)[number];
export type AuditEntry = {
  at: string; from: EnvelopeState; event: EnvelopeEvent; to: EnvelopeState; note: string;
};
export type EnvelopeSpec = {
  intent: string;
  leagueId: string;
  franchiseId: string;
  subject: { players: string[]; picks: string[] };
  expected: {
    player?: string;
    rosterStatus?: (typeof ROSTER_STATUSES)[number];
    trade?: {
      tradeId: string; offering: string; accepting: string;
      offeringGives: string[]; acceptingGives: string[];
    };
    lineup?: { week: number; starters: string[]; baseline: string[] };
  };
  gravity: (typeof GRAVITIES)[number];
  undo: (typeof UNDO_CLASSES)[number];
  target: { kind: 'mapped'; url: string } | { kind: 'unmapped' };
  deadline?: string;
};
export type Receipt = {
  correlationId: string; spec: EnvelopeSpec; state: EnvelopeState; audit: AuditEntry[];
};
export type EnvelopeDemo = { kind: 'fixture'; receipt: Receipt };

export const MFL_KEY_STATES = ['absent', 'connected', 'rejected', 'unreachable', 'unavailable'] as const;
export type MFLKeyStatus = {
  state: (typeof MFL_KEY_STATES)[number];
  league: string;
  season: number;
  verifiedAt?: string;
  detail?: string;
};

export type LineupPlayer = { id: string; position?: Position };
export type LineupProblem = { subject: string; kind: 'short' | 'over' | 'unknown'; message: string };
export type LineupCheck = { full: boolean; legal: boolean; problems: LineupProblem[] };
export type LineupReading = {
  franchise: string;
  week: number;
  starterCount: number;
  starters: LineupPlayer[];
  bench: LineupPlayer[];
  check: LineupCheck;
  provenance: Provenance;
  rulesSource: Provenance;
};

export type TradeAsset = { token: string; name: string; position?: Position };
export type TradeOffer = {
  tradeId: string;
  direction: 'to_you' | 'by_you';
  otherId: string;
  otherName: string;
  give: TradeAsset[];
  get: TradeAsset[];
  expires: string;
  comments: string;
};
export type TradeReading = { offers: TradeOffer[]; provenance: Provenance };

export type PulsePlayer = {
  id: string;
  name: string;
  position: string;
  team: string;
  score: number;
  secondsRemaining: number;
};
export type PulseSide = {
  franchiseId: string;
  name: string;
  score: number;
  secondsRemaining: number;
  playing: number;
  yetToPlay: number;
  starters: PulsePlayer[];
};
export type PulseMatchup = { home: PulseSide; away: PulseSide };
export type PulseReading = { week: number; matchups: PulseMatchup[]; provenance: Provenance };
export type Alert = {
  kind: string;
  urgency: (typeof URGENCIES)[number];
  title: string;
  detail: string;
  node: string;
  workspace: string;
  at: string;
  provenance: Provenance;
};
export type Unavailable = { kind: string; note: string };
export type AlertReading = { alerts: Alert[]; unavailable: Unavailable[] };
