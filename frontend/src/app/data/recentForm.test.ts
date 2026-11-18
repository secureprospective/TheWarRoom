import { expect, it } from 'vitest';
import { parsePulseResults, recentForm, type PulseResults } from './nativePulse';

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

it('omits unresolved or unrecognized results and rejects an empty scope or limit', () => {
  const results = [week(1, 'Final', 'unknown'), week(2, 'No result yet', 'mine'),
    week(3, 'Final', 'mine')];
  expect(recentForm(results, 'mine', 2026)).toEqual([{ week: 3, result: 'W' }]);
  expect(recentForm(results, '', 2026)).toEqual([]);
  expect(recentForm(results, 'mine', 2026, 0)).toEqual([]);
});
