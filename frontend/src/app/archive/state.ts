import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import { archiveAPI } from './api';
import type { Archive } from './contract';
export type { Archive } from './contract';
type ArchiveState = { archive?: Archive; pending: boolean; error: string; season: number;
  team: string | null; other: string };
export function createArchiveState(read: (season: number, team: string, other: string) => Promise<Archive>,
  mine: () => string) {
  const store = createStore<ArchiveState>(() =>
    ({ pending: false, error: '', season: 0, team: null, other: '' }));
  let revision = 0;
  async function load() {
    const request = ++revision;
    store.setState({ pending: true, error: '' });
    const { season, team, other } = store.getState();
    try {
      const archive = await read(season, team ?? mine(), other);
      if (request === revision)
        store.setState({ archive, season: archive.season, team: archive.team, other: archive.other });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
  function choose(action: string, value: string) {
    if (action === 'season') {
      if (!/^20\d{2}$/.test(value)) throw new Error('League archive: invalid season');
      store.setState({ season: Number(value) });
    } else if (action === 'team' || action === 'other') {
      if (value && !/^\d{4}$/.test(value)) throw new Error('League archive: invalid franchise');
      store.setState({ [action]: value });
    } else if (action !== 'refresh') throw new Error('League archive: invalid action');
    return load();
  }
  return { use: () => useStore(store), peek: store.getState, load, choose };
}
export const archiveState = createArchiveState(archiveAPI.read, () => commands.read().franchiseId ?? '');
