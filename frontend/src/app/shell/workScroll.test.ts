import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { readdirSync, readFileSync } from 'node:fs';
import { clockDom } from '../clock/testDom';
import { scrollWorkIntoView, useWorkAnchor } from './workScroll';

const globals = ['document', 'window', 'IS_REACT_ACT_ENVIRONMENT', 'ResizeObserver'].map(key => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => { for (const root of roots.splice(0)) root.unmount(); });
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});

it.each(['start', 'nearest'] as const)('scrolls only .wbody for %s, never any shell ancestor', block => {
  const shell = { scrollTop: 56 };
  const scroll = { scrollTop: 100, clientTop: 0, clientHeight: 600,
    getBoundingClientRect: () => ({ top: 60 }), parentElement: shell };
  const element = { closest: vi.fn(() => scroll),
    getBoundingClientRect: () => ({ top: 700, bottom: 750 }),
    scrollIntoView: () => { shell.scrollTop = 999; } };
  scrollWorkIntoView(element as unknown as HTMLElement, block);
  expect(scroll.scrollTop).toBe(block === 'start' ? 740 : 190);
  expect(shell.scrollTop).toBe(56);
  expect(element.closest).toHaveBeenCalledWith('.wbody');
});

it('nearest leaves visible content still and missing work columns never scroll the document', () => {
  const scroll = { scrollTop: 100, clientTop: 0, clientHeight: 600,
    getBoundingClientRect: () => ({ top: 60 }) };
  const element = { closest: () => scroll, getBoundingClientRect: () => ({ top: 400, bottom: 430 }) };
  scrollWorkIntoView(element as unknown as HTMLElement, 'nearest');
  expect(scroll.scrollTop).toBe(100);
  scrollWorkIntoView({ closest: () => null } as unknown as HTMLElement);
});

it.each(['.contract-table tbody tr', '.pool-table tbody tr'])
('anchors %s for row-menu, Inspector insertion and Close through async resize', async selector => {
  const dom = clockDom([]);
  let capture!: (event: Event) => void, resized!: () => void;
  dom.document.addEventListener = (name: string, listener: () => void) => {
    if (name === 'click') capture = listener;
  };
  Object.defineProperty(globalThis, 'document', { configurable: true, writable: true, value: dom.document });
  Object.defineProperty(globalThis, 'window', { configurable: true, writable: true,
    value: { document: dom.document, HTMLIFrameElement: class {} } });
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true });
  Object.defineProperty(globalThis, 'ResizeObserver', { configurable: true, value: class {
    constructor(callback: () => void) { resized = callback; }
    observe() {} disconnect() {}
  } });
  const shell = { scrollTop: 0 };
  const scroll = { scrollTop: 100, addEventListener() {}, removeEventListener() {},
    getBoundingClientRect: () => ({ top: 60 }) };
  let layout = 500;
  const row = { isConnected: true,
    getBoundingClientRect: () => ({ top: layout - scroll.scrollTop,
      bottom: layout - scroll.scrollTop + 30 }) };
  const element = { closest: () => scroll, contains: (target: unknown) => target === row,
    querySelectorAll: () => [row] };
  function Probe() {
    useWorkAnchor({ current: element as unknown as HTMLElement }, selector);
    return null;
  }
  const root = createRoot(dom.element); roots.push(root);
  await act(async () => { root.render(createElement(Probe)); });
  capture({ target: { closest: () => row } } as unknown as Event);
  layout += 330; resized();
  expect(row.getBoundingClientRect().top).toBe(400);
  capture({ target: { closest: () => null } } as unknown as Event);
  layout += 170; resized();
  expect(row.getBoundingClientRect().top).toBe(400);
  capture({ target: { closest: () => null } } as unknown as Event);
  layout -= 500; resized();
  expect(row.getBoundingClientRect().top).toBe(400);
  expect(shell.scrollTop).toBe(0);
});

it('no frontend call uses native scrollIntoView', () => {
  function scan(dir: string): string[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap(file => {
      const path = `${dir}/${file.name}`;
      if (file.isDirectory()) return scan(path);
      return /\.(tsx?|jsx?)$/.test(path) && !path.includes('.test.') &&
        /\.scrollIntoView\s*\(/.test(readFileSync(path, 'utf8')) ? [path] : [];
    });
  }
  expect(scan('src')).toEqual([]);
});
