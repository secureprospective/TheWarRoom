import { describe, expect, it } from 'vitest';
import { bySlot, pairStarters, type PulseStarterRow } from './pulseModel';

const row = (slot: string, player: string) =>
  ({ slot, player, name: player, final: false }) as PulseStarterRow;

// Christopher 11-21: live scoring lists players in the same position order as the Franchise HQ lineup.
describe('lineup order on live screens', () => {
  it('sorts offense, flex, defense, then kickers, keeping lineup order inside a position', () => {
    const slots = ['S', 'PK', 'LB', 'QB', 'CB', 'RB', 'WR', 'DT', 'TE', 'DE', 'RB'];
    expect([...slots].sort(bySlot))
      .toEqual(['QB', 'RB', 'RB', 'WR', 'TE', 'DT', 'DE', 'LB', 'CB', 'S', 'PK']);
  });
  it('pairs offense, then defense, then kickers', () => {
    const groups = pairStarters([row('PK', 'k'), row('LB', 'l'), row('QB', 'q')], [row('QB', 'q2')]);
    expect(groups.map(g => [g.heading, g.pairs.map(p => p.slot)])).toEqual([
      ['OFFENSE', ['QB']], ['DEFENSE', ['LB']], ['KICKERS', ['PK']]]);
  });
});
