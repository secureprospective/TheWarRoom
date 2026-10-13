import type { ReactNode } from 'react';
import { Act } from '../commands/Act';
import { RangeBar } from '../market/ReadingView';
import { reportSort, sortedRows } from './sortState';
import type { Report } from './contract';
import '../draftreports/reports.css';

// WeekBars is a row's values as bars on the report's one scale, zero to barMax.
export function WeekBars({ bars, labels, max }: { bars: number[]; labels: string[]; max: number }) {
  const width = 300 / Math.max(bars.length, 1);
  return <svg className="report-bars" viewBox="0 0 300 40" role="img"
    aria-label={bars.map((b, i) => `${labels[i]}: ${b.toFixed(2)}`).join(', ')}>
    <line className="axis" x1="0" x2="300" y1="39.5" y2="39.5" />
    {bars.map((b, i) => <rect key={i} x={i * width + 1} width={Math.max(width - 2, 1)}
      y={39 - (b / max) * 38} height={(b / max) * 38}>
      <title>{`${labels[i]}: ${b.toFixed(2)}`}</title>
    </rect>)}
  </svg>;
}
type CellRenderer = (cell: string, row: number, column: number) => ReactNode;
function ReportTable({ report, renderCell }: { report: Report; renderCell?: CellRenderer }) {
  const sort = reportSort.use()[report.id];
  const rows = sortedRows(report.rows, sort);
  const sortLabel = (i: number) => sort?.column === i && !sort.desc
    ? report.sortDescending : report.sortAscending;
  return <div className="draft-table-scroll" tabIndex={0} role="region" aria-label={report.title}>
    <table className={`report-table ${report.columns.length <= 3 ? 'report-compact'
      : report.columns.length >= 6 ? 'report-dense' : ''}`}>
      <thead><tr>{report.columns.map((c, i) =>
      <th key={i} scope="col"
        aria-sort={sort?.column === i ? (sort.desc ? 'descending' : 'ascending') : 'none'}>
        {report.rows.every(row => row.sort) && report.sortAscending ?
          <Act verb="report.sort" args={{ id: report.id, column: i }}
            label={`${sortLabel(i)} · ${c}`}>
            {c}
          </Act> : c}
      </th>)}</tr></thead>
      <tbody>{rows.map((row, i) => [row.group && row.group !== rows[i - 1]?.group &&
        <tr key={`g${i}`} className="report-group"><th scope="colgroup" colSpan={report.columns.length}>
          {row.group}</th></tr>,
      <tr key={i} className={row.mine ? 'report-mine' : undefined}
        aria-current={row.mine ? 'true' : undefined}>{row.cells.map((cell, j) =>
        <td key={j} title={row.titles?.[j]}
          className={report.columns[j] === 'Local time' ? 'draft-local-time' : undefined}>
          {renderCell ? renderCell(cell, i, j) : cell.split('\n').map((line, k) =>
            <div className="draft-cell-line" key={k}>{line}</div>)}
          {j === row.cells.length - 1 && row.range && <RangeBar range={row.range} />}
          {j === row.cells.length - 1 && row.bars && report.barMax && report.barLabels &&
            <WeekBars bars={row.bars} labels={report.barLabels} max={report.barMax} />}
        </td>)}</tr>])}</tbody>
    </table>
  </div>;
}
// A report that heads its page (the Race standings) takes the page heading level.
export function ReportView({ report, howLabel, heading: Heading = 'h5', renderCell }:
  { report: Report; howLabel: string; heading?: 'h4' | 'h5'; renderCell?: CellRenderer }) {
  return <section className="draft-report" data-endpoint={report.id} aria-label={report.title}>
    <Heading>{report.title}</Heading>
    <p>{report.source}{report.asOf && <> · <span className="draft-local-time">{report.asOf}</span></>}</p>
    {report.note && <p role={report.status === 'failed' ? 'alert' : 'status'}>{report.note}</p>}
    {report.summary.length > 0 && <div className="report-summary"
      aria-label={report.id === 'M-154' ? 'Bye-week shortfalls' : 'Report summary'}>
      {report.id === 'M-154' && <h6>Bye-week shortfalls</h6>}
      <table className="draft-summary"><tbody>{report.summary.map((row, i) =>
      <tr key={i}><th scope="row">{row.cells[0]}</th><td>{row.cells[1]}</td></tr>)}</tbody></table></div>}
    {report.link && <div className="market-controls">
      <Act verb="endpoint.open" args={{ id: report.link.endpoint }} label={report.link.label}>
        {report.link.label}
      </Act>
    </div>}
    {report.status !== 'failed' && report.status !== 'empty' && report.rows.length > 0 &&
      (report.id === 'M-089' ?
        <div>{report.rows.map((row, i) => <details key={i} className="draft-pick-detail">
          <summary>{row.cells[0]} · {row.cells[1]}</summary>
          {row.cells[2].split('\n').map((line, j) => <p key={j}>{line}</p>)}
        </details>)}</div> : report.id === 'M-102' ?
        <div>{report.rows.map((row, i) => <details key={i} className="draft-pick-detail">
          <summary>{row.cells[0]} · {row.cells[2]}</summary>
          <dl className="kv">{row.cells.map((cell, j) => <div key={j}>
            <dt>{report.columns[j]}</dt>
            <dd className={report.columns[j] === 'Local time' ? 'draft-local-time' : undefined}>{cell}</dd>
          </div>)}</dl>
        </details>)}</div> : <ReportTable report={report} renderCell={renderCell} />)}
    <details className="market-how"><summary>{howLabel}</summary><p>{report.how}</p></details>
  </section>;
}
