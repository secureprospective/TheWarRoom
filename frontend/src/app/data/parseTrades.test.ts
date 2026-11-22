import { describe, expect, it } from 'vitest';
import { parseTrades } from './parseTrades';
import { parseReceipt } from './parseEnvelope';
import { goTradeReading, tradeReading, tradeReceipt } from './tradeTestData';

type Wire = { offers: Record<string, unknown>[] };
function wire(change: (value: Wire) => void): unknown {
  const value = goTradeReading() as Wire;
  change(value);
  return value;
}
function receipt(change: (trade: Record<string, unknown>, spec: Record<string, unknown>) => void) {
  const value = JSON.parse(JSON.stringify(tradeReceipt()));
  change(value.spec.expected.trade, value.spec);
  return value;
}

describe('trade boundary', () => {
  it('parses the Go reading, including a zero expiry', () => {
    expect(parseTrades(goTradeReading())).toEqual(tradeReading());
    const zero = wire((v) => { v.offers[0].expires = '0001-01-01T00:00:00Z'; });
    expect(parseTrades(zero).offers[0].expires).toBe('0001-01-01T00:00:00Z');
  });
  it('rejects malformed fields at their paths', () => {
    expect(() => parseTrades(wire((v) => { v.offers[0].give = null; })))
      .toThrow('trades.offers[0].give: expected array');
    expect(() => parseTrades(wire((v) => { v.offers[0].direction = 'sideways'; })))
      .toThrow('trades.offers[0].direction');
    expect(() => parseTrades(wire((v) => { v.offers[0].expires = ''; })))
      .toThrow('trades.offers[0].expires: expected timestamp');
    expect(() => parseTrades({ offers: null, provenance: tradeReading().provenance }))
      .toThrow('trades.offers: expected array');
  });
  it('parses a trade.accept receipt and holds its invariants', () => {
    expect(parseReceipt(JSON.parse(JSON.stringify(tradeReceipt())))).toEqual(tradeReceipt());
    expect(() => parseReceipt(receipt((t) => { t.accepting = '0007'; })))
      .toThrow('.expected.trade.offering: must differ from accepting');
    expect(() => parseReceipt(receipt((t) => { t.accepting = '0003'; })))
      .toThrow('.expected.trade.accepting: must equal franchiseId');
    expect(() => parseReceipt(receipt((t) => { t.acceptingGives = []; })))
      .toThrow('.expected.trade.acceptingGives: expected nonempty array');
    expect(() => parseReceipt(receipt((_t, spec) => {
      (spec.expected as Record<string, unknown>).player = '16195';
    }))).toThrow('.expected.player: forbidden for trade.accept');
    expect(() => parseReceipt(receipt((_t, spec) => { spec.intent = 'roster.ir'; })))
      .toThrow('.expected.trade: not a trade intent');
  });
});
