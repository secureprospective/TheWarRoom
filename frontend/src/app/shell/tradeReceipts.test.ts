import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it } from 'vitest';
import { heldSnapshot } from '../data/lineupTestData';
import { tradeReceipt } from '../data/tradeTestData';
import { parseReceipt } from '../data/parseEnvelope';
import { EnvelopeRail } from './EnvelopeRail';
import { TradePlan, RemoveTradePlan } from './TradePlan';
import { tradeAssetLabel } from './tradeLabels';

it('renders every named pick on the receipt and never exposes an unknown token', () => {
  const receipt = tradeReceipt();
  receipt.tradeAssets = [{ token: 'FP_0007_2027_1', name: '2027 round 1 · Arizona Cardinals' }];
  receipt.spec.expected.trade!.acceptingGives.push('unknown');
  const parsed = parseReceipt(JSON.parse(JSON.stringify(receipt)));
  const html = renderToStaticMarkup(createElement(EnvelopeRail, {
    receipt: parsed, snapshot: heldSnapshot(),
  }));
  expect(html).toContain('+2027 round 1 · Arizona Cardinals');
  expect(html).toContain('Unnamed asset');
  expect(html).not.toContain('FP_');
  expect(html).not.toContain('unknown');
  expect(tradeAssetLabel({ token: 'unknown', name: '' })).toBe('Unnamed asset');
  expect(() => parseReceipt({ ...receipt, tradeAssets: [{ token: 'pick', name: 1 }] })).toThrow();
});

it('offers the same local removal on the rail and trade desk only before hand-off', () => {
  for (const intent of ['trade.reject', 'trade.revoke', 'trade.propose']) {
    const receipt = tradeReceipt();
    receipt.spec.intent = intent;
    const remove = RemoveTradePlan({ receipt })!;
    expect(remove.props.verb).toBe('trade.builder');
    expect(remove.props.args).toEqual({ action: 'remove', mine: '0025', id: receipt.correlationId });
    for (const component of [TradePlan, EnvelopeRail]) {
      const markup = () => renderToStaticMarkup(createElement(component, {
        receipt, snapshot: heldSnapshot(),
      }));
      expect(markup()).toContain('Remove plan');
      receipt.audit.push({ at: '2026-10-07T21:01:00Z', from: 'ready', to: 'handed_off',
        event: 'hand_off', note: '' });
      expect(markup()).not.toContain('Remove plan');
      receipt.audit.pop();
      receipt.state = 'withdrawn';
      expect(markup()).not.toContain('Remove plan');
      receipt.state = 'ready';
    }
  }
  expect(RemoveTradePlan({ receipt: tradeReceipt() })).toBeNull();
});
