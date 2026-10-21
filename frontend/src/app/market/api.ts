import type * as Bindings from '../../../wailsjs/go/market/API';
import type { market } from '../../../wailsjs/go/models';

// The one place that touches Market's Go bindings: the historical Market surfaces, the asset
// board and the player market reports. Callers parse what these return.
export type RawMarketScope = market.RawMarketScope;
function desktop(): typeof Bindings | undefined {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go) return undefined;
  return (window as unknown as { go: { market: { API: typeof Bindings } } }).go.market.API;
}
function bindings(missing: string): typeof Bindings {
  const api = desktop();
  if (!api) throw new Error(missing);
  return api;
}
const values = 'Player values require the desktop app; no browser fixture is substituted.';
const reports = 'Player market reports require the desktop app; no browser fixture is substituted.';
const historical = 'Historical Market needs the desktop app';

export const marketAPI = {
  // hasDesktop is false in a plain browser, where the surfaces read as not wired.
  hasDesktop: (): boolean => desktop() !== undefined,
  lists: async (raw: RawMarketScope): Promise<unknown> => bindings(historical).TargetMarketLists(raw),
  clock: async (raw: RawMarketScope): Promise<unknown> => bindings(historical).TargetPickClock(raw),
  stretches: async (raw: RawMarketScope): Promise<unknown> =>
    bindings(historical).TargetMarketStretches(raw),
  profile: async (raw: RawMarketScope, kind: string): Promise<unknown> =>
    bindings(historical).TargetAssetProfile({ scope: raw, kind } as market.RawMarketAsset),
  assetBoard: async (): Promise<unknown> => bindings(values).TargetAssetBoard(),
  reports: async (): Promise<unknown> => bindings(reports).TargetMarketReports(),
  refreshReports: async (force: boolean): Promise<void> =>
    bindings(reports).TargetMarketReportsRefresh(force),
};

// subscribeSeason calls changed whenever the season's held feeds change, until the returned stop.
export function subscribeSeason(changed: () => void): () => void {
  let disposed = false;
  let stop: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(m => {
    if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
      stop = m.EventsOn('target:season', changed);
  });
  return () => { disposed = true; stop?.(); };
}
