import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ReportView } from './ReportView';
import { ReportsView } from '../draftreports/Reports';
import { parseBoard, range } from './contract';
import { sortedRows, reportSort } from './sortState';
import { endpointById, placementRoute } from '../registry';
import { PoolFrame, PoolInspectorView } from '../pool/Pool';
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
    expect(endpoint.status).toBe('built');
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

it('pool columns and the Inspector render the same Go facts, with missing shown as a dash', () => {
  const reading = parsePool(poolWire.pool);
  const row = { ...reading.value.rows[0], market: wire.facts };
  reading.value.rows = [row];
  const table = html(createElement(PoolFrame, { reading, refreshing: false }));
  const inspector = html(createElement(PoolInspectorView, { row, plans: [], busy: '' }));
  for (const fact of wire.facts) {
    expect(table).toContain(fact.label);
    expect(table).toContain(fact.text);
    expect(inspector).toContain(fact.text);
    expect(table).toContain(`title="${fact.reason}"`);
  }
  const missing = { ...row, age: '27', market: wire.missing };
  const empty = html(createElement(PoolInspectorView, { row: missing, plans: [], busy: '' }));
  expect(empty.match(/<dd>—<\/dd>/g)).toHaveLength(4);
  expect(empty).toContain('unavailable; refresh from MFL to check.');
  expect(empty).not.toContain('<dd>0</dd>');
});
