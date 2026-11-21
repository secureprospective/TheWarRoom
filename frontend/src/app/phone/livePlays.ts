import type { FieldState, Game, Play } from '../now/liveContract';

/** Side is the mock's field side: L is the away club (defends the left goal), R the home club. */
export type Side = 'L' | 'R';
/** Spot is a field state on the mock's yard axis: x 0 is the left goal line, 100 the right. */
export type Spot = { poss: Side; x: number; down: number; dist: number };
export type PlayType = 'run' | 'pass' | 'inc' | 'int' | 'td' | 'sack' | 'fumble' | 'penalty' | 'fg' | 'fgmiss'
  | 'punt';
export type FieldPlay = { id: string; type: PlayType; how: 'run' | 'pass'; dir: 1 | -1; poss: Side;
  start: Spot; end?: Spot; gain: number; air: number; yac: number; lat1: number; kick: number;
  pen?: { team: string; name: string; moved: number }; scoring?: 'td' | 'fg'; score: { L: number; R: number };
  first: boolean; flip: boolean; q: number; clock: string; text: string; title: string; tag: string };

export function fieldSpot(state: FieldState | null, game: Game): Spot | undefined {
  if (!state || state.toEndzone < 0) return undefined;
  const poss: Side | undefined = state.possession === game.home.id ? 'R'
    : state.possession === game.away.id ? 'L' : undefined;
  if (!poss) return undefined;
  return { poss, x: poss === 'R' ? state.toEndzone : 100 - state.toEndzone, down: state.down,
    dist: state.distance };
}

const RUN = new RegExp(`\\b(up the middle|(left|right) (end|tackle|guard)|scrambles|rushe?s|kneels)\\b`, 'i');

function classify(text: string): PlayType | undefined {
  if (/^PENALTY\b/i.test(text) || /No Play/i.test(text)) return 'penalty';
  if (/field goal/i.test(text)) return /is GOOD/i.test(text) ? 'fg' : 'fgmiss';
  if (/\bpunts\b/i.test(text)) return 'punt';
  if (/\bkicks\b|extra point|two-point|timeout|end of (quarter|half|game)/i.test(text)) return undefined;
  if (/INTERCEPTED/i.test(text)) return 'int';
  if (/TOUCHDOWN/i.test(text)) return 'td';
  if (/FUMBLES/i.test(text)) return 'fumble';
  if (/\bsacked\b/i.test(text)) return 'sack';
  if (/pass incomplete/i.test(text)) return 'inc';
  if (/\bpass\b/i.test(text) && /\bfor\b/i.test(text)) return 'pass';
  if (RUN.test(text) || /\bfor (-?\d+) yards?\b|for no gain/i.test(text)) return 'run';
  return undefined;
}

function textYards(text: string): number | undefined {
  if (/for no gain/i.test(text)) return 0;
  const loss = text.match(/loss of (\d+) yards?/i);
  if (loss) return -Number(loss[1]);
  const m = text.match(/\bfor (-?\d+) yards?\b/i);
  return m ? Number(m[1]) : undefined;
}

const TITLES: Record<PlayType, (gain: number) => string> = {
  run: g => g ? `${Math.abs(g)}-yard ${g < 0 ? 'loss' : 'run'}` : 'Run, no gain',
  pass: g => `${g}-yard catch`, inc: () => 'Incomplete', int: () => 'Intercepted', td: () => 'Touchdown',
  sack: g => `Sack, ${g} yards`, fumble: () => 'Fumble', penalty: () => 'Flag on the play',
  fg: () => 'Field goal', fgmiss: () => 'Field goal no good', punt: () => 'Punt',
};
const TAGS: Partial<Record<PlayType, string>> = { td: 'TOUCHDOWN', fg: 'FIELD GOAL', penalty: 'FLAG',
  int: 'TURNOVER', fumble: 'FUMBLE', fgmiss: 'NO GOOD', sack: 'SACK' };

/** fieldPlay reads one held play into the mock's play shape. Type and yards come only from the play's own
 *  text and start/end field states; anything not read that way returns undefined (the plain play card). */
export function fieldPlay(play: Play, game: Game): FieldPlay | undefined {
  const type = classify(play.text);
  const start = fieldSpot(play.start, game);
  if (!type || !start) return undefined;
  const end = fieldSpot(play.end, game);
  const dir = start.poss === 'L' ? 1 : -1;
  const offense = start.poss === 'L' ? game.away.abbreviation : game.home.abbreviation;
  const pm = play.text.match(/PENALTY on ([A-Z]{2,3})-[^,]*,\s*([^,]+?),\s*(\d+) yards?/i);
  const pen = pm ? { team: pm[1].toUpperCase(), name: pm[2].trim(), moved: Number(pm[3]) } : undefined;
  const toEnd = play.start!.toEndzone;
  const sameSide = end && end.poss === start.poss;
  let gain = type === 'td' ? toEnd
    : type === 'inc' ? 0
      : sameSide && end ? (end.x - start.x) * dir
        : type === 'penalty' && pen ? (pen.team === offense ? -pen.moved : pen.moved)
          : textYards(play.text) ?? 0;
  if (['fg', 'fgmiss', 'punt', 'int'].includes(type)) gain = 0;
  const how = type === 'td' && /\bpass\b/i.test(play.text) ? 'pass' : 'run';
  const kickMatch = play.text.match(/(\d+) yard field goal|punts (\d+) yards?/i);
  const kick = Number(kickMatch?.slice(1).find(Boolean) ?? 0);
  const air = type === 'pass' || (type === 'td' && how === 'pass') ? Math.max(gain, 1)
    : type === 'inc' ? /\bdeep\b/i.test(play.text) ? 20 : /\bshort\b/i.test(play.text) ? 7 : 10
      : type === 'int' && end ? Math.min(50, Math.max(5, Math.abs(end.x - start.x))) : 0;
  const flip = ['td', 'fg', 'fgmiss', 'punt', 'int'].includes(type) || (!!end && end.poss !== start.poss);
  const first = !flip && !!end && end.down === 1 && start.down > 0
    && ['run', 'pass', 'penalty'].includes(type);
  const scoring = type === 'td' ? 'td' : type === 'fg' ? 'fg' : undefined;
  return { id: play.id, type, how, dir, poss: start.poss, start, end, gain, air, yac: 0,
    lat1: /\bleft\b/i.test(play.text) ? -0.3 : /\bright\b/i.test(play.text) ? 0.3 : 0, kick, pen, scoring,
    score: { L: play.awayScore, R: play.homeScore }, first, flip, q: play.period, clock: play.clock,
    text: play.text, title: TITLES[type](gain), tag: first ? '1ST DOWN' : TAGS[type] ?? '' };
}

/** freshPlays: the plays held since the last one shown, newest few only. The first read shows none, so
 *  opening the page never replays history. */
export function freshPlays(lastSeen: number | undefined, game: Game, keep = 3): Play[] {
  if (lastSeen === undefined) return [];
  const all = [...game.plays];
  if (game.last && !all.some(p => p.id === game.last!.id)) all.push(game.last);
  return all.filter(p => p.sequence > lastSeen).sort((a, b) => a.sequence - b.sequence).slice(-keep);
}

export function latestSequence(game: Game): number {
  return Math.max(game.last?.sequence ?? -1, ...game.plays.map(p => p.sequence), -1);
}

/** PlayQueue runs one moment at a time; past its limit the oldest waiting moment is dropped. */
export class PlayQueue<T> {
  private waiting: T[] = [];
  private running = false;
  private closed = false;
  constructor(private readonly run: (item: T) => Promise<void>, private readonly limit = 3) {}
  push(item: T) {
    if (this.closed) return;
    this.waiting.push(item);
    while (this.waiting.length > this.limit) this.waiting.shift();
    void this.drain();
  }
  close() { this.closed = true; this.waiting = []; }
  private async drain() {
    if (this.running) return;
    this.running = true;
    try {
      for (let next = this.waiting.shift(); next !== undefined && !this.closed; next = this.waiting.shift()) {
        try { await this.run(next); } catch { /* one failed moment must not stop the rest */ }
      }
    } finally { this.running = false; }
  }
}

/** presentation: reduced motion gets the mock's plain text banner instead of field motion. */
export function presentation(reduced: boolean): 'banner' | 'motion' { return reduced ? 'banner' : 'motion'; }

/** driveStart: where the current drive began, from the newest plays run by the side with the ball. */
export function driveStart(game: Game, spot: Spot): number {
  let x = spot.x;
  for (const p of [...game.plays].sort((a, b) => b.sequence - a.sequence)) {
    const s = fieldSpot(p.start, game);
    if (!s || s.poss !== spot.poss) break;
    x = s.x;
  }
  return x;
}

const ORD = ['1ST', '2ND', '3RD', '4TH'];
export function spotWords(x: number, game: Game): string {
  const r = Math.round(x);
  return r === 50 ? '50' : r < 50 ? `${game.away.abbreviation} ${r}` : `${game.home.abbreviation} ${100 - r}`;
}
export function downWords(s: Spot): string {
  if (s.down < 1) return '';
  const goal = (s.poss === 'L' ? 100 - s.x : s.x) <= s.dist;
  return `${ORD[s.down - 1]} & ${goal ? 'GOAL' : s.dist}`;
}

/** bannerText is the mock's reduced-motion banner: the moment in words and where the ball now is. */
export function bannerText(p: FieldPlay, game: Game, after?: Spot): { big: string; small: string } {
  const ab = (s: Side) => s === 'L' ? game.away.abbreviation : game.home.abbreviation;
  const big = p.scoring === 'td' ? `TOUCHDOWN ${ab(p.poss)}`
    : p.scoring === 'fg' ? `FIELD GOAL ${ab(p.poss)} · ${p.kick} YDS` : p.title.toUpperCase();
  const small = p.scoring ? `${ab('L')} ${p.score.L} · ${ab('R')} ${p.score.R}`
    : !after ? '' : p.flip ? `${ab(after.poss)} ball at ${spotWords(after.x, game)}`
      : `${downWords(after)} at ${spotWords(after.x, game)}`;
  return { big, small };
}
