import { afterEach, describe, expect, it, vi } from 'vitest';
import { researchAPI, researchAvailable, subscribeResearch, subscribeResearchPlayer } from './api';
import { createResearchState } from './state';
import { commands } from '../commands/registry';
import wire from './wire.testdata.json';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const flush = () => vi.dynamicImportSettled();

describe('Research transport adapter', () => {
  it('uses the feature namespace for all six operations and decodes both readings', async () => {
    const player = { ...wire.players[0], report: wire.news, articles: wire.articles,
      howLabel: 'How this is counted' };
    const api = {
      TargetResearch: vi.fn(async () => wire),
      TargetResearchPlayer: vi.fn(async () => player),
      TargetResearchRefresh: vi.fn(async () => {}),
      TargetResearchNewsRefresh: vi.fn(async () => {}),
      TargetResearchStatus: vi.fn(async () => 'Reading fantasy points'),
      TargetOpenResearchArticle: vi.fn(async () => {}),
    };
    vi.stubGlobal('window', { go: { research: { API: api } } });
    const scope = { mine: '0001', search: '', position: '', team: '', group: 'QB',
      trend: 'topAdds', depth: 'team', injuries: 'all' };
    expect(researchAvailable()).toBe(true);
    expect((await researchAPI.read(scope)).news.id).toBe('M-058');
    expect(api.TargetResearch).toHaveBeenCalledWith(scope);
    expect((await researchAPI.player!(player.id)).id).toBe(player.id);
    expect(api.TargetResearchPlayer).toHaveBeenCalledWith(player.id);
    await researchAPI.refresh(true);
    expect(api.TargetResearchRefresh).toHaveBeenCalledWith(true);
    await researchAPI.news('0001');
    expect(api.TargetResearchNewsRefresh).toHaveBeenCalledWith('0001');
    expect(await researchAPI.status!()).toBe('Reading fantasy points');
    expect(api.TargetResearchStatus).toHaveBeenCalledOnce();
    await researchAPI.open('16162', 'rotowire640841');
    expect(api.TargetOpenResearchArticle).toHaveBeenCalledWith('16162', 'rotowire640841');
  });

  it('does not substitute fixtures when desktop transport is missing', async () => {
    for (const window of [undefined, {}]) {
      vi.stubGlobal('window', window);
      expect(researchAvailable()).toBe(false);
      const state = createResearchState(researchAPI, () => '');
      await state.load();
      expect(state.peek().error).toContain('no browser fixture is substituted');
      expect(state.peek().pending).toBe(false);
      const stop = subscribeResearch(() => { throw new Error('browser event fired'); });
      await flush();
      stop();
    }
  });

  it('delivers season events and unsubscribes on disposal', async () => {
    const unsubscribe = vi.fn();
    let callback: (() => void) | undefined;
    const EventsOnMultiple = vi.fn((_name: string, changed: () => void) => {
      callback = changed;
      return unsubscribe;
    });
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const changed = vi.fn();
    const stop = subscribeResearch(changed);
    await flush();
    expect(EventsOnMultiple).toHaveBeenCalledWith('target:season', changed, -1);
    callback?.();
    expect(changed).toHaveBeenCalledOnce();
    stop();
    expect(unsubscribe).toHaveBeenCalledOnce();
  });

  it('keeps Inspector on the existing alert bus and returns its cleanup', () => {
    const changed = vi.fn();
    const stop = vi.fn();
    const subscription = vi.spyOn(commands.alerts, 'onSeasonChange').mockImplementation(callback => {
      callback();
      return stop;
    });
    try {
      const dispose = subscribeResearchPlayer(changed);
      expect(subscription).toHaveBeenCalledWith(changed);
      expect(changed).toHaveBeenCalledOnce();
      dispose();
      expect(stop).toHaveBeenCalledOnce();
    } finally {
      subscription.mockRestore();
    }
  });

  it('does not subscribe after disposal before runtime import resolution', async () => {
    const EventsOnMultiple = vi.fn();
    vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
    const stop = subscribeResearch(vi.fn());
    stop();
    await flush();
    expect(EventsOnMultiple).not.toHaveBeenCalled();
  });
});
