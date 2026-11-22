import { expect, it } from 'vitest';
import { parsePulseActivity, parsePulseFranchiseSchedule, parsePulseOdds, parsePulseResults,
  parsePulseStandings } from
  './nativePulse';
import { scoreSources } from '../shell/NativeLeaguePulse';

it('accepts only the member-safe activity projection for the requested league', () => {
  const value = { league: 'native.x', revision: 3, events: [{ id: 'notice.1', kind: 'notice',
    when: '2026-11-18T12:00:00Z', summary: 'Trade deadline is Saturday.',
    franchises: ['franchise.1'] }] };
  expect(parsePulseActivity(value, 'native.x').events[0]).toEqual({ ...value.events[0] });
  expect(() => parsePulseActivity({ ...value, league: 'native.other' }, 'native.x'))
    .toThrow('another league');
  expect(() => parsePulseActivity({ ...value, events: [{ ...value.events[0], kind: 'bid' }] }, 'native.x'))
    .toThrow('expected notice | trade | roster');
  const privateField = { ...value.events[0], reason: 'private' };
  expect(() => parsePulseActivity({ ...value, events: [privateField] }, 'native.x'))
    .toThrow('unexpected field');
  const invalidTime = { ...value.events[0], when: 'tomorrow' };
  expect(() => parsePulseActivity({ ...value, events: [invalidTime] }, 'native.x'))
    .toThrow('expected RFC3339 timestamp');
});

it('parses a franchise schedule only for its requested league, team and season', () => {
  const schedule = parsePulseFranchiseSchedule({ league: 'native.x', franchise: 'franchise.1', season: 2026,
    revision: 7, message: 'Published', games: [{ week: 8, game: 'g1', opponent: 'franchise.2',
      venue: 'Away', kickoff: '2026-11-15T18:00:00Z' }, { week: 9, venue: 'Bye' }] }, 'native.x',
  'franchise.1', 2026);
  expect(schedule.games).toEqual([{ week: 8, opponent: 'franchise.2', venue: 'Away',
    kickoff: '2026-11-15T18:00:00Z' }, { week: 9, opponent: '', venue: 'Bye', kickoff: undefined }]);
  expect(() => parsePulseFranchiseSchedule({ league: 'native.x', franchise: 'other', season: 2026,
    revision: 7, message: '', games: [] }, 'native.x', 'franchise.1', 2026))
    .toThrow('another franchise or season');
});

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

// The server labels live scores with the evidence it consulted (#246); Pulse reads it, never refuses it.
it('reads the live-score evidence on results, standings and odds', () => {
  const evidence = { reduced: false, stale: false, live: 'g12', season: 2026, week: 10,
    asOf: '2026-11-15T18:00:00Z', source: 'ESPN', provisional: true,
    sources: { espn: '2026-11-15T18:00:00Z' } };
  const results = parsePulseResults({ league: 'native.x', season: 2026, week: 10, revision: 1,
    published: true, words: '', games: [], missed: [], scores: [], evidence }, 'native.x', 10);
  expect(results.evidence?.source).toBe('ESPN');
  expect(scoreSources(results)).toBe('ESPN · provisional');
  expect(scoreSources({ ...results, evidence: { ...results.evidence!, stale: true } }))
    .toBe('ESPN · held · provisional');
  expect(() => parsePulseResults({ league: 'native.x', season: 2026, week: 10, revision: 1,
    published: true, words: '', games: [], missed: [], scores: [], evidence: { ...evidence, stale: 'no' } },
  'native.x', 10)).toThrow('expected true or false');
  expect(parsePulseStandings({ league: 'native.x', name: 'L', season: 2026, revision: 1, status: '',
    words: '', through: 9, decided: 0, pending: 0, criteria: [], noData: [], computed: [], evidence },
    'native.x', 2026).season).toBe(2026);
  expect(parsePulseOdds({ league: 'native.x', name: 'L', season: 2026, revision: 1, status: '', words: '',
    powerWords: '', gameWords: '', oddsWords: '', week: 10, runs: 0, seed: 0, rows: [], games: [],
    evidence },
  'native.x', 2026).week).toBe(10);
});

it('reads a standings streak from the held standings row without filling an absent streak', () => {
  const row = { rank: 1, franchise: 'franchise.1', name: 'Alpha', wins: 4, losses: 2, ties: 0,
    record: '4-2-0', streak: 'W3', pointsFor: 1235000, pointsWords: '123.5 points scored', why: [] };
  const base = { league: 'native.x', name: 'L', season: 2026, revision: 1, status: 'Ranked',
    words: 'Standings through week 6', through: 6, decided: 6, pending: 0, criteria: [], noData: [] };
  expect(parsePulseStandings({ ...base, computed: [row] }, 'native.x', 2026).rows[0].streak).toBe('W3');
  expect(parsePulseStandings({ ...base, computed: [{ ...row, streak: undefined }] }, 'native.x', 2026)
    .rows[0].streak).toBeUndefined();
  expect(() => parsePulseStandings({ ...base, computed: [{ ...row, streak: '0' }] }, 'native.x', 2026))
    .toThrow('invalid current streak');
});

it('preserves a uint64 odds seed as an exact decimal string', () => {
  const seed = '18446744073709551615';
  const odds = parsePulseOdds({ league: 'native.x', name: 'L', season: 2026, revision: 1, status: '',
    words: '', powerWords: '', gameWords: '', oddsWords: '', week: 10, runs: 0, seed, rows: [], games: [] },
  'native.x', 2026);
  expect(odds.seed).toBe(seed);
});
