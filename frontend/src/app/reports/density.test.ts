import { readFileSync } from 'node:fs';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { parse } from 'postcss';
import { expect, it } from 'vitest';
import { ReportView } from './ReportView';
import { parseBoard } from './contract';
import { PoolFrame } from '../pool/Pool';
import { parsePool } from '../pool/provider';
import { AssetBoardView } from '../assets/AssetBoardView';
import { heldSnapshot } from '../data/lineupTestData';
import reportWire from './wire.testdata.json';
import poolWire from '../pool/wire.testdata.json';

// Resolve the inherited custom property at the rendered cell's consumer rule without a browser.
function padding(markup: string, sheet: string, selector: string) {
  expect(markup).toContain('<td');
  const density = markup.match(/data-density="([^"]+)"/)![1];
  let inherited: string | undefined;
  parse(readFileSync('src/app/look/look.css', 'utf8')).walkRules(rule => {
    if (!rule.selector.includes(`[data-density="${density}"]`)) return;
    rule.walkDecls('--density-table-padding', decl => { inherited = decl.value; });
  });
  let cell = '';
  parse(readFileSync(sheet, 'utf8')).walkRules(rule => {
    if (rule.selector !== selector) return;
    rule.walkDecls('padding', decl => { cell = decl.value; });
  });
  expect(cell).not.toBe('');
  return cell.replace(/var\(--density-table-padding, ([^)]+)\)/,
    (_match, fallback: string) => inherited && inherited !== 'initial' ? inherited : fallback);
}

it.each(['report', 'dense report', 'pool', 'asset board'])
('resolves different narrative and matrix padding on rendered %s cells, preserving tactical', kind => {
  const report = parseBoard(reportWire.board).reports[0];
  if (kind === 'dense report') {
    report.columns = ['One', 'Two', 'Three', 'Four', 'Five', 'Six'];
    report.rows = [{ cells: ['Main\nSub-line', '2', '3', '4', '5', '6'], range: null }];
  }
  const table = kind === 'pool'
    ? createElement(PoolFrame, { reading: parsePool(poolWire.pool), refreshing: false })
    : kind === 'asset board' ? createElement(AssetBoardView, { snapshot: heldSnapshot() })
    : createElement(ReportView, { report, howLabel: 'How' });
  const sheet = kind === 'pool' ? 'src/app/pool/pool.css' : kind === 'asset board'
    ? 'src/app/shell/shell.css' : 'src/app/draftreports/reports.css';
  const selector = kind === 'pool' ? '.twr-app .pool-table :is(th, td)' : kind === 'asset board'
    ? '.twr-app .report-table :is(th, td)' : kind === 'dense report'
      ? '.draft-report .draft-table-scroll table.report-dense :is(th, td)'
      : '.draft-report .draft-table-scroll :is(th, td)';
  const render = (density: string) => renderToStaticMarkup(createElement('div',
    { className: 'twr-app', 'data-density': density }, table));
  expect(padding(render('narrative'), sheet, selector)).toBe('12px 10px');
  expect(padding(render('matrix'), sheet, selector)).toBe('3px 4px');
  expect(padding(render('tactical'), sheet, selector)).toBe(kind === 'pool' ? '4px 6px' :
    kind === 'dense report' ? '8px 2px' : '8px 6px');
  if (kind === 'dense report') {
    expect(render('matrix')).toContain('class="draft-cell-line">Main</div>');
    expect(render('matrix')).toContain('class="draft-cell-line">Sub-line</div>');
  }
});
