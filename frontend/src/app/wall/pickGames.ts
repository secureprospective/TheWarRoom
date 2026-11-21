import type { Game } from '../now/liveContract';
import { chooseWallLayout } from './layout';

/** How far ahead of kickoff a not-yet-started game may fill an empty phone. */
export const preWindowMs = 12 * 3_600_000;
const kick = (g: Game) => Date.parse(g.kickoff);

/** wallCount reads the ?n= override: a whole number 1-9, else the default six. */
export function wallCount(search: string): number {
  const n = Number(new URLSearchParams(search).get('n'));
  return Number.isInteger(n) && n >= 1 && n <= 9 ? n : 6;
}

/** pickGames chooses the wall's games: live first (earliest kickoff first), then games about to start
 *  (soonest first, only within preWindowMs) while there is room, never more than `cap`. */
export function pickGames(games: Game[], cap: number, now: number): Game[] {
  const byKick = (a: Game, b: Game) => (kick(a) || 0) - (kick(b) || 0);
  const live = games.filter(g => g.status === 'in').sort(byKick);
  const soon = games.filter(g => g.status === 'pre' && kick(g) >= now - 3_600_000
    && kick(g) - now <= preWindowMs).sort(byKick);
  return [...live, ...soon].slice(0, cap);
}

/** nextKickoff is the earliest future kickoff on the board, in Central time, or '' when none is held. */
export function nextKickoff(games: Game[], now: number): string {
  const next = games.filter(g => g.status === 'pre' && kick(g) > now).sort((a, b) => kick(a) - kick(b))[0];
  if (!next) return '';
  return `Next kickoff ${new Date(kick(next)).toLocaleString('en-US', { weekday: 'short', month: 'short',
    day: 'numeric', hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' })} CT`;
}

/** wallGrid chooses cols, rows and scale for n phones inside viewport w x h (defaults 1920x1080). */
export function wallGrid(n: number, w = 1920, h = 1080) {
  return chooseWallLayout(n, w, h);
}
