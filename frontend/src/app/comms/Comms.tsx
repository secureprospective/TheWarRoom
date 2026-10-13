import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportView } from '../reports/ReportView';
import { commsState } from './state';
import type { Comms } from './contract';
import './comms.css';

export function CommsView({ comms, panel, limit }: { comms: Comms; panel: boolean; limit: number }) {
  const dispatch = commands.dispatch;
  return <>
    <section data-endpoint="C-05" className="league-feed" aria-label={comms.feed.title}>
      <h4>{comms.feed.title}</h4>
      <p>{comms.feed.source} · {comms.feed.asOf}</p>
      <div className="market-controls">
        <label>Franchise{' '}<select value={comms.team}
          onChange={e => dispatch('comms.read', { action: 'team', value: e.target.value })}>
          {comms.teams.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select></label>
        <label>Kind{' '}<select value={comms.kind}
          onChange={e => dispatch('comms.read', { action: 'kind', value: e.target.value })}>
          {comms.kinds.map(k => <option key={k.value} value={k.value}>{k.label}</option>)}
        </select></label>
        <Act variant="button" verb="comms.read" args={{ action: 'refresh', value: '' }}>Refresh from MFL</Act>
      </div>
      {comms.feed.note && <p role={comms.feed.status === 'failed' ? 'alert' : 'status'}>{comms.feed.note}</p>}
      <ol className="league-events">{comms.events.slice(0, panel ? limit : undefined).map((event, i) =>
        <li key={i}><time>{event.date}</time><p>{event.sentence}</p></li>)}</ol>
      {panel && comms.events.length > limit &&
        <Act variant="button" verb="comms.read" args={{ action: 'more', value: '' }}>Show more</Act>}
      <details className="market-how"><summary>How this is counted</summary><p>{comms.feed.how}</p></details>
    </section>
    {!panel && <ReportView report={comms.transactions} howLabel="How this is counted" heading="h4" />}
    {panel && <section data-endpoint="M-120" className="league-board" aria-label="League board">
      <h4>League board</h4>
      <p>{comms.leagueTalk}</p>
      <Act variant="button"
        verb="comms.read" args={{ action: 'open', value: 'league-board' }}>Open the league board</Act>
      <p role="status">{comms.boardNote}</p>
      {comms.boardPosts.map((report, i) => <section key={report.id} data-endpoint={report.id}>
        {report.rows.length > 0 || report.status === 'failed' || report.status === 'stale' ?
          <ReportView report={report} howLabel="How this is counted" /> : <h5>{report.title}</h5>}
        <Act variant="button"
          verb="comms.read" args={{ action: 'open', value: i === 0 ? 'message-board' : 'polls' }}
          label={`Open on MFL · ${report.title}`}>Open on MFL</Act>
      </section>)}
    </section>}
  </>;
}

export default function CommsMount({ panel = false }: { panel?: boolean }) {
  const state = commsState.use();
  useEffect(() => {
    if (!commsState.peek().pending) void commsState.load();
    let disposed = false;
    let stop: (() => void) | undefined;
    void import('../../../wailsjs/runtime/runtime').then(m => {
      if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
        stop = m.EventsOn('target:season', () => { void commsState.load(); });
    });
    return () => { disposed = true; stop?.(); };
  }, []);
  return <section className="comms-reading" aria-label={panel ? 'League communications' : 'Digests'}>
    {!panel && <h4>Digests</h4>}
    {state.pending && <p role="status">Reading league communications…</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.comms && <CommsView comms={state.comms} panel={panel} limit={state.limit} />}
  </section>;
}
