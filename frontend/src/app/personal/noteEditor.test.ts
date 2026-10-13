import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { clockDom, TestElement } from '../clock/testDom';
import { commands } from '../commands/registry';
import { renderState } from '../shell/state';
import { NoteEditor } from './PersonalLists';
import { personalState } from './state';
import { parsePersonal, type Entry, type ListChange } from './contract';
import wire from './wire.testdata.json';

function find(root: TestElement, tag: string): TestElement {
  if (root.tagName === tag) return root;
  for (const child of root.children) {
    if (!(child instanceof TestElement)) continue;
    try { return find(child, tag); } catch { /* Search the next branch. */ }
  }
  throw new Error(`Missing ${tag}`);
}
function props(node: TestElement) {
  const key = Object.keys(node).find(key => key.startsWith('__reactProps'))!;
  return (node as unknown as Record<string, {
    onChange(e: { target: { value: string } }): void; onClick(): void; value: string; disabled: boolean;
  }>)[key];
}
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

it('clears kept notes, acknowledges locally and lists only this player newest first', async () => {
  const { document, root: host, element } = clockDom([]);
  const entries: Entry[] = [
    { kind: 'note', entity: 'player', id: '0816', text: 'Older', at: '2026-10-10T10:00:00Z', order: 1 },
    { kind: 'note', entity: 'player', id: '0817', text: 'Other player',
      at: '2026-10-11T10:00:00Z', order: 2 },
  ];
  const change = vi.fn(async (raw: ListChange) => {
    entries.push({ kind: 'note', entity: raw.entity, id: raw.id, text: raw.text!,
      at: '2026-10-13T10:00:00Z', order: 3 });
  });
  vi.stubGlobal('document', document);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('window', { document, HTMLIFrameElement: class {}, go: { main: { App: {
    TargetPersonal: async () => ({ ...parsePersonal(wire), entries }), TargetPersonalChange: change,
  } } } });
  vi.spyOn(commands, 'use').mockReturnValue({ ...renderState(commands.read()), franchiseId: '0001' });
  const { FixtureProvider } = await import('../data/provider');
  commands.loadSnapshot(await new FixtureProvider().snapshot());
  commands.dispatch('franchise.set', { franchiseId: '0001' });
  await personalState.load('0001');
  const root = createRoot(element);
  try {
    await act(async () => { root.render(createElement(NoteEditor, { id: '0816', entity: 'player' })); });
    await act(async () => { props(find(host, 'TEXTAREA')).onChange({ target: { value: 'New note' } }); });
    expect(props(find(host, 'TEXTAREA')).value).toBe('New note');
    await act(async () => {
      props(find(host, 'BUTTON')).onClick();
      await new Promise(resolve => setTimeout(resolve, 25));
    });
    expect(props(find(host, 'TEXTAREA')).value).toBe('');
    expect(props(find(host, 'BUTTON')).disabled).toBe(true);
    expect(host.textContent).toContain('Kept ·');
    expect(host.textContent).toContain('New note');
    expect(host.textContent).not.toContain('Other player');
    expect(host.textContent.indexOf('New note')).toBeLessThan(host.textContent.indexOf('Older'));
    expect(change).toHaveBeenCalledTimes(1);
  } finally { await act(async () => { root.unmount(); }); }
});

it('does not acknowledge a failed write', async () => {
  const { createPersonalState } = await import('./state');
  const state = createPersonalState({ read: async () => parsePersonal(wire),
    change: async () => { throw new Error('Save failed'); }, handoff: async () => {} });
  await state.change('0001', 'note', { id: '0816', entity: 'player', action: 'add', text: 'Keep me' });
  expect(state.peek('0001').savedNote).toBeUndefined();
  expect(state.peek('0001').error).toContain('Save failed');
});
