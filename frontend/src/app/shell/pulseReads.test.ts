import { expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import { parsePulseBracket, parsePulseDirectory, parsePulseLineup, parsePulseResults,
  parsePulseStandings, parsePulseWeek, pointsLabel } from '../data/nativePulse';
import { readNativePulse, readPulseMatchup, readPulseWeekResults } from './pulseReads';
import { applyFilter, filterCounts, groupRows, pulseRows, seedRows, starterRows, statusLabel }
  from './pulseModel';
import pulse from './pulse.wire.testdata.json';

const LEAGUE = 'native.test';
const port = (results: unknown = pulse.results): HostedPort => vi.fn(async request => {
  const franchise = 'franchise' in request.arguments ? String(request.arguments.franchise) : '';
  const lineups = pulse.lineups as Record<string, unknown>;
  const data: Record<string, unknown> = {
    'native.directory.read': pulse.directory, 'week.read': pulse.week,
    'standings.read': pulse.standings, 'bracket.read': pulse.bracket, 'results.read': results,
    'lineup.read': lineups[franchise],
    'projections.read': { ...pulse.projections, players: pulse.projections.players.filter(p =>
      String((request.arguments as { players?: string }).players ?? '').split(',').includes(p.player)) },
  };
  if (data[request.command] === undefined) throw new Error(`No fixture: ${request.command}`);
  return { label: 'Fixture', writable: false, football: '', data: data[request.command] };
});

it('reads the league, the week, the standings and the playoff field through the one port', async () => {
  const reading = await readNativePulse(LEAGUE, 'franchise.01', port());
  expect(reading.errors).toEqual([]);
  expect(reading.name).toBe('Held league');
  expect(reading.season).toBe(2026);
  expect(reading.seats).toHaveLength(32);
  expect(reading.week).toBe(2);
  expect(reading.lastWeek).toBe(14);
  expect(reading.standings?.rows[0].record).toBe('2-1');
  expect(reading.bracket?.fields.map(f => f.conference)).toEqual(['East', 'West']);
  expect(reading.bracket?.perConference).toBe(7);
});

it('scores one week and lines both lineups up with their teams and projections', async () => {
  const results = await readPulseWeekResults(LEAGUE, 2026, 2, port());
  expect(results.published).toBe(true);
  expect(results.games).toHaveLength(16);
  const mine = results.games.find(g => g.away === 'franchise.32')!;
  expect(mine.awayPoints).toBe(3);
  expect(mine.homePoints).toBe(1.5);
  expect(statusLabel(mine, true)).toBe('FINAL');
  const score = results.scores.find(s => s.franchise === 'franchise.01')!;
  const matchup = await readPulseMatchup(LEAGUE, 2026, 2, ['franchise.01', 'franchise.32'],
    { 'franchise.01': score.starters.map(s => s.player) }, port());
  expect(matchup.errors).toEqual([]);
  const rows = starterRows(score, matchup.lineups['franchise.01'], matchup.projections);
  expect(rows).toEqual([{ slot: 'LB', player: '1001', name: 'franchise.01 starter', team: 'AAA',
    locked: true, points: 1.5, projection: 9.5, words: '1.5 points' }]);
  // Another franchise's lineup is private, so its starters keep their points and lose the rest.
  const theirs = starterRows(results.scores.find(s => s.franchise === 'franchise.32'), undefined,
    matchup.projections);
  expect(theirs[0].team).toBeUndefined();
  expect(theirs[0].locked).toBeUndefined();
  expect(theirs[0].points).toBe(3);
});

it('says the schedule is unpublished rather than inventing a week', async () => {
  const results = await readPulseWeekResults(LEAGUE, 2026, 2, port(pulse.unpublished));
  expect(results.published).toBe(false);
  expect(results.games).toEqual([]);
  expect(results.words).toContain('not published yet');
});

it('refuses a reading that belongs to another league, season, week or franchise', () => {
  const wrong = (value: object, patch: object) => ({ ...value, ...patch });
  expect(() => parsePulseResults(wrong(pulse.results, { league: 'other' }), LEAGUE, 2)).toThrow();
  expect(() => parsePulseResults(wrong(pulse.results, { week: 3 }), LEAGUE, 2)).toThrow();
  expect(() => parsePulseResults(wrong(pulse.results, { published: 'yes' }), LEAGUE, 2)).toThrow();
  expect(() => parsePulseStandings(wrong(pulse.standings, { season: 2025 }), LEAGUE, 2026)).toThrow();
  expect(() => parsePulseBracket(wrong(pulse.bracket, { league: 'other' }), LEAGUE, 2026)).toThrow();
  expect(() => parsePulseDirectory(wrong(pulse.directory, { league: 'other' }), LEAGUE)).toThrow();
  expect(() => parsePulseWeek(pulse.week, LEAGUE, 'franchise.02', 2026)).toThrow();
  expect(() => parsePulseLineup(pulse.lineups['franchise.01'], LEAGUE, 'franchise.01', 3)).toThrow();
  expect(() => parsePulseResults({ ...pulse.results, surprise: 1 }, LEAGUE, 2)).toThrow();
});

it('counts and filters the week the way the strip and the filter bar report it', () => {
  const results = parsePulseResults(pulse.results, LEAGUE, 2);
  const rows = pulseRows(results, 'franchise.01', parsePulseDirectory(pulse.directory, LEAGUE).seats,
    true);
  expect(filterCounts(rows)).toEqual({ all: 16, mine: 1, live: 0, close: 16 });
  expect(applyFilter(rows, 'mine').map(r => r.key)).toEqual(['franchise.32-franchise.01']);
  expect(groupRows(rows)[0].heading).toBe('MY MATCHUP');
  expect(groupRows(rows).slice(1).map(g => g.heading))
    .toEqual(['DIVISION.1 · 4', 'DIVISION.2 · 4', 'DIVISION.3 · 4', 'DIVISION.4 · 3']);
  const undecided = { ...results, games: results.games.map(g => ({ ...g, status: 'No result yet',
    homePoints: undefined, awayPoints: undefined })) };
  const open = pulseRows(undecided, 'franchise.01', [], true);
  expect(filterCounts(open)).toEqual({ all: 16, mine: 1, live: 16, close: 0 });
  expect(pulseRows(undecided, 'franchise.01', [], false)[0].status).toBe('WAITING');
});

it('seeds the Race field from the playoff read and puts the rest on the bubble', () => {
  const standings = parsePulseStandings(pulse.standings, LEAGUE, 2026);
  const bracket = parsePulseBracket(pulse.bracket, LEAGUE, 2026);
  const rows = seedRows(standings, bracket, 'East', 'franchise.01');
  expect(rows.filter(r => r.inField)).toHaveLength(7);
  expect(rows[0].seed).toBe(1);
  expect(rows[0].bye).toBe(true);
  expect(rows[0].record).not.toBe('');
  expect(rows.filter(r => !r.inField)[0].seed).toBeUndefined();
  expect(seedRows(standings, undefined, 'East', 'franchise.01').every(r => !r.inField)).toBe(true);
});

it('writes a score to one decimal and an absent score as a dash', () => {
  expect(pointsLabel(1.5)).toBe('1.5');
  expect(pointsLabel(0)).toBe('0.0');
  expect(pointsLabel(undefined)).toBe('–');
});
