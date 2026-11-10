import { expect, it } from 'vitest';
import { parsePulseResults } from './nativePulse';

// A week imported with its final team results carries the recorded verdict on each score.
it('reads a week whose scores carry a recorded result', () => {
  const lineup = { status: 'Valid', words: 'Lineup is valid', problems: [], flagged: false };
  const results = parsePulseResults({ league: 'native.x', season: 2026, week: 8, revision: 7,
    published: true, words: 'Week 8', games: [], missed: [],
    scores: [{ recordedResult: 'W', franchise: 'franchise.1', name: 'Dallas Cowboys',
      status: 'Complete', points: 1917000, words: 'Recorded result', starters: [], adjustment: 0,
      changes: [], lineup }] }, 'native.x', 8);
  expect(results.scores[0].points).toBe(191.7);
});

it('reads a starter\'s stat line in whole units', () => {
  const lineup = { status: 'Valid', words: 'Lineup is valid', problems: [], flagged: false };
  const starter = { player: '0001', name: 'Purdy, Brock', position: 'QB', words: '30.6 points',
    points: 306000, line: { passing_yards: 2850000, sack: 5000 } };
  const read = (line: unknown) => parsePulseResults({ league: 'native.x', season: 2026, week: 9,
    revision: 7, published: false, words: 'Week 9', games: [], missed: [],
    scores: [{ franchise: 'franchise.1', name: 'Dallas Cowboys', status: 'Complete', points: 306000,
      words: '30.6 points', starters: [{ ...starter, line }], adjustment: 0, changes: [], lineup }] },
  'native.x', 9).scores[0].starters[0].line;
  expect(read(starter.line)).toEqual({ passing_yards: 285, sack: 0.5 });
  expect(() => read({ passing_yards: '285' })).toThrow('expected a points value');
});
