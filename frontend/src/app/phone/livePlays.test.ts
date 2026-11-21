import { describe, expect, it, vi } from 'vitest';
import type { FieldState, Game, Play } from '../now/liveContract';
import { bannerText, driveStart, fieldPlay, fieldSpot, freshPlays, PlayQueue, presentation } from './livePlays';

const at = (possession: string, toEndzone: number, down = 1, distance = 10): FieldState =>
  ({ possession, down, distance, yardLine: 0, toEndzone });
let seq = 0;
const mk = (text: string, start: FieldState | null, end: FieldState | null, score = [0, 0]): Play =>
  ({ id: `p${++seq}`, sequence: seq, text, clock: '8:42', period: 2, at: '2026-11-15T18:00:00Z',
    awayScore: score[0], homeScore: score[1], start, end, participants: [] });
const game = (plays: Play[] = []): Game => ({ id: 'g', kickoff: '2026-11-15T18:00:00Z',
  home: { id: 'H', abbreviation: 'KC', score: 0 }, away: { id: 'A', abbreviation: 'BUF', score: 0 },
  status: 'in', clock: '8:42', period: 2, state: at('H', 57, 2, 7), last: plays.at(-1) ?? null, box: [],
  plays });

describe('fieldSpot', () => {
  it('places the away offense driving right and the home offense driving left', () => {
    expect(fieldSpot(at('A', 70), game())).toMatchObject({ poss: 'L', x: 30 });
    expect(fieldSpot(at('H', 57, 2, 7), game())).toMatchObject({ poss: 'R', x: 57, down: 2, dist: 7 });
  });
  it('returns nothing for an unknown possession or spot', () => {
    expect(fieldSpot(at('X', 50), game())).toBeUndefined();
    expect(fieldSpot(at('A', -1), game())).toBeUndefined();
    expect(fieldSpot(null, game())).toBeUndefined();
  });
});

describe('fieldPlay', () => {
  const g = game();
  it('reads a run and its yards from the field states', () => {
    const p = fieldPlay(mk('I.Pacheco up the middle to KC 48 for 9 yards.', at('H', 57), at('H', 48, 2, 1)), g);
    expect(p).toMatchObject({ type: 'run', gain: 9, dir: -1, poss: 'R', first: false });
    expect(p?.start.x).toBe(57);
  });
  it('marks a first down when a run moves the chains', () => {
    const p = fieldPlay(mk('J.Cook right end to KC 30 for 22 yards.', at('A', 52, 3, 2), at('A', 30)), g);
    expect(p).toMatchObject({ type: 'run', gain: 22, dir: 1, first: true });
  });
  it('reads a completed pass and an incomplete one', () => {
    expect(fieldPlay(mk('P.Mahomes pass short right to T.Kelce to BUF 40 for 17 yards.', at('H', 57),
      at('H', 40)), g)).toMatchObject({ type: 'pass', gain: 17, air: 17, yac: 0 });
    expect(fieldPlay(mk('P.Mahomes pass incomplete deep left to X.Worthy.', at('H', 57, 1),
      at('H', 57, 2)), g)).toMatchObject({ type: 'inc', gain: 0 });
  });
  it('reads a sack loss from the text when the states do not move it', () => {
    expect(fieldPlay(mk('J.Allen sacked at BUF 25 for -8 yards (C.Jones).', at('A', 67), null), g))
      .toMatchObject({ type: 'sack', gain: -8 });
  });
  it('reads an interception as a turnover', () => {
    expect(fieldPlay(mk('J.Allen pass deep middle INTERCEPTED by T.McDuffie at KC 20.', at('A', 45),
      at('H', 80)), g)).toMatchObject({ type: 'int', flip: true });
  });
  it('reads a touchdown run and a touchdown pass with the score', () => {
    expect(fieldPlay(mk('I.Pacheco left end for 12 yards, TOUCHDOWN.', at('H', 12), at('A', 75), [3, 13]), g))
      .toMatchObject({ type: 'td', how: 'run', gain: 12, scoring: 'td', score: { L: 3, R: 13 } });
    expect(fieldPlay(mk('P.Mahomes pass short left to T.Kelce for 8 yards, TOUCHDOWN.', at('H', 8), null), g))
      .toMatchObject({ type: 'td', how: 'pass', gain: 8 });
  });
  it('reads field goals good and no good', () => {
    expect(fieldPlay(mk('H.Butker 44 yard field goal is GOOD, Center-J.Winchester.', at('H', 26, 4), null), g))
      .toMatchObject({ type: 'fg', kick: 44, scoring: 'fg' });
    expect(fieldPlay(mk('T.Bass 52 yard field goal is No Good, Wide Right.', at('A', 34, 4), null), g))
      .toMatchObject({ type: 'fgmiss', kick: 52 });
  });
  it('reads a penalty with the team, the foul and the yards', () => {
    const p = fieldPlay(mk('PENALTY on KC-W.Morris, False Start, 5 yards, enforced at KC 43 - No Play.',
      at('H', 57, 2, 7), at('H', 62, 2, 12)), g);
    expect(p).toMatchObject({ type: 'penalty', gain: -5, pen: { team: 'KC', name: 'False Start', moved: 5 } });
  });
  it('reads a punt', () => {
    expect(fieldPlay(mk('T.Araiza punts 48 yards to BUF 12, Center-J.Winchester.', at('H', 60, 4),
      at('A', 88)), g)).toMatchObject({ type: 'punt', kick: 48, flip: true });
  });
  it('leaves unknown plays and plays without a start spot to the plain card', () => {
    expect(fieldPlay(mk('Timeout #1 by BUF at 08:42.', at('H', 57), at('H', 57)), g)).toBeUndefined();
    expect(fieldPlay(mk('H.Butker kicks 65 yards from KC 35 to end zone, Touchback.', null, null), g))
      .toBeUndefined();
    expect(fieldPlay(mk('I.Pacheco up the middle for 3 yards.', null, at('H', 50)), g)).toBeUndefined();
  });
});

describe('freshPlays', () => {
  it('holds back history on the first read and returns only later plays after', () => {
    const a = mk('one', null, null), b = mk('two', null, null), c = mk('three', null, null);
    expect(freshPlays(undefined, game([a, b]))).toEqual([]);
    expect(freshPlays(b.sequence, game([a, b, c])).map(p => p.id)).toEqual([c.id]);
  });
  it('keeps only the newest few when a backlog arrives', () => {
    const list = Array.from({ length: 8 }, (_, i) => mk(`p${i}`, null, null));
    expect(freshPlays(list[0].sequence, game(list)).map(p => p.text)).toEqual(['p5', 'p6', 'p7']);
  });
});

describe('PlayQueue', () => {
  it('runs one moment at a time, in order', async () => {
    const order: string[] = [];
    let release: () => void = () => {};
    const q = new PlayQueue<string>(async item => {
      order.push(`start ${item}`);
      if (item === 'a') await new Promise<void>(r => { release = r; });
      order.push(`end ${item}`);
    });
    q.push('a'); q.push('b');
    await Promise.resolve();
    expect(order).toEqual(['start a']);
    release();
    await vi.waitFor(() => expect(order).toEqual(['start a', 'end a', 'start b', 'end b']));
  });
  it('drops the oldest waiting moment past its limit and stops on close', async () => {
    const seen: number[] = [];
    let release: () => void = () => {};
    const q = new PlayQueue<number>(async n => { seen.push(n); if (n === 0) await new Promise<void>(r => {
      release = r; }); }, 2);
    [0, 1, 2, 3].forEach(n => q.push(n));
    release();
    await vi.waitFor(() => expect(seen).toEqual([0, 2, 3]));
    q.close(); q.push(9);
    await Promise.resolve();
    expect(seen).toEqual([0, 2, 3]);
  });
});

describe('presentation', () => {
  it('shows the text banner instead of motion when reduced motion is asked for', () => {
    expect(presentation(true)).toBe('banner');
    expect(presentation(false)).toBe('motion');
  });
  it('words the banner the way the mock does', () => {
    const g = game();
    const td = fieldPlay(mk('I.Pacheco left end for 12 yards, TOUCHDOWN.', at('H', 12), null, [3, 13]), g)!;
    expect(bannerText(td, g)).toEqual({ big: 'TOUCHDOWN KC', small: 'BUF 3 · KC 13' });
    const run = fieldPlay(mk('J.Cook right end for 4 yards.', at('A', 70), at('A', 66, 2, 6)), g)!;
    expect(bannerText(run, g, fieldSpot(at('A', 66, 2, 6), g))).toEqual({ big: '4-YARD RUN',
      small: '2ND & 6 at BUF 34' });
  });
});

describe('driveStart', () => {
  it('walks back through the plays run by the side with the ball', () => {
    const g = game([mk('a', at('A', 40), null), mk('b', at('H', 75), null), mk('c', at('H', 66), null)]);
    expect(driveStart(g, fieldSpot(at('H', 57), g)!)).toBe(75);
  });
});
