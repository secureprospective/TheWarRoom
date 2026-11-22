import { useEffect } from 'react';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { marketReportState } from './marketState';
import { subscribeSeason } from '../market/api';

export default function MarketReports() {
  const state = marketReportState.use();
  const selected = commands.use().endpointIds;
  useEffect(() => {
    if (!marketReportState.peek().pending) void marketReportState.refresh();
    return subscribeSeason(() => { void marketReportState.refresh(); });
  }, []);
  return <>
    {state.pending && <p role="status">{state.board?.loading}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.board && <ReportsView board={state.board} selected={selected} refreshVerb="market.reports" />}
  </>;
}
