import { describe, expect, it } from 'vitest';
import clock from './fixtures/clock.json';
import demo from './fixtures/envelope-demo.json';
import { parseClock, parseEnvelopeDemo, parseReceipt } from './parse';

describe('clock and envelope boundaries', () => {
  it('parses real deterministic fixtures without inventing windows', () => {
    const c = parseClock(clock);
    expect(c.provenance.kind).toBe('fixture');
    expect(c.value.phase).toBe('OFFSEASON');
    expect(c.value.deadlines).toEqual([]);
    expect(c.value.windows.every((w) => w.status === 'unknown')).toBe(true);
    const e = parseEnvelopeDemo(demo);
    expect(e.kind).toBe('fixture');
    expect(e.receipt.audit.map((entry) => entry.to)).toEqual([
      'ready', 'handed_off', 'not_yet_done', 'landed',
    ]);
    expect(e.receipt.spec.franchiseId).toBe('0001');
  });
  it.each([
    ['bad urgency', (c: typeof clock) => {
      (c.value.deadlines as unknown[]).push({
        id: 'x', label: 'x', urgency: 'U9', pinned: false, promoted: false,
      });
    }, 'clock.value.deadlines[0].urgency:'],
    ['guessed window', (c: typeof clock) => { c.value.windows[0].status = 'open'; },
      'clock.value.windows[0].status:'],
    ['missing window', (c: typeof clock) => { c.value.windows.pop(); }, 'clock.value.windows:'],
    ['bad phase', (c: typeof clock) => { c.value.phase = 'SPRING'; }, 'clock.value.phase:'],
    ['empty phase', (c: typeof clock) => { c.value.phase = ''; }, 'clock.value.phase:'],
    ['undated but urgent', (c: typeof clock) => {
      (c.value.deadlines as unknown[]).push({
        id: 'x', label: 'x', urgency: 'U2', pinned: true, promoted: true,
      });
    }, 'clock.value.deadlines[0].at:'],
  ])('rejects clock corruption: %s', (_name, corrupt, message) => {
    const copy = JSON.parse(JSON.stringify(clock)); corrupt(copy);
    expect(() => parseClock(copy)).toThrowError(message);
  });
  it('parses a bare receipt with the same rules', () => {
    expect(parseReceipt(demo.receipt).state).toBe('landed');
    const copy = JSON.parse(JSON.stringify(demo.receipt));
    copy.state = 'ready';
    expect(() => parseReceipt(copy)).toThrowError('receipt.state: must match audit head');
  });
  it('rejects an unmapped target escaping draft, even without a URL', () => {
    const copy = JSON.parse(JSON.stringify(demo));
    copy.receipt.spec.target = { kind: 'unmapped' };
    expect(() => parseEnvelopeDemo(copy)).toThrowError(
      'envelope.receipt.state: unmapped target must stay draft or blocked',
    );
  });
  it.each([
    ['bad state', (e: typeof demo) => { e.receipt.state = 'accepted'; }, 'envelope.receipt.state:'],
    ['bad event', (e: typeof demo) => { e.receipt.audit[0].event = 'accept'; },
      'envelope.receipt.audit[0].event:'],
    ['unmapped landing', (e: typeof demo) => { e.receipt.spec.target = { kind: 'unmapped', url: '' }; },
      'envelope.receipt.spec.target.url:'],
    ['unsafe URL', (e: typeof demo) => { e.receipt.spec.target.url = 'javascript:alert(1)'; },
      'envelope.receipt.spec.target.url:'],
    ['trail mismatch', (e: typeof demo) => { e.receipt.audit[1].from = 'draft'; },
      'envelope.receipt.audit: discontinuous trail'],
    ['non-UTC audit', (e: typeof demo) => { e.receipt.audit[0].at = '2026-10-05T12:00:00+01:00'; },
      'envelope.receipt.audit[0].at: expected UTC timestamp'],
  ])('rejects envelope corruption: %s', (_name, corrupt, message) => {
    const copy = JSON.parse(JSON.stringify(demo)); corrupt(copy);
    expect(() => parseEnvelopeDemo(copy)).toThrowError(message);
  });
});


describe('clock week boundary', () => {
  it('keeps an absent week absent', () => {
    expect(parseClock(clock).value).not.toHaveProperty('week');
  });
  it.each([1, 5, 18])('accepts week %s', (week) => {
    expect(parseClock({ ...clock, value: { ...clock.value, week } }).value.week).toBe(week);
  });
  it.each([0, 19, '5', 4.5, null])('rejects week %s', (week) => {
    expect(() => parseClock({ ...clock, value: { ...clock.value, week } })).toThrow('clock.value.week:');
  });
});
