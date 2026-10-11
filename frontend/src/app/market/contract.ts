import type { Provenance } from '../data/contract';
export type MarketScope = {
  from: number;
  to: number;
  phases: string[] | null;
  assets: string[] | null;
  byAge: boolean;
};
export type MarketRange = { value: number; lo: number; hi: number; text: string };
export type MarketLine = {
  key: string;
  label: string;
  detail: string;
  count: number;
  range: MarketRange | null;
  mark: string;
};
export type MarketSection = { id: string; title: string; note: string; rows: MarketLine[] };
export type MarketSurface = { summary: string; sections: MarketSection[] };
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
  provenance?: Omit<Provenance, 'kind'> & { kind: 'archive' };
  value?: T;
};
// The shell every archive reading carries, whatever its payload; HowCounted shows only this.
export type ReadingShell = Omit<Reading<unknown>, 'value'>;
export type SurfaceReading = Reading<MarketSurface>;
export type MarketProvider = {
  lists(scope: MarketScope): Promise<SurfaceReading>;
  clock(scope: MarketScope): Promise<SurfaceReading>;
  stretches(scope: MarketScope): Promise<SurfaceReading>;
  profile(scope: MarketScope, kind: string): Promise<SurfaceReading>;
};
export const phases = [
  ['winter', 'After the season'],
  ['opening', 'Before NFL draft'],
  ['between', 'NFL to rookie draft'],
  ['rookieDraft', 'Rookie draft'],
  ['summer', 'Draft to kickoff'],
  ['early', 'Weeks 1–4'],
  ['midseason', 'Midseason'],
  ['deadline', 'Deadline run-in'],
] as const;
export const assets = [
  'QB',
  'RB',
  'WR',
  'TE',
  'PK',
  'DL',
  'LB',
  'DB',
  'pick1:next',
  'pick2:next',
  'pick3:next',
  'pick1:later',
  'pick2:later',
  'pick3:later',
];
export type MarketFilter = {
  field:
    | 'phase'
    | 'asset'
    | 'age'
    | 'window'
    | 'reset'
    | 'retry'
    | 'inspect'
    | 'draft-segment'
    | 'draft-nfl'
    | 'draft-position'
    | 'draft-seasons'
    | 'draft-from'
    | 'draft-reset'
    | 'draft-retry'
    | import('../counterparties/contract').CounterFilter['field']
    | import('../pool/contract').PoolFilter['field'];
  value: string;
};

export const assetLabels: Record<string, string> = {
  'pick1:next': 'Next draft · 1st',
  'pick2:next': 'Next draft · 2nd',
  'pick3:next': 'Next draft · 3rd+',
  'pick1:later': 'Later drafts · 1st',
  'pick2:later': 'Later drafts · 2nd',
  'pick3:later': 'Later drafts · 3rd+',
};
