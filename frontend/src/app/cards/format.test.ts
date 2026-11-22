import { describe, expect, it } from 'vitest';
import { ageAt, formatMoney } from './format';

describe('money in integer cents', () => {
  it.each([
    [1_200_000_000, '$12.0M'],
    [0, '$0.0M'],
    [-120_000_000, '$-1.2M'],
    [-1, '$0.0M'],
    [4_999_999, '$0.0M'],
    [5_000_000, '$0.1M'],
  ])('formats %s as %s', (cents, expected) => expect(formatMoney(cents)).toBe(expected));
});

describe('age against an injected UTC date', () => {
  const asOf = new Date('2026-10-06T00:00:00Z');
  it.each([
    ['2000-10-06', 26],
    ['2000-10-07', 25],
    ['2000-10-05', 26],
    ['2026-10-06', 0],
    ['2027-10-06', undefined],
  ])('handles birthday %s', (date, expected) => {
    expect(ageAt(Date.parse(`${date}T00:00:00Z`) / 1000, asOf)).toBe(expected);
  });
  it('does not turn missing birthdate into zero', () =>
    expect(ageAt(undefined, asOf)).toBeUndefined());
  it('does not depend on the machine time zone', () => {
    expect(
      ageAt(
        Date.parse('2000-10-07T00:00:00Z') / 1000,
        new Date('2026-10-06T23:30:00-02:00'),
      ),
    ).toBe(26);
  });
  it('keeps leap-day birthdays pending until March in a non-leap year', () => {
    const born = Date.parse('2000-02-29T00:00:00Z') / 1000;
    expect(ageAt(born, new Date('2025-02-28T00:00:00Z'))).toBe(24);
    expect(ageAt(born, new Date('2025-03-01T00:00:00Z'))).toBe(25);
  });
  it('omits age for an invalid injected clock', () =>
    expect(ageAt(0, new Date('invalid'))).toBeUndefined());
});
