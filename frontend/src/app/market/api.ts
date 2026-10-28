import type * as Bindings from '../../../wailsjs/go/market/API';

// The one place that touches Market's Go bindings: the historical Market surfaces, the asset
// board and the player market reports. Callers parse what these return.
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

export const marketAPI = {
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
