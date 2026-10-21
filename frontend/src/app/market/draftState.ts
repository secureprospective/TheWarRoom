import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { readDraftLab } from '../draftreports/api';
import { unavailable } from './provider';
import type { MarketFilter, SurfaceReading } from './contract';
import { FILTER_KEYS, keepFilter, restoreFilter, subset } from '../shell/viewFilters';

export type DraftScope = {
  from: number;
  to: number;
  seasons: number;
  group: string;
  segments: string[];
  nfl: string[];
};
export const segments = [
  ['r1a', '1.01–1.10'], ['r1b', '1.11–1.32'], ['r2a', '2.01–2.16'],
  ['r2b', '2.17–2.32'], ['r3', 'Round 3'], ['r4', 'Round 4'], ['r5', 'Round 5 on'],
] as const;
export const nflRounds = [
  ['n1', 'NFL 1st'], ['n2', 'NFL 2nd'], ['n3', 'NFL 3rd'],
  ['n4', 'NFL 4th–7th'], ['nx', 'None that spring'],
] as const;
const positions = ['QB', 'RB', 'WR', 'TE', 'PK', 'DL', 'LB', 'DB'];
const initial = (): DraftScope => ({
  from: 2017, to: 2026, seasons: 3, group: 'DL', segments: [], nfl: [],
});
// parseDraftScope reads a kept draft filter (S-10); anything but the shape draftFiltered() makes is null.
export function parseDraftScope(v: unknown): DraftScope | null {
  if (typeof v !== 'object' || v === null) return null;
  const r = v as Record<string, unknown>;
  const kept = {
    segments: subset(r.segments, segments.map(([id]) => id)),
    nfl: subset(r.nfl, nflRounds.map(([id]) => id)),
  };
  const all = initial();
  const recent = draftFiltered(all, { field: 'draft-from', value: 'recent' }).from;
  if (![all.from, recent].includes(r.from as number) || r.to !== all.to ||
    ![2, 3, 4].includes(r.seasons as number) || !positions.includes(r.group as string) ||
    kept.segments === undefined || kept.nfl === undefined) return null;
  return { from: r.from as number, to: all.to, seasons: r.seasons as number, group: r.group as string,
    segments: kept.segments, nfl: kept.nfl };
}
type DraftReader = (scope: DraftScope) => Promise<SurfaceReading>;
function absent(reason: string, status: 'failed' | 'not_wired' = 'failed'): SurfaceReading {
  return { ...unavailable(reason, status), threshold: 40 };
}
export const readDraft: DraftReader = readDraftLab;

function toggle(xs: string[], value: string, allowed: readonly string[]) {
  if (!allowed.includes(value)) throw new Error('Draft: invalid filter');
  const next = xs.includes(value) ? xs.filter(x => x !== value) : [...xs, value];
  return next.length === allowed.length ? [] : allowed.filter(x => next.includes(x));
}
export function draftFiltered(scope: DraftScope, change: MarketFilter): DraftScope {
  switch (change.field) {
    case 'draft-segment': return {
      ...scope, segments: toggle(scope.segments, change.value, segments.map(([id]) => id)),
    };
    case 'draft-nfl': return {
      ...scope, nfl: toggle(scope.nfl, change.value, nflRounds.map(([id]) => id)),
    };
    case 'draft-position':
      if (!positions.includes(change.value))
        throw new Error('Draft: invalid position');
      return { ...scope, group: change.value };
    case 'draft-seasons':
      if (!['2', '3', '4'].includes(change.value)) throw new Error('Draft: invalid judgment');
      return { ...scope, seasons: Number(change.value) };
    case 'draft-from':
      if (!['all', 'recent'].includes(change.value)) throw new Error('Draft: invalid class range');
      return { ...scope, from: change.value === 'recent' ? 2024 : 2017 };
    case 'draft-reset': return initial();
    default: return scope;
  }
}
function pickScope(scope: DraftScope, kind: string): DraftScope {
  const round = kind.split(':')[0];
  const ids = round === 'pick1' ? ['r1a', 'r1b'] : round === 'pick2' ? ['r2a', 'r2b'] : ['r3', 'r4', 'r5'];
  return { ...scope, segments: ids };
}
export function createDraftState(reader: DraftReader) {
  const store = createStore<{
    scope: DraftScope;
    reading?: SurfaceReading;
    pending: boolean;
    picks: Record<string, SurfaceReading>;
  }>(() => ({
    scope: restoreFilter(FILTER_KEYS.draft, parseDraftScope, initial()), pending: false, picks: {},
  }));
  let revision = 0;
  const seen = new Map<string, string>();
  async function read(scope: DraftScope) {
    try { return await reader(scope); }
    catch (cause) { return absent(String(cause)); }
  }
  async function refresh(scope = store.getState().scope) {
    const token = ++revision;
    keepFilter(FILTER_KEYS.draft, scope);
    store.setState({ scope: { ...scope }, pending: true });
    const reading = await read(scope);
    if (token === revision) store.setState({ reading, pending: false });
  }
  async function pick(kind: string) {
    const scope = pickScope(store.getState().scope, kind), signature = JSON.stringify(scope);
    if (seen.get(kind) === signature && store.getState().picks[kind]) return;
    const token = revision, reading = await read(scope);
    if (token !== revision) return;
    seen.set(kind, signature);
    store.setState(s => ({ picks: { ...s.picks, [kind]: reading } }));
  }
  return {
    use: () => useStore(store), peek: store.getState, refresh, pick,
    change: (change: MarketFilter) => {
      if (change.field === 'draft-retry') seen.clear();
      return refresh(draftFiltered(store.getState().scope, change));
    },
  };
}
export const draftState = createDraftState(readDraft);
