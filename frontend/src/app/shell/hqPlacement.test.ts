import { expect, it } from 'vitest';
import { fitsSlot, placePlayer } from './hqPlacement';
import type { HQSlot } from './HQBoard';
const settings = { positions: [{ code: 'LB', min: 1, max: 3 }, { code: 'QB', min: 1, max: 1 }] };
const players = [{ id: 'a', position: 'LB' }, { id: 'b', position: 'LB' }, { id: 'c', position: 'LB' },
  { id: 'q', position: 'QB' }];
const slots: HQSlot[] = [{ code: 'LB', side: 'def', id: 'a' }, { code: 'D·FLX', side: 'def', id: 'b' },
  { code: 'QB', side: 'off', id: 'q' }];
const values = new Map([['a', 10], ['b', 5], ['c', 15], ['q', 20]]);
it('rejects incompatible drops without changing any slot', () => {
  expect(placePlayer(slots, players, settings, values, 'c', 2)).toBe(slots);
  expect(slots.map(s => s.id)).toEqual(['a', 'b', 'q']);
});
it('swaps compatible starters and cascades displaced players to weaker compatible slots', () => {
  expect(placePlayer(slots, players, settings, values, 'a', 1).map(s => s.id)).toEqual(['b', 'a', 'q']);
  expect(placePlayer(slots, players, settings, values, 'c', 0).map(s => s.id)).toEqual(['c', 'a', 'q']);
});
it('autoplaces into the weakest compatible slot and autofills a benched starter', () => {
  expect(placePlayer(slots, players, settings, values, 'c').map(s => s.id)).toEqual(['a', 'c', 'q']);
  expect(placePlayer(slots, players, settings, values, 'a', undefined, true).map(s => s.id))
    .toEqual(['c', 'b', 'q']);
});
it('never invents projections to choose an autofill', () => {
  expect(placePlayer(slots, players, settings, new Map(), 'a', undefined, true)[0].id).toBeUndefined();
});
it('changes eligibility when the position allowance changes', () => {
  expect(fitsSlot(players[0], slots[1], settings)).toBe(true);
  expect(fitsSlot(players[0], slots[1], { positions: [{ code: 'LB', min: 0, max: 0 }] })).toBe(false);
});

it('keeps quarterbacks, kickers and offensive line out of skill flex slots', () => {
 const slot: HQSlot = { code: 'FLEX', side: 'off' };
 for (const position of ['QB', 'K', 'OL'])
  expect(fitsSlot({ id: position, position }, slot, { positions: [] })).toBe(false);
 for (const position of ['RB', 'WR', 'TE'])
  expect(fitsSlot({ id: position, position }, slot, { positions: [] })).toBe(true);
});
