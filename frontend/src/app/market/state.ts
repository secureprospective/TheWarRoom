import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import {
  assets,
  phases,
  type MarketFilter,
  type MarketProvider,
  type MarketScope,
  type SurfaceReading,
} from './contract';
import { marketProvider, unavailable } from './provider';

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
export function createMarketState(provider: MarketProvider) {
  const store = createStore<State>(() => ({
    scope: { from: 2017, to: 2026, phases: null, assets: null, byAge: false },
    profiles: {},
    pending: false,
  }));
  let revision = 0;
  const profileScopes = new Map<string, string>();
  const safe = (read: Promise<SurfaceReading>) => read.catch((cause) => unavailable(String(cause)));
  async function refresh(scope = store.getState().scope) {
    const token = ++revision;
    store.setState({ scope, pending: true });
    const [lists, clock, stretches] = await Promise.all([
      safe(provider.lists(scope)),
      safe(provider.clock(scope)),
      safe(provider.stretches(scope)),
    ]);
    if (token !== revision) return;
    store.setState({
      lists,
      clock,
      stretches,
      applied: scope,
      pending: false,
      profiles: store.getState().profiles,
    });
  }
  async function profile(kind: string) {
    const before = store.getState();
    const scope = before.applied ?? before.scope;
    const signature = JSON.stringify(scope);
    if (before.profiles[kind] && profileScopes.get(kind) === signature) return;
    const token = revision;
    const reading = await safe(provider.profile(scope, kind));
    if (token === revision) {
      profileScopes.set(kind, signature);
      store.setState((s) => ({ profiles: { ...s.profiles, [kind]: reading } }));
    }
  }
  return {
    use: () => useStore(store),
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
