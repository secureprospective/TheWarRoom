import { describe, expect, it } from 'vitest';
import { countdown, grade } from './urgency';
import { deadlineLabel, localDate, phaseLabels, windowLabels } from './labels';

const now = Date.parse('2026-10-06T12:00:00Z');
const at = (remaining: number) => new Date(now + remaining).toISOString();

describe('Go urgency boundaries', () => {
  it.each([
    [3_600_000, 'U2', '1h 0m'],
    [3_599_999, 'U3', '59m'],
    [172_800_000, 'U2', '2d 0h'],
    [172_800_001, 'U1', '2d 0h'],
    [-1, 'U3', 'passed'],
    [undefined, 'U0', '—'],
  ] as const)('grades %s milliseconds as %s', (remaining, urgency, label) => {
    const date = remaining === undefined ? undefined : at(remaining);
    expect(grade(date, now)).toBe(urgency);
    expect(countdown(date, now)).toBe(label);
  });
  it.each([
    [0, 'under 1m'], [59_999, 'under 1m'], [60_000, '1m'],
    [47 * 60_000, '47m'], [192 * 60_000, '3h 12m'], [148 * 3_600_000, '6d 4h'],
  ])('shows the two largest time units for %s', (remaining, label) => {
    expect(countdown(at(remaining), now)).toBe(label);
  });
  it('labels real calendar kinds without changing human labels or guessing dates', () => {
    expect(deadlineLabel('TRADE_DEADLINE')).toBe('Trade deadline');
    expect(deadlineLabel('Commissioner meeting')).toBe('Commissioner meeting');
    expect(localDate(undefined)).toBe('Date unknown');
    expect(localDate(at(0))).toBe(
      new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(now)),
    );
    expect(Object.values(phaseLabels)).toEqual(['Offseason', 'Regular season', 'Playoffs']);
    expect(Object.keys(windowLabels)).toHaveLength(7);
  });
});
