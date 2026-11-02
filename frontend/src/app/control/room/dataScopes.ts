import type { sourcesAPI } from '../../datasources/api';

/** The scopes Go lists for a refresh (the data sources reading) or for a cache clear. */
export const dataScopes = (mode: 'refresh' | 'clear', api: typeof sourcesAPI) =>
  mode === 'clear' ? api.cacheScopes() : api.read().then(r => r.scopes);
