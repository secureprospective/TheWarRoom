import { selectProvider } from '../data/provider';

const unavailable = async (): Promise<never> => {
  throw new Error('Data and sources needs the desktop app. No sample data is substituted.');
};

export const sourcesAPI = {
  read: unavailable,
  crosswalk: unavailable,
  crosswalkReport: unavailable,
  loadCrosswalk: unavailable,
  refresh: (_scope: string) => unavailable(),
  folder: (_path: string) => unavailable(),
  photos: (_on: boolean) => unavailable(),
  cacheScopes: unavailable,
  cacheClear: (_scope: string) => unavailable(),
};

export const subscribeSources = (changed: () => void) => selectProvider().onSeasonChange(changed);
