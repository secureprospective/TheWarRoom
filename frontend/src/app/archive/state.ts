import { createReadQuery } from '../data/query';
import { commands } from '../commands/registry';
import { archiveAPI } from './api';
import type { Archive } from './contract';
export type { Archive } from './contract';
type ArchiveSelection = { archive?: Archive; season: number; team: string | null; other: string };
export function createArchiveState(read: (season: number, team: string, other: string) => Promise<Archive>,
  mine: () => string) {
  const query = createReadQuery('archive.TargetArchive', {
    initial: (): ArchiveSelection => ({ season: 0, team: null, other: '' }), read,
    accept: archive => ({ archive, season: archive.season, team: archive.team, other: archive.other }),
  });
  const load = () => query.load(() => {
    const { season, team, other } = query.peek();
    return [season, team ?? mine(), other];
  });
  function choose(action: string, value: string) {
    if (action === 'season') {
      if (!/^20\d{2}$/.test(value)) throw new Error('League archive: invalid season');
      query.set({ season: Number(value) });
    } else if (action === 'team' || action === 'other') {
      if (value && !/^\d{4}$/.test(value)) throw new Error('League archive: invalid franchise');
      query.set({ [action]: value });
    } else if (action !== 'refresh') throw new Error('League archive: invalid action');
    return load();
  }
  return { use: query.use, peek: query.peek, load, choose };
}
export const archiveState = createArchiveState(archiveAPI.read, () => commands.read().franchiseId ?? '');
