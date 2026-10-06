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
