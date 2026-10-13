import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const csvPath = fileURLToPath(new URL('../../docs/ui/endpoint-registry.csv', import.meta.url));
export const outputPath = fileURLToPath(new URL('../src/app/registry/endpoints.gen.ts', import.meta.url));

export const commissionerPath = fileURLToPath(new URL(
  '../src/app/registry/commissioner.gen.ts', import.meta.url,
));

export function parseCSV(source) {
  const rows = [];
  let row = [], field = '', quoted = false, closed = false;
  for (let i = 0; i < source.length; i++) {
    const char = source[i];
    if (quoted) {
      if (char !== '"') field += char;
      else if (source[i + 1] === '"') { field += '"'; i++; }
      else { quoted = false; closed = true; }
    } else if (char === ',' || char === '\n' || char === '\r') {
      row.push(field);
      field = '';
      closed = false;
      if (char !== ',') {
        rows.push(row);
        row = [];
        if (char === '\r' && source[i + 1] === '\n') i++;
      }
    } else if (char === '"' && !field && !closed) quoted = true;
    else {
      if (closed || char === '"') throw new Error(`CSV: unexpected character at ${i}`);
      field += char;
    }
  }
  if (quoted) throw new Error('CSV: unterminated quoted field');
  if (field || row.length || closed) rows.push([...row, field]);
  return rows;
}

const workspaces = {
  Home: ['Seasonal card', 'Alert tray', 'Digests'],
  'War Room': ['Market', 'Valuations and pool', 'Research'],
  'Franchise HQ': ['Lineup and roster', 'Contracts and cap', 'My moves'],
  'Trade Floor': ['Trade desk', 'Offers history', 'Counterparties', 'Draft room'],
  'League Pulse': ['Now', 'Race', 'Archive'],
  'Control Room': ['App', 'Data and sources', 'League'],
};
const placements = [
  ...Object.entries(workspaces).flatMap(([node, names]) => names.map((name) => `${node} › ${name}`)),
  'Inspector', 'Comms', 'Calendar', 'Command bar', 'Status strip', 'Report drawer',
];
const fields = [
  'id', 'source', 'name', 'kind', 'gm_question', 'when', 'gravity', 'clock', 'data', 'placement',
  'verb', 'disposition', 'ring', 'status', 'sol_ids', 'disposition_note', 'merged_into',
  'merged_place', 'gravity_reason', 'ring_note', 'gravity_note', 'governs',
];
// Registry prose the app never reads stays in the CSV and out of the entry bundle.
const docOnly = ['when', 'data', 'sol_ids', 'gravity_reason', 'ring_note', 'gravity_note'];
const emitted = fields.filter((key) => !docOnly.includes(key) && key !== 'governs');

export function readEndpoints(source) {
  const [header, ...data] = parseCSV(source);
  if (!header || fields.some((field) => !header.includes(field)) || new Set(header).size !== header.length) {
    throw new Error('CSV: missing normalized fields or duplicate columns');
  }
  const rows = data.map((values, index) => {
    if (values.length !== header.length) throw new Error(`CSV: row ${index + 2} width`);
    const row = Object.fromEntries(header.map((key, i) => [key, values[i]]));
    if (!/^(CM-\d{3}|M-\d{3}|[LCSX]-\d{2}(\.\d+)?)$/.test(row.id)) {
      throw new Error(`CSV: invalid id ${row.id}`);
    }
    if (!/^G[0-3]$/.test(row.gravity) || !/^[0-9]$/.test(row.ring)) {
      throw new Error(`${row.id}: invalid gravity or ring`);
    }
    if (!['kept', 'merged', 'retired'].includes(row.disposition)) throw new Error(`${row.id}: disposition`);
    if (row.disposition === 'kept' ? !placements.includes(row.placement) : row.placement !== '') {
      throw new Error(`${row.id}: placement`);
    }
    row.merged_into = row.merged_into ? row.merged_into.split(' ') : [];
    row.governs = row.governs ? row.governs.split(' ') : [];
    row.ring = Number(row.ring);
    const targets = row.merged_into.length > 0;
    const place = row.merged_place !== '';
    if (row.disposition === 'merged' ? targets === place : targets || place) {
      throw new Error(`${row.id}: merge target kind`);
    }
    if (place && !placements.includes(row.merged_place)) throw new Error(`${row.id}: merged place`);
    return row;
  });
  const ids = new Map(rows.map((row) => [row.id, row]));
  if (ids.size !== rows.length) throw new Error('CSV: duplicate endpoint id');
  for (const row of rows) {
    for (const target of row.merged_into) {
      if (ids.get(target)?.disposition !== 'kept') throw new Error(`${row.id}: target ${target} is not kept`);
    }
  }
  validateGoverns(rows, ids);
  return rows;
}

function validateGoverns(rows, ids) {
  const controls = rows.filter((row) => row.id.startsWith('CM-'));
  const expected = Array.from({ length: 56 }, (_, i) => `CM-${String(i + 1).padStart(3, '0')}`);
  if (controls.length !== expected.length || expected.some((id) => !ids.has(id))) {
    throw new Error('CSV: CM ids must be contiguous CM-001..CM-056');
  }
  for (const row of rows) {
    const commissioner = row.id.startsWith('CM-');
    if (commissioner && row.disposition !== 'kept') throw new Error(`${row.id}: commissioner must be kept`);
    if (commissioner && !row.governs.length) throw new Error(`${row.id}: empty governs`);
    if (!commissioner && row.governs.length) throw new Error(`${row.id}: owner has governs`);
    if (new Set(row.governs).size !== row.governs.length) {
      throw new Error(`${row.id}: duplicate governs`);
    }
    for (const target of row.governs) {
      if (target === 'ledger-wide') {
        if (!['CM-039', 'CM-056'].includes(row.id) || row.governs.length !== 1) {
          throw new Error(`${row.id}: invalid ledger-wide governs`);
        }
      } else if (!ids.has(target) || target.startsWith('CM-')) {
        throw new Error(`${row.id}: governs target ${target} is not an owner id`);
      }
    }
    if (['CM-039', 'CM-056'].includes(row.id) && row.governs[0] !== 'ledger-wide') {
      throw new Error(`${row.id}: requires ledger-wide governs`);
    }
  }
}

function literal(value, width = 80) {
  if (typeof value !== 'string') return JSON.stringify(value);
  const chunks = [];
  let chunk = '';
  for (const char of value) {
    if (JSON.stringify(chunk + char).length > width) { chunks.push(JSON.stringify(chunk)); chunk = ''; }
    chunk += char;
  }
  chunks.push(JSON.stringify(chunk));
  return chunks.join(' +\n      ');
}

export function generateEndpoints(source) {
  return generateTable(readEndpoints(source).filter((row) => !row.id.startsWith('CM-')));
}

export function generateCommissioner(source) {
  return generateTable(readEndpoints(source).filter((row) => row.id.startsWith('CM-')), true);
}

function generateTable(rows, commissioner = false) {
  const keys = commissioner ? [...emitted, 'governs'] : emitted;
  const idType = commissioner ? 'CommissionerId' : 'EndpointId';
  const rowType = commissioner ? 'CommissionerControl' : 'Endpoint';
  const tableName = commissioner ? 'commissionerControls' : 'endpoints';
  const lines = [
    '// Generated by scripts/gen-endpoints.mjs from docs/ui/endpoint-registry.csv. Do not edit.',
    ...(commissioner ? ["import type { EndpointId as OwnerEndpointId } from './endpoints.gen';"] : []),
    `export type ${idType} =`, ...rows.map((row) => `  | ${JSON.stringify(row.id)}`), ';',
    'export type Placement =', ...placements.map((place) => `  | ${JSON.stringify(place)}`), ';',
    'const places = [', ...placements.map(place => `  ${JSON.stringify(place)},`), '] as const;',
    `type ${rowType}Fields = {`,
    ...keys.filter((key) => !['disposition', 'placement', 'merged_into', 'merged_place'].includes(key))
      .map((key) => `  ${key}: ${
      key === 'id' ? idType : key === 'gravity' ? "'G0' | 'G1' | 'G2' | 'G3'" :
      key === 'ring' ? commissioner ? '0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9' : '0 | 1 | 2 | 3 | 4' :
      key === 'governs' ? "readonly (OwnerEndpointId | 'ledger-wide')[]" :
      key === 'disposition' ? "'kept' | 'merged' | 'retired'" :
      ['placement', 'merged_place'].includes(key) ? "Placement | ''" : 'string'
    };`),
    '};',
    ...(commissioner ? [
      `export type ${rowType} = Readonly<${rowType}Fields & {`,
      "  disposition: 'kept'; placement: Placement; merged_into: readonly []; merged_place: '';",
      '}>;',
    ] : [
      `export type ${rowType} = Readonly<${rowType}Fields & (`,
      "  | { disposition: 'kept'; placement: Placement; merged_into: readonly []; merged_place: '' }",
      "  | { disposition: 'retired'; placement: ''; merged_into: readonly []; merged_place: '' }",
      "  | { disposition: 'merged'; placement: ''; merged_into: readonly []; merged_place: Placement }",
      "  | { disposition: 'merged'; placement: '';",
      `      merged_into: readonly [${idType}, ...${idType}[]]; merged_place: '' }`,
      ')>;',
    ]), `export const ${tableName}: readonly ${rowType}[] = Object.freeze([`,
  ];
  for (const row of rows) {
    lines.push('  Object.freeze({');
    for (const key of keys) {
      const value = key === 'governs'
        ? `Object.freeze([\n${row[key].map(id => `      ${JSON.stringify(id)},`).join('\n')}\n    ] as const)`
        : key === 'merged_into' ? `Object.freeze(${JSON.stringify(row[key])} as const)`
        : ['placement', 'merged_place'].includes(key) && row[key]
          ? `places[${placements.indexOf(row[key])}]` : literal(row[key], commissioner ? 68 : 80);
      lines.push(`    ${key}: ${value},`);
    }
    lines.push('  } as const),');
  }
  return [...lines, ']);', ''].join('\n');
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  mkdirSync(dirname(outputPath), { recursive: true });
  const source = readFileSync(csvPath, 'utf8');
  writeFileSync(outputPath, generateEndpoints(source));
  writeFileSync(commissionerPath, generateCommissioner(source));
}
