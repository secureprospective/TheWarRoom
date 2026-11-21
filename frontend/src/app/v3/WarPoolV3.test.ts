import { describe, expect, it } from 'vitest';
import { posMatch } from './WarPoolV3';

describe('posMatch', () => {
  it('matches the pool chips: All, one position, or any defensive player under IDP', () => {
    expect(posMatch('All', 'K')).toBe(true);
    expect(posMatch('WR', 'WR')).toBe(true);
    expect(posMatch('WR', 'TE')).toBe(false);
    expect(['DT', 'LB', 'CB', 'S', 'DE'].every(p => posMatch('IDP', p))).toBe(true);
    expect(posMatch('IDP', 'QB')).toBe(false);
  });
});
