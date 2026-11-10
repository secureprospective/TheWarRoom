import { describe, expect, it } from 'vitest';
import { leanWords, lensValue, lensWords, parseWarRoomLens, signed, verdictWords } from './warroomLens';

const side = (franchise: string, extra: object = {}) => ({ franchise, direction: 'contend', set: false,
  nowWeight: 0.6, before: 292.9, after: 288.4, net: -4.5, out: 9.7, in: 5.2, spare: [],
  needs: [{ position: 'S', state: 'hole', have: 0, starts: 1 },
    { position: 'QB', state: 'ok', have: 2, starts: 1 },
    { position: 'TE', state: 'thin', have: 1, starts: 1 }],
  ...extra });
const reading = (them: object) => ({ league: 'native.a', season: 2026, starterN: 21, you: side('f1'), them,
  assets: [{ token: '100', now: 16.5, dynasty: 17.2, you: 6.5, them: 10, valued: true, words: '', age: 22.9,
    ageKnown: true }, { token: 'pick:2027:1:f1', now: 0, dynasty: 0, you: 0, them: 0, valued: false,
    words: 'No draft on record', age: 0, ageKnown: false }],
  verdict: 'bad', tune: [{ kind: 'trim', token: '100', you: -1.6, them: 1.6 }],
  notes: ['No draft on record'] });

describe('trade lens reading', () => {
  it('reads both sides, orders needs by position and keeps unknown ages unknown', () => {
    const lens = parseWarRoomLens(reading(side('f2', { net: 8.1 })), 'native.a', 'f1');
    expect(lens.you.needs.map(n => n.position)).toEqual(['QB', 'TE', 'S']);
    expect(lens.them?.net).toBe(8.1);
    expect(lens.assets.get('100')?.age).toBe(22.9);
    expect(lens.assets.get('pick:2027:1:f1')?.age).toBeUndefined();
    expect(lensValue(lens.assets.get('100'), 'them')).toBe('10.0');
    expect(lensValue(lens.assets.get('pick:2027:1:f1'), 'you')).toBe('—');
    expect(verdictWords(lens).tag).toBe('BAD FOR YOU');
  });
  it('has no partner side until one is chosen, and refuses another league or franchise', () => {
    const empty = side('', { needs: [] });
    expect(parseWarRoomLens(reading(empty), 'native.a', 'f1').them).toBeUndefined();
    expect(() => parseWarRoomLens(reading(empty), 'native.b', 'f1')).toThrow('another league');
    expect(() => parseWarRoomLens(reading(empty), 'native.a', 'f9')).toThrow('another franchise');
  });
  it('says whose direction it is, the real weights, and names holes before thin spots', () => {
    const lens = parseWarRoomLens(reading(side('f2')), 'native.a', 'f1');
    expect(lensWords(lens.you)).toBe('our read · win now: 60% this season, 40% long term');
    expect(lensWords({ ...lens.you, set: true, direction: 'rebuild', nowWeight: 0 }))
      .toBe('rebuild: long-term value only');
    expect(leanWords(lens.you)).toContain('(60%)');
    expect(leanWords(lens.you)).toContain('Starters at S move them most.');
    expect(signed(-0.04)).toBe('+0.0');
    expect(signed(-4.54)).toBe('−4.5');
  });
});
