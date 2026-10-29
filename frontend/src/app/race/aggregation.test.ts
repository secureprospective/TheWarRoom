import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import { clockDom, TestElement } from '../clock/testDom';
import { commands } from '../commands/registry';
import Race from './Race';
import wire from './wire.testdata.json';

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map(key => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
afterEach(() => {
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
function descendants(node: TestElement): TestElement[] {
  return [node, ...node.children.flatMap(child => child instanceof TestElement ? descendants(child) : [])];
}
function props(node: TestElement) {
  const key = Object.keys(node).find(key => key.startsWith('__reactProps'))!;
  return (node as unknown as Record<string, {
    value: string; onChange(event: { target: { value: string } }): void;
  }>)[key];
}

it('starts with Starting lineup and re-reads the power board for each roster count', async () => {
  const { document, root: host, element } = clockDom([]);
  const TargetRace = vi.fn(async () => wire.race);
  const TargetRaceRefresh = vi.fn(async () => {});
  vi.stubGlobal('document', document);
  vi.stubGlobal('window', { document, HTMLIFrameElement: class {},
    go: { race: { API: { TargetRace, TargetRaceRefresh } } } });
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  commands.dispatch('endpoint.open', { id: 'M-008' });
  const root = createRoot(element);
  const select = () => descendants(host).find(node => node.tagName === 'SELECT')!;
  try {
    await act(async () => { root.render(createElement(Race)); });
    expect(select()).toBeDefined();
    expect(select().options.map(option => option.textContent))
      .toEqual(['Whole roster', 'Starting lineup', 'Best 21 (any position)']);
    expect(props(select()).value).toBe('lineup');
    expect(TargetRace.mock.calls[0]).toEqual([0, '', '', 'lineup']);
    const refreshes = TargetRaceRefresh.mock.calls.length;
    for (const mode of ['sum', 'topn', 'lineup']) {
      await act(async () => {
        await props(select()).onChange({ target: { value: mode } });
      });
      expect(TargetRace).toHaveBeenLastCalledWith(3, '0001', '', mode);
      expect(props(select()).value).toBe(mode);
    }
    expect(TargetRaceRefresh).toHaveBeenCalledTimes(refreshes);
    await act(async () => { commands.dispatch('endpoint.open', { id: 'C-02' }); });
    expect(select()).toBeDefined();
    expect(props(select()).value).toBe('lineup');
    await act(async () => { commands.dispatch('endpoint.open', { id: 'M-161' }); });
    expect(select()).toBeUndefined();
  } finally {
    await act(async () => { root.unmount(); });
  }
});
