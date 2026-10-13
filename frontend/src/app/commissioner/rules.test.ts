import { describe, expect, it } from 'vitest';
import { parseRules } from './rules';
import wire from './rules.wire.testdata.json';

const rows = () => parseRules(wire).sections.flatMap(s => s.rows);

describe('Commissioner rules wire', () => {
  it('reads the real Go rules reading with values, sources and waiting changes in words', () => {
    const reading = parseRules(wire);
    expect(reading.revision).toBe(3);
    expect(reading.rollover).toMatch(/^Needs a ruling/);
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
  it('refuses a reading Go did not send', () => {
    expect(() => parseRules({ ...wire, gravity: 'G1' })).toThrow();
    const [first] = wire.sections;
    const bad = { ...first, rows: [{ ...first.rows[0], state: 'set' }] };
    expect(() => parseRules({ ...wire, sections: [bad] })).toThrow();
    const schema = { ...first.rows[0].schema, kind: 'float' };
    const kind = { ...first, rows: [{ ...first.rows[0], schema }] };
    expect(() => parseRules({ ...wire, sections: [kind] })).toThrow();
  });
});
