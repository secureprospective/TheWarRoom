import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { parseSources, parseOutcome, type DataProvider } from './provider';
import { createDataState } from './state';
import { RefreshScopes, Parameters, HeldRead } from './DataSources';
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
  photos: async () => parseOutcome(wire.result), parameter: async () => {},
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
    expect(markup).toContain(d.scopes[3].progress);
    expect(markup.match(/disabled=""/g)?.length).toBe(6);
    for (const s of d.scopes) expect(markup).toContain(s.note.replaceAll("'", '&#x27;'));
    const unset = html(createElement(RefreshScopes, { data: { ...d, archiveDir: '' }, busy: '' }));
    expect(unset).toContain('disabled=""');
  });
  it('keeps every parameter, default and reset, but gates changes to Admin', () => {
    const markup = html(createElement(Parameters, { params: wire.params as main.ParamsResult,
      busy: '', words: data().words }));
    expect(markup).toContain('Test weight');
    expect(markup).not.toContain('test.weight');
    expect(markup).toContain('Overridden');
    expect(markup).toContain('Reset');
    expect(markup).toContain(`aria-label="${data().words.clear}"`);
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

it('fits every table in the 680px panel without hidden or nested scrolling', async () => {
  const { readFileSync } = await import('node:fs');
  const css = readFileSync('src/app/datasources/sources.css', 'utf8');
  const d = data();
  const reports = [d.health, d.signals, d.facts, d.checks, report(wire.crosswalk, 'Crosswalk')];
  expect(reports.map(r => r.columns.length)).toEqual([3, 4, 2, 3, 3]);
  expect(css).toContain('width: 100%; table-layout: fixed;');
  expect(css).toContain('min-width: 0; padding: 10px 8px; white-space: normal; overflow-wrap: anywhere;');
  expect(css).toContain('.draft-table-scroll { overflow: visible; max-height: none; }');
  expect(css).toContain('td input { width: 100%; }');
  expect(css).toContain('input { box-sizing: border-box; min-width: 0; }');
  expect(css).toContain('grid-template-columns: minmax(0, 1fr) minmax(0, 3fr)');
  expect(css).toContain('white-space: normal; overflow-wrap: anywhere; }');
  const panel = 680, padding = 16, minimum = 0;
  for (const r of reports) {
    expect(r.columns.length * (minimum + padding)).toBeLessThan(panel);
    if (r.rows.length && r.status === 'ok') {
      const markup = html(createElement(ReportView, { report: r, howLabel: d.words.how }));
      expect(markup.match(/scope="col"/g)).toHaveLength(r.columns.length);
    }
  }
  const parameters = html(createElement(Parameters, { params: wire.params as main.ParamsResult,
    busy: '', words: d.words }));
  expect(parameters.match(/scope="col"/g)).toHaveLength(4);
  const widths = [1, 2, 3, 4].map(i => Number(css.match(new RegExp(
    `th:nth-child\\(${i}\\) \\{ width: (\\d+)%;`))?.[1]));
  expect(widths).toEqual([46, 14, 16, 24]);
  expect(widths.reduce((sum, width) => sum + width, 0)).toBe(100);
  for (const width of widths) expect(panel * width / 100).toBeGreaterThan(padding + minimum);
  const crosswalk = html(createElement(ReportView, { report: reports[4], howLabel: d.words.crosswalkHow }));
  expect(crosswalk).toContain('Gosnell, Stephen');
  expect(crosswalk).toContain('WR · MIN');
  expect(crosswalk).not.toContain('0816');
  expect(crosswalk).not.toContain('Cincinnati Bengals');
});

it('places and retains progress, success and failure beside every scope button', async () => {
  for (const scope of data().scopes) {
    const p = provider();
    let finish!: (v: ReturnType<typeof parseOutcome>) => void;
    p.refresh = async () => new Promise(resolve => { finish = resolve; });
    const state = createDataState(p);
    const pending = state.run({ kind: 'refresh', scope: scope.id });
    const busy = html(createElement(RefreshScopes, { data: data(), busy: state.read().busy }));
    expect(busy).toContain(`${scope.action}</button><span role="status" aria-live="polite">Refreshing…`);
    finish({ ok: true, error: '', summary: `Finished ${scope.label}` });
    await pending;
    expect(state.read().outcomes[scope.id]).toBe(`Finished ${scope.label}`);
    await state.run({ kind: 'read' });
    expect(state.read().outcomes[scope.id]).toBe(`Finished ${scope.label}`);
    expect(state.read().outcomes.read).toBe(data().words.readResult);
    const done = html(createElement(RefreshScopes, { data: data(), busy: '',
      outcomes: state.read().outcomes }));
    expect(done).toContain(`aria-live="polite">Finished ${scope.label}`);
    p.refresh = async () => ({ ok: false, error: `Failed ${scope.label}`, summary: '' });
    await state.run({ kind: 'refresh', scope: scope.id });
    expect(state.read().outcomes[scope.id]).toBe(`Failed ${scope.label}`);
    await state.run({ kind: 'read' });
    expect(state.read().outcomes[scope.id]).toBe(`Failed ${scope.label}`);
    p.refresh = async () => { throw new Error('transport unavailable'); };
    await state.run({ kind: 'refresh', scope: scope.id });
    expect(state.read().outcomes[scope.id]).toContain('transport unavailable');
  }
});

it('keeps held-read progress and its Go result beside the button, including failures', async () => {
  const p = provider();
  const state = createDataState(p);
  let finish!: (v: ReturnType<typeof data>) => void;
  p.read = async () => new Promise(resolve => { finish = resolve; });
  const pending = state.run({ kind: 'read' });
  expect(state.read().busy).toBe('read');
  const busy = html(createElement(HeldRead, { words: data().words, busy: state.read().busy }));
  expect(busy).toContain('disabled=""');
  expect(busy).toContain('Read held records</button><span role="status" aria-live="polite">Refreshing…');
  finish(data());
  await pending;
  const done = html(createElement(HeldRead, { words: data().words, busy: state.read().busy,
    result: state.read().outcomes.read }));
  expect(done).toContain(data().words.readResult);
  p.read = async () => { throw new Error('held reader unavailable'); };
  await state.run({ kind: 'read' });
  expect(state.read().outcomes.read).toContain('held reader unavailable');
  it('turns photos off when they are on, and keeps Go\'s result line', async () => {
    const asked: boolean[] = [];
    const p = { ...provider(), photos: async (on: boolean) => {
      asked.push(on); return { ok: true, error: '', summary: 'Player photos are off.' }; } };
    const state = createDataState(p);
    await state.run({ kind: 'read' });
    expect(state.read().data!.photosOn).toBe(true);
    await state.run({ kind: 'photos', on: false });
    expect(asked).toEqual([false]);
    expect(state.read().outcomes.photos).toBe('Player photos are off.');
    const markup = html(createElement(RefreshScopes, { data: state.read().data!, busy: '' }));
    expect(markup).toContain('Turn photos off');
  });
});
