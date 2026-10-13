import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
// Plain Node generator is also exercised by Vitest; no runtime application import.
const { parseCSV, generateEndpoints, generateCommissioner, readEndpoints, csvPath, outputPath,
  commissionerPath } =
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
    expect(generateCommissioner(source)).toBe(readFileSync(commissionerPath, 'utf8'));
    expect(generateCommissioner(source)).not.toBe(readFileSync(commissionerPath, 'utf8') + '// stale');
  });
  it('rejects corrupt normalized data before generation', () => {
    expect(() => readEndpoints(source.replace('G2,live,', 'G9,live,'))).toThrow('invalid gravity');
    expect(() => readEndpoints(source.replace('League Pulse › Now', 'Unknown place'))).toThrow('placement');
    expect(() => readEndpoints(source.replace('M-003 M-005', 'M-999 M-005'))).toThrow('not kept');
    expect(() => readEndpoints(source.replace('id,source,', 'id,id,'))).toThrow('duplicate columns');
  });
});

it('pools repeated placement strings without changing the generated registry contract', () => {
  const generated = generateEndpoints(source);
  expect(generated).toContain('const places = [');
  expect(generated).toContain('placement: places[');
  expect(generated.match(/"Trade Floor › Draft room"/g)).toHaveLength(2);
});

it('keeps registry prose the app never reads out of the generated table', () => {
  const generated = generateEndpoints(source);
  for (const key of ['when', 'data', 'sol_ids', 'gravity_reason', 'ring_note', 'gravity_note']) {
    expect(generated).not.toMatch(new RegExp(`^    ${key}:`, 'm'));
  }
  expect(generated).toMatch(/^    gm_question:/m);
});

function changeRow(id: string, field: string, value: string, input = source): string {
  const [header, ...rows] = parseCSV(input) as string[][];
  const column = header.indexOf(field);
  const row = rows.find(values => values[0] === id)!;
  row[column] = value;
  return [header, ...rows].map(values => values.map(cell =>
    `"${cell.replaceAll('"', '""')}"`).join(',')).join('\n');
}

it.each([
  ['CM-001', 'governs', 'M-999', 'not an owner id'],
  ['CM-001', 'governs', 'CM-002', 'not an owner id'],
  ['CM-001', 'governs', '', 'empty governs'],
  ['CM-001', 'governs', 'M-091 M-091', 'duplicate governs'],
  ['CM-001', 'governs', 'ledger-wide', 'invalid ledger-wide'],
  ['CM-039', 'governs', 'ledger-wide M-091', 'invalid ledger-wide'],
  ['CM-039', 'governs', 'M-091', 'requires ledger-wide'],
  ['CM-056', 'governs', 'M-091', 'requires ledger-wide'],
  ['M-001', 'governs', 'M-002', 'owner has governs'],
  ['CM-028', 'id', 'CM-057', 'contiguous CM-001..CM-056'],
  ['CM-028', 'id', 'CM-027', 'duplicate endpoint id'],
  ['CM-001', 'ring', '10', 'invalid gravity or ring'],
])('rejects %s %s = %s at both generation boundaries', (id, field, value, message) => {
  const mutated = changeRow(id, field, value);
  expect(() => generateEndpoints(mutated)).toThrow(message);
  expect(() => generateCommissioner(mutated)).toThrow(message);
});

it('rejects a retired commissioner record before emitting its kept-only type', () => {
  const retired = changeRow('CM-001', 'disposition', 'retired', changeRow('CM-001', 'placement', ''));
  expect(() => generateCommissioner(retired)).toThrow('commissioner must be kept');
});

it('keeps all controls in their separate module, leaving the owner table unchanged', () => {
  const rows = readEndpoints(source) as { id: string; ring: number; governs: string[] }[];
  expect(rows.filter(row => row.id.startsWith('CM-'))).toHaveLength(56);
  expect(generateEndpoints(source)).not.toMatch(/CM-\d{3}|commish\./);
  expect(generateEndpoints(source)).not.toMatch(/governs/);
  const controls = generateCommissioner(source);
  expect(controls).toContain('export type CommissionerId =');
  expect(controls).toContain('readonly (OwnerEndpointId | \'ledger-wide\')[]');
  for (let n = 1; n <= 56; n++) {
    const id = `CM-${String(n).padStart(3, '0')}`;
    expect(controls).toContain(`id: "${id}"`);
    const row = rows.find(row => row.id === id)!;
    expect(row.ring).toBe((n >= 46 && n <= 50) || n === 55 ? 6 : 4);
    expect(row.governs.length).toBeGreaterThan(0);
  }
  expect(() => readEndpoints(changeRow('CM-001', 'ring', '9'))).not.toThrow();
  expect(() => readEndpoints(changeRow('CM-001', 'ring', '0'))).not.toThrow();
});
