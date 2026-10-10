import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { counterpartProvider } from './provider';
import type { ReadingShell } from '../market/contract';
import type { MarketFilter } from '../market/contract';
import type { CounterScope, CounterpartiesReading, TeamPanelReading } from './contract';

export const sortKeys = [
  'name',
  'n',
  'perSeason',
  'now',
  'later',
  'day',
  'since',
  'picks',
  'age',
  'partners',
] as const;
export type SortKey = (typeof sortKeys)[number];
export const tiers = [
  ['contender', 'Contenders'],
  ['hunt', 'In the hunt'],
  ['rebuild', 'Rebuilding'],
] as const;
export const windows = [
  ['spring', 'Offseason'],
  ['summer', 'Draft to kickoff'],
  ['season', 'In season'],
] as const;
export const postures = [
  ['now', 'Bought for now'],
  ['later', 'Built for later'],
  ['swap', 'Like for like'],
] as const;
const initial = (): CounterScope => ({
  from: 2017,
  to: 2026,
  tiers: [],
  windows: [],
  postures: [],
  team: '',
});
function toggle(values: string[], value: string, allowed: readonly (readonly [string, string])[]) {
  if (!allowed.some(([id]) => id === value)) throw new Error('Counterparties: invalid filter');
  const next = values.includes(value) ? values.filter((v) => v !== value) : [...values, value];
  return next.length === allowed.length
    ? []
    : allowed.map(([id]) => id).filter((id) => next.includes(id));
}
export function filtered(scope: CounterScope, change: MarketFilter): CounterScope {
  switch (change.field) {
    case 'counterpart-tier':
      return { ...scope, tiers: toggle(scope.tiers, change.value, tiers) };
    case 'counterpart-window':
      return { ...scope, windows: toggle(scope.windows, change.value, windows) };
    case 'counterpart-posture':
      return { ...scope, postures: toggle(scope.postures, change.value, postures) };
    case 'counterpart-team':
      return { ...scope, team: change.value };
    case 'counterpart-reset':
      return initial();
    default:
      return scope;
  }
}
function failure(cause: unknown): ReadingShell {
  return {
    status: 'failed',
    absenceReason: `Counterparties: ${String(cause)}. Retry the reading.`,
    count: 0,
    threshold: 40,
    baseline: '',
    cutoff: '',
    method: '',
    inputSHA: '',
    buildStamp: '',
    howCounted: '',
  };
}
export function createCounterState(provider: typeof counterpartProvider) {
  const store = createStore<{
    sort: { key: SortKey; descending: boolean };
    scope: CounterScope;
    applied: CounterScope;
    reading?: CounterpartiesReading;
    pending: boolean;
    panels: Record<string, TeamPanelReading>;
    teamPending: Record<string, boolean>;
  }>(() => ({
    sort: { key: 'day', descending: true },
    scope: initial(),
    applied: initial(),
    pending: false,
    panels: {},
    teamPending: {},
  }));
  let revision = 0;
  const seen = new Map<string, string>();
  const teamTokens = new Map<string, number>();
  async function refresh(scope = store.getState().scope) {
    const token = ++revision;
    store.setState({ scope, pending: true });
    let reading: CounterpartiesReading;
    try {
      reading = await provider.surface(scope);
    } catch (cause) {
      reading = failure(cause);
    }
    if (token === revision) store.setState({ reading, applied: { ...scope }, pending: false });
  }
  async function team(fid: string) {
    const scope = { ...store.getState().applied, team: '' },
      signature = JSON.stringify(scope);
    if (seen.get(fid) === signature) return;
    seen.set(fid, signature);
    const token = (teamTokens.get(fid) ?? 0) + 1;
    teamTokens.set(fid, token);
    store.setState((s) => ({ teamPending: { ...s.teamPending, [fid]: true } }));
    let reading: TeamPanelReading;
    try {
      reading = await provider.team(scope, fid);
    } catch (cause) {
      reading = { ...failure(cause), threshold: 1 };
    }
    if (teamTokens.get(fid) !== token) return;
    store.setState((s) => ({
      panels: { ...s.panels, [fid]: reading },
      teamPending: { ...s.teamPending, [fid]: false },
    }));
  }
  return {
    use: () => useStore(store),
    peek: store.getState,
    refresh,
    team,
    change: (change: MarketFilter) => {
      if (change.field === 'counterpart-sort') {
        if (!sortKeys.includes(change.value as SortKey)) throw new Error('Counterparties: invalid sort');
        const previous = store.getState().sort;
        store.setState({
          sort: {
            key: change.value as SortKey,
            descending: previous.key === change.value ? !previous.descending : false,
          },
        });
        return Promise.resolve();
      }
      if (change.field === 'counterpart-retry') seen.clear();
      return refresh(filtered(store.getState().scope, change));
    },
  };
}
export const counterState = createCounterState(counterpartProvider);
