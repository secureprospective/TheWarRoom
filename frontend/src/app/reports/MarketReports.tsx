import { useEffect } from 'react';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { marketReportState } from './marketState';

export default function MarketReports() {
  const state = marketReportState.use();
  const selected = commands.use().endpointIds;
  useEffect(() => {
    if (!marketReportState.peek().pending) void marketReportState.refresh();
    let disposed = false;
    let stop: (() => void) | undefined;
    void import('../../../wailsjs/runtime/runtime').then(m => {
      if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
        stop = m.EventsOn('target:season', () => { void marketReportState.refresh(); });
    });
    return () => { disposed = true; stop?.(); };
  }, []);
  return <>
    {state.pending && <p role="status">{state.board?.loading}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.board && <ReportsView board={state.board} selected={selected} refreshVerb="market.reports" />}
  </>;
}
