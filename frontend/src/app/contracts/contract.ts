import type { Provenance } from '../data/contract';

export type Amount = { cents: number; text: string };
export type Year = {
  season: number;
  cost: Amount;
  inferred: boolean;
  tag: 'Observed' | 'Inferred';
  reason: string;
};
export type Row = {
  id: string;
  franchiseId: string;
  team: string;
  name: string;
  position: string;
  salary: Amount;
  expiration: number;
  status: string;
  info: string;
};
export type Table = {
  directorySource: string;
  directoryAsOf: string;
  rows: Row[];
  teams: { id: string; name: string }[];
  positions: string[];
};
export type Cap = {
  team: string;
  season: number;
  cap: Amount | null;
  used: Amount;
  room: Amount | null;
  capSource: string;
  future: Year[];
};
export type Adjustment = {
  id: string;
  team: string;
  franchiseId: string;
  amount: Amount;
  description: string;
};
export type PlayerContract = { player: Row; years: Year[] };
export type Reading<T> = {
  status: 'ok' | 'thin' | 'failed' | 'not_wired';
  absenceReason: string | null;
  value?: T;
  provenance: Provenance;
  count: number;
  threshold: number;
  baseline: string;
  cutoff: string;
  method: string;
  inputSHA: string;
  buildStamp: string;
  howCounted: string;
};
export type Readings = {
  table: Reading<Table>;
  cap: Reading<Cap>;
  adjustments: Reading<{ rows: Adjustment[] }>;
  cantCut: Reading<string[]>;
};
export type Scope = {
  mine: string;
  team: string;
  position: string;
  sort: 'name' | 'team' | 'position' | 'salary' | 'expiration';
  desc: boolean;
};
export interface ContractsProvider {
  contracts(scope: Scope): Promise<Readings>;
  player(id: string): Promise<Reading<PlayerContract>>;
  onChange(listener: () => void): () => void;
}
