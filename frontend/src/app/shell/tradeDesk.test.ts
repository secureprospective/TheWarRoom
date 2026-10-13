import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { commands, createCommands } from '../commands/registry';
import '../commands/tradeAccept';
import { LiveProvider } from '../data/provider';
import type { Receipt, TradeReading } from '../data/contract';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { dotNote, tradeOffer, tradeReading, tradeReceipt } from '../data/tradeTestData';
import { clockDom, TestElement } from '../clock/testDom';
import TradeDesk, { tradeExpiry } from './TradeDesk';
import { EnvelopeRail } from './EnvelopeRail';
import EntityInspector from './EntityInspector';

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => { for (const root of roots.splice(0)) root.unmount(); });
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
async function flush() {
  for (let i = 0; i < 4; i += 1) await new Promise((resolve) => setTimeout(resolve, 0));
}
function buttons(element: TestElement): string[] {
  return element.children.flatMap((child): string[] => {
    if (!(child instanceof TestElement)) return [];
    return child.tagName === 'BUTTON' ? [child.textContent] : buttons(child);
  });
}
function hasLabel(element: TestElement, label: string): boolean {
  return element.getAttribute('aria-label') === label || element.children.some(
    (child) => child instanceof TestElement && hasLabel(child, label));
}
async function setup(reading: TradeReading = tradeReading(), held: Receipt[] = []) {
  const snapshot = heldSnapshot();
  snapshot.franchises.value.push({ id: '0007', name: 'Rivals' });
  const provider = new LiveProvider();
  const trades = vi.spyOn(provider, 'trades').mockResolvedValue(reading);
  vi.spyOn(provider, 'lineup').mockResolvedValue(heldLineup());
  const drafted = vi.spyOn(provider, 'draftTradeAccept').mockResolvedValue(tradeReceipt());
  const moves = vi.spyOn(provider, 'moves').mockResolvedValue(held);
  const listeners = new Set<() => void>();
  vi.spyOn(provider, 'onMovesChange').mockImplementation((listener) => {
    listeners.add(listener);
    return () => { listeners.delete(listener); };
  });
  vi.spyOn(provider, 'onSeasonChange').mockImplementation(() => () => {});
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: '0025' });
  await c.trades.refresh('0025');
  for (const key of ['use', 'useMoves', 'loadMoves', 'onMovesChange'] as const) {
    vi.spyOn(commands, key).mockImplementation(c[key]);
  }
  for (const key of ['refresh', 'peek', 'subscribe', 'onSeasonChange', 'onMovesChange'] as const) {
    vi.spyOn(commands.trades, key).mockImplementation(c.trades[key]);
  }
  const dom = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
  const root = createRoot(dom.element);
  roots.push(root);
  await act(async () => { root.render(createElement(TradeDesk, { snapshot })); await flush(); });
  const element = dom.element as unknown as TestElement;
  return {
    c, snapshot, root, element, trades, drafted, moves,
    text: () => element.textContent,
    buttons: () => buttons(element),
    plan: async () => {
      await act(async () => { c.dispatch('trade.accept.plan', { tradeId: '901' }); await flush(); });
    },
    event: async () => {
      await act(async () => { for (const listener of listeners) listener(); await flush(); });
    },
  };
}
const handedText = "Opened MFL's trade desk. Accept offer 901 there.";

describe('Trade desk', () => {
  it('lists both groups with names, positions, unnamed assets, expiry and comments', async () => {
    const t = await setup(tradeReading([tradeOffer('to_you', '901'), tradeOffer('by_you', '902')]));
    for (const part of [
      'Offered to you · 1', 'Your offers · 1', 'Rivals', 'You give', 'You get', 'Given, Player (WR)',
      'Unnamed asset', 'Talked on ProBoards',
    ]) expect(t.text()).toContain(part);
    expect(t.buttons().filter((label) => label === 'Plan accept')).toHaveLength(1);
    expect(hasLabel(t.element, 'Trade accept plan')).toBe(false);
  });
  it.each(['by_you', 'to_you'] as const)(
    'hides a removed exit plan beside a pending %s offer on the desk and inspector', async (direction) => {
      const receipt = tradeReceipt();
      receipt.spec.intent = direction === 'by_you' ? 'trade.revoke' : 'trade.reject';
      receipt.state = 'withdrawn';
      receipt.audit.push({ at: '2026-10-07T21:01:00Z', from: 'ready', to: 'withdrawn',
        event: 'withdraw', note: 'Removed by you before it was handed off.' });
      const t = await setup(tradeReading([tradeOffer(direction)]), [receipt]);
      await t.event();
      const check = () => {
        expect(t.text()).toContain('Pending offer on MFL');
        expect(t.text()).not.toContain('Withdrawn');
        expect(t.text()).not.toContain('Removed by you');
        expect(t.buttons()).toContain(direction === 'by_you' ? 'Plan withdrawal' : 'Plan rejection');
        expect(t.buttons()).not.toContain('Remove plan');
      };
      check();
      await act(async () => {
        t.root.render(createElement(EntityInspector, {
          entity: { kind: 'offer', id: '901', franchiseId: '0025' }, snapshot: t.snapshot,
        }));
        await flush();
      });
      check();
    },
  );
  it.each(['by_you', 'to_you'] as const)(
    'keeps a Ready exit plan and its removal beside a pending %s offer', async (direction) => {
      const receipt = tradeReceipt();
      receipt.spec.intent = direction === 'by_you' ? 'trade.revoke' : 'trade.reject';
      const t = await setup(tradeReading([tradeOffer(direction)]), [receipt]);
      await t.event();
      expect(t.text()).toContain('Pending offer on MFL');
      expect(t.text()).toContain('Complete this yourself on MFL');
      expect(t.buttons()).toContain('Remove plan');
      expect(t.buttons()).toContain(direction === 'by_you'
        ? "Open MFL's trade desk to withdraw it" : "Open MFL's trade desk to reject");
      expect(t.buttons()).not.toContain(direction === 'by_you' ? 'Plan withdrawal' : 'Plan rejection');
    },
  );
  it('states relative expiry and never invents one', () => {
    const now = Date.parse('2026-10-07T14:00:00Z');
    expect(tradeExpiry('2026-10-09T18:00:00Z', now)).toBe('expires in 2d 4h');
    expect(tradeExpiry('2026-10-07T14:30:00Z', now)).toBe('expires in less than 1h');
    expect(tradeExpiry('2026-10-07T13:00:00Z', now)).toBe('expired');
    expect(tradeExpiry('0001-01-01T00:00:00Z', now)).toBe('Expiry unknown');
  });
  it('says when there are no offers, and hides offers when the feed failed', async () => {
    const empty = await setup(tradeReading([]));
    expect(empty.text()).toContain('No pending offers');
    expect(empty.text()).not.toContain('Offered to you');
    expect(empty.text()).not.toContain('Your offers');
    const fresh = heldLineup().provenance;
    const failed = await setup(tradeReading([tradeOffer()], {
      ...fresh, freshness: { ...fresh.freshness, state: 'fail', note: 'MFL timed out' },
    }));
    expect(failed.text()).toContain('Pending trades unavailable · MFL timed out');
    expect(failed.text()).not.toContain('Given, Player');
  });
  it('plans, hands off, and follows the plan through DOT review to Landed', async () => {
    const t = await setup();
    const blocked = tradeReceipt('blocked', 'offer 901 is no longer pending');
    t.drafted.mockResolvedValueOnce(blocked);
    t.moves.mockResolvedValue([blocked]);
    await t.plan();
    expect(t.drafted).toHaveBeenCalledWith('0025', '901');
    expect(t.text()).toContain('Not ready · offer 901 is no longer pending');
    expect(t.buttons()).toContain('Plan accept');

    t.moves.mockResolvedValue([tradeReceipt('ready')]);
    await t.plan();
    expect(t.text()).toContain("Accept this offer on MFL's trade desk; this plan does not accept it.");
    expect(t.buttons()).toContain('Open MFL trade desk');
    expect(t.buttons()).not.toContain('Plan accept');

    t.moves.mockResolvedValue([tradeReceipt('dot_review', dotNote)]);
    t.trades.mockResolvedValue(tradeReading([]));
    await t.event();
    expect(t.text()).toContain('Saved trade plans');
    expect(t.text()).toContain(`DOT review · ${dotNote}`);
    expect(t.text()).toContain(handedText);

    t.moves.mockResolvedValue([tradeReceipt('landed', 'TRADE row matched')]);
    await t.event();
    expect(t.text()).toContain('Landed · TRADE row matched');
    expect(t.text()).not.toContain(handedText);
  });
  it('keeps unhanded plans for vanished offers off the desk', async () => {
    const t = await setup(tradeReading([]), [tradeReceipt('blocked', 'offer 901 is no longer pending')]);
    await t.event();
    expect(t.text()).not.toContain('Saved trade plans');
  });
  it('drafts once for repeated clicks', async () => {
    const t = await setup();
    let resolve!: (receipt: Receipt) => void;
    t.drafted.mockReturnValueOnce(new Promise((done) => { resolve = done; }));
    await act(async () => {
      t.c.dispatch('trade.accept.plan', { tradeId: '901' });
      t.c.dispatch('trade.accept.plan', { tradeId: '901' });
      await flush();
    });
    expect(t.drafted).toHaveBeenCalledTimes(1);
    expect(t.buttons()).toContain('Planning accept…');
    await act(async () => { resolve(tradeReceipt()); await flush(); });
  });
  it('heads the My moves rail with the offer, the other franchise and the swap', async () => {
    const t = await setup();
    const last = t.snapshot.players.value.find((p) => p.id === '16195')!.name!.split(',')[0];
    await act(async () => {
      t.root.render(createElement(EnvelopeRail, { receipt: tradeReceipt(), snapshot: t.snapshot }));
    });
    expect(t.text()).toContain('Offer 901 · Rivals');
    expect(t.text()).toContain(`+Unnamed asset −${last}`);
    expect(t.text()).toContain('Trade accept');
    expect(t.buttons()).toContain('Open MFL trade desk');
  });
});
