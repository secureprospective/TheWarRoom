import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { array, integer, object, text } from '../data/parseValues';
import { parseBoard, type Board } from '../reports/contract';
import { choice, type Choice } from '../race/contract';
import { commands } from '../commands/registry';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type Archive = { board: Board; season: number; seasons: Choice[]; team: string; other: string;
  teams: Choice[] };
export function parseArchive(value: unknown): Archive {
  const p = 'League archive';
  const r = object(value, p, ['board', 'season', 'seasons', 'team', 'other', 'teams']);
  const a = { board: parseBoard(r.board), season: integer(r.season, p),
    seasons: array(r.seasons, p, choice), team: text(r.team, p), other: text(r.other, p),
    teams: array(r.teams, p, choice) };
  if (a.seasons.length && !a.seasons.some(s => s.value === String(a.season)))
    throw new Error(`${p}: season outside choices`);
  if (a.teams.length && [a.team, a.other].some(t => t && !a.teams.some(c => c.value === t)))
    throw new Error(`${p}: franchise outside choices`);
  if (!a.board.reports.length) throw new Error(`${p}: missing reports`);
  return a;
}
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
export const archiveState = createArchiveState(async (season, team, other) => {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League archive reports require the desktop app; no browser fixture is substituted.');
  const bindings = (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
  return parseArchive(await bindings.TargetArchive(season, team, other));
}, () => commands.read().franchiseId ?? '');
