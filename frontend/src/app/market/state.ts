import { createReadQuery } from '../data/query';
import {
  assets,
  phases,
  type MarketFilter,
  type MarketProvider,
  type MarketScope,
  type SurfaceReading,
} from './contract';
import { marketProvider, marketScopeInput, unavailable } from './provider';
import { FILTER_KEYS, keepFilter, restoreFilter, subset } from '../shell/viewFilters';

type State = {
  scope: MarketScope;
  lists?: SurfaceReading;
  clock?: SurfaceReading;
  stretches?: SurfaceReading;
  profiles: Record<string, SurfaceReading>;
  pending: boolean;
  applied?: MarketScope;
};
function toggle(selected: string[] | null, value: string, all: readonly string[]): string[] | null {
  if (!all.includes(value)) throw new Error(`Market filter: unknown ${value}`);
  const next = selected
    ? selected.includes(value)
      ? selected.filter((id) => id !== value)
      : [...selected, value]
    : [value];
  return next.length === 0 || next.length === all.length
    ? null
    : all.filter((id) => next.includes(id));
}
export function filtered(scope: MarketScope, change: MarketFilter): MarketScope {
  if (change.field === 'phase')
    return {
      ...scope,
      phases: toggle(
        scope.phases,
        change.value,
        phases.map(([id]) => id),
      ),
    };
  if (change.field === 'asset')
    return { ...scope, assets: toggle(scope.assets, change.value, assets) };
  if (change.field === 'age') return { ...scope, byAge: change.value === 'split' };
  if (change.field === 'window') {
    const windows: Record<string, string[]> = {
      spring: ['winter', 'opening', 'between'],
      summer: ['rookieDraft', 'summer'],
      season: ['early', 'midseason', 'deadline'],
    };
    if (!windows[change.value]) throw new Error('Market filter: unknown window');
    return { ...scope, phases: windows[change.value] };
  }
  return { ...scope, phases: null, assets: null, byAge: false };
}
const allScope = (): MarketScope => ({ from: 2017, to: 2026, phases: null, assets: null, byAge: false });
// parseMarketScope reads a kept market filter (S-10); anything but the shape filtered() makes is null.
export function parseMarketScope(v: unknown): MarketScope | null {
  if (typeof v !== 'object' || v === null) return null;
  const r = v as Record<string, unknown>;
  const phaseIds = phases.map(([id]) => id);
  const kept = {
    phases: r.phases === null ? null : subset(r.phases, phaseIds),
    assets: r.assets === null ? null : subset(r.assets, assets),
  };
  const all = allScope();
  if (r.from !== all.from || r.to !== all.to || typeof r.byAge !== 'boolean' ||
    kept.phases === undefined || kept.assets === undefined) return null;
  return { ...all, phases: kept.phases, assets: kept.assets, byAge: r.byAge };
}
export function createMarketState(provider: MarketProvider) {
  const safe = (read: Promise<SurfaceReading>) => read.catch((cause) => unavailable(String(cause)));
  const query = createReadQuery('market.TargetMarketLists', {
    initial: (): State => ({
      scope: restoreFilter(FILTER_KEYS.market, parseMarketScope, allScope()), profiles: {}, pending: false,
    }), companions: ['market.TargetPickClock', 'market.TargetMarketStretches'],
    arguments: ([scope]: [MarketScope]): [ReturnType<typeof marketScopeInput>] => [marketScopeInput(scope)],
    read: async (request: MarketScope) => {
      const [lists, clock, stretches] = await Promise.all([
        safe(provider.lists(request)), safe(provider.clock(request)), safe(provider.stretches(request)),
      ]);
      return { lists, clock, stretches };
    }, accept: (value, [scope]) => ({ ...value, applied: scope }),
  });
  const store = query.store;
  function detail(kind: string) {
    return createReadQuery('market.TargetAssetProfile', {
      initial: () => ({}), cacheSize: 1,
      arguments: ([request]: [{ scope: MarketScope; kind: string }]):
        [{ scope: ReturnType<typeof marketScopeInput>; kind: string }] => [
          { ...request, scope: marketScopeInput(request.scope) },
        ],
      read: ({ scope, kind }: { scope: MarketScope; kind: string }) => safe(provider.profile(scope, kind)),
      accept: reading => {
        store.setState(s => ({ profiles: { ...s.profiles, [kind]: reading } }));
        return {};
      },
    });
  }
  const details = new Map<string, ReturnType<typeof detail>>();
  async function refresh(scope = store.getState().scope) {
    for (const reading of details.values()) reading.fence();
    keepFilter(FILTER_KEYS.market, scope);
    store.setState({ scope });
    await query.load([scope]);
  }
  async function profile(kind: string) {
    const before = store.getState();
    let reading = details.get(kind);
    if (!reading) { reading = detail(kind); details.set(kind, reading); }
    await reading.load([{ scope: before.applied ?? before.scope, kind }], { policy: 'cache-first' });
  }
  return {
    use: query.use,
    peek: store.getState,
    refresh,
    profile,
    change: (change: MarketFilter) =>
      refresh(
        change.field === 'retry'
          ? store.getState().scope
          : filtered(store.getState().scope, change),
      ),
  };
}
export const marketState = createMarketState(marketProvider);
export const prefetchMarket = () => {
  if (!marketState.peek().lists && !marketState.peek().pending) void marketState.refresh();
};
