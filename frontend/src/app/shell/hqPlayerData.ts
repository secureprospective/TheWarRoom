import { array, integer, object, text, timestamp } from '../data/parseValues';
import type { Snapshot } from '../data/contract';
import type { Board, Report } from '../reports/contract';
import type { PlayerReading } from '../research/contract';
import { unavailable } from './hqModel';

export function finite(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new
    Error(`${path}: expected finite number`);
  return value;
}
const nullable = (v: unknown, p: string) => v === null ? undefined : finite(v, p);
export type Usage = { snapShare?: number; targetShare?: number; carryShare?: number; weeksUsed: number;
  opponent: string; allowedPoints?: number; allowedRank?: number; label: string; reason: string };
export function parseUsage(value: unknown): Usage {
  const r = object(value, 'Player usage', ['snapShare', 'targetShare', 'carryShare', 'weeksUsed',
    'opponent', 'allowedPoints', 'allowedRank', 'teams', 'label', 'reason']);
  const share = (v: unknown) => {
    const n = nullable(v, 'Share');
    if (n !== undefined && (n < 0 || n > 1)) throw new Error('Share outside 0–1');
    return n;
  };
  integer(r.teams, 'Teams');
  return { snapShare: share(r.snapShare), targetShare: share(r.targetShare), carryShare: share(r.carryShare),
    weeksUsed: integer(r.weeksUsed, 'Weeks used'), opponent: text(r.opponent, 'Opponent'),
    allowedPoints: nullable(r.allowedPoints, 'Allowed points'),
    allowedRank: r.allowedRank === null ? undefined : integer(r.allowedRank, 'Allowed rank', 1),
    label: text(r.label, 'Matchup'), reason: text(r.reason, 'Usage reason') };
}
export type Arc = { age: number; points: { season: number; age: number; level: number; survival: number }[];
  peakStart: number; peakEnd: number; reason: string };
export function parseArc(value: unknown): Arc {
  const r = object(value, 'Age arc', ['age', 'points', 'peakStart', 'peakEnd', 'paramsFallback']);
  return { age: finite(r.age, 'Age'), peakStart: integer(r.peakStart, 'Peak start'), peakEnd:
    integer(r.peakEnd, 'Peak end'),
    reason: r.paramsFallback === undefined ? '' : text(r.paramsFallback, 'Arc fallback'),
    points: array(r.points, 'Arc points', (v, p) => {
      const row = object(v, p, ['season', 'age', 'level', 'survival']);
      return { season: integer(row.season, p), age: finite(row.age, p), level: finite(row.level, p),
        survival: finite(row.survival, p) };
    }) };
}
export type Injury = { id: string; status: string; details: string; returnSeason?: number;
  returnWeek?: number; returnReason: string };
export function parseInjuries(value: unknown): Injury[] {
  return array(value, 'Injuries', (v, p) => {
    const r = object(v, p, ['id', 'status', 'details', 'expReturn', 'returnSeason', 'returnWeek',
      'returnReason']);
    if (r.expReturn !== undefined) text(r.expReturn, p);
    return { id: text(r.id, p), status: text(r.status, p), details: r.details === undefined ? '' :
      text(r.details, p),
      returnSeason: r.returnSeason === undefined ? undefined : integer(r.returnSeason, p),
      returnWeek: r.returnWeek === undefined ? undefined : integer(r.returnWeek, p),
      returnReason: r.returnReason === undefined ? '' : text(r.returnReason, p) };
  });
}
export type Note = { text: string; at: string; order: number };
export function parseNotes(value: unknown): { player: string; notes: Note[]; count: number } {
  const r = object(value, 'My notes', ['player', 'notes', 'count']);
  const player = text(r.player, 'Notes player');
  const notes = array(r.notes, 'Notes', (v, p) => {
    const row = object(v, p, ['kind', 'entity', 'id', 'text', 'at', 'order']);
    if (row.kind !== 'note' || row.entity !== 'player' || row.id !== player) throw new
      Error('Notes belong to another player');
    return { text: text(row.text, p), at: timestamp(row.at, p), order: integer(row.order, p, 1) };
  });
  const count = integer(r.count, 'Note count');
  if (count !== notes.length) throw new Error('Note count mismatch');
  return { player, notes, count };
}
export function researchFact(reading: PlayerReading | undefined, label: string): string | undefined {
  const fact = reading?.report.rows.find(r => r.cells[0] === label)?.cells[1];
  return !fact || /^(?:not held|None of|Cannot price)/i.test(fact) ? undefined : fact;
}
export function formPoints(reading?: PlayerReading): { week: number; points: number }[] {
  const weekly = reading?.report.rows.flatMap(r => {
    const match = /^Form · Wk (\d+)$/.exec(r.cells[0]);
    return match && /^-?\d+(?:\.\d+)?$/.test(r.cells[1])
      ? [{ week: Number(match[1]), points: Number(r.cells[1]) }] : [];
  }) ?? [];
  if (weekly.length) return weekly;
  const fact = researchFact(reading, 'Form') ?? '';
  return Array.from(fact.matchAll(/W(\d+): (-?\d+(?:\.\d+)?)/g), m => ({ week: Number(m[1]), points:
    Number(m[2]) }));
}
export function projections(snapshot: Snapshot, board?: Board, week?: number): Map<string, number> {
  const result = new Map<string, number>();
  const projection = board?.reports.find(r => r.id === 'M-046');
  if (!projection || !week || !new RegExp(`week ${week}(?:\\D|$)`, 'i').test(projection.title)
    || !['ok', 'stale'].includes(projection.status)) return result;
  // This report exposes names, not IDs. Only unique exact identities are safe to join.
  for (const player of snapshot.players.value) {
    if (!player.name || snapshot.players.value.filter(p => p.name === player.name).length !== 1) continue;
    const rows = projection.rows.filter(r => r.cells[0] === player.name);
    if (rows.length === 1 && /^-?\d+(?:\.\d+)?$/.test(rows[0].cells[1])) result.set(player.id,
      Number(rows[0].cells[1]));
  }
  return result;
}
export function totalProjection(ids: string[], values: Map<string, number>): string {
  return ids.every(id => values.has(id)) ? ids.reduce((sum, id) => sum + values.get(id)!, 0).toFixed(1) :
    unavailable;
}
export function ordinal(rank: number): string {
  const suffix = rank % 100 >= 11 && rank % 100 <= 13 ? 'th' :
    ({ 1: 'st', 2: 'nd', 3: 'rd' } as Record<number, string>)[rank % 10] ?? 'th';
  return `${rank}${suffix}`;
}
export function standingFromReport(report?: Report): { record?: string; standing?: string } {
  if (!report || !['ok', 'stale'].includes(report.status)) return {};
  const row = report.rows.find(r => r.mine);
  if (!row) return {};
  const group = report.rows.filter(r => r.group === row.group);
  return { record: row.cells[report.columns.indexOf('Record')],
    standing: `${ordinal(group.indexOf(row) + 1)}${row.group ? ` ${row.group}` : ''}` };
}

export function byeWeeks(snapshot: Snapshot, reports?: Report[]): Map<string, number> {
  const rows = reports?.find(r => r.id === 'M-154' && ['ok', 'stale'].includes(r.status))?.rows ?? [];
  const result = new Map<string, number>();
  for (const player of snapshot.players.value) {
    if (!player.name || snapshot.players.value.filter(p => p.name === player.name).length !== 1) continue;
    const row = rows.find(r => r.cells[0] === player.name);
    if (row && /^\d+$/.test(row.cells[3])) result.set(player.id, Number(row.cells[3]));
  }
  return result;
}
