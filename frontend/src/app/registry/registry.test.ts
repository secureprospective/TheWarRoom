import { describe, it, expect, vi } from 'vitest';
import { renderToPipeableStream, renderToStaticMarkup } from 'react-dom/server';
import { PassThrough } from 'node:stream';
import { createElement } from 'react';
import { FixtureProvider } from '../data/provider';
import { createCommands, commands } from '../commands/registry';
import { renderState } from '../shell/state';
import { TargetApp } from '../shell/TargetApp';
import { searchCandidates, rankResults, runResult, indexResult } from '../commands/search';
import { endpoints, endpointById, endpointsAt, resolve, placementRoute, surfacePlaces } from './index';
import type { Endpoint, EndpointId, Placement } from './index';
import { assertEndpointRoutes } from './routes';
import { EndpointIndex } from './EndpointIndex';
import { nodes, nodeKeys } from '../shell/nodes';

function renderSettledApp(): Promise<string> {
  return new Promise((accept, reject) => {
    const output = new PassThrough();
    let html = '';
    output.on('data', (chunk: Buffer) => { html += chunk.toString(); });
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
    expect(resolve('M-006')).toEqual({ kind: 'retired', note: endpointById('M-006').disposition_note });
    expect(() => endpointById('bad' as EndpointId)).toThrow('Unknown endpoint');
  });
  it('indexes kept endpoints in ring, gravity, id order, always below the short-list guard', () => {
    const placements = new Set(endpoints.filter((r) => r.placement).map((r) => r.placement as Placement));
    for (const place of placements) {
      const rows = endpointsAt(place);
      expect(rows.length).toBeLessThanOrEqual(80);
      expect(rows.every((r) => r.disposition === 'kept' && r.placement === place)).toBe(true);
      expect([...rows]).toEqual([...rows].sort((a, b) =>
        a.ring - b.ring || b.gravity.localeCompare(a.gravity) || a.id.localeCompare(b.id)));
      const html = renderToStaticMarkup(createElement(EndpointIndex, { placement: place }));
      for (const row of rows) {
        expect(html).toContain(row.id);
        expect(html).toContain(`not wired · ring ${row.ring}`);
      }
    }
    for (const node of nodeKeys) {
      for (const w of nodes[node].workspaces) {
        const place = `${nodes[node].label} › ${w.label.replace(' (role-gated)', '')}`;
        expect(placementRoute(place)).toEqual({ node, workspace: w.slug });
      }
    }
  });
  it('gives every kept row two real routes; merged rows reach kept targets or a reachable place', () => {
    const routes = assertEndpointRoutes(endpoints, candidates);
    for (const row of endpoints) {
      expect([...routes.get(row.id)!]).toEqual(row.disposition === 'kept'
        ? ['nav', 'command'] : row.disposition === 'merged' ? ['merged-into'] : []);
    }
  });
  it('renders every reachable workspace and surface index without a snapshot', async () => {
    // SSR otherwise reads Zustand's initial server snapshot, not dispatched navigation.
    const projection = vi.spyOn(commands, 'use').mockImplementation(() => renderState(commands.read()));
    try {
      for (const node of nodeKeys) {
        for (const w of nodes[node].workspaces) {
          commands.dispatch('nav.open', { node, workspace: w.slug } as Parameters<
            typeof commands.registry['nav.open']['run']
          >[0]);
          const place = `${nodes[node].label} › ${w.label.replace(' (role-gated)', '')}`;
          const html = renderToStaticMarkup(createElement(TargetApp));
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
    } finally { projection.mockRestore(); }
  });
  it('kills an unplaced kept row, unreachable merge and a retired target', () => {
    const mutate = (id: EndpointId, patch: Partial<Endpoint>) => endpoints.map((r) =>
      r.id === id ? { ...r, ...patch } as Endpoint : r);
    expect(() => assertEndpointRoutes(mutate('M-001', { placement: '' }), candidates))
      .toThrow('M-001: fewer than two routes');
    expect(() => assertEndpointRoutes(mutate('M-150', { merged_into: ['M-006'] }), candidates))
      .toThrow('merge target not kept and reachable');
    const retired = indexResult({ label: endpointById('M-006').name, kind: 'endpoint', aliases: [],
      invocation: { verb: 'endpoint.open', args: { id: 'M-006' } } });
    expect(() => assertEndpointRoutes(endpoints, [...candidates, retired]))
      .toThrow('M-006: retired row reachable');
    expect(() => assertEndpointRoutes(mutate('M-145', { merged_place: '' }), candidates))
      .toThrow('merge target not kept and reachable');
  });
});

describe('Endpoint verbs and search', () => {
  it('offers all kept endpoints by their own name and never offers retired or merged rows', () => {
    const offered = candidates.filter((r) => r.kind === 'endpoint');
    expect(offered).toHaveLength(209);
    for (const row of endpoints) {
      const match = rankResults(row.name, candidates).find((r) =>
        r.invocation.verb === 'endpoint.open' && r.invocation.args.id === row.id);
      expect(Boolean(match)).toBe(row.disposition === 'kept');
    }
    const result = rankResults('M-033', candidates)[0];
    expect(result.label).toContain('M-033 · Trades O=05 · Trade Floor › Trade desk and offers');
    runResult(executor, result);
    expect(executor.read().node).toBe('trade');
    expect(executor.read().endpointIds).toEqual(['M-033']);
  });
  it('ranks endpoints after equally matching places and before players', () => {
    const result = rankResults('same', [
      indexResult({ label: 'same player', aliases: [], kind: 'player',
        invocation: { verb: 'inspector.open', args: {
          subject: { kind: 'player', id: '1', franchiseId: '1' },
        } } }),
      indexResult({ label: 'same endpoint', aliases: [], kind: 'endpoint',
        invocation: { verb: 'endpoint.open', args: { id: 'M-001' } } }),
      indexResult({ label: 'same place', aliases: [], kind: 'place',
        invocation: { verb: 'nav.open', args: { node: 'home' } } }),
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
      const place = row.merged_place || endpointById(row.merged_into[row.merged_into.length - 1]).placement;
      const route = placementRoute(place);
      if (route) expect(c.read().node).toBe(route.node);
      else expect(c.read().endpointSurface).toBe(place);
    }
    const before = c.read();
    expect(() => c.dispatch('endpoint.open', { id: 'M-006' })).toThrow('retired');
    expect(c.read()).toBe(before);
  });
});
