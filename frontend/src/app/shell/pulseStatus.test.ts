import { expect, it } from 'vitest';
import type { PulseGame, PulseResults, PulseStarter } from '../data/nativePulse';
import type { Game as LiveNflGame } from '../now/liveContract';
import { matchupPlay, pulseRows, statusLabel } from './pulseModel';

const game = (extra: Partial<PulseGame>): PulseGame => ({ week: 10, home: '0001', away: '0002', homeName: 'A',
  awayName: 'B', status: 'No result yet', words: '', why: [], ...extra });

it('says where an undecided game stands', () => {
  expect(statusLabel(game({}), false)).toBe('WAITING');
  expect(statusLabel(game({}), true)).toBe('TO PLAY');
  expect(statusLabel(game({ awayProvisional: 12.5 }), true)).toBe('IN PLAY');
  expect(statusLabel(game({ status: 'Final', winner: '0001' }), true)).toBe('FINAL');
  expect(statusLabel(game({}), true, { on: 2, left: 3 })).toBe('LIVE');
  expect(statusLabel(game({}), true, { on: 0, left: 4 })).toBe('4 LEFT');
  expect(statusLabel(game({}), true, { on: 0, left: 0 })).toBe('DONE');
});

const nfl = (away: string, home: string, status: 'pre' | 'in' | 'post') =>
  ({ away: { abbreviation: away }, home: { abbreviation: home }, status }) as unknown as LiveNflGame;
const starter = (team: string | undefined, points?: number): PulseStarter =>
  ({ player: `p-${team}-${points}`, name: 'x', words: '', team, points });
const results = (home: PulseStarter[], away: PulseStarter[]): PulseResults => ({
  league: 'l', season: 2026, week: 10, revision: 1, published: true, words: '', missed: [],
  games: [game({})], scores: [
    { franchise: '0001', name: 'A', status: 'Incomplete', words: '', starters: home, adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: false },
    { franchise: '0002', name: 'B', status: 'Incomplete', words: '', starters: away, adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: false }] });

// Sunday 4:40 PM: late games on, the night games to come. Only a matchup with a starter on the
// field is LIVE; the rest say how many starters are left, or that every one is done.
it('is LIVE only while a starter is on the field', () => {
  const board = [nfl('SEA', 'LV', 'in'), nfl('PIT', 'CIN', 'pre'), nfl('NYG', 'DAL', 'post')];
  const on = results([starter('SEA'), starter('PIT')], [starter('NYG', 12)]);
  expect(matchupPlay(on, on.games[0], board)).toEqual({ on: 1, left: 1 });
  expect(pulseRows(on, '0001', [], true, board)[0].status).toBe('LIVE');
  const waiting = results([starter('PIT')], [starter('CIN'), starter('DAL')]);
  expect(pulseRows(waiting, '0001', [], true, board)[0].status).toBe('2 LEFT');
  // A starter whose game is over but not yet settled is neither on the field nor left; a bye neither.
  const done = results([starter('NYG'), starter('BUF')], [starter('DAL', 8)]);
  expect(pulseRows(done, '0001', [], true, board)[0].status).toBe('DONE');
  // An unknown club, or no board, never claims LIVE: it falls back to the scores.
  expect(pulseRows(results([starter(undefined)], []), '0001', [], true, board)[0].status).toBe('TO PLAY');
  expect(pulseRows(on, '0001', [], true, [])[0].status).toBe('TO PLAY');
  expect(pulseRows(on, '0001', [], false, board)[0].status).toBe('WAITING');
});
