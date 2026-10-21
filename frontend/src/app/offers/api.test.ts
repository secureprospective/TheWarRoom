import { afterEach, describe, expect, it, vi } from 'vitest';
import { personalWire, poolWire, rawBid, readOffersWire, subscribeOffers,
  subscribePersonal, subscribePool } from './api';
import { commands } from '../commands/registry';
import type { BidFormState } from '../pool/contract';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  vi.restoreAllMocks();
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const flush = async () => {
  await vi.dynamicImportSettled();
  await new Promise(resolve => setTimeout(resolve, 20));
};
const form: BidFormState = {
  planId: '', playerId: '13593', playerName: 'Smith', salary: '5', years: 3,
  snipe: false, beat: '', evidence: '', closesOpen: false, closesDate: '', closesTime: '',
};

describe('Offers cluster transport adapter', () => {
  it('routes all thirteen bindings through the feature namespace', async () => {
    const names = ['TargetOffers', 'TargetPersonal', 'TargetPersonalChange', 'TargetPersonalHandOff',
      'TargetPool', 'TargetPoolRefresh', 'TargetBidPlans', 'TargetBidPreview', 'TargetBidDraft',
      'TargetBidRevise', 'TargetBidHandOff', 'TargetBidPosted', 'TargetBidWithdraw'];
    const api = Object.fromEntries(names.map(name => [name, vi.fn(async () => ({ method: name }))]));
    vi.stubGlobal('window', { go: { offers: { API: api } } });
    const scope = { from: 2013, to: 2026, direction: 'all', window: '', team: '' };
    await readOffersWire(scope);
    expect(api.TargetOffers).toHaveBeenCalledWith(scope);
    await personalWire.read('0001');
    await personalWire.change('0001', 'watch', { action: 'add', entity: 'player', id: '13593' });
    await personalWire.handoff('watch');
    expect(api.TargetPersonal).toHaveBeenCalledWith('0001');
    expect(api.TargetPersonalChange).toHaveBeenCalledWith({
      mine: '0001', kind: 'watch', action: 'add', entity: 'player', id: '13593', text: '',
    });
    expect(api.TargetPersonalHandOff).toHaveBeenCalledWith('watch');
    await poolWire.refresh(true);
    await poolWire.pool('0001', { position: 'WR', sort: 'name', desc: false });
    expect(api.TargetPool).toHaveBeenCalledWith({
      franchiseId: '0001', position: 'WR', sort: 'name', desc: false, showNoTeam: false,
    });
    expect(api.TargetPoolRefresh).toHaveBeenCalledWith(true);
    await poolWire.plans('0001');
    expect(api.TargetBidPlans).toHaveBeenCalledWith('0001');
    await poolWire.preview('0001', form);
    await poolWire.draft('0001', form);
    await poolWire.revise('bid-1', '0001', form);
    expect(api.TargetBidPreview).toHaveBeenCalledWith(rawBid('0001', form));
    expect(api.TargetBidDraft).toHaveBeenCalledWith(rawBid('0001', form));
    expect(api.TargetBidRevise).toHaveBeenCalledWith('bid-1', rawBid('0001', form));
    for (const action of ['handOff', 'posted', 'withdraw'] as const) await poolWire[action]('bid-1');
    for (const method of ['TargetBidHandOff', 'TargetBidPosted', 'TargetBidWithdraw'])
      expect(api[method]).toHaveBeenCalledWith('bid-1');
    for (const method of names) expect(api[method]).toHaveBeenCalledOnce();
  });

  it('keeps browser refusal behavior instead of substituting local plans', async () => {
    for (const window of [undefined, {}]) {
      vi.stubGlobal('window', window);
      expect(() => personalWire.read('0001')).toThrow('Private lists need the desktop app');
      expect(() => poolWire.plans('0001')).toThrow('The free-agent pool needs the desktop app');
      await expect(readOffersWire({ from: 2013, to: 2026, direction: 'all', window: '', team: '' }))
        .rejects.toThrow();
    }
  });

  it('delivers season events and removes both pool subscriptions', async () => {
    const listeners = new Map<string, () => void>();
    const stops = new Map<string, ReturnType<typeof vi.fn>>();
    const EventsOnMultiple = vi.fn((name: string, changed: () => void) => {
      listeners.set(name, changed);
      const stop = vi.fn();
      stops.set(name, stop);
      return stop;
    });
    const moveStop = vi.fn();
    let moveListener = () => {};
    vi.spyOn(commands, 'onMovesChange').mockImplementation(changed => {
      moveListener = changed; return moveStop;
    });
    vi.stubGlobal('window', { go: {}, runtime: { EventsOnMultiple } });
    const changed = vi.fn();
    const stop = subscribePool(changed);
    try {
      await flush();
      expect([...listeners.keys()]).toEqual(['target:season']);
      changed.mockClear();
      moveListener();
      listeners.get('target:season')?.();
      expect(changed).toHaveBeenCalledTimes(2);
    } finally { stop(); }
    expect(moveStop).toHaveBeenCalledOnce();
    for (const cleanup of stops.values()) expect(cleanup).toHaveBeenCalledOnce();
  });

  it('does not subscribe after disposal before the runtime import resolves', async () => {
    const EventsOnMultiple = vi.fn();
    vi.stubGlobal('window', { go: {}, runtime: { EventsOnMultiple } });
    const stops = [subscribeOffers(vi.fn()), subscribePool(vi.fn()), subscribePersonal(vi.fn())];
    for (const stop of stops) stop();
    await flush();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });

  it('retains the personal-list alert bus and its cleanup', () => {
    const stop = vi.fn();
    const listener = vi.fn();
    vi.spyOn(commands.alerts, 'onSeasonChange').mockImplementation(changed => { changed(); return stop; });
    const dispose = subscribePersonal(listener);
    expect(listener).toHaveBeenCalledOnce();
    dispose();
    expect(stop).toHaveBeenCalledOnce();
  });
});
