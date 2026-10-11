import { Act } from '../commands/Act';
import { RangeBar } from '../market/ReadingView';
import { reportSort, sortedRows } from './sortState';
import type { Report } from './contract';
import '../draftreports/reports.css';

function ReportTable({ report }: { report: Report }) {
  const sort = reportSort.use()[report.id];
  const rows = sortedRows(report.rows, sort);
  const sortLabel = (i: number) => sort?.column === i && !sort.desc
    ? report.sortDescending : report.sortAscending;
  return <div className="draft-table-scroll" tabIndex={0} role="region" aria-label={report.title}>
    <table><thead><tr>{report.columns.map((c, i) =>
      <th key={i} scope="col"
        aria-sort={sort?.column === i ? (sort.desc ? 'descending' : 'ascending') : 'none'}>
        {report.rows.every(row => row.sort) && report.sortAscending ?
          <Act verb="report.sort" args={{ id: report.id, column: i }}
            label={`${sortLabel(i)} · ${c}`}>
            {c}
          </Act> : c}
      </th>)}</tr></thead>
      <tbody>{rows.map((row, i) => <tr key={i}>{row.cells.map((cell, j) =>
        <td key={j} title={row.titles?.[j]}
          className={report.columns[j] === 'Local time' ? 'draft-local-time' : undefined}>
          {cell.split('\n').map((line, k) => <div className="draft-cell-line" key={k}>{line}</div>)}
          {j === row.cells.length - 1 && row.range && <RangeBar range={row.range} />}
        </td>)}</tr>)}</tbody>
    </table>
  </div>;
}
export function ReportView({ report, howLabel }: { report: Report; howLabel: string }) {
  return <section className="draft-report" data-endpoint={report.id} aria-label={report.title}>
    <h5>{report.title}</h5>
    <p>{report.source} · <span className="draft-local-time">{report.asOf}</span></p>
    {report.note && <p role={report.status === 'failed' ? 'alert' : 'status'}>{report.note}</p>}
    {report.summary.length > 0 && <table className="draft-summary"><tbody>{report.summary.map((row, i) =>
      <tr key={i}><th scope="row">{row.cells[0]}</th><td>{row.cells[1]}</td></tr>)}</tbody></table>}
    {report.link && <div className="market-controls">
      <Act verb="endpoint.open" args={{ id: report.link.endpoint }} label={report.link.label}>
        {report.link.label}
      </Act>
    </div>}
    {report.status !== 'failed' && report.status !== 'empty' && report.rows.length > 0 &&
      (report.id === 'M-102' ?
        <div>{report.rows.map((row, i) => <details key={i} className="draft-pick-detail">
          <summary>{row.cells[0]} · {row.cells[2]}</summary>
          <dl className="kv">{row.cells.map((cell, j) => <div key={j}>
            <dt>{report.columns[j]}</dt>
            <dd className={report.columns[j] === 'Local time' ? 'draft-local-time' : undefined}>{cell}</dd>
          </div>)}</dl>
        </details>)}</div> : <ReportTable report={report} />)}
    <details className="market-how"><summary>{howLabel}</summary><p>{report.how}</p></details>
  </section>;
}
