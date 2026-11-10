import { expect, it } from 'vitest';
import type { PulseBracket, PulseStandings } from '../data/nativePulse';
import { bracketRounds, seedRows, tieGroups, tieStake, type PulseSeedRow } from './pulseModel';

const seed = (n: number, name: string) => ({ seed: n, franchise: `f.${name}`, name, division: 'East',
  rank: n, kind: '', bye: n === 1, byHand: false });
const bracket = { fields: [{ conference: 'NFC', seeds: [seed(1, 'Dallas'), seed(2, 'Chicago'),
  seed(3, 'Seattle')] },
  { conference: 'AFC', seeds: [seed(1, 'Denver')] }],
shape: [{ id: 'wildcard_2', week: 14, homeSeed: 2, awaySeed: 3 },
  { id: 'division_1', week: 15, homeSeed: 1, awayFrom: 'lowest remaining seed' },
  { id: 'championship', week: 17, homeFrom: 'conference winner',
    awayFrom: 'conference winner' }] } as PulseBracket;

it('lays the bracket out in the league\'s own shape, the final shared between sides', () => {
  const read = bracketRounds(bracket);
  expect(read.final).toEqual({ heading: 'THE FINAL · WK 17', sides: ['NFC champion', 'AFC champion'] });
  expect(read.fields[0].rounds.map(r => r.heading)).toEqual(['WILD CARD · WK 14', 'DIVISIONAL · WK 15']);
  expect(read.fields[0].rounds[0].games[0]).toEqual({ id: 'wildcard_2',
    home: { seed: 2, words: 'Chicago' }, away: { seed: 3, words: 'Seattle' } });
  expect(read.fields[0].rounds[1].games[0].away).toEqual({ words: 'lowest remaining seed' });
  expect(read.fields[1].rounds[0].games[0].home).toEqual({ seed: 2, words: 'seed 2' });
});

it('reads each tie once, with the step that split it, and leaves out the whole-table split', () => {
  const all = 'Win percentage put Dallas ahead of Chicago and Seattle.';
  const tie =
    'Chicago and Seattle were level on win percentage. Points put Chicago (955.0 points) ahead of Seattle.';
  const level =
    'Denver and Miami remain level on the available tie-breakers, '
    + 'so they are listed in the league\'s franchise order.';
  const rows = [['Dallas', [all]], ['Chicago', [all, tie]], ['Seattle', [all, tie]], ['Denver', [level]],
    ['Miami', [level]]] as const;
  const standings =
    { rows: rows.map(([name, why]) => ({ franchise: `f.${name}`, why })) } as unknown as PulseStandings;
  const side = ['Dallas', 'Chicago', 'Seattle', 'Denver', 'Miami'].map(name =>
    ({ franchise: `f.${name}`, name }) as PulseSeedRow);
  expect(tieGroups(standings, side.slice(0, 3)).map(t => [t.step, t.teams.map(x => x.name)]))
    .toEqual([['Points', ['Chicago', 'Seattle']]]);
  expect(tieGroups(standings, side).map(t => [t.step, t.split, t.edge]))
    .toEqual([['Win percentage', true, ''], ['Points', true, '955.0 points'],
      ['Franchise order', false, '']]);
  const [points] = tieGroups(standings, side.slice(0, 3));
  expect(tieStake(points)).toBe('');
  expect(tieStake({ ...points, teams: [{ ...side[1], inField: true }, side[2]] }))
    .toBe('PLAYOFF SEAT ON THE LINE');
  expect(tieStake({ ...points, teams: [{ ...side[1], bye: true }, side[2]] })).toBe('BYE ON THE LINE');
});

it('keeps the other side out of a side\'s bubble', () => {
  const row = (name: string, conference: string) => ({ franchise: `f.${name}`, name, conference, division: '',
    rank: 1, record: '', pointsWords: '', pointsFor: 0, why: [] });
  const standings = { rows: [row('Dallas', 'NFC'), row('Denver', 'AFC'), row('Miami', 'AFC'),
    row('Detroit', 'NFC')] } as unknown as PulseStandings;
  expect(seedRows(standings, bracket, 'AFC', '').map(r => r.name)).toEqual(['Denver', 'Miami']);
});
