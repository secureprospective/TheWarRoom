import { subscribeDraft } from './api';
import { useEffect, type ReactNode } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import type { EndpointId } from '../registry';
import { ReportView } from '../reports/ReportView';
export { ReportView } from '../reports/ReportView';
import { reportState } from './state';
import type { Board } from './provider';
import './reports.css';

export function ReportNavigation({ board, selected }: { board: Board; selected: readonly string[] }) {
  const report = board.reports.find(r => selected.includes(r.id)) ?? board.reports[0];
  return <><h4>{board.title}</h4>
    <div className="market-controls" aria-label={board.title}>
      {board.reports.map(r => <Act variant="text"
        key={r.id} verb="endpoint.open" args={{ id: r.id as EndpointId }}
        label={r.title} active={report.id === r.id}>{r.title}</Act>)}
    </div></>;
}
export function ReportsView({ board, selected, refreshVerb = 'draft.reports', controls }: {
  board: Board; selected: readonly string[];
  refreshVerb?: 'draft.reports' | 'market.reports' | 'race.reports' | 'now.reports' | 'research.read';
  controls?: ReactNode;
}) {
  const report = board.reports.find(r => selected.includes(r.id)) ?? board.reports[0];
  return <div className="market draft-reports">
    <ReportNavigation board={board} selected={selected} />
    <div className="market-controls">
      {refreshVerb === 'race.reports' || refreshVerb === 'now.reports' || refreshVerb === 'research.read'
        ? <Act variant="button"
          verb={refreshVerb} args={{ action: 'refresh', value: '' }} label={board.refresh}>
          {board.refresh}
        </Act>
        : <Act variant="button"
          verb={refreshVerb} args={{ action: 'refresh' }} label={board.refresh}>{board.refresh}</Act>}
      {board.mflLabel && <Act variant="button"
        verb="draft.reports" args={{ action: 'mfl' }} disabled={!board.mflUrl}
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
    return subscribeDraft(() => { void reportState.refresh(); });
  }, []);
  return <>
    {state.pending && <p role="status">{state.board?.loading ?? 'Reading MFL draft reports…'}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.board && <ReportsView board={state.board} selected={selected} />}
  </>;
}
