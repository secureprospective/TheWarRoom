import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { clockDom } from '../clock/testDom';
import { commands } from '../commands/registry';
import StatusStrip from './StatusStrip';

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

it('rereads health after refresh completion and rejects an older response', async () => {
  const dom = clockDom([]);
  let changed!: () => void;
  vi.spyOn(commands.alerts, 'onSeasonChange').mockImplementation(listener => {
    changed = listener;
    return () => {};
  });
  type Status = { health: string; state: string; gameday: string };
  const pending: ((status: Status) => void)[] = [];
  const read = vi.fn(() => new Promise<Status>(resolve => { pending.push(resolve); }));
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {},
    go: { control: { API: { TargetShellStatus: read } } } });
  const root = createRoot(dom.element);
  try {
    await act(async () => { root.render(createElement(StatusStrip)); });
    await act(async () => {
      pending[0]({ health: 'Source failures · last good data', state: 'stale', gameday: 'Week 5' });
    });
    expect(dom.element.textContent).toContain('Source failures');
    await act(async () => { changed(); changed(); });
    expect(read).toHaveBeenCalledTimes(3);
    await act(async () => {
      pending[2]({ health: 'Sources healthy', state: 'live', gameday: 'Week 5' });
    });
    await act(async () => {
      pending[1]({ health: 'Source failures', state: 'stale', gameday: 'Week 5' });
    });
    expect(dom.element.textContent).toContain('Sources healthy');
    expect(dom.element.textContent).not.toContain('Source failures');
  } finally {
    await act(async () => { root.unmount(); });
  }
});
