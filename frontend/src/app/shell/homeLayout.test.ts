import { expect, it } from 'vitest';
import { bump, defaultLayout, DEFAULTS, pack, parseLayout, place } from './homeLayout';

it('packs both phase defaults into the 3 × 2 grid', () => {
  expect(pack(DEFAULTS.in)).toHaveLength(5);
  expect(pack(DEFAULTS.off)).toHaveLength(5);
});
it('a full grid asks for a tile to bump, and the bumped tile lands in overflow', () => {
  expect(place(defaultLayout(), 'in', 'cap', '1x1')).toBeUndefined();
  const next = bump(defaultLayout(), 'in', { id: 'cap', span: '1x1' }, 'race')!;
  expect(next.over).toEqual(['race']);
  expect(next.tiles.map(t => t[0])).toContain('cap');
});
it('a saved layout that no longer packs falls back to the default', () => {
  expect(parseLayout(JSON.stringify({ mode: 'custom', tiles: [['news', '1x2'], ['wire', '1x2'],
    ['mu', '1x2'], ['race', '1x1']], over: [], pins: {} })).mode).toBe('default');
  expect(parseLayout('not json').mode).toBe('default');
});
