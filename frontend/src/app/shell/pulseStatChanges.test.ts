import { expect, it } from 'vitest';
import { changePoints, changeVerdict, statChanges, type PulseStarterRow } from './pulseModel';

const starter = (player: string, was?: number, now?: number): PulseStarterRow => ({ slot: 'RB', player,
  name: `Player ${player}`, points: now, projection: undefined, words: '', final: true,
  revision: was === undefined || now === undefined ? undefined
    : { was, now, stats: [{ stat: 'rushing_yards', label: 'Rushing yards (0 to 999)',
      was: 88, now: 94 }] } });

it('lists only re-scored starters, biggest move first, and says what moved', () => {
  const rows = statChanges([starter('1', 18.8, 19.4), starter('2')], [starter('3', 10, 8.5)]);
  expect(rows.map(r => [r.player, r.left])).toEqual([['3', false], ['1', true]]);
  expect(rows[1].stats).toBe('rushing yards 88 → 94');
  expect(changePoints(rows[1].delta, true)).toBe('+0.6');
  expect(changePoints(rows[0].delta, true)).toBe('−1.5');
  expect(changePoints(0.05)).toBe('0.05');
});

it('says whether a change decided the matchup', () => {
  const [theirs, mine] = statChanges([starter('1', 18.8, 19.4)], [starter('3', 10, 8.5)]);
  const names = { left: 'Aces', right: 'Kings', mine: true };
  expect(changeVerdict(mine, 5, false, names)).toBe('no change · W stands');
  expect(changeVerdict(mine, 0.4, false, names)).toBe('changed · L → W');
  expect(changeVerdict(mine, 0.6, false, names)).toBe('changed · T → W');
  expect(changeVerdict(theirs, 1, false, { ...names, mine: false })).toBe('changed · Kings → Aces');
  expect(changeVerdict(theirs, -3, false, { ...names, mine: false })).toBe('no change · Kings still wins');
  expect(changeVerdict(mine, undefined, false, names)).toBe('week not final');
  expect(changeVerdict(mine, 5, true, names)).toBe('recorded result stands');
});
