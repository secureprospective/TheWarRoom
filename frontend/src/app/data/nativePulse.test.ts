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

it('reads how the final stats moved a starter, and which totals are recorded ones', () => {
  const lineup = { status: 'Valid', words: 'Lineup is valid', problems: [], flagged: false };
  const starter = { player: '13589', name: 'Gibbs, Jahmyr', position: 'RB', words: '19.4 points',
    points: 194000,
    revision: { was: 188000, now: 194000,
      stats: [{ stat: 'rushing_yards', label: 'Rushing yards', was: 880000, now: 940000 }] } };
  const score = (extra: object) => parsePulseResults({ league: 'native.x', season: 2026, week: 9,
    revision: 1, published: true, words: '', games: [], missed: [],
    scores: [{ franchise: '0001', name: 'A', status: 'Complete', points: 194000, words: '19.4 points',
      starters: [starter], adjustment: 0, changes: [], lineup, ...extra }] }, 'native.x', 9).scores[0];
  expect(score({}).starters[0].revision).toEqual({ was: 18.8, now: 19.4,
    stats: [{ stat: 'rushing_yards', label: 'Rushing yards', was: 88, now: 94 }] });
  expect(score({}).recorded).toBe(false);
  expect(score({ recordedResult: 'W' }).recorded).toBe(true);
});
