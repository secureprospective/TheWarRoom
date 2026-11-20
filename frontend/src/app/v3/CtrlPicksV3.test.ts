import { describe, expect, it } from 'vitest';
import { defaultPickYear, filterPicks, pickHow, pickNumber, pickStatus } from './CtrlPicksV3';
import type { HostedRoster } from '../commissioner/hostedReadings';

type P = HostedRoster['picks'][number];
const pick = (o: Partial<P> = {}): P => ({ season: 2027, round: 1, slot: 1, original: 'A', owner: 'A',
  compensatory: false, correction: '', player: '', skipped: false, used: false, ...o });
const read = (...picks: P[]) => ({ season: 2026, picks,
  franchises: [{ value: 'A', label: 'Alpha' }, { value: 'B', label: 'Bravo' }] }) as unknown as HostedRoster;
const all = { year: 'all', round: 'all', trade: 'all', search: '' };

describe('pick words', () => {
  it('words how a pick came to its owner', () => {
    expect(pickHow(pick())).toBe('Original');
    expect(pickHow(pick({ owner: 'B' }))).toBe('Traded');
    expect(pickHow(pick({ compensatory: true }))).toBe('Compensatory');
    expect(pickHow(pick({ correction: 'x', owner: 'B' }))).toBe('Correction');
  });
  it('words status and number', () => {
    expect(pickStatus(pick({ player: 'p' }))).toBe('USED');
    expect(pickStatus(pick({ skipped: true }))).toBe('SKIPPED');
    expect(pickStatus(pick())).toBe('ACTIVE');
    expect(pickNumber(pick({ round: 2, slot: 4 }))).toBe('2.04');
  });
});

describe('defaultPickYear', () => {
  it('picks the first season after the league season', () => {
    expect(defaultPickYear(read(pick({ season: 2026 }), pick({ season: 2028 }), pick()))).toBe('2027');
  });
  it('falls back to the last season, then all', () => {
    expect(defaultPickYear({ ...read(pick({ season: 2025 })) })).toBe('2025');
    expect(defaultPickYear(undefined)).toBe('all');
  });
});

describe('filterPicks', () => {
  const reading = read(pick({ round: 2, slot: 1 }), pick({ slot: 2, owner: 'B' }),
    pick({ season: 2028 }));
  it('orders by season, round, slot', () => {
    expect(filterPicks(reading, all).map(p => `${p.season}:${p.round}.${p.slot}`))
      .toEqual(['2027:1.2', '2027:2.1', '2028:1.1']);
  });
  it('filters by year, round and traded', () => {
    expect(filterPicks(reading, { ...all, year: '2028' })).toHaveLength(1);
    expect(filterPicks(reading, { ...all, round: '2' })).toHaveLength(1);
    expect(filterPicks(reading, { ...all, trade: 'traded' })).toHaveLength(1);
  });
  it('searches team labels and ids', () => {
    expect(filterPicks(reading, { ...all, search: ' bravo ' })).toHaveLength(1);
    expect(filterPicks(reading, { ...all, search: 'zzz' })).toHaveLength(0);
  });
});
