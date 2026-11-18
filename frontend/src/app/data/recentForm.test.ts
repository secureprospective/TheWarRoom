import { expect, it } from 'vitest';
import { parsePulseResults, recentForm, recentPlayerForm, type PulseResults } from './nativePulse';

function week(week: number, status: string, winner?: string, options: {
  season?: number; published?: boolean; revision?: number; bye?: string;
} = {}): PulseResults {
  return parsePulseResults({ league: 'test', season: options.season ?? 2026, week,
    revision: options.revision ?? 1, published: options.published ?? true, words: `Week ${week}`,
    games: status ? [{ week, home: 'mine', away: 'other', homeName: 'Mine', awayName: 'Other',
      status, winner, words: '', why: [] }] : [], missed: [], scores: [], bye: options.bye }, 'test', week);
}

it('derives the last five published wins, losses, ties and byes in week order', () => {
  const results = [week(8, 'Final', 'mine', { revision: 2 }), week(3, 'No result yet'),
    week(1, 'Final', 'mine'), week(7, 'Final', 'mine', { published: false }),
    week(5, 'Final', 'other'), week(4, '', undefined, { bye: 'mine' }),
    week(2, 'Tie'), week(6, 'Final', 'other'), week(6, 'Final', 'mine', { season: 2025 }),
    week(9, 'Final', 'mine'), week(8, 'Final', 'other', { revision: 1 })];
  expect(recentForm(results, 'mine', 2026)).toEqual([
    { week: 4, result: 'B' }, { week: 5, result: 'L' }, { week: 6, result: 'L' },
    { week: 8, result: 'W' }, { week: 9, result: 'W' },
  ]);
});

it('counts a final without a winner as a tie; omits unresolved results and rejects empty scope', () => {
  const results = [week(1, 'Final', 'unknown'), week(2, 'No result yet', 'mine'),
    week(3, 'Final', 'mine'), week(4, 'Final')];
  expect(recentForm(results, 'mine', 2026)).toEqual([{ week: 3, result: 'W' }, { week: 4, result: 'T' }]);
  expect(recentForm(results, '', 2026)).toEqual([]);
  expect(recentForm(results, 'mine', 2026, 0)).toEqual([]);
});

it('derives player points only for started weeks and uses the highest published revision', () => {
  const withStarter = (weekNum: number, revision: number, points: number): PulseResults => ({
    ...week(weekNum, 'Final', 'mine', { revision }),
    scores: [{ franchise: 'mine', name: 'Mine', status: 'Final', points: 0, words: '',
      starters: [{ player: 'player-1', name: 'Player', words: '', points }], adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: true }],
  });
  const notStarted = { ...week(4, 'Final'), scores: [{ franchise: 'mine', name: 'Mine', status: 'Final',
    words: '', starters: [], adjustment: 0, lineupStatus: '', lineupWords: '', flagged: false,
    recorded: true }] };
  expect(recentPlayerForm([withStarter(1, 1, 5), withStarter(1, 2, 8), notStarted,
    withStarter(3, 1, 0)], 'player-1')).toEqual([{ week: 1, points: 8 }, { week: 3, points: 0 }]);
});
