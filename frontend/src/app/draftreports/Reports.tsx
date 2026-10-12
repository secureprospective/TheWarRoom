import { useEffect, type ReactNode } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import type { EndpointId } from '../registry';
import { ReportView } from '../reports/ReportView';
export { ReportView } from '../reports/ReportView';
import { reportState } from './state';
import type { Board } from './provider';
import './reports.css';

export function ReportsView({ board, selected, refreshVerb = 'draft.reports', controls }: {
  board: Board; selected: readonly string[];
  refreshVerb?: 'draft.reports' | 'market.reports' | 'race.reports' | 'now.reports' | 'research.read'
    | 'archive.reports';
  controls?: ReactNode;
}) {
  const report = board.reports.find(r => selected.includes(r.id)) ?? board.reports[0];
  return <div className="market draft-reports">
    <h4>{board.title}</h4>
    <div className="market-controls" aria-label={board.title}>
      {board.reports.map(r => <Act key={r.id} verb="endpoint.open" args={{ id: r.id as EndpointId }}
        label={r.title} active={report.id === r.id}>{r.title}</Act>)}
    </div>
    <div className="market-controls">
      {refreshVerb === 'race.reports' || refreshVerb === 'now.reports' || refreshVerb === 'research.read'
        || refreshVerb === 'archive.reports'
        ? <Act verb={refreshVerb} args={{ action: 'refresh', value: '' }} label={board.refresh}>
          {board.refresh}
        </Act>
        : <Act verb={refreshVerb} args={{ action: 'refresh' }} label={board.refresh}>{board.refresh}</Act>}
      {board.mflLabel && <Act verb="draft.reports" args={{ action: 'mfl' }} disabled={!board.mflUrl}
        label={board.mflLabel}>{board.mflLabel}</Act>}
    </div>
    {controls}
    <ReportView key={report.id} report={report} howLabel={board.howLabel} />
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
