import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { createProxy, createLedgerState, parseLedger, parseReceipt, type Port } from './proxy';
import { LedgerEntries } from './LeagueLedger';
import wire from './wire.testdata.json';

const receipt = {
  entry: 'act.test', league: wire.league, revision: 3, recordedAt: '2026-10-13T12:00:00Z',
  message: 'Recorded in the league ledger.',
};
const port = (over: Partial<Port> = {}): Port => ({
  execute: async () => receipt, ledger: async () => wire,
  leagues: async () => [{ id: wire.league, name: wire.name }], ...over,
});

describe('Commissioner spine', () => {
  it('reads the real Go owner wire without private entries and labels every value', () => {
    const reading = parseLedger(wire);
    expect(reading.entries).toHaveLength(1);
    expect(JSON.stringify(reading)).not.toContain('Private correction');
    expect(reading.entries[0].what).toBe('Created league “Our league”');
    const html = renderToStaticMarkup(createElement(LedgerEntries, { reading }));
    for (const label of ['League version: 2', 'By creator', 'Reason:', 'Safeguard:', 'Start our own league'])
      expect(html).toContain(label);
    expect(html).not.toContain('G3');
  });
  it('passes the exact command to Go and returns its receipt, never choosing gravity or role', async () => {
    let sent = '';
    const proxy = createProxy(port({ execute: async body => { sent = body; return receipt; } }));
    const body = '{"command":"commish.correction.revert","reason":"typed by human","gravity":"G0"}';
    expect(await proxy.execute(body)).toEqual(receipt);
    expect(sent).toBe(body);
    expect(await proxy.ledger(wire.league)).toEqual(parseLedger(wire));
    expect(await proxy.leagues()).toEqual([{ id: wire.league, name: wire.name }]);
  });
  it('returns Go refusals without fabricating a landed receipt', async () => {
    const proxy = createProxy(port({ execute: async () => {
      throw new Error('give a reason for this change');
    } }));
    await expect(proxy.execute('{}')).rejects.toThrow('give a reason');
    expect(() => parseReceipt({ ...receipt, revision: 0 })).toThrow();
    expect(() => parseLedger({ ...wire, privatePayload: 'secret' })).toThrow();
    expect(() => parseLedger({ ...wire, entries: [{ ...wire.entries[0], role: 'commissioner' }] })).toThrow();
  });
  it('refreshes and selects leagues through the registered reader, clearing failed readings', async () => {
    const state = createLedgerState(createProxy(port()));
    await state.choose();
    expect(state.peek().selected).toBe(wire.league);
    expect(state.peek().reading?.entries).toHaveLength(1);
    expect(state.peek().pending).toBe(false);
    await state.choose('native.missing');
    expect(state.peek().error).toContain('Choose another league');
    expect(state.peek().reading).toBeUndefined();
    await state.choose();
    expect(state.peek().error).toBe('');
    const view = readFileSync(new URL('./LeagueLedger.tsx', import.meta.url), 'utf8');
    expect(view).toContain('verb="commissioner.read"');
    expect(view).not.toContain('<' + 'button');
  });
  it('keeps the proxy behind a dynamic import and the ledger inside the existing lazy League room', () => {
    const registry = readFileSync(new URL('../commands/registry.ts', import.meta.url), 'utf8');
    const room = readFileSync(new URL('../control/LeagueRoom.tsx', import.meta.url), 'utf8');
    expect(registry).toContain("import('../commissioner/proxy')");
    expect(registry).not.toMatch(/import .* from ['"].*commissioner/);
    expect(room).toContain("lazy(() => import('../commissioner/LeagueLedger'))");
    const budget = readFileSync(new URL('../../../scripts/bundle-budget.mjs', import.meta.url), 'utf8');
    expect(budget).toContain('Commissioner proxy in owner entry');
  });
});
