import { parseArchive, type Archive } from './contract';
import type * as Bindings from '../../../wailsjs/go/archive/API';

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League archive reports require the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { archive: { API: typeof Bindings } } }).go.archive.API;
}

export const archiveAPI = {
  read: async (season: number, team: string, other: string): Promise<Archive> =>
    parseArchive(await bindings().TargetArchive(season, team, other)),
};

export function subscribeArchive(changed: () => void): () => void {
  let disposed = false;
  let stop: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(m => {
    if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
      stop = m.EventsOn('target:season', changed);
  });
  return () => { disposed = true; stop?.(); };
}
