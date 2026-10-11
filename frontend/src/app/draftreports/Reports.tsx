import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import type { EndpointId } from '../registry';
import { RangeBar } from '../market/ReadingView';
import { reportState } from './state';
import type { Board, Report } from './provider';
import './reports.css';

export function ReportView({ report, howLabel }: { report: Report; howLabel: string }) {
  return <section className="draft-report" data-endpoint={report.id} aria-label={report.title}>
    <h5>{report.title}</h5>
    <p>{report.source} · <span className="draft-local-time">{report.asOf}</span></p>
    {report.note && <p role={report.status === 'failed' ? 'alert' : 'status'}>{report.note}</p>}
    {report.summary.length > 0 && <table className="draft-summary"><tbody>{report.summary.map((row, i) =>
      <tr key={i}><th scope="row">{row.cells[0]}</th><td>{row.cells[1]}</td></tr>)}</tbody></table>}
    {report.seeEveryPick && <div className="market-controls">
      <Act verb="endpoint.open" args={{ id: 'M-063' }} label={report.seeEveryPick}>{report.seeEveryPick}</Act>
    </div>}
    {report.status !== 'failed' && report.status !== 'empty' && report.rows.length > 0 &&
      (report.id === 'M-102' ?
      <div>{report.rows.map((row, i) => <details key={i} className="draft-pick-detail">
        <summary>{row.cells[0]} · {row.cells[2]}</summary>
        <dl className="kv">{row.cells.map((cell, j) => <div key={j}>
          <dt>{report.columns[j]}</dt>
          <dd className={report.columns[j] === 'Local time' ? 'draft-local-time' : undefined}>{cell}</dd>
        </div>)}</dl>
      </details>)}</div> :
      <div className="draft-table-scroll" tabIndex={0} role="region" aria-label={report.title}>
        <table><thead><tr>{report.columns.map((c, i) => <th key={i} scope="col">{c}</th>)}</tr></thead>
          <tbody>{report.rows.map((row, i) => <tr key={i}>{row.cells.map((cell, j) =>
            <td key={j} className={report.columns[j] === 'Local time' ? 'draft-local-time' : undefined}>
              {cell.split('\n').map((line, k) => <div className="draft-cell-line" key={k}>{line}</div>)}
              {j === row.cells.length - 1 && row.range &&
              <RangeBar range={row.range} />}</td>)}</tr>)}</tbody>
        </table>
      </div>)}
    <details className="market-how"><summary>{howLabel}</summary><p>{report.how}</p></details>
  </section>;
}
export function ReportsView({ board, selected }: { board: Board; selected: readonly string[] }) {
  const report = board.reports.find(r => selected.includes(r.id)) ?? board.reports[0];
  return <div className="market draft-reports">
    <h4>{board.title}</h4>
    <div className="market-controls" aria-label={board.title}>
      {board.reports.map(r => <Act key={r.id} verb="endpoint.open" args={{ id: r.id as EndpointId }}
        label={r.title} active={report.id === r.id}>{r.title}</Act>)}
    </div>
    <div className="market-controls">
      <Act verb="draft.reports" args={{ action: 'refresh' }} label={board.refresh}>{board.refresh}</Act>
      <Act verb="draft.reports" args={{ action: 'mfl' }} disabled={!board.mflUrl}
        label={board.mflLabel}>{board.mflLabel}</Act>
    </div>
    <ReportView report={report} howLabel={board.howLabel} />
  </div>;
}
export default function Reports() {
  const state = reportState.use();
  const selected = commands.use().endpointIds;
  useEffect(() => {
    if (!reportState.peek().pending) void reportState.refresh();
    const runtime = import('../../../wailsjs/runtime/runtime');
    let disposed = false;
    let stop: (() => void) | undefined;
    void runtime.then(m => {
      if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
        stop = m.EventsOn('target:season', () => { void reportState.refresh(); });
    });
    return () => { disposed = true; stop?.(); };
  }, []);
  return <>
    {state.pending && <p role="status">{state.board?.loading ?? 'Reading MFL draft reports…'}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.board && <ReportsView board={state.board} selected={selected} />}
  </>;
}
