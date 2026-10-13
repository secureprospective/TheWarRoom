import { readFileSync } from 'node:fs';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ReportView } from './ReportView';
import { ReportsView } from '../draftreports/Reports';
import { parseBoard, range } from './contract';
import { sortedRows, reportSort } from './sortState';
import { endpointById, placementRoute } from '../registry';
import { PoolInspectorView } from '../pool/Pool';
import { parsePool } from '../pool/provider';
import wire from './wire.testdata.json';
import poolWire from '../pool/wire.testdata.json';

const board = () => parseBoard(wire.board);
const html = renderToStaticMarkup;
describe('Go player market reports', () => {
  it.each(wire.board.reports.map(r => [r.id]))('%s renders Go values and both report routes', id => {
    const b = board();
    const r = b.reports.find(r => r.id === id)!;
    const markup = html(createElement(ReportView, { report: r, howLabel: b.howLabel }));
    for (const text of [r.title, r.source, r.asOf, b.howLabel, r.columns[1], r.rows[0].cells[1]])
      expect(markup).toContain(text.replaceAll('&', '&amp;').replaceAll("'", '&#x27;'));
    const endpoint = endpointById(id as Parameters<typeof endpointById>[0]);
    expect(endpoint.status).not.toBe('spec');
    expect(placementRoute(endpoint.placement)).toEqual({ node: 'war', workspace: 'valuations-and-pool' });
    const view = html(createElement(ReportsView,
      { board: b, selected: [id], refreshVerb: 'market.reports' }));
    expect(view).toContain(`data-endpoint="${id}"`);
    expect(view).toContain(`aria-label="${r.title}" aria-pressed="true"`);
    expect(view).toContain(b.refresh);
    expect(view).not.toContain('Open MFL draft');
  });
  it('shows honest units and full min–max ranges, not grades or league dollars', () => {
    for (const id of ['M-067', 'M-069']) {
      const r = board().reports.find(r => r.id === id)!;
      const markup = html(createElement(ReportView, { report: r, howLabel: board().howLabel }));
      expect(markup).toContain('MFL-wide');
      expect(markup).toContain('class="range"');
      expect(markup).not.toContain('class="reference"');
      expect(markup).not.toContain('grade="');
      expect(markup).not.toContain('$');
      expect(r.rows[0].range?.hi).toBeGreaterThan(r.rows[0].range!.lo);
    }
    expect(board().reports.find(r => r.id === 'M-069')!.how).toContain('not this league\'s dollars');
  });
  it('keeps a group together under one heading and sorts inside it', () => {
    const rows = [
      { cells: ['A low', '1'], range: null, sort: [0, 1], group: 'East' },
      { cells: ['A high', '9'], range: null, sort: [1, 9], group: 'East' },
      { cells: ['B mid', '5'], range: null, sort: [2, 5], group: 'West' },
    ];
    expect(sortedRows(rows, { column: 1, desc: true }).map(r => r.cells[0]))
      .toEqual(['A high', 'A low', 'B mid']);
    const report = { id: 'M-002', title: 'Standings', columns: ['Franchise', 'Points'], rows, summary: [],
      source: 'Source: MFL', asOf: 'As of: Oct 11, 2026, 2:14 PM', note: '', how: 'how', status: 'ok' };
    const html = renderToStaticMarkup(
      createElement(ReportView, { report: report as never, howLabel: 'How' }));
    expect(html.match(/scope="colgroup"/g)).toHaveLength(2);
    expect(html).toMatch(/colspan="2">East<\/th>/i);
  });
  it('sorts by Go numeric keys, never parsed display strings, with stable ties', () => {
    const rows = [
      { cells: ['Ten', 'unparseable'], range: null, sort: [0, 10] },
      { cells: ['Two', 'unparseable'], range: null, sort: [1, 2] },
      { cells: ['Also two', 'unparseable'], range: null, sort: [2, 2] },
    ];
    expect(sortedRows(rows, { column: 1, desc: false }).map(r => r.cells[0]))
      .toEqual(['Two', 'Also two', 'Ten']);
    expect(sortedRows(rows, { column: 1, desc: true }).map(r => r.cells[0]))
      .toEqual(['Ten', 'Two', 'Also two']);
    expect(rows[0].cells[0]).toBe('Ten');
    reportSort.set({ id: 'M-067', column: 1 });
    expect(() => reportSort.set({ id: 'M-067', column: 3 })).toThrow();
    expect(() => reportSort.set({ id: 'unknown', column: 1 })).toThrow();
  });
  it('labels independent missing or empty feeds without invented rows', () => {
    for (const status of ['failed', 'empty'] as const) {
      const r = { ...board().reports[0], status, rows: [], note: 'MFL reports no draft picks.' };
      const markup = html(createElement(ReportView, { report: r, howLabel: board().howLabel }));
      expect(markup).toContain(r.note);
      expect(markup).not.toContain('<table');
    }
  });
  it('rejects malformed sorting metadata and ranges', () => {
    const b = wire.board;
    const report = b.reports[0];
    expect(() => parseBoard({ ...b, reports: [{ ...report,
      rows: [{ ...report.rows[0], sort: [NaN] }] }] })).toThrow();
    expect(() => range({ value: 2, lo: 3, hi: 4, text: 'Bad' }, 'range')).toThrow();
  });
});

it('the Inspector keeps full Go facts and ranges, with missing shown as a dash', () => {
  const reading = parsePool(poolWire.pool);
  const row = { ...reading.value.rows[0], market: wire.facts };
  reading.value.rows = [row];
  const inspector = html(createElement(PoolInspectorView, { row, plans: [], busy: '' }));
  for (const fact of wire.facts) {
    expect(inspector).toContain(fact.text);
    expect(inspector).toContain(`title="${fact.reason}"`);
  }
  const missing = { ...row, age: '27', market: wire.missing };
  const empty = html(createElement(PoolInspectorView, { row: missing, plans: [], busy: '' }));
  expect(empty.match(/<dd>—<\/dd>/g)).toHaveLength(4);
  expect(empty).toContain('unavailable; refresh from MFL to check.');
  expect(empty).not.toContain('<dd>0</dd>');
});


it('fits compact and wide report tables to their pane with a local scroller', () => {
  const report = board().reports[0];
  for (const n of [2, 3, 4]) {
    const r = { ...report, columns: Array.from({ length: n }, (_, i) => `Column ${i}`) };
    const markup = html(createElement(ReportView, { report: r, howLabel: board().howLabel }));
    expect(markup.includes('class="report-compact"')).toBe(n <= 3);
    expect(markup).toContain('class="draft-table-scroll"');
  }
  const css = readFileSync('src/app/draftreports/reports.css', 'utf8');
  expect(css).toContain('.draft-report .draft-table-scroll table { width: 100%; table-layout: fixed; }');
  expect(css).toContain('.draft-table-scroll { min-width: 0; overflow: auto; max-width: 100%;');
});
