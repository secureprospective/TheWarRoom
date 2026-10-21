import { array, integer, object, text } from '../data/parseValues';
import { parseBoard, type Board } from '../reports/contract';
import { choice, type Choice } from '../race/contract';

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
