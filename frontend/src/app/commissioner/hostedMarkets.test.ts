import { expect, it } from 'vitest';
import { parseHostedResults, parseHostedStandings } from './hostedMarkets';

// The league server sends its live-score evidence on results and standings, and provisional points on a game
// being played. HQ reads both in its one refresh; a refused field there leaves the lineup read-only.
const evidence = { sources: ['espn'], reduced: false, stale: false, live: true, season: 2026, week: 10,
  asOf: '2026-11-15T18:00:00Z', source: 'ESPN live', provisional: true };

it('reads live results with score evidence and provisional points', () => {
  const read = parseHostedResults({ league: 'l', season: 2026, week: 10, revision: 4, published: true,
    words: 'Week 10', missed: [], evidence, games: [{ week: 10, home: 'a', away: 'b', homeName: 'A',
      awayName: 'B', status: 'live', homeProvisional: 41.5, awayProvisional: 38.2, words: 'A leads',
      why: [] }] });
  expect(read.games).toEqual([{ names: 'A v B', words: 'A leads', why: [] }]);
});

it('reads standings that carry score evidence', () => {
  const read = parseHostedStandings({ league: 'l', name: 'L', season: 2026, revision: 4, status: 'live',
    words: 'Through week 9', through: 9, decided: 144, pending: 0, criteria: [], noData: [], evidence,
    computed: [{ rank: 1, franchise: 'a', name: 'A', wins: 9, losses: 0, ties: 0, record: '9-0',
      pointsFor: 1000, pointsWords: '1000.0', why: [] }] });
  expect(read.computed).toEqual([{ rank: 1, franchise: 'a', name: 'A', record: '9-0',
    points: '1000.0', why: [] }]);
});
