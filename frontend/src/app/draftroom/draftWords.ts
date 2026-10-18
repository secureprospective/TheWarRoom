import { localTime } from '../clock/localTime';
import type { Checkpoint, Draft, Pick, Player, Terms } from './draft';

const dollars = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD',
  minimumFractionDigits: 0 });
const exact = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });

// Salaries travel as whole cents; a salary with odd cents shows them.
export const money = (cents: number) => (cents % 100 === 0 ? dollars : exact).format(cents / 100);
const plural = (n: number, one: string) => `${n} ${one}${n === 1 ? '' : 's'}`;
export const contractWords = (t: Terms) => `${money(t.salaryCents)} for ${plural(t.years, 'year')}` +
  (t.rfaAfter ? ', then restricted free agency' : '');

// Time left on a pick clock, from the two biggest units that matter at that size.
export function countdownWords(ms: number): string {
  const seconds = Math.ceil(ms / 1000);
  if (seconds <= 0) return 'Time is up';
  const [h, m, s] = [Math.floor(seconds / 3600), Math.floor((seconds % 3600) / 60), seconds % 60];
  if (h > 0) return m > 0 ? `${plural(h, 'hour')} ${plural(m, 'minute')}` : plural(h, 'hour');
  if (m > 0) return s > 0 ? `${plural(m, 'minute')} ${plural(s, 'second')}` : plural(m, 'minute');
  return plural(s, 'second');
}
// A moment in the viewer's own time zone, never a fixed one.
export const when = (iso: string) => localTime(iso);
export const nanosToMs = (ns: number) => ns / 1e6;

export const franchiseName = (draft: Draft, id: string) =>
  draft.owners.find(o => o.id === id)?.name ?? id;
export const playerName = (draft: Draft, id: string) => draft.players[id]?.name ?? `Player ${id}`;
export const playerLine = (p: Player) => [p.name, [p.position, p.team].filter(Boolean).join(' · ')]
  .filter(Boolean).join(', ');

const sameSlot = (a: { round: number; original: string }, b: { round: number; original: string }) =>
  a.round === b.round && a.original === b.original;
export const isOnClock = (draft: Draft, pick: Pick) =>
  draft.onClock !== null && pick.extra === 0 && sameSlot(draft.onClock, pick);

const madeBy: Record<string, string> = {
  owner: 'Picked by the franchise', commissioner: 'Chosen by the commissioner',
  inserted: 'Extra pick added by the commissioner',
  overrode: 'Chosen by the commissioner after the clock skipped it',
};

// What happened to a pick, in words a GM reads.
export function pickWords(draft: Draft, pick: Pick): string {
  if (pick.player !== '') {
    return (madeBy[pick.how] ?? 'Picked') + (pick.changed ? ', then changed by the commissioner' : '');
  }
  if (pick.skipped) return pick.how === 'expired' ? 'Skipped: the clock ran out' : 'Skipped';
  if (pick.forfeited) return 'Given up: the franchise is over its pick limit';
  if (isOnClock(draft, pick)) return draft.onClock?.paused ? 'On the clock, paused' : 'On the clock';
  return 'Waiting for its turn';
}
export const checkpointWords = (c: Checkpoint) => `${c.words} (${when(c.at)})`;
