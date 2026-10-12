import { array, integer, object } from '../data/parseValues';
import { parseBoard, report, type Board, type Report } from '../reports/contract';
import { choice, type Choice } from '../race/contract';

export type Now = { week: number; weeks: Choice[]; matchup: Report; board: Board };
export function parseNow(v: unknown): Now {
  const p = 'League Pulse now';
  const r = object(v, p, ['week', 'weeks', 'matchup', 'board']);
  const now = {
    week: integer(r.week, p), weeks: array(r.weeks, p, choice),
    matchup: report(r.matchup, `${p}.matchup`), board: parseBoard(r.board),
  };
  if (now.matchup.id !== 'X-03') throw new Error(`${p}: the matchup must be X-03`);
  if (now.week !== 0 && !now.weeks.some(w => w.value === String(now.week)))
    throw new Error(`${p}: week outside the choices`);
  return now;
}
