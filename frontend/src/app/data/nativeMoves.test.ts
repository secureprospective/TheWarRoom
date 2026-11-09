import { describe, expect, it } from 'vitest';
import wire from './native-moves.wire.testdata.json';
import { parseNativeMoves } from './nativeMoves';
import { parseReceipt } from './parseEnvelope';

const copy = () => JSON.parse(JSON.stringify(wire));

describe('native HQ move receipt adapter', () => {
  it('reads real Go receipts without pretending they are MFL receipts', () => {
    const r = parseNativeMoves(wire);
    expect(r.planned).toEqual([]);
    expect(r.reason).toBe('native plans not yet stored');
    expect(r.moves.map(m => m.state)).toContain('handed_off');
    expect(r.moves.map(m => m.state)).toContain('needs_you');
    expect(r.moves.every(m => m.spec.target.kind === 'native')).toBe(true);
    for (const receipt of wire.moves) expect(() => parseReceipt(receipt)).toThrow();
  });

  it.each(['handed_off', 'needs_you', 'landed'])('accepts the native %s state', state => {
    const raw = copy();
    raw.moves[0].state = state;
    raw.moves[0].audit[0].to = state;
    expect(parseNativeMoves(raw).moves[0].state).toBe(state);
  });

  it.each(['state', 'scope', 'target', 'audit', 'plans', 'url', 'unknown'])(
    'rejects malformed %s', change => {
    const raw = copy();
    if (change === 'state') raw.moves[0].state = 'ready';
    if (change === 'scope') raw.moves[0].spec.franchiseId = 'franchise.99';
    if (change === 'target') raw.moves[0].spec.target.kind = 'mapped';
    if (change === 'audit') raw.moves[0].audit[0].event = 'match';
    if (change === 'plans') raw.planned = raw.moves;
    if (change === 'url') raw.moves[0].spec.target.url = 'https://example.test';
    if (change === 'unknown') raw.moves[0].spec.expected.player = '5001';
    expect(() => parseNativeMoves(raw)).toThrow();
  });
});
