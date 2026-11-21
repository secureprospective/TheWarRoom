import { describe, expect, it } from 'vitest';
import { calendarRows } from './WarPlansV3';

describe('calendarRows', () => {
  it('puts open windows first, then upcoming by date, and drops past events', () => {
    const now = Date.parse('2026-11-20T12:00:00Z');
    const rows = calendarRows([
      { id: 'past', label: 'Past', at: '2026-11-01T00:00:00Z' },
      { id: 'late', label: 'Late', at: '2026-12-20T00:00:00Z' },
      { id: 'soon', label: 'Soon', at: '2026-11-25T00:00:00Z' },
      { id: 'open', label: 'Open', at: '2026-11-01T00:00:00Z', live: true },
    ], now);
    expect(rows.map(r => r.id)).toEqual(['open', 'soon', 'late']);
  });
});
