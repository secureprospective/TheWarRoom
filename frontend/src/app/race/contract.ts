import { array, integer, object, text } from '../data/parseValues';
import { parseBoard, report, type Board, type Report } from '../reports/contract';

export type Choice = { value: string; label: string };
export type Race = {
  standings: Report; board: Board; week: number; weeks: Choice[]; team: string; teams: Choice[];
};
const choice = (v: unknown, p: string): Choice => {
  const c = object(v, p, ['value', 'label']);
  return { value: text(c.value, p), label: text(c.label, p) };
};
export function parseRace(v: unknown): Race {
  const p = 'League race';
  const r = object(v, p, ['standings', 'board', 'week', 'weeks', 'team', 'teams']);
  const race = {
    standings: report(r.standings, `${p}.standings`), board: parseBoard(r.board),
    week: integer(r.week, p), weeks: array(r.weeks, p, choice),
    team: text(r.team, p), teams: array(r.teams, p, choice),
  };
  if (race.standings.id !== 'M-002') throw new Error(`${p}: standings must be M-002`);
  if (race.week !== 0 && !race.weeks.some(w => w.value === String(race.week)))
    throw new Error(`${p}: week outside the choices`);
  if (race.team && !race.teams.some(t => t.value === race.team))
    throw new Error(`${p}: team outside the choices`);
  return race;
}
