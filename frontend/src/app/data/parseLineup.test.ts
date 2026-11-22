import { describe, expect, it } from 'vitest';
import { parseLineup } from './parseLineup';
import { heldLineup } from './lineupTestData';

describe('held lineup boundary', () => {
  it('parses the 0025 reading asserted by the Go held-reading test', () => {
    const value = heldLineup();
    expect(parseLineup(value)).toEqual(value);
    expect(value.starters).toHaveLength(21);
    expect(value.bench).toHaveLength(17);
    expect(value.check).toEqual({ full: true, legal: true, problems: [] });
    delete value.starters[0].position;
    expect(parseLineup(value).starters[0]).toEqual({ id: '16150' });
  });
  it.each(['starters', 'bench'] as const)('requires the %s array, including empty arrays', (key) => {
    for (const bad of [null, undefined]) {
      expect(() => parseLineup({ ...heldLineup(), [key]: bad })).toThrow(`lineup.${key}`);
    }
  });
  it.each([null, undefined])('requires problems rather than %s', (problems) => {
    const r = heldLineup();
    expect(() => parseLineup({ ...r, check: { ...r.check, problems } })).toThrow('lineup.check.problems');
  });
  it('rejects unknown kinds and bad positions with their paths', () => {
    const r = heldLineup();
    expect(() => parseLineup({ ...r, check: { ...r.check, problems: [{
      subject: 'QB', kind: 'broken', message: 'bad',
    }] } })).toThrow('lineup.check.problems[0].kind');
    expect(() => parseLineup({ ...r, starters: [{ id: '16150', position: 'PK' }] }))
      .toThrow('lineup.starters[0].position');
  });
  it.each(['provenance', 'rulesSource'] as const)('reuses strict provenance for %s', (key) => {
    const r = heldLineup();
    expect(() => parseLineup({ ...r, [key]: { ...r[key], kind: 'invented' } }))
      .toThrow(`lineup.${key}.kind`);
  });
  it('rejects obsolete fields and missing booleans', () => {
    const r = heldLineup();
    expect(() => parseLineup({ ...r, check: { ...r.check, counts: {} } }))
      .toThrow('lineup.check.counts');
    expect(() => parseLineup({ ...r, check: { legal: true, problems: [] } }))
      .toThrow('lineup.check.full');
    expect(() => parseLineup({ ...r, starterCount: undefined })).toThrow('lineup.starterCount');
  });
});
