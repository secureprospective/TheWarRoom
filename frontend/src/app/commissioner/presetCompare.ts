import { ledgerBindings as bindings } from './api';
import { array, integer, object, text } from '../data/parseValues';
import type { Change } from './forms';

// The Go preset comparison (TargetCommissionerPresetCompare): every setting where the league differs from
// a preset, in plain words. Applying it is TargetCommissionerPresetApply, which records the chosen settings
// as the same rules changes a hand edit sends; a dry run returns those changes for the ordinary preview.
export type PresetDifference = {
  field: string; label: string; section: string; league: string; preset: string; nowOnly: boolean;
};
export type PresetComparison = {
  league: string; preset: string; presetLabel: string; revision: number; summary: string;
  differences: PresetDifference[]; presets: { id: string; label: string }[];
};
export type PresetApply = {
  league: string; preset: string; fields: string[]; reason: string; expected: number; request: string;
  effective: Change['effective']; dryRun?: boolean;
};

export function parsePresetComparison(v: unknown): PresetComparison {
  const r = object(v, 'Preset comparison',
    ['league', 'preset', 'presetLabel', 'revision', 'summary', 'differences', 'presets']);
  return {
    league: text(r.league, 'League'), preset: text(r.preset, 'Preset'),
    presetLabel: text(r.presetLabel, 'Preset name'), revision: integer(r.revision, 'League version'),
    summary: text(r.summary, 'Summary'),
    differences: array(r.differences, 'Differences', (d, p) => {
      const x = object(d, p, ['field', 'label', 'section', 'league', 'preset', 'value', 'nowOnly']);
      return { field: text(x.field, p), label: text(x.label, p), section: text(x.section, p),
        league: text(x.league, p), preset: text(x.preset, p), nowOnly: x.nowOnly === true };
    }),
    presets: array(r.presets, 'Presets', (d, p) => {
      const x = object(d, p, ['id', 'label', 'description', 'suggestedTeams']);
      return { id: text(x.id, p), label: text(x.label, p) };
    }),
  };
}

// The rules changes a preset apply sends, as the bodies the ordinary preview reads.
export function parsePresetChanges(v: unknown): string[] {
  const r = object(v, 'Preset changes', ['changes', 'receipts']);
  return array(r.changes, 'Changes', d => JSON.stringify(d));
}

export const readPresetComparison = async (league: string, preset: string) =>
  parsePresetComparison(await bindings().TargetCommissionerPresetCompare(league, preset));

export const presetChanges = async (apply: PresetApply) =>
  parsePresetChanges(await bindings().TargetCommissionerPresetApply(JSON.stringify({ ...apply, dryRun: true })));

export async function applyPreset(apply: PresetApply): Promise<number> {
  const result = object(await bindings().TargetCommissionerPresetApply(JSON.stringify({ ...apply, dryRun: false })),
    'Preset change', ['changes', 'receipts']);
  return array(result.receipts, 'Receipts', d => d).length;
}
