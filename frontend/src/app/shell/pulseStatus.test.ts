import { expect, it } from 'vitest';
import type { PulseGame } from '../data/nativePulse';
import { statusLabel } from './pulseModel';

const game = (extra: Partial<PulseGame>): PulseGame => ({ week: 10, home: '0001', away: '0002', homeName: 'A',
  awayName: 'B', status: 'No result yet', words: '', why: [], ...extra });

it('says where an undecided game stands', () => {
  expect(statusLabel(game({}), false)).toBe('WAITING');
  expect(statusLabel(game({}), true, true)).toBe('LIVE');
  expect(statusLabel(game({}), true, false)).toBe('TO PLAY');
  expect(statusLabel(game({ awayProvisional: 12.5 }), true, false)).toBe('IN PLAY');
  expect(statusLabel(game({ status: 'Final', winner: '0001' }), true, false)).toBe('FINAL');
});
