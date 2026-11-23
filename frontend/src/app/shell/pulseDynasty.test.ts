import { expect, it } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { PulseOdds, PulseOddsRow } from '../data/nativePulse';
import { Power } from './PulseRace';
import { Power as PowerV3 } from '../v3/PulseRaceV3';

const row = (name: string, power: number, dynastyPower?: number): PulseOddsRow => ({ franchise: `f.${name}`,
  name, rank: power, record: '1-0-0', power, powerScore: 0.5, dynastyPower, expected: 100 });
const odds = (rows: PulseOddsRow[]) => ({ powerWords: 'words', rows, games: [] }) as unknown as PulseOdds;
const reading = (rows: PulseOddsRow[]) => ({ odds: odds(rows), errors: [] }) as never;

it('shows the dynasty rank beside the power rank, and nothing when it is missing', () => {
  const some = renderToStaticMarkup(createElement(Power,
    { reading: reading([row('Dallas', 3, 12), row('Miami', 4)]), mine: '' }));
  expect(some).toContain('DYNASTY');
  expect(some).toContain('#12');
  const none = renderToStaticMarkup(createElement(Power,
    { reading: reading([row('Dallas', 3), row('Miami', 4, 0)]), mine: '' }));
  expect(none).not.toMatch(/dynasty/i);
  expect(none).not.toContain('#0');
});

it('v3 power rows say Dynasty #N only when the rank is held', () => {
  const draw = (rows: PulseOddsRow[]) => renderToStaticMarkup(createElement(PowerV3,
    { odds: odds(rows), mine: '', badge: null, errors: [] }));
  expect(draw([row('Dallas', 3, 12)])).toContain('Dynasty #12');
  expect(draw([row('Dallas', 3), row('Miami', 4, 0)])).not.toMatch(/dynasty|#0/i);
});
