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
