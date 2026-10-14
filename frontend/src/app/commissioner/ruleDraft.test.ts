import { describe, expect, it } from 'vitest';
import { blankDraft, draftFrom, editDraft, scaled, toNatural, type Draft } from './ruleDraft';
import { parseRules, type RuleRow } from './rules';
import legacy from './rules.legacy.wire.testdata.json';

const rows = parseRules(legacy).sections.flatMap(s => s.rows);
const row = (field: string): RuleRow => rows.find(r => r.field === field) as RuleRow;
const typed = (r: RuleRow, draft: Draft) => toNatural(r.schema, draft, r.label, r.unit === 'percent');
const text = (v: string): Draft => ({ t: 'text', v });
// Instants compare as instants: the form writes milliseconds that Go leaves out.
const same = (a: unknown, b: unknown): boolean => {
  if (typeof a === 'string' && typeof b === 'string' && /^\d{4}-\d\d-\d\dT/.test(a))
    return new Date(a).getTime() === new Date(b).getTime();
  if (Array.isArray(a) && Array.isArray(b)) return a.length === b.length && a.every((v, i) => same(v, b[i]));
  if (a && b && typeof a === 'object' && typeof b === 'object') {
    const [x, y] = [a, b] as Record<string, unknown>[];
    return Object.keys(x).length === Object.keys(y).length && Object.keys(x).every(k => same(x[k], y[k]));
  }
  return a === b;
};

describe('Rule value drafts', () => {
  it('turns every Legacy rule into a draft and back to the value Go sent', () => {
    const set = rows.filter(r => r.state === 'Set');
    expect(set.length).toBeGreaterThan(90);
    for (const r of set) {
      const back = typed(r, draftFrom(r.schema, r.current, r.unit === 'percent'));
      expect(back, r.field).toMatchObject({ ok: true });
      expect(same(back.ok && back.value, r.current), `${r.field} changed on the way through`).toBe(true);
    }
  });
  it('types money in dollars, shares in percent and times in the unit they divide into', () => {
    const cap = row('cap.amount');
    expect(draftFrom(cap.schema, 12_500_000_050)).toEqual(text('125000000.5'));
    expect(typed(cap, text('125000000.50'))).toEqual({ ok: true, value: 12_500_000_050 });
    expect(typed(cap, text('1.234'))).toMatchObject({ ok: false });
    const share = row('cap.irShare');
    expect(draftFrom(share.schema, 3550, true)).toEqual(text('35.5'));
    expect(typed(share, text('35.5'))).toEqual({ ok: true, value: 3550 });
    const clock = row('draft.clock');
    expect(draftFrom(clock.schema, 21_600_000_000_000)).toEqual({ t: 'time', v: '6', unit: 'hours' });
    expect(typed(clock, { t: 'time', v: '1.5', unit: 'days' }))
      .toEqual({ ok: true, value: 129_600_000_000_000 });
    expect(typed(clock, { t: 'time', v: 'soon', unit: 'days' })).toMatchObject({ ok: false });
  });
  it('holds a number to Go’s bounds and says so in words', () => {
    const seats = row('trades.dotSeats');
    expect(typed(seats, text('3'))).toEqual({ ok: false, error: 'Trade review seats: must be at least 4' });
    expect(typed(seats, text('11'))).toEqual({ ok: false, error: 'Trade review seats: must be at most 10' });
    expect(typed(seats, text('7'))).toEqual({ ok: true, value: 7 });
    expect(typed(seats, text('7.5'))).toMatchObject({ ok: false });
    expect(typed(seats, text(''))).toMatchObject({ ok: false });
    expect(typed(row('cap.irShare'), text('101'))).toEqual({
      ok: false, error: 'IR salary share: must be at most 100%' });
  });
  it('builds a range from its two ends, and lists and maps from their entries', () => {
    const limits = row('roster.limits');
    const blank = blankDraft(limits.schema);
    expect(typed(limits, blank)).toMatchObject({ ok: false });
    const filled: Draft = { t: 'rec', f: { Min: text('20'), Max: text('70') } };
    expect(typed(limits, filled)).toEqual({ ok: true, value: { Min: 20, Max: 70 } });
    const ir = row('ir.eligible');
    expect(typed(ir, { t: 'list', items: [text('IR'), text('PUP')] }))
      .toEqual({ ok: true, value: ['IR', 'PUP'] });
    const lineup = row('lineup.positions');
    const two = { t: 'map', items: [{ k: 'QB', v: filled }, { k: 'QB', v: filled }] } as Draft;
    expect(typed(lineup, two)).toMatchObject({ ok: false, error: expect.stringContaining('two entries') });
    expect(typed(lineup, { t: 'map', items: [{ k: ' ', v: filled }] })).toMatchObject({ ok: false });
  });
  it('writes an on or off value and leaves a blank date out of a record', () => {
    expect(typed(row('league.dynasty'), { t: 'flag', v: true })).toEqual({ ok: true, value: true });
    expect(scaled(-12_500, 4)).toBe('-1.25');
    const windows = row('calendar.windows');
    const back = typed(windows, draftFrom(windows.schema, windows.current));
    expect(back).toMatchObject({ ok: true });
    const open = (back.ok ? back.value : {}) as Record<string, Record<string, unknown>>;
    expect(Object.values(open).every(w => w.DatedStart === undefined)).toBe(true);
  });
  it('adds and removes entries at any depth, and ignores a path that leads nowhere', () => {
    const tag = row('contracts.extension');
    const draft = draftFrom(tag.schema, tag.current);
    const floors = (d: Draft) => ((d as { f: Record<string, Draft> }).f.Floors as { items: unknown[] }).items;
    const before = floors(draft).length;
    const added = editDraft(tag.schema, draft, { action: 'add', path: ['Floors'], index: 0 });
    expect(floors(added)).toHaveLength(before + 1);
    expect(floors(added).at(-1)).toEqual({ k: '', v: { t: 'text', v: '' } });
    const removed = editDraft(tag.schema, added, { action: 'remove', path: ['Floors'], index: 0 });
    expect(floors(removed)).toHaveLength(before);
    expect(editDraft(tag.schema, draft, { action: 'add', path: ['Nowhere'], index: 0 })).toEqual(draft);
    expect(editDraft(tag.schema, draft, { action: 'add', path: ['Floors', 99], index: 0 })).toEqual(draft);
    const tiers = row('bids.tiers');
    const list = draftFrom(tiers.schema, tiers.current);
    const grown = editDraft(tiers.schema, list, { action: 'add', path: [], index: 0 });
    const count = (d: Draft) => (d as { items: unknown[] }).items.length;
    expect(count(grown)).toBe(count(list) + 1);
  });
});
