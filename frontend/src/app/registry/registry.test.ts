import { describe, it, expect, vi } from 'vitest';
import { renderToPipeableStream, renderToStaticMarkup } from 'react-dom/server';
import { PassThrough } from 'node:stream';
import { createElement } from 'react';
import { FixtureProvider } from '../data/provider';
import { createCommands, commands } from '../commands/registry';
import { renderState } from '../shell/state';
import { TargetApp } from '../shell/TargetApp';
import { searchCandidates, rankResults, runResult, indexResult } from '../commands/search';
import {
  endpoints,
  endpointById,
  endpointsAt,
  resolve,
  placementRoute,
  surfacePlaces,
} from './index';
import type { Endpoint, EndpointId, Placement } from './index';
import { assertEndpointRoutes } from './routes';
import { EndpointIndex } from './EndpointIndex';
import { nodes, nodeKeys } from '../shell/nodes';

function renderSettledApp(): Promise<string> {
  return new Promise((accept, reject) => {
    const output = new PassThrough();
    let html = '';
    output.on('data', (chunk: Buffer) => {
      html += chunk.toString();
    });
    output.on('end', () => accept(html));
    output.on('error', reject);
    const stream = renderToPipeableStream(createElement(TargetApp), {
      onAllReady: () => stream.pipe(output),
      onError: reject,
    });
  });
}

const snapshot = await new FixtureProvider().snapshot();
const executor = createCommands(() => undefined);
const candidates = searchCandidates(executor, snapshot);

describe('Endpoint map', () => {
  it('freezes the records and targets and resolves every disposition without invention', () => {
    expect(endpoints).toHaveLength(270);
    expect(Object.isFrozen(endpoints)).toBe(true);
    for (const row of endpoints) {
      expect(Object.isFrozen(row)).toBe(true);
      expect(Object.isFrozen(row.merged_into)).toBe(true);
      const result = resolve(row.id);
      expect(result.kind).toBe(row.disposition === 'kept' ? 'view' : row.disposition);
    }
    expect(resolve('M-150')).toEqual({ kind: 'merged', into: ['M-003', 'M-005'] });
    expect(resolve('M-145')).toEqual({ kind: 'merged', place: 'Inspector' });
    expect(resolve('M-006')).toEqual({
      kind: 'retired',
      note: endpointById('M-006').disposition_note,
    });
    expect(() => endpointById('bad' as EndpointId)).toThrow('Unknown endpoint');
  });
  it('indexes kept endpoints in ring, gravity, id order, always below the short-list guard', () => {
    const placements = new Set(
      endpoints.filter((r) => r.placement).map((r) => r.placement as Placement),
    );
    for (const place of placements) {
      const rows = endpointsAt(place);
      expect(rows.length).toBeLessThanOrEqual(80);
      expect(rows.every((r) => r.disposition === 'kept' && r.placement === place)).toBe(true);
      expect([...rows]).toEqual(
        [...rows].sort(
          (a, b) =>
            a.ring - b.ring || b.gravity.localeCompare(a.gravity) || a.id.localeCompare(b.id),
        ),
      );
      const html = renderToStaticMarkup(createElement(EndpointIndex, { placement: place }));
      for (const row of rows) {
        expect(html).toContain(row.id);
        const item = html.split('<li').find((part) => part.includes(`<b>${row.id} ·`));
        const labels: Record<string, string> = {
          landed: 'landed live',
          built: 'built, not yet landed live',
          spec: 'not wired',
        };
        expect(item).toContain(`${labels[row.status]} · ring ${row.ring}`);
      }
    }
    for (const node of nodeKeys) {
      for (const w of nodes[node].workspaces) {
        const place = `${nodes[node].label} › ${w.label.replace(' (role-gated)', '')}`;
        expect(placementRoute(place)).toEqual({ node, workspace: w.slug });
      }
    }
  });
  it('distinguishes live-Landed Acts from built Acts and keeps every other row spec', () => {
    const built = [
      'S-01', 'S-02', 'S-04', 'S-05', 'S-08', 'S-11', 'S-12', 'S-13',
      'M-087', 'M-096', 'M-149', 'M-117', 'M-116', 'M-118', 'M-103', 'M-101', 'M-154',
      'M-151', 'M-159', 'M-167', 'M-169',
      'C-05', 'M-032', 'M-120', 'M-123',
      'L-34', 'C-07', 'C-08', 'C-09', 'S-09',
      'M-046', 'M-047', 'M-067', 'M-069',
      'M-063', 'M-064', 'M-065', 'M-066', 'M-068', 'M-102', 'M-176', 'M-179', 'M-183',
      'M-028',
      'M-029',
      'M-025',
      'M-031',
      'M-078',
      'M-039',
      'M-056',
      'M-194', 'M-193', 'M-195',
      'C-06.7',
      'C-06.5',
      'C-06.1', 'C-06.3', 'M-036', 'M-196',
      'C-06.6',
      'C-06.2', 'C-06.4', 'M-199', 'M-033', 'M-034',
      'X-04',
      ...Array.from({ length: 32 }, (_, i) => `L-${String(i + 1).padStart(2, '0')}`),
      'L-33',
      'L-35', 'L-36',
      'M-030', 'M-148', 'M-153', 'C-03', 'C-04', 'S-03', 'S-06', 'S-07',
      'M-002', 'M-008', 'C-02', 'M-009', 'M-010', 'M-011', 'M-012', 'M-013', 'M-014',
      'M-071', 'M-074', 'M-076', 'M-077', 'M-090', 'M-161', 'M-170',
      'X-03', 'M-003', 'M-004', 'M-005', 'M-007', 'M-015', 'M-016', 'M-018', 'M-021', 'M-022',
      'M-072', 'M-088',
    ];
    for (const row of endpoints) {
      let status = 'spec';
      if (row.id === 'M-027') status = 'landed';
      if (built.includes(row.id)) status = 'built';
      expect(row.status).toBe(status);
    }
    const checks: [EndpointId, string][] = [
      ['M-027', 'landed live'],
      ['M-028', 'built, not yet landed live'],
      ['M-029', 'built, not yet landed live'],
      ['M-033', 'built, not yet landed live'],
      ['X-04', 'built, not yet landed live'],
      ['M-034', 'built, not yet landed live'],
    ];
    for (const [id, label] of checks) {
      const row = endpointById(id);
      const html = renderToStaticMarkup(
        createElement(EndpointIndex, {
          placement: row.placement as Placement,
        }),
      );
      const item = html.split('<li').find((part) => part.includes(`<b>${id} ·`));
      expect(item).toContain(`<small>${label} · ring ${row.ring}</small>`);
      if (row.status !== 'spec') expect(item).not.toContain('not wired');
    }
  });
  it('gives every kept row two real routes; merged rows reach kept targets or a reachable place', () => {
    const routes = assertEndpointRoutes(endpoints, candidates);
    for (const row of endpoints) {
      expect([...routes.get(row.id)!]).toEqual(
        row.disposition === 'kept'
          ? ['nav', 'command']
          : row.disposition === 'merged'
            ? ['merged-into']
            : [],
      );
    }
  });
  it('renders every reachable workspace and surface index without a snapshot', async () => {
    // SSR otherwise reads Zustand's initial server snapshot, not dispatched navigation.
    const projection = vi
      .spyOn(commands, 'use')
      .mockImplementation(() => renderState(commands.read()));
    try {
      for (const node of nodeKeys) {
        for (const w of nodes[node].workspaces) {
          commands.dispatch('nav.open', { node, workspace: w.slug } as Parameters<
            (typeof commands.registry)['nav.open']['run']
          >[0]);
          const place = `${nodes[node].label} › ${w.label.replace(' (role-gated)', '')}`;
          const html = await renderSettledApp();
          expect(html).toContain(`${place} endpoint index`);
          for (const row of endpointsAt(place as Placement)) expect(html).toContain(row.id);
        }
      }
      for (const place of surfacePlaces) {
        commands.dispatch('surface.open', { place });
        const html = await renderSettledApp();
        expect(html).toContain(`${place} endpoint index`);
      }
      expect(() => assertEndpointRoutes(endpoints, searchCandidates(executor))).not.toThrow();
      commands.dispatch('overlays.close', {});
      commands.dispatch('nav.open', { node: 'home', workspace: 'seasonal-card' });
    } finally {
      projection.mockRestore();
    }
  });
  it('kills an unplaced kept row, unreachable merge and a retired target', () => {
    const mutate = (id: EndpointId, patch: Partial<Endpoint>) =>
      endpoints.map((r) => (r.id === id ? ({ ...r, ...patch } as Endpoint) : r));
    expect(() => assertEndpointRoutes(mutate('M-001', { placement: '' }), candidates)).toThrow(
      'M-001: fewer than two routes',
    );
    expect(() =>
      assertEndpointRoutes(mutate('M-150', { merged_into: ['M-006'] }), candidates),
    ).toThrow('merge target not kept and reachable');
    const retired = indexResult({
      label: endpointById('M-006').name,
      kind: 'endpoint',
      aliases: [],
      invocation: { verb: 'endpoint.open', args: { id: 'M-006' } },
    });
    expect(() => assertEndpointRoutes(endpoints, [...candidates, retired])).toThrow(
      'M-006: retired row reachable',
    );
    expect(() => assertEndpointRoutes(mutate('M-145', { merged_place: '' }), candidates)).toThrow(
      'merge target not kept and reachable',
    );
  });
});

describe('Endpoint verbs and search', () => {
  it('offers all kept endpoints by their own name and never offers retired or merged rows', () => {
    const offered = candidates.filter((r) => r.kind === 'endpoint');
    expect(offered).toHaveLength(204);
    for (const row of endpoints) {
      const match = rankResults(row.name, candidates).find(
        (r) => r.invocation.verb === 'endpoint.open' && r.invocation.args.id === row.id,
      );
      expect(Boolean(match)).toBe(row.disposition === 'kept');
    }
    const result = rankResults('M-033', candidates)[0];
    expect(result.label).toContain('M-033 · Trades O=05 · Trade Floor › Trade desk');
    runResult(executor, result);
    expect(executor.read().node).toBe('trade');
    expect(executor.read().endpointIds).toEqual(['M-033']);
  });
  it('ranks endpoints after equally matching places and before players', () => {
    const result = rankResults('same', [
      indexResult({
        label: 'same player',
        aliases: [],
        kind: 'player',
        invocation: {
          verb: 'inspector.open',
          args: {
            subject: { kind: 'player', id: '1', franchiseId: '1' },
          },
        },
      }),
      indexResult({
        label: 'same endpoint',
        aliases: [],
        kind: 'endpoint',
        invocation: { verb: 'endpoint.open', args: { id: 'M-001' } },
      }),
      indexResult({
        label: 'same place',
        aliases: [],
        kind: 'place',
        invocation: { verb: 'nav.open', args: { node: 'home' } },
      }),
    ]);
    expect(result.map((r) => r.kind)).toEqual(['place', 'endpoint', 'player']);
  });
  it('dispatches every kept endpoint to its workspace or shell surface and preserves highlight ids', () => {
    const c = createCommands(() => undefined);
    expect(c.registry['endpoint.open'].gravity).toBe('G0');
    expect(c.registry['endpoint.open'].undo).toBe('instant');
    for (const row of endpoints.filter((r) => r.disposition === 'kept')) {
      c.dispatch('commandbar.open', {});
      c.dispatch('endpoint.open', { id: row.id });
      expect(c.read().endpointIds).toEqual([row.id]);
      const route = placementRoute(row.placement);
      if (route) {
        expect(c.read().node).toBe(route.node);
        expect(c.read().workspace[route.node]).toBe(route.workspace);
      } else {
        expect(c.read().endpointSurface).toBe(row.placement);
        if (row.placement === 'Inspector') expect(c.read().inspector).toBe('rest');
        if (row.placement === 'Comms') expect(c.read().comms).toBe(true);
        if (row.placement === 'Calendar') expect(c.read().summoned).toBe('calendar');
      }
      expect(c.read().commandbar).toBe(row.placement === 'Command bar');
    }
    for (const place of surfacePlaces) {
      c.dispatch('surface.open', { place });
      expect(c.read().endpointSurface).toBe(place);
    }
    c.dispatch('surface.close', {});
    expect(c.read().endpointSurface).toBeNull();
  });
  it('opens all merge targets, keeps their highlight ids, opens merged places and refuses retirement', () => {
    const c = createCommands(() => undefined);
    for (const row of endpoints.filter((r) => r.disposition === 'merged')) {
      c.dispatch('endpoint.open', { id: row.id });
      expect(c.read().endpointIds).toEqual(row.merged_into.length ? row.merged_into : [row.id]);
      const place =
        row.merged_place || endpointById(row.merged_into[row.merged_into.length - 1]).placement;
      const route = placementRoute(place);
      if (route) expect(c.read().node).toBe(route.node);
      else expect(c.read().endpointSurface).toBe(place);
    }
    const before = c.read();
    expect(() => c.dispatch('endpoint.open', { id: 'M-006' })).toThrow('retired');
    expect(c.read()).toBe(before);
  });
});

it('Draft room mounts the shared lazy Draft lab rather than a placeholder', async () => {
  const use = vi.spyOn(commands, 'use').mockImplementation(() => renderState(commands.read()));
  commands.dispatch('nav.open', { node: 'trade', workspace: 'draft-room' });
  const html = await renderSettledApp();
  expect(html).toContain('Draft lab');
  expect(html).not.toContain('Draft room</h4><p class="not-wired"');
  const { readFileSync } = await import('node:fs');
  const source = readFileSync('src/app/market/Market.tsx', 'utf8');
  expect(source).toContain("s.workspace[s.node] === 'draft-room'");
  expect(source).toContain("lazy(() => import('./Draft'))");
  commands.dispatch('nav.open', { node: 'home', workspace: 'seasonal-card' });
  use.mockRestore();
});

it('reveals rendered endpoints instead of index cards, preserving controls that fit', async () => {
  const { revealEndpoint } = await import('./EndpointIndex');
  const work = { scrollTop: 800, clientTop: 0, clientHeight: 600,
    getBoundingClientRect: () => ({top: 50}), querySelectorAll: () => [indexedSurface, surface] };
  const card = { focus: vi.fn(), closest: () => work,
    getBoundingClientRect: () => ({top: 1800, bottom: 1900}) };
  const index = { closest: () => work, querySelector: () => card };
  const parent = {closest: () => work,
    getBoundingClientRect: () => ({top: 1000-work.scrollTop, bottom: 2000-work.scrollTop})};
  const surface = { dataset: {endpoint: 'M-064'},
    closest: (s: string) => s === '.endpoint-index' ? null : work,
    getBoundingClientRect: () => ({top: 1150-work.scrollTop, bottom: 2000-work.scrollTop}),
    parentElement: parent,
    previousElementSibling: {classList: {contains: () => true}} };
  const indexedSurface = {...surface,
    closest: (s: string) => s === '.endpoint-index' ? index : work};
  revealEndpoint(index as unknown as HTMLElement, ['M-064']);
  expect(card.focus).not.toHaveBeenCalled();
  expect(work.scrollTop).toBe(950);
  surface.previousElementSibling.classList.contains = () => false;
  work.scrollTop = 800;
  revealEndpoint(index as unknown as HTMLElement, ['M-064']);
  expect(work.scrollTop).toBe(1100);
  work.scrollTop = 800;
  revealEndpoint(index as unknown as HTMLElement, ['M-999']);
  expect(card.focus).toHaveBeenCalledWith({preventScroll: true});
  expect(work.scrollTop).toBe(2050);
  card.focus.mockClear();
  work.scrollTop = 300;
  expect(revealEndpoint(index as unknown as HTMLElement, ['M-999'], false)).toBe(false);
  expect(card.focus).not.toHaveBeenCalled();
  expect(work.scrollTop).toBe(300);
});

it('reveals a lazily mounted endpoint once and stops watching after it arrives', async () => {
  const { act } = await import('react');
  const { createRoot } = await import('react-dom/client');
  const { clockDom } = await import('../clock/testDom');
  const dom = clockDom([]);
  const keys = ['document', 'window', 'IS_REACT_ACT_ENVIRONMENT', 'MutationObserver'];
  const saved = keys.map(key => [key, Object.getOwnPropertyDescriptor(globalThis, key)] as const);
  const surfaces: HTMLElement[] = [];
  const work = { scrollTop: 800, clientTop: 0, clientHeight: 600,
    getBoundingClientRect: () => ({top: 50}), querySelectorAll: () => surfaces };
  const create = dom.document.createElement.bind(dom.document);
  dom.document.createElement = tag => {
    const element = create(tag);
    Object.assign(element, { closest: () => work });
    return element;
  };
  let arrival!: () => void;
  const stop = vi.fn();
  Object.defineProperty(globalThis, 'document', {configurable: true, value: dom.document});
  Object.defineProperty(globalThis, 'window', {configurable: true,
    value: { document: dom.document, HTMLIFrameElement: class {} }});
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', {configurable: true, value: true});
  Object.defineProperty(globalThis, 'MutationObserver', {configurable: true, value: class {
    constructor(callback: () => void) { arrival = callback; }
    observe() {} disconnect() { stop(); }
  }});
  const root = createRoot(dom.element);
  try {
    await act(async () => { commands.dispatch('endpoint.open', {id: 'M-064'}); });
    await act(async () => {
      root.render(createElement(EndpointIndex, {placement: 'Trade Floor › Draft room'}));
    });
    expect(arrival).toBeTypeOf('function');
    surfaces.push({ dataset: {endpoint: 'M-064'}, closest: (s: string) =>
      s === '.endpoint-index' ? null : work,
      getBoundingClientRect: () => ({top: 350, bottom: 1000}) } as unknown as HTMLElement);
    arrival();
    expect(work.scrollTop).toBe(1100);
    expect(stop).toHaveBeenCalledTimes(1);
  } finally {
    await act(async () => { root.unmount(); });
    for (const [key, descriptor] of saved) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor);
      else Reflect.deleteProperty(globalThis, key);
    }
  }
});
