import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { readdirSync, readFileSync } from 'node:fs';
import { clockDom } from '../clock/testDom';
import { revealWorkSection, scrollWorkIntoView, useWorkAnchor, workSettleWindow } from './workScroll';

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
  const scroll = { scrollTop: 100, clientHeight: 600, addEventListener() {}, removeEventListener() {},
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
  capture({ target: { closest: (query: string) => query === selector ? row : null } } as unknown as Event);
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

async function anchorProbe(selector: string) {
  const dom = clockDom([]);
  const captures = new Map<string, (event: Event) => void>();
  let resized!: () => void;
  dom.document.addEventListener = (name: string, listener: () => void) => { captures.set(name, listener); };
  Object.defineProperty(globalThis, 'document', { configurable: true, value: dom.document });
  Object.defineProperty(globalThis, 'window', { configurable: true, value: {
    document: dom.document, HTMLIFrameElement: class {} } });
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true });
  Object.defineProperty(globalThis, 'ResizeObserver', { configurable: true, value: class {
    constructor(callback: () => void) { resized = callback; } observe() {} disconnect() {}
  } });
  const scroll = { scrollTop: 500, clientTop: 0, clientHeight: 600,
    addEventListener() {}, removeEventListener() {},
    getBoundingClientRect: () => ({ top: 60 }) };
  function node(y: number) {
    return { y, height: 30, isConnected: true, inside: true,
      getBoundingClientRect() {
        return { top: this.y - scroll.scrollTop, bottom: this.y - scroll.scrollTop + this.height };
      },
      closest(query: string): unknown {
        return query === selector ? this : query === '.wbody' ? scroll : null;
      } };
  }
  const row = node(700), control = node(650);
  const panel = { closest: () => null };
  control.closest = query => query === 'button, input, select, textarea, summary' ? control :
    query === '.bid-form, .contract-move' ? panel : null;
  const element = { closest: () => scroll, contains: (target: { inside?: boolean }) => target.inside,
    querySelectorAll: () => [row] };
  const ref = { current: element as unknown as HTMLElement };
  function Probe() { useWorkAnchor(ref, selector); return null; }
  const root = createRoot(dom.element); roots.push(root);
  await act(async () => { root.render(createElement(Probe)); });
  return { scroll, row, control, capture: (target: unknown, event = 'click') =>
    captures.get(event)!({ target } as unknown as Event), resize: () => resized() };
}
it.each(['bid', 'contract', 'cut', 'buyout'])
('A: %s card stays at its top through collapse and re-sort', async () => {
  const p = await anchorProbe('.bid-plan');
  p.capture(p.row);
  const top = p.row.getBoundingClientRect().top;
  p.row.y -= 160; p.resize();
  expect(p.row.getBoundingClientRect().top).toBe(top);
  p.row.y += 740; p.resize();
  expect(p.row.getBoundingClientRect().top).toBe(top);
  expect(readFileSync('src/app/shell/MyMoves.tsx', 'utf8')).toContain("useWorkAnchor(root, '.bid-plan')");
  expect(readFileSync('src/app/pool/Pool.tsx', 'utf8')).toContain(', .bid-plan');
});
it.each(['row-first', 'inspector-first'])
('E: %s panel edit holds its touched control, not the hidden row', async order => {
  const p = await anchorProbe('.contract-table tbody tr');
  if (order === 'row-first') p.capture(p.row);
  else p.capture({ closest: () => null });
  p.row.y = 1600;
  p.capture(p.control);
  p.row.y += 390; p.resize();
  expect(p.control.getBoundingClientRect().top).toBe(150);
  expect(p.scroll.scrollTop).toBe(500);
  p.capture(p.control, 'change');
  p.control.y += 80; p.row.y += 80; p.resize();
  expect(p.control.getBoundingClientRect().top).toBe(150);
});
it('G: inline Cancel/Save anchor on the preceding row after the touched control disappears', async () => {
  const p = await anchorProbe('.pool-table tbody tr');
  const panel = { closest: () => ({ previousElementSibling: p.row }) };
  p.control.closest = query => query === 'button, input, select, textarea, summary' ? p.control :
    query === '.bid-form, .contract-move' ? panel : null;
  p.capture(p.control);
  p.control.isConnected = false;
  p.row.y += 280; p.resize();
  expect(p.row.getBoundingClientRect().top).toBe(200);
});

it('A: terminal collapse reveals a card whose buttons were visible but top was hidden', async () => {
  const p = await anchorProbe('.bid-plan');
  p.row.y = 350; p.row.height = 400;
  p.capture(p.row);
  p.row.height = 30; p.resize();
  expect(p.row.getBoundingClientRect().top).toBe(60);
});

it('releases the row anchor for Contract Acts so resize cannot undo the form reveal', async () => {
  const p = await anchorProbe('.contract-table tbody tr');
  p.capture(p.row);
  p.capture({ closest: (query: string) => query === 'button'
    ? { getAttribute: () => 'Contract Acts…' } : null });
  p.scroll.scrollTop = 0;
  p.row.y += 300;
  p.resize();
  expect(p.scroll.scrollTop).toBe(0);
});

it.each(['wheel', 'scroll', 'click', 'keydown', 'input', 'timeout'])
('settles a cold section after growth above it, then yields permanently to %s', event => {
  vi.useFakeTimers();
  try {
    const document = new EventTarget();
    Object.defineProperty(globalThis, 'document', { configurable: true, value: document });
    let resized!: () => void;
    const disconnect = vi.fn();
    Object.defineProperty(globalThis, 'ResizeObserver', { configurable: true, value: class {
      constructor(callback: () => void) { resized = callback; }
      observe() {} disconnect() { disconnect(); }
    } });
    const scroll = Object.assign(new EventTarget(), {
      scrollTop: 0, clientTop: 0, clientHeight: 600, children: [],
      getBoundingClientRect: () => ({ top: 60 }),
      classList: { contains: (name: string) => name === 'wbody' },
    });
    let above = 430;
    const section = { isConnected: true, parentElement: scroll, tagName: 'SECTION', focus: vi.fn(),
      closest: () => scroll,
      getBoundingClientRect: () => ({ top: 60 + above - scroll.scrollTop, bottom: 900 }),
    };
    revealWorkSection(section as unknown as HTMLElement);
    expect(scroll.scrollTop).toBe(430);
    above += 330; resized();
    expect(section.getBoundingClientRect().top).toBe(60);
    // A delayed event from our own scroll must not release the anchor.
    scroll.dispatchEvent(new Event('scroll'));
    above += 100; resized();
    expect(section.getBoundingClientRect().top).toBe(60);
    if (event === 'timeout') vi.advanceTimersByTime(workSettleWindow);
    else if (event === 'scroll') {
      scroll.scrollTop -= 150;
      scroll.dispatchEvent(new Event('scroll'));
    } else document.dispatchEvent(new Event(event));
    const userTop = scroll.scrollTop;
    above += 500; resized();
    expect(scroll.scrollTop).toBe(userTop);
    expect(disconnect).toHaveBeenCalled();
  } finally { vi.useRealTimers(); }
});
