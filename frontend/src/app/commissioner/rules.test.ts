import { describe, expect, it } from 'vitest';
import { parseRules } from './rules';
import wire from './rules.wire.testdata.json';
import legacy from './rules.legacy.wire.testdata.json';

const rows = () => parseRules(wire).sections.flatMap(s => s.rows);

describe('Commissioner rules wire', () => {
  it('reads the real Go rules reading with values, sources and waiting changes in words', () => {
    const reading = parseRules(wire);
    expect(reading.revision).toBe(3);
    expect(reading.rollover).toBe('The next rollover has no date yet; the commissioner sets it.');
    const limits = rows().find(r => r.field === 'roster.limits');
    expect(limits).toMatchObject({ label: 'Active roster size', value: '20 to 70', state: 'Set' });
    expect(limits?.schema.fields.map(f => f.label)).toEqual(['Minimum', 'Maximum']);
    const cap = rows().find(r => r.field === 'cap.amount');
    expect(cap).toMatchObject({ value: 'Needs a ruling', state: 'Needs a ruling' });
    expect(cap?.pending[0]).toMatchObject({ value: '$125,000,000', when: 'Waits for the next rollover' });
    const bids = rows().find(r => r.field === 'bids.ufa');
    expect(bids?.schema.choices)
      .toEqual([{ value: 'blind', label: 'Blind' }, { value: 'open', label: 'Open' }]);
    expect(rows().every(r => r.citation !== '' && !/confidence/i.test(JSON.stringify(r)))).toBe(true);
  });
  it('carries what the form needs: the value in force, its unit and the safeguard for each timing', () => {
    const all = parseRules(legacy).sections.flatMap(s => s.rows);
    const share = all.find(r => r.field === 'cap.irShare');
    expect(share).toMatchObject({ unit: 'percent', current: 10000 });
    expect(all.find(r => r.field === 'roster.limits')).toMatchObject({
      unit: '', current: { Min: 18, Max: 80 }, safeguards: { nextRollover: 'hold', now: 'reason' } });
    expect(all.find(r => r.field === 'calendar.tradeDeadline')?.current).toBeNull();
    expect(all.find(r => r.field === 'scoring.catalogue')?.value.length).toBeGreaterThan(2000);
  });
  it('refuses a reading Go did not send', () => {
    const [head] = wire.sections;
    const row = head.rows[0];
    const withRow = (change: object) => ({ ...wire, sections: [{ ...head, rows: [{ ...row, ...change }] }] });
    expect(() => parseRules(withRow({ unit: 'dollars' }))).toThrow();
    expect(() => parseRules(withRow({ safeguards: { nextRollover: 'ask', now: 'hold' } }))).toThrow();
    expect(() => parseRules({ ...wire, gravity: 'G1' })).toThrow();
    const [first] = wire.sections;
    const bad = { ...first, rows: [{ ...first.rows[0], state: 'set' }] };
    expect(() => parseRules({ ...wire, sections: [bad] })).toThrow();
    const schema = { ...first.rows[0].schema, kind: 'float' };
    const kind = { ...first, rows: [{ ...first.rows[0], schema }] };
    expect(() => parseRules({ ...wire, sections: [kind] })).toThrow();
  });
});
