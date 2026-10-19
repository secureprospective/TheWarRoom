import { unscale } from './ruleDraft';
import type { ActDefinition, Field } from './actTable';
import type { Change } from './forms';
import type { ActReadings } from './actReadings';

export type ActValues = Record<string, string | boolean | string[]>;
function converted(field: Field, raw: ActValues[string]): unknown {
  if (field.kind === 'boolean') return raw === true;
  if (field.kind === 'order' || field.kind === 'starters') return raw;
  const value = String(raw ?? '').trim();
  if (['pick', 'scope', 'shape', 'scheduleGame'].includes(field.kind))
    return JSON.parse(value) as object;
  if (field.kind === 'instant') return new Date(value).toISOString();
  if (['number', 'money', 'decimal', 'bracketSlot'].includes(field.kind)) {
    return unscale(value, field.kind === 'money' ? 2 : field.kind === 'decimal' ? 4 : 0);
  }
  return value;
}
function missing(field: Field, raw: ActValues[string]): boolean {
  if (field.optional && (raw === undefined || raw === '')) return false;
  if (field.kind === 'boolean') return false;
  if (Array.isArray(raw)) return raw.length === 0 || raw.some(v => !v) || new Set(raw).size !== raw.length;
  if (raw === undefined || String(raw).trim() === '') return true;
  if (field.kind === 'instant') return !Number.isFinite(new Date(String(raw)).getTime());
  if (['number', 'money', 'decimal', 'bracketSlot'].includes(field.kind)) {
    return unscale(String(raw), field.kind === 'money' ? 2 : field.kind === 'decimal' ? 4 : 0) === null;
  }
  return false;
}
export function actChange(def: ActDefinition, values: ActValues, held: ActReadings, reason: string,
  dryRun = false): Change | null {
  const filled = { ...values };
  for (const field of def.fields) if (field.kind === 'self') filled[field.path] = held.principal ?? '';
  if (def.fields.some(field => missing(field, filled[field.path]))) return null;
  const args: Record<string, unknown> = {};
  for (const field of def.fields) {
    if (field.optional && !filled[field.path]) continue;
    const value = converted(field, filled[field.path]);
    const [head, tail] = field.path.split('.');
    if (tail) args[head] = { ...(args[head] as object | undefined), [tail]: value };
    else args[head] = value;
  }
  if (def.command === 'commish.abilities.set' && args.franchise === '*') {
    delete args.franchise;
    args.all = true;
  }
  if (def.command === 'schedule.edit') {
    const game = args.game as { week: number; id: string; published: boolean };
    args.week = game.week;
    args.published = game.published;
    args.games = [{ id: game.id, home: args.home, away: args.away }];
    delete args.game;
    delete args.home;
    delete args.away;
  }
  if (def.command === 'season.rollover') args.dry_run = dryRun;
  return { league: held.league, command: def.command, args: def.root ? { [def.root]: args } : args,
    reason: reason.trim(), expected: held.revision, effective: 'now',
    visibility: def.private ? 'private' : 'league' };
}
