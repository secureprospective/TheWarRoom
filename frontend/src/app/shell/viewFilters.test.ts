import { describe, it, expect, vi } from 'vitest';
import { FILTER_KEYS, keepFilter, restoreFilter } from './viewFilters';
import { createMarketState, filtered, parseMarketScope } from '../market/state';
import { createDraftState, draftFiltered, parseDraftScope } from '../market/draftState';
import type { MarketProvider } from '../market/contract';
import { unavailable } from '../market/provider';

function memory() {
  const m = new Map<string, string>();
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v), m };
}

describe('S-10 filter persistence', () => {
  it('keeps a market filter and reads back exactly the shape filtered() made', () => {
    const all = { from: 2017, to: 2026, phases: null, assets: null, byAge: false };
    const summer = filtered(all, { field: 'window', value: 'summer' });
    const scope = filtered(summer, { field: 'age', value: 'split' });
    const s = memory();
    keepFilter(FILTER_KEYS.market, scope, s);
    expect(restoreFilter(FILTER_KEYS.market, parseMarketScope, all, s)).toEqual(scope);
  });
  it('drops a stored filter this build does not recognise, never half-applying it', () => {
    const all = { from: 2017, to: 2026, phases: null, assets: null, byAge: false };
    const s = memory();
    for (const bad of ['{', '"x"', JSON.stringify({ ...all, phases: ['summer', 'nope'] }),
      JSON.stringify({ ...all, from: 1999 }), JSON.stringify({ ...all, byAge: 'yes' })]) {
      s.setItem(FILTER_KEYS.market, bad);
      expect(restoreFilter(FILTER_KEYS.market, parseMarketScope, all, s)).toEqual(all);
    }
  });
  it('keeps the draft room filter, recent classes and all', () => {
    const start = createDraftState(async () => unavailable('x')).peek().scope;
    const scope = draftFiltered(draftFiltered(start, { field: 'draft-from', value: 'recent' }),
      { field: 'draft-segment', value: 'r2a' });
    expect(parseDraftScope(JSON.parse(JSON.stringify(scope)))).toEqual(scope);
    expect(parseDraftScope({ ...scope, group: 'K' })).toBeNull();
    expect(parseDraftScope({ ...scope, segments: ['r9'] })).toBeNull();
  });
  it('a market change stores the new scope and Reset stores the full view', async () => {
    const store = memory();
    vi.stubGlobal('window', { localStorage: store });
    const none = async () => unavailable('x');
    const provider = { lists: none, clock: none, stretches: none, profile: none } as unknown as
      MarketProvider;
    const market = createMarketState(provider);
    await market.change({ field: 'asset', value: 'QB' });
    expect(JSON.parse(store.getItem(FILTER_KEYS.market) ?? 'null').assets).toEqual(['QB']);
    expect(createMarketState(provider).peek().scope.assets).toEqual(['QB']);
    await market.change({ field: 'reset', value: '' });
    expect(createMarketState(provider).peek().scope.assets).toBeNull();
    delete (globalThis as { window?: unknown }).window;
  });
});
