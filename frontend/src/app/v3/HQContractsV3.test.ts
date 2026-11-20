import { describe, expect, it } from 'vitest';
import { contractRows } from './HQContractsV3';
import type { NativeHQ } from '../commissioner/hqReading';

const reading = {
  snapshot: {
    league: { value: { season: 2026 } },
    players: { value: [{ id: 'a', name: 'Long Deal', position: 'LB' },
      { id: 'b', name: 'Ends Now', position: 'RB' }] },
    rosters: { value: [{ franchiseId: 'f1', players: [
      { id: 'b', salary: 150_000_000, yearsRemaining: 1, contractStatus: 'UFA', rosterStatus: 'ROSTER' },
      { id: 'a', salary: 1_350_000_000, yearsRemaining: 4, contractStatus: 'RFA', rosterStatus: 'ROSTER' },
    ] }] },
  },
  contracts: new Map([['a', [2026, 2027, 2028, 2029].map(season => ({ season, cents: 1_350_000_000 }))]]),
} as unknown as NativeHQ;

describe('contractRows', () => {
  it('lists the largest salary first, with each season it is owed and the year it ends', () => {
    const rows = contractRows(reading, 'f1');
    expect(rows.map(r => r.name)).toEqual(['Long Deal', 'Ends Now']);
    expect([...rows[0].years.keys()]).toEqual([2026, 2027, 2028, 2029]);
    expect(rows[0].ends).toBe(2029);
  });
  it('falls back to years remaining when the league holds no season-by-season costs', () => {
    const row = contractRows(reading, 'f1')[1];
    expect(row.ends).toBe(2026);
    expect(row.years.get(2026)).toBe(150_000_000);
    expect(row.status).toBe('UFA');
  });
});
