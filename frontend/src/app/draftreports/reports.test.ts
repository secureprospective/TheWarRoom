import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { endpointById, placementRoute } from '../registry';
import { ReportView, ReportsView } from './Reports';
import { parseBoard, type ReportProvider } from './provider';
import { createReportState } from './state';
import wire from './wire.testdata.json';

const board = () => parseBoard(wire);
const html = (node: Parameters<typeof renderToStaticMarkup>[0]) => renderToStaticMarkup(node);
describe('Go-authored draft reports', () => {
  it.each(wire.reports.map(r => [r.id]))('%s renders its Go report and provenance', (id) => {
    const b = board();
    const r = b.reports.find(r => r.id === id)!;
    const markup = html(createElement(ReportView, { report: r, howLabel: b.howLabel }));
    for (const word of [r.title, r.source, r.asOf, b.howLabel, r.columns[0]]) expect(markup).toContain(word);
    expect(markup).toContain(`data-endpoint="${id}"`);
    expect(markup).toContain(r.rows[0].cells[0]);
    expect(markup).toContain(r.how.replaceAll("'", '&#x27;'));
  });
  it('opens the endpoint-selected report and supplies nav and command routes', () => {
    for (const report of board().reports) {
      const id = report.id as Parameters<typeof endpointById>[0];
      expect(endpointById(id).status).toBe('built');
      const route = placementRoute(endpointById(id).placement);
      expect(route).toMatchObject({ node: 'trade', workspace: 'draft-room' });
      const markup = html(createElement(ReportsView, { board: board(), selected: [id] }));
      expect(markup).toContain(`data-endpoint="${id}"`);
      expect(markup).toContain(`aria-label="${report.title}" aria-pressed="true"`);
    }
  });
  it('shows a point distance bar, not a letter grade, with a baseline', () => {
    const r = board().reports.find(r => r.id === 'M-065')!;
    const markup = html(createElement(ReportView, { report: r, howLabel: board().howLabel }));
    expect(markup).toContain('+10.00 points/game');
    expect(markup).toContain('150% of same-round expected value');
    expect(markup).toContain('class="reference"');
    expect(markup).not.toContain('Grade');
  });
  it('does not render invented rows for unavailable or empty exports', () => {
    const r = board().reports[0];
    for (const status of ['failed', 'empty'] as const) {
      const report = { ...r, status, rows: [], note: 'Export unavailable' };
      const markup = html(createElement(ReportView, { report, howLabel: board().howLabel }));
      expect(markup).toContain('Export unavailable');
      expect(markup).not.toContain('<table');
      expect(markup).toContain('How this is counted');
    }
  });
  it('rejects malformed IPC output', () => {
    expect(() => parseBoard({ ...wire, extra: true })).toThrow();
    const wrongColumns = { ...wire, reports: [{ ...wire.reports[0], columns: [] }] };
    expect(() => parseBoard(wrongColumns)).toThrow(/column count/);
    expect(() => parseBoard({ ...wire, mflUrl: 'https://bad/action' })).toThrow(/read link/);
  });
});

describe('held report requests', () => {
  it('keeps the last good view on refresh failure and opens MFL only explicitly', async () => {
    let fail = false;
    let opens = 0;
    const reader: ReportProvider = {
      read: async () => { if (fail) throw new Error('offline'); return board(); },
      refresh: async () => {}, openMFL: async () => { opens++; },
    };
    const state = createReportState(reader);
    await state.refresh();
    expect(state.peek().board).toEqual(board());
    expect(opens).toBe(0);
    fail = true;
    await state.action('refresh');
    expect(state.peek().error).toContain('offline');
    expect(state.peek().board).toEqual(board());
    await state.action('mfl');
    expect(opens).toBe(1);
  });
});

it('rejects an older read after a newer refresh has landed', async () => {
  const replies: ((b: ReturnType<typeof board>) => void)[] = [];
  const state = createReportState({
    refresh: async () => {}, openMFL: async () => {},
    read: () => new Promise(resolve => { replies.push(resolve); }),
  });
  const old = state.refresh();
  await Promise.resolve();
  const newer = state.refresh();
  await Promise.resolve();
  const current = { ...board(), title: 'Newer Go reading' };
  replies[1](current);
  await newer;
  replies[0](board());
  await old;
  expect(state.peek().board).toEqual(current);
  expect(state.peek().pending).toBe(false);
});
