import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { parseReceipt } from './parseEnvelope';
import { lineupReceipt } from './lineupReceiptTestData';
import ir from './fixtures/envelope-demo.json';

describe('intent-aware lineup boundary', () => {
  it('parses the Go DraftLineup test starter set and keeps IR receipts unchanged', () => {
    const raw: { id: string; position: string }[] = JSON.parse(readFileSync(
      '../internal/lineup/testdata/players-0025.json', 'utf8',
    ));
    const receipt = lineupReceipt();
    const baseline = raw.map((p) => p.id);
    const firstWR = raw.find((p) => p.position === 'WR')!.id;
    const starters = baseline.map((id) => id === firstWR ? '16387' : id);
    receipt.spec.expected.lineup = { week: 5, baseline, starters };
    receipt.spec.subject.players = [...starters].reverse();
    expect(parseReceipt(receipt)).toEqual(receipt);
    expect(parseReceipt(ir.receipt)).toEqual(ir.receipt);
    receipt.spec.expected.lineup.baseline = [];
    expect(parseReceipt(receipt).spec.expected.lineup?.baseline).toEqual([]);
  });
  it.each([
    ['lineup', undefined], ['lineup', null], ['lineup.week', 0], ['lineup.week', 1.5],
    ['lineup.starters', []], ['lineup.starters', ['16150', '16150']],
    ['lineup.starters', null], ['lineup.starters', ['bad']],
    ['lineup.baseline', undefined], ['lineup.baseline', null], ['lineup.baseline', ['bad']],
    ['player', '16150'], ['rosterStatus', 'ROSTER'], ['lineup.extra', 'unexpected'],
  ])('rejects expected.%s at its path', (field, value) => {
    const receipt = JSON.parse(JSON.stringify(lineupReceipt()));
    const parts = field.split('.');
    if (parts.length === 1) receipt.spec.expected[field] = value;
    else receipt.spec.expected.lineup[parts[1]] = value;
    expect(() => parseReceipt(receipt)).toThrow(`receipt.spec.expected.${field}`);
  });
  it.each([{ players: [] }, { players: ['16150'] }, { players: ['16150', '16150'] }])(
    'requires subjects to equal starters: $players', ({ players }) => {
    const receipt = lineupReceipt();
    receipt.spec.subject.players = players;
    expect(() => parseReceipt(receipt)).toThrow('receipt.spec.subject.players');
    },
  );
  it('forbids lineup fields on IR', () => {
    const receipt = JSON.parse(JSON.stringify(ir.receipt));
    receipt.spec.expected.lineup = lineupReceipt().spec.expected.lineup;
    expect(() => parseReceipt(receipt)).toThrow('receipt.spec.expected.lineup');
  });
});
