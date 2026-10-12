import { TargetDataSources, TargetDataRefresh, SetLeagueArchiveDir, TargetCrosswalk,
  GetParams, SetParam, ResetParam } from '../../../wailsjs/go/main/App';
import { object, text, array, choice } from '../data/parseValues';
import { report, type Report } from '../reports/contract';
import type { main } from '../../../wailsjs/go/models';

export type Scope = { id: string; label: string; note: string; action: string; progress: string };
export type Sources = { health: Report; signals: Report; facts: Report; checks: Report;
  archiveDir: string; scopes: Scope[]; words: Record<string, string> };
export type Outcome = { ok: boolean; error: string; summary: string };
export function parseSources(v: unknown): Sources {
  const p = 'Data and sources';
  const r = object(v, p, ['health', 'signals', 'facts', 'checks', 'archiveDir', 'scopes', 'words']);
  const scopes = array(r.scopes, p, (v, p) => {
    const s = object(v, p, ['id', 'label', 'note', 'action', 'progress']);
    return { id: choice(s.id, p, ['league', 'signals', 'crosswalk', 'archive']),
      label: text(s.label, p), note: text(s.note, p), action: text(s.action, p),
      progress: text(s.progress, p) };
  });
  if (scopes.length !== 4 || new Set(scopes.map(s => s.id)).size !== 4)
    throw new Error(`${p}: refresh scopes missing or repeated`);
  const keys = ['title', 'refresh', 'folder', 'setFolder', 'read', 'readProgress', 'readResult', 'how',
    'crosswalkHow', 'parameters', 'permission', 'find', 'empty', 'league', 'setting', 'value', 'default',
    'override', 'range', 'overridden', 'fitted', 'handSet', 'apply', 'reset', 'clear', 'kept'];
  const rawWords = object(r.words, p, keys);
  const words = Object.fromEntries(keys.map(k => [k, text(rawWords[k], `${p}.${k}`)]));
  return { health: report(r.health, p), signals: report(r.signals, p), facts: report(r.facts, p),
    checks: report(r.checks, p), archiveDir: text(r.archiveDir, p), scopes, words };
}
export function parseOutcome(v: unknown): Outcome {
  const r = object(v, 'Data Act', ['ok', 'error', 'summary']);
  if (typeof r.ok !== 'boolean') throw new Error('Data Act: invalid result');
  const result = { ok: r.ok, error: text(r.error, 'Data Act'), summary: text(r.summary, 'Data Act') };
  if (!result.ok && !result.error) throw new Error('Data Act: missing failure reason');
  return result;
}
export type DataProvider = {
  read: () => Promise<Sources>; crosswalk: () => Promise<Report>; params: () => Promise<main.ParamsResult>;
  refresh: (scope: string) => Promise<Outcome>; folder: (path: string) => Promise<Outcome>;
  parameter: (key: string, position: string, value: number | null) => Promise<void>;
};
function desktop() {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Data and sources needs the desktop app. No sample data is substituted.');
}
export const dataProvider: DataProvider = {
  read: async () => { desktop(); return parseSources(await TargetDataSources()); },
  crosswalk: async () => { desktop(); return report(await TargetCrosswalk(), 'Crosswalk'); },
  params: async () => { desktop(); const p = await GetParams(); if (!p.ok) throw new Error(p.error);
    return p; },
  refresh: async scope => { desktop(); return parseOutcome(await TargetDataRefresh(scope)); },
  folder: async path => { desktop(); return parseOutcome(await SetLeagueArchiveDir(path)); },
  parameter: async (key, position, value) => {
    desktop(); const r = value === null ? await ResetParam(key, position) : await SetParam(key,
      position, value);
    if (!r.ok) throw new Error(r.error);
  },
};
