import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { parseSources, parseOutcome, type DataProvider } from './provider';
import { createDataState } from './state';
import { RefreshScopes, Parameters } from './DataSources';
import { report } from '../reports/contract';
import { ReportView } from '../reports/ReportView';
import { endpointById, placementRoute } from '../registry';
import { commands } from '../commands/registry';
import type { main } from '../../../wailsjs/go/models';
import wire from './wire.testdata.json';
const data = () => parseSources(wire.data);
const provider = (): DataProvider => ({
  read: async () => data(), crosswalk: async () => report(wire.crosswalk, 'Crosswalk'),
  params: async () => wire.params as main.ParamsResult,
  refresh: async () => parseOutcome(wire.result), folder: async () => parseOutcome(wire.result),
    parameter: async () => {},
});
const html = renderToStaticMarkup;
describe('Control Room data and sources', () => {
  it('renders Go health and honest unchecked facts, not a successful compile', () => {
    const d = data();
    for (const r of [d.health, d.facts, d.checks]) {
      const markup = html(createElement(ReportView, { report: r, howLabel: 'How this is counted' }));
      expect(markup).toContain(r.title);
      expect(markup).toContain(`data-endpoint="${r.id}"`);
    }
    expect(d.facts.status).toBe('empty');
    expect(d.facts.note).toContain('No manifest yet');
    expect(d.checks.rows.every(r => r.cells[2] === 'Not run yet')).toBe(true);
  });
  it('offers one Act per Go scope, a labelled folder and progress, disabling duplicate work', () => {
    const d = data();
    const markup = html(createElement(RefreshScopes, { data: d, busy: 'archive' }));
    expect(markup).toContain('id="archive-folder"');
    expect(markup).toContain('Refreshing League archive');
    expect(markup.match(/disabled=""/g)?.length).toBe(5);
    for (const s of d.scopes) expect(markup).toContain(s.note.replaceAll("'", '&#x27;'));
    const unset = html(createElement(RefreshScopes, { data: { ...d, archiveDir: '' }, busy: '' }));
    expect(unset).toContain('disabled=""');
  });
  it('keeps every parameter, default and reset, but gates changes to Admin', () => {
    const markup = html(createElement(Parameters, { params: wire.params as main.ParamsResult, busy: '' }));
    expect(markup).toContain('test.weight');
    expect(markup).toContain('Overridden');
    expect(markup).toContain('Reset');
    expect(markup).toContain('values are read-only');
    expect(() => commands.dispatch('data.parameter', { kind: 'parameter', key: 'test.weight',
      position: 'WR', value: 0.7 })).toThrow(/permission denied/);
  });
  it('routes all five built rows to the lazy workspace', () => {
    for (const id of ['C-07', 'C-08', 'C-09', 'L-34', 'S-09'] as const) {
      const e = endpointById(id);
      expect(e.status).toBe('built');
      expect(placementRoute(e.placement)).toEqual({ node: 'control', workspace: 'data-and-sources' });
    }
    expect(commands.registry['refresh.run'].args).toEqual(['scope']);
  });
  it('refuses malformed wire and unknown or duplicate scopes', () => {
    expect(() => parseSources({ ...wire.data, extra: 1 })).toThrow(/unexpected/);
    expect(() => parseSources({ ...wire.data, scopes: [] })).toThrow(/scopes/);
    expect(() => parseSources({ ...wire.data,
      scopes: Array(4).fill(wire.data.scopes[0]) })).toThrow(/repeated/);
    expect(() => parseSources({ ...wire.data, health: { ...wire.data.health,
      columns: [] } })).toThrow(/column count/);
    expect(() => parseOutcome({ ok: false, error: '', summary: '' })).toThrow(/reason/);
  });
});
describe('scoped refresh state', () => {
  it('holds the last good reading and reports failures', async () => {
    const p = provider();
    const state = createDataState(p);
    await state.run({ kind: 'read' });
    const good = state.read().data;
    p.read = async () => { throw new Error('local read failed'); };
    await state.run({ kind: 'read' });
    expect(state.read().data).toBe(good);
    expect(state.read().error).toContain('local read failed');
    expect(state.read().busy).toBe('');
    p.read = async () => data();
    p.refresh = async () => ({ ok: false, error: 'Bad archive: 2014/rules.json', summary: '' });
    await state.run({ kind: 'refresh', scope: 'archive' });
    expect(state.read().error).toContain('2014/rules.json');
  });
  it('serializes Acts and shows the result without a blind second refresh', async () => {
    const p = provider();
    let finish!: (v: ReturnType<typeof parseOutcome>) => void;
    let calls = 0;
    p.refresh = async () => { calls++; return new Promise(resolve => { finish = resolve; }); };
    const state = createDataState(p);
    const pending = state.run({ kind: 'refresh', scope: 'archive' });
    expect(state.read().busy).toBe('archive');
    await state.run({ kind: 'refresh', scope: 'league' });
    expect(calls).toBe(1);
    finish(parseOutcome(wire.result));
    await pending;
    expect(state.read().busy).toBe('');
    expect(state.read().result).toBe(wire.result.summary);
  });
});
