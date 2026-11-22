import { expect, it } from 'vitest';
import { boxLine, compactLine } from './statLine';

const qb = { passing_td: 2, pass_completion: 21, pass_attempt: 30, passing_yards: 285,
  rushing_attempt: 4, rushing_yards: 12, interception_thrown: 1 };
const lb = { tackle: 5, assist: 3, sack: 1.5, forced_fumble: 1 };

it('says a starter in the phrases of the position played', () => {
  expect(compactLine(qb, 'QB')).toEqual(['2 TD', '21/30', '285 yds']);
  expect(compactLine({ rushing_attempt: 14, rushing_yards: 72, reception: 3, receiving_yards: 20,
    target: 4 }, 'RB')).toEqual(['14-72 rush', '3-20 rec']);
  expect(compactLine({ reception: 5, receiving_yards: 80, target: 8, receiving_td: 1 }, 'WR'))
    .toEqual(['1 TD', '5-80 rec', '8 tgt']);
  expect(compactLine({ fg_made: 2, fg_attempt: 3, pat_made: 3 }, 'PK')).toEqual(['2/3 FG', '3 XP']);
  expect(compactLine(lb, 'LB')).toEqual(['8 tkl', '1.5 sack']);
});

it('says nothing for a starter with no line and never invents a zero', () => {
  expect(compactLine(undefined, 'QB')).toEqual([]);
  expect(compactLine({ passing_yards: 0, rushing_yards: 0 }, 'QB')).toEqual([]);
  expect(boxLine(undefined, 'WR')).toEqual(['–', '–', '–']);
});

it('lays a starter out for the box score', () => {
  expect(boxLine(qb, 'QB')).toEqual(['21/30, 285 yds', '4 car, 12 yds', '2 TD · 1 INT']);
  expect(boxLine(lb, 'LB')).toEqual(['8 tkl', '1.5 sack', '1 FF']);
});
