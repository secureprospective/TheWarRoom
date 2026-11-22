import { lazy, Suspense, useEffect } from 'react';
const LiveGame = lazy(() => import('./LiveGame'));
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { ReportView } from '../reports/ReportView';
import { nowState } from './state';
import type { Now as NowBoard } from './contract';

export function NowView({ now, selected }: { now: NowBoard; selected: readonly string[] }) {
  return <>
    <div className="market-controls race-choices" aria-label="Week shown">
      <label>Week{' '}
        <select value={String(now.week)} disabled={now.weeks.length === 0}
          onChange={e => commands.dispatch('now.reports', { action: 'week', value: e.target.value })}>
          {now.weeks.map(w => <option key={w.value} value={w.value}>{w.label}</option>)}
        </select>
      </label>
    </div>
    <div className="draft-reports race-standings">
      <ReportView report={now.matchup} howLabel={now.board.howLabel} heading="h4" />
    </div>
    <Suspense fallback={<p role="status">Reading held games…</p>}><LiveGame /></Suspense>
    <div className="market-controls race-choices">
      <label>Weekly reports{' '}
        <select value={String(now.reportWeek)} disabled={now.reportWeeks.length === 0}
          onChange={e => commands.dispatch('now.reports', { action: 'reports-week', value: e.target.value })}>
          {now.reportWeeks.map(w => <option key={w.value} value={w.value}>{w.label}</option>)}
        </select>
      </label>
    </div>
    <ReportsView board={now.board} selected={selected} refreshVerb="now.reports" />
  </>;
}

export default function Now() {
  const state = nowState.use();
  const selected = commands.use().endpointIds;
  const mine = commands.use().franchiseId;
  useEffect(() => {
    if (!nowState.peek().pending) void nowState.load(true);
    let disposed = false;
    let stop: (() => void) | undefined;
    void import('../../../wailsjs/runtime/runtime').then(m => {
      if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
        stop = m.EventsOn('target:season', () => { void nowState.load(false); });
    });
    return () => { disposed = true; stop?.(); };
  }, []);
  useEffect(() => {
    if (nowState.peek().now) void nowState.load(false);
  }, [mine]);
  return <NowBody now={state.now} pending={state.pending} error={state.error} selected={selected} />;
}

export function NowBody({ now, pending, error, selected }: {
  now?: NowBoard; pending: boolean; error: string; selected: readonly string[];
}) {
  return <section className="pulse-now" aria-label="League Pulse now">
    {pending && <p role="status">{now?.board.loading ?? 'Reading the week…'}</p>}
    {error && <p role="alert">{error}</p>}
    {now?.waiting && <p role="status">{now.waiting}</p>}
    {now && !now.waiting && <NowView now={now} selected={selected} />}
  </section>;
}
