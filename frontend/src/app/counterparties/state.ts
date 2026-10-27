import { createReadQuery, readKey } from '../data/query';
import { counterpartProvider } from './api';
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
  type State = {
    sort: { key: SortKey; descending: boolean };
    scope: CounterScope;
    applied: CounterScope;
    reading?: CounterpartiesReading;
    panels: Record<string, TeamPanelReading>;
    teamPending: Record<string, boolean>;
  };
  const query = createReadQuery('counterparties.TargetCounterparties', {
    initial: (): State => ({
      sort: { key: 'day', descending: true }, scope: initial(), applied: initial(),
      panels: {}, teamPending: {},
    }), read: async scope => {
      try { return await provider.surface(scope); }
      catch (cause) { return failure(cause); }
    }, accept: (reading, [scope]) => ({ reading, applied: { ...scope } }),
  });
  const store = query.store;
  function detail(fid: string) {
    const reading = createReadQuery('counterparties.TargetCounterpartTeam', {
      initial: () => ({}), cacheSize: 1,
      read: async ({ scope, fid }) => {
        try { return await provider.team(scope, fid); }
        catch (cause) { return { ...failure(cause), threshold: 1 }; }
      }, accept: value => {
        store.setState(s => ({ panels: { ...s.panels, [fid]: value } }));
        return {};
      },
    });
    reading.store.subscribe(s => {
      store.setState(current => ({ teamPending: { ...current.teamPending, [fid]: s.pending } }));
    });
    return { reading, requested: '' };
  }
  const details = new Map<string, ReturnType<typeof detail>>();
  async function refresh(scope = store.getState().scope) {
    store.setState({ scope });
    await query.load([scope]);
  }
  async function team(fid: string) {
    const scope = { ...store.getState().applied, team: '' };
    let panel = details.get(fid);
    if (!panel) { panel = detail(fid); details.set(fid, panel); }
    const args: [{ scope: CounterScope; fid: string }] = [{ scope, fid }];
    const key = readKey('counterparties.TargetCounterpartTeam', args);
    if (panel.reading.peek().pending && panel.requested === key) return;
    panel.requested = key;
    await panel.reading.load(args, { policy: 'cache-first' });
  }
  return {
    use: query.use,
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
      if (change.field === 'counterpart-retry') {
        for (const panel of details.values()) panel.reading.invalidate();
      }
      return refresh(filtered(store.getState().scope, change));
    },
  };
}
export const counterState = createCounterState(counterpartProvider);
