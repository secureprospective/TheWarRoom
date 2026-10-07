import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
// Plain Node generator is also exercised by Vitest; no runtime application import.
const { parseCSV, generateEndpoints, readEndpoints, csvPath, outputPath } =
  // @ts-expect-error Node script has no TypeScript declaration.
  await import('../../../scripts/gen-endpoints.mjs');

const source = readFileSync(csvPath, 'utf8');
describe('Registry boundary and freshness', () => {
  it('parses RFC 4180 commas, escaped quotes, quoted newlines, CRLF and empty final fields', () => {
    expect(parseCSV('a,b,c\r\n"a,b","x""y","line\nnext"\r\nz,,\r\n'))
      .toEqual([['a', 'b', 'c'], ['a,b', 'x"y', 'line\nnext'], ['z', '', '']]);
    expect(parseCSV('"",tail,')).toEqual([['', 'tail', '']]);
    expect(parseCSV('')).toEqual([]);
    expect(parseCSV('"a\r\nb"')).toEqual([['a\r\nb']]);
  });
  it('rejects unterminated quotes, stray quotes and characters after closing quotes', () => {
    for (const value of ['"unterminated', 'a"b', '"x"tail']) {
      expect(() => parseCSV(value)).toThrow(/CSV:/);
    }
  });
  it('regenerates exactly the checked-in typed, frozen file', () => {
    expect(generateEndpoints(source)).toBe(readFileSync(outputPath, 'utf8'));
    expect(generateEndpoints(source)).not.toBe(readFileSync(outputPath, 'utf8') + '// stale');
  });
  it('rejects corrupt normalized data before generation', () => {
    expect(() => readEndpoints(source.replace('G2,live,', 'G9,live,'))).toThrow('invalid gravity');
    expect(() => readEndpoints(source.replace('League Pulse › Now', 'Unknown place'))).toThrow('placement');
    expect(() => readEndpoints(source.replace('M-003 M-005', 'M-999 M-005'))).toThrow('not kept');
    expect(() => readEndpoints(source.replace('id,source,', 'id,id,'))).toThrow('duplicate columns');
  });
});
