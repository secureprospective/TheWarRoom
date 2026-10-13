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
      expect(endpointById(id).status).not.toBe('spec');
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

it('renders one Go-written pick per line in grid and recap cells', () => {
  for (const id of ['M-064', 'M-066']) {
    const report = { ...board().reports.find(r => r.id === id)!,
      rows: [{ cells: id === 'M-064' ? ['1', '1.01 · First\n1.02 · Second', 'No recorded pick'] :
        ['One', '2', '1.01 · First\n1.02 · Second'], range: null }] };
    const markup = html(createElement(ReportView, { report, howLabel: board().howLabel }));
    expect(markup).toContain('<div class="draft-cell-line">1.01 · First</div>' +
      '<div class="draft-cell-line">1.02 · Second</div>');
  }
});

it('shows live-room status and the Go-written see-every-pick endpoint Act', () => {
  const report = board().reports.find(r => r.id === 'M-183')!;
  const markup = html(createElement(ReportView, { report, howLabel: board().howLabel }));
  expect(markup).toContain('2026 rookie draft complete');
  expect(markup).not.toContain('MFL does not report who is on the clock.');
  expect(markup).toContain('See every pick');
  expect(markup).toContain('aria-label="See every pick"');
  expect(markup).toContain('data-endpoint="M-183"');
  const bad = { ...wire, reports: [{ ...wire.reports[8], summary: [{cells: ['bad'],range: null}] }] };
  expect(() => parseBoard(bad)).toThrow(/status line/);
});

it('contains the wide grid in a zero-minimum shell track and local table scroller', async () => {
  const { readFileSync } = await import('node:fs');
  const css = readFileSync('src/app/draftreports/reports.css', 'utf8');
  const shell = readFileSync('src/app/shell/shell.css', 'utf8');
  const bodyRule = shell.match(/\.twr-app \.wbody \{([^}]+)\}/)?.[1];
  expect(bodyRule).toContain('grid-template-columns: minmax(0, 1fr)');
  for (const selector of ['.draft-reports', '.draft-report']) {
    const rule = css.slice(css.indexOf(`${selector} {`)).split('}')[0];
    expect(rule).toContain('min-width: 0');
    expect(rule).toContain('max-width: 100%');
  }
  const scroll = css.slice(css.indexOf('.draft-table-scroll {')).split('}')[0];
  expect(scroll).toContain('overflow: auto');
  expect(scroll).toContain('max-width: 100%');
  const report = board().reports.find(r => r.id === 'M-064')!;
  const columns = ['Round', ...Array.from({length: 32}, (_, i) => `Franchise ${i+1}`)];
  const markup = html(createElement(ReportView, { report: { ...report, columns,
    rows: [{ cells: columns.map((_, i) => i ? '1.01 · A long player name' : '1'), range: null }] },
    howLabel: board().howLabel }));
  expect(markup).toContain('class="draft-table-scroll"');
  expect(markup.match(/scope="col"/g)).toHaveLength(33);
});

it('keeps local-time cells on one line', async () => {
  const { readFileSync } = await import('node:fs');
  const css = readFileSync('src/app/draftreports/reports.css', 'utf8');
  expect(css).toMatch(/\.draft-report \.draft-local-time \{ white-space: nowrap;/);
  const markup = html(createElement(ReportView,
    { report: board().reports[0], howLabel: board().howLabel }));
  expect(markup).toContain('class="draft-local-time"');
});

it('See every pick dispatches M-063 through the existing Act', async () => {
  const { Act } = await import('../commands/Act');
  const { commands } = await import('../commands/registry');
  const report = board().reports.find(r => r.id === 'M-183')!;
  const view = ReportView({report, howLabel: board().howLabel});
  type Node = {type?: unknown; props?: {label?: string; className?: string; children?: unknown}};
  const controls = view.props.children.find((child: unknown) =>
    (child as Node)?.props?.className === 'market-controls') as Node;
  const link = controls.props?.children as Node & {props: Parameters<typeof Act>[0]};
  expect(link.type).toBe(Act);
  expect(link.props.label).toBe('See every pick');
  expect(link.props.verb).toBe('endpoint.open');
  expect(link.props.args).toEqual({id: 'M-063'});
  commands.dispatch('endpoint.open', {id: 'M-183'});
  Act(link.props).props.onClick();
  expect(commands.read().endpointIds).toEqual(['M-063']);
});
