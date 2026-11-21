// Desktop v3 Control Room, Feedback tab: problem reports sent from the league's app screens, read via
// commish.feedback.read and refreshed from c.feedbackRefreshVersion.
import type { ReactNode } from 'react';
import { hostedWorkflow } from '../control/nativeApi';
import { memberDisplayName } from '../commissioner/people';
import type { ProblemReport } from '../commissioner/problemReports';
import { useProblemReports } from '../commissioner/HostedControlRoomFeedback';
import type { ControlBodyProps } from './ControlRoomV3';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; tag?: ReactNode }) {
  return <section className="pc panel sz-L">
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** One-line device and release summary for a problem report row. */
export function reportMeta(
  report: Pick<ProblemReport, 'appRelease' | 'viewportWidth' | 'userAgent'>,
): string {
  return `App release ${report.appRelease} · ${report.viewportWidth}px`
    + ` · User agent: ${report.userAgent || 'Unavailable'}`;
}

export function CtrlFeedbackV3({ c, league, port = hostedWorkflow, badge, people }: ControlBodyProps) {
  const { reports, error } = useProblemReports(league, port, c.feedbackRefreshVersion);

  const foot = reports
    ? (reports.length ? 'Showing up to 100 newest reports.' : 'No reports have been sent.')
    : 'Reading reports…';

  return <div className="wbody scr scr-ctrl"><div className="lay"
    style={{ gridTemplateColumns: 'minmax(0,1fr)' }}>
    <Panel title="Feedback" badge={badge}
      tag={reports ? <span className="tag o">{reports.length}</span> : null} foot={foot}>
      {error && <St><span role="alert">{error}</span></St>}
      {!reports && !error && <St><span role="status">Reading reports…</span></St>}
      {reports?.length === 0 && <St>No reports have been sent.</St>}
      {reports && reports.length > 0 && reports.map(report => <div key={report.id} className="row"
        style={{ height: 'auto', minHeight: 30, padding: '8px 10px', whiteSpace: 'normal',
          alignItems: 'start', gridTemplateColumns: 'minmax(0,1fr)' }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '8px' }}>
          <span className="tag">{report.route}</span>
          <b title={report.principal}>{memberDisplayName(report.principal, people)}</b>
          <span className="sp" />
          <time className="m" dateTime={report.createdAt}>
            {new Date(report.createdAt).toLocaleString()}</time>
        </div>
        <p style={{ margin: '4px 0', whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
          {report.text}</p>
        <div className="m" style={{ overflowWrap: 'anywhere' }}>{reportMeta(report)}</div>
      </div>)}
    </Panel>
  </div></div>;
}
