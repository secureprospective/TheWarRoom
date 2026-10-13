import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import { clockDom, TestElement } from '../clock/testDom';
import { PoolInspectorView, PoolScreen } from './Pool';
import { createPoolState } from './state';
import { parseBoard, parsePool, poolProvider } from './provider';
import wire from './wire.testdata.json';

it('reveals the mounted form and focuses salary after the Inspector requests a bid', async () => {
  const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map(key => ({
    key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
  }));
  const dom = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
  const scroll = { scrollTop: 900, clientTop: 0, clientHeight: 600,
    classList: { contains: (name: string) => name === 'wbody' },
    getBoundingClientRect: () => ({ top: 50 }), addEventListener: vi.fn(), removeEventListener: vi.fn() };
  let focused: TestElement | undefined;
  const focus = vi.spyOn(TestElement.prototype, 'focus').mockImplementation(function (this: TestElement) {
    focused = this;
  });
  const closest = Object.getOwnPropertyDescriptor(TestElement.prototype, 'closest');
  const box = Object.getOwnPropertyDescriptor(TestElement.prototype, 'getBoundingClientRect');
  Object.assign(TestElement.prototype, { closest: () => scroll,
    getBoundingClientRect: () => ({ top: -750, bottom: -550 }) });
  const query = vi.spyOn(TestElement.prototype, 'querySelector')
    .mockImplementation(function (this: TestElement, selector) {
    if (!selector.startsWith('input:')) return undefined!;
    const fields = (node: TestElement): TestElement[] => node.children.flatMap(child =>
      child instanceof TestElement ? child.tagName === 'INPUT' ? [child] : fields(child) : []);
    return fields(this).find(field => !field.getAttribute('disabled')) ?? undefined!;
  });
  const state = createPoolState({ franchise: () => '0025',
    provider: { ...poolProvider, pool: vi.fn(async () => parsePool(wire.pool)),
      plans: vi.fn(async () => ({ ...parseBoard(wire.board), value: { plans: [], summary: 'No plans' } })),
      preview: vi.fn(async () => { throw new Error('Preview unavailable'); }) } });
  const root = createRoot(dom.element);
  try {
    await state.reload();
    const row = state.peek().reading!.value.rows.find(player => !player.planId)!;
    const inspector = PoolInspectorView({ row, plans: [], busy: '' });
    const button = inspector.props.children.at(-1);
    expect(button.props.args).toEqual({ field: 'pool-plan', value: row.id });
    const render = () => root.render(createElement(PoolScreen, { state: state.peek(), franchiseId: '0025' }));
    await act(async () => { render(); });
    await act(async () => { await state.change(button.props.args); render(); });
    expect(scroll.scrollTop).toBe(100);
    expect(focus).toHaveBeenCalledWith({ preventScroll: true });
    const first = focused!;
    expect(first.tagName).toBe('INPUT');
    expect(first.getAttribute('placeholder')).toBe('5.25');
    const calls = focus.mock.calls.length;
    scroll.scrollTop = 333;
    await act(async () => { state.edit('salary', '6'); render(); });
    expect(focus.mock.calls.length).toBe(calls);
    expect(scroll.scrollTop).toBe(333);
  } finally {
    await act(async () => { root.unmount(); });
    query.mockRestore();
    focus.mockRestore();
    for (const [name, descriptor] of [['closest', closest], ['getBoundingClientRect', box]] as const) {
      if (descriptor) Object.defineProperty(TestElement.prototype, name, descriptor);
      else Reflect.deleteProperty(TestElement.prototype, name);
    }
    for (const { key, descriptor } of globals) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor);
      else Reflect.deleteProperty(globalThis, key);
    }
  }
});
