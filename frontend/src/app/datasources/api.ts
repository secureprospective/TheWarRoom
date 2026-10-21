import type * as Bindings from '../../../wailsjs/go/sources/API';
import { selectProvider } from '../data/provider';

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Data and sources needs the desktop app. No sample data is substituted.');
  return (window as unknown as { go: { sources: { API: typeof Bindings } } }).go.sources.API;
}

export const sourcesAPI = {
  read: async () => bindings().TargetDataSources(),
  crosswalk: async () => bindings().TargetCrosswalk(),
  crosswalkReport: async () => bindings().GetCrosswalkReport(),
  loadCrosswalk: async () => bindings().LoadCrosswalk(),
  refresh: async (scope: string) => bindings().TargetDataRefresh(scope),
  folder: async (path: string) => bindings().SetLeagueArchiveDir(path),
  cacheScopes: async () => bindings().TargetCacheScopes(),
  cacheClear: async (scope: string) => bindings().TargetCacheClear(scope),
};

export function subscribeSources(changed: () => void): () => void {
  return selectProvider().onSeasonChange(changed);
}
