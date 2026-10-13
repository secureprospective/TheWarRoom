import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { clockDom, TestElement } from '../clock/testDom';
import { commands } from '../commands/registry';
import { endpoints, endpointById, surfacePlaces } from './index';
import { nodes, nodeKeys } from '../shell/nodes';
import ReportDrawer, { reportGroups } from './ReportDrawer';

function descendants(root: TestElement): TestElement[] {
  return [root, ...root.children.flatMap(child => child instanceof TestElement ? descendants(child) : [])];
}
function props(node: TestElement) {
  const key = Object.keys(node).find(key => key.startsWith('__reactProps'))!;
  return (node as unknown as Record<string, {
    onChange(event: { target: { value: string } }): void; onClick(): void; value: string;
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
  commands.dispatch('overlays.close', {});
});

it('lists every kept, placed read in nav order, including hybrids and unlanded reports', () => {
  const groups = reportGroups('');
  const rows = groups.flatMap(group => group.rows);
  const expected = endpoints.filter(row => row.disposition === 'kept' && row.placement &&
    row.kind.split(/[ /()]+/).includes('Read'));
  expect(rows.map(row => row.id).sort()).toEqual(expected.map(row => row.id).sort());
  expect(rows.length).toBeGreaterThan(8);
  expect(rows.some(row => row.kind === 'Decide/Read')).toBe(true);
  expect(rows.some(row => row.kind === 'Read (gated)')).toBe(true);
  expect(rows.some(row => row.status === 'spec')).toBe(true);
  const places = [...nodeKeys.flatMap(node => nodes[node].workspaces
    .map(w => `${nodes[node].label} › ${w.label}`)), ...surfacePlaces];
  expect(groups.map(group => group.place)).toEqual(places.filter(place =>
    expected.some(row => row.placement === place)));
});

it('filters case-insensitive name, GM question, id and place without truncating matches', () => {
  const report = endpointById('C-02');
  for (const query of [report.name.toUpperCase(), report.gm_question, ' c-02 ']) {
    expect(reportGroups(query).flatMap(group => group.rows).map(row => row.id)).toContain('C-02');
  }
  const race = reportGroups('league pulse › race');
  expect(race).toHaveLength(1);
  expect(race[0].place).toBe('League Pulse › Race');
  expect(race[0].rows.length).toBeGreaterThan(8);
  expect(reportGroups('no such report 123')).toEqual([]);
});

it('typing narrows the drawer; a report Act opens its id and closes the drawer', async () => {
  const { document, root: host, element } = clockDom([]);
  vi.stubGlobal('document', document);
  vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  commands.dispatch('surface.open', { place: 'Report drawer' });
  expect(commands.read().endpointSurface).toBe('Report drawer');
  const root = createRoot(element);
  const report = endpointById('C-02');
  const input = () => descendants(host).find(node => node.tagName === 'INPUT')!;
  const buttons = () => descendants(host).filter(node => node.tagName === 'BUTTON');
  try {
    await act(async () => { root.render(createElement(ReportDrawer)); });
    expect(host.textContent).toContain(report.name);
    expect(host.textContent).toContain(report.gm_question);
    expect(host.textContent).toContain('League Pulse › Race');
    expect(host.textContent).toContain('Not wired');
    expect(host.textContent).toContain('Built, not yet landed live');
    const total = buttons().length;
    await act(async () => { props(input()).onChange({ target: { value: 'c-02' } }); });
    expect(props(input()).value).toBe('c-02');
    expect(buttons()).toHaveLength(1);
    expect(host.textContent).toContain(report.name);
    expect(host.textContent).not.toContain(endpointById('M-002').name);
    await act(async () => { props(buttons()[0]).onClick(); });
    expect(commands.history().at(-1)).toEqual({ id: 'endpoint.open', args: { id: 'C-02' } });
    expect(commands.read()).toMatchObject({ node: 'pulse', workspace: { pulse: 'race' },
      endpointSurface: null, endpointIds: ['C-02'] });
    await act(async () => { props(input()).onChange({ target: { value: 'no such report 123' } }); });
    expect(buttons()).toHaveLength(0);
    expect(host.textContent).toContain('No reports match.');
    await act(async () => { props(input()).onChange({ target: { value: '' } }); });
    expect(buttons()).toHaveLength(total);
  } finally { await act(async () => { root.unmount(); }); }
});
