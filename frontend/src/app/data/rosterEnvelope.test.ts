import { describe, expect, it } from 'vitest';
import { parseReceipt } from './parseEnvelope';
import ir from './fixtures/envelope-demo.json';

describe('IR and taxi receipt boundary', () => {
  it.each(['IR', 'TAXI_SQUAD', 'ROSTER'])('accepts taxi status %s without changing the receipt', (status) => {
    const receipt = structuredClone(ir.receipt);
    receipt.spec.intent = 'roster.taxi';
    receipt.spec.expected.rosterStatus = status;
    expect(parseReceipt(receipt)).toEqual(receipt);
  });
  it.each(['TAXI', 'active', '', null, 1])('rejects bad status %s at its path', (status) => {
    const receipt = {
      ...ir.receipt,
      spec: {
        ...ir.receipt.spec, intent: 'roster.taxi',
        expected: { ...ir.receipt.spec.expected, rosterStatus: status },
      },
    };
    expect(() => parseReceipt(receipt)).toThrow('receipt.spec.expected.rosterStatus');
  });
});
