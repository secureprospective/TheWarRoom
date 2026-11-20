import { useEffect, useState } from 'react';
import { ProblemReportsRead, parseProblemReports, type ProblemReport } from './problemReports';
import type { HostedPort } from './hostedRoom';
import { hostedWorkflow } from '../control/nativeApi';
import './problemReports.css';

/** The league's problem reports, read again whenever refreshVersion changes; shared by the old room
 *  and the desktop v3 Feedback body. */
export function useProblemReports(league: string, port: HostedPort = hostedWorkflow, refreshVersion = 0) {
  const [reports, setReports] = useState<ProblemReport[]>();
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    setReports(undefined);
    setError('');
    void port({ command: ProblemReportsRead, mode: 'read', arguments: { league } })
      .then(response => {
        const result = parseProblemReports(response.data, league);
        if (active) setReports(result);
      }).catch(cause => {
        if (active) setError(cause instanceof Error ? cause.message : String(cause));
      });
    return () => { active = false; };
  }, [league, port, refreshVersion]);
  return { reports, error };
}

export function HostedControlRoomFeedback({ league, port = hostedWorkflow, refreshVersion = 0 }: {
  league: string; port?: HostedPort; refreshVersion?: number;
}) {
  const { reports, error } = useProblemReports(league, port, refreshVersion);

  return <section className="hosted-feedback" aria-label="Problem reports">
    <h2>Problem reports</h2>
    {error && <p role="alert">{error}</p>}
    {!reports && !error && <p role="status">Reading reports…</p>}
    {reports?.length === 0 && <p>No reports have been sent.</p>}
    {reports && reports.length > 0 && <>
      <p>Showing up to 100 newest reports.</p>
      <ol>
      {reports.map(report => <li key={report.id}>
        <header><strong>{report.principal}</strong>
          <time dateTime={report.createdAt}>{new Date(report.createdAt).toLocaleString()}</time></header>
        <p className="hosted-feedback-text">{report.text}</p>
        <dl>
          <div><dt>Screen</dt><dd>{report.route}</dd></div>
          <div><dt>App release</dt><dd>{report.appRelease}</dd></div>
          <div><dt>Viewport</dt><dd>{report.viewportWidth}px</dd></div>
          <div><dt>User agent</dt><dd>{report.userAgent || 'Unavailable'}</dd></div>
        </dl>
      </li>)}
      </ol>
    </>}
  </section>;
}
