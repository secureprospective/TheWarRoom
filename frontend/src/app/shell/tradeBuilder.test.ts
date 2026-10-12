import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import tradeWire from './trade-wire.testdata.json';
import acquisitionWire from '../contracts/acquisition-wire.testdata.json';
import { parseBuilder } from './tradeBuilder';
import { parseAcquisition } from '../contracts/acquisition';
import { parseReceipt } from '../data/parseEnvelope';
import { TradePlan } from './TradePlan';
import { tradeReceipt } from '../data/tradeTestData';
import { endpointById } from '../registry';

describe('Trade builder and acquisition wire boundaries', () => {
  it('reads Go-authored real roster cents, names and full projected rosters', () => {
    const r = parseBuilder(tradeWire);
    expect(r.value!.preview!.teams).toHaveLength(2);
    const teams = r.value!.preview!.teams;
    expect(teams[0].figures[0].after).toBe('$110,170,000.46');
    expect(teams[1].figures[0].after).toBe('$125,859,999.54');
    expect(teams[0].roster).toHaveLength(35);
    expect(teams[1].roster).toHaveLength(48);
    expect(r.value!.preview!.post).toContain('Shakir, Khalil');
    expect(r.value!.preview!.post).not.toContain('15761');
    expect(() => parseBuilder({ ...tradeWire, value: { ...tradeWire.value, extra: 1 } })).toThrow();
  });
  it('reads the forced cut charge and incoming season costs without client arithmetic', () => {
    const r = parseAcquisition(acquisitionWire);
    expect(r.value!.deadCap).toEqual({ cents: 945000000, text: '$9,450,000' });
    expect(r.value!.seasons).toHaveLength(3);
    expect(r.value!.team[0].after).toBe('$129,380,000');
    expect(r.value!.cutName).toBe('McLaurin, Terry');
    const view = readFileSync('src/app/contracts/Acquisitions.tsx', 'utf8');
    expect(view).toContain('Incoming cap hit by season');
    expect(view).toContain('cut dead cap this season');
    expect(view).not.toContain('reduce(');
  });
  it('validates the actor for all new receipts and uses G3 safe hand-offs', () => {
    for (const intent of ['trade.propose', 'trade.reject', 'trade.revoke']) {
      const r = tradeReceipt('ready');
      r.spec.intent = intent;
      if (intent !== 'trade.reject') r.spec.franchiseId = r.spec.expected.trade!.offering;
      expect(parseReceipt(r).spec.intent).toBe(intent);
      const wrong = structuredClone(r);
      wrong.spec.franchiseId = '9999';
      expect(() => parseReceipt(wrong)).toThrow('must equal franchiseId');
    }
    const withdraw = tradeReceipt('ready');
    withdraw.spec.intent = 'trade.revoke';
    withdraw.spec.franchiseId = withdraw.spec.expected.trade!.offering;
    const html = renderToStaticMarkup(createElement(TradePlan, { receipt: withdraw }));
    expect(html).toContain('Open MFL&#x27;s trade desk to withdraw it');
    expect(html).toContain('withdraw link acts at once');
    const source = readFileSync('src/app/commands/registry.ts', 'utf8');
    expect(source).toContain("id: 'trade.builder', label: 'Trade builder', gravity: 'G3'");
  });
  it('records built rows, the waiver deferral and Christopher’s retirement', () => {
    for (const id of ['M-033', 'M-034', 'M-199', 'C-06.2', 'C-06.4'] as const)
      expect(endpointById(id).status).not.toBe('spec');
    expect(endpointById('M-192').disposition).toBe('retired');
    expect(endpointById('M-192').placement).toBe('');
    expect(endpointById('M-192').disposition_note).toContain('retired 2026-10-11 (Christopher)');
    expect(endpointById('M-024').ring).toBe(4);
    expect(endpointById('M-024').disposition_note).toContain('native claim/order mechanics remain open');
  });
});
