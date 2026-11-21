import { useEffect, useState, type FormEvent } from 'react';
import { ProblemReportCreate } from '../commissioner/problemReports';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct } from '../shell/HQAct';
import type { HostedPort } from '../commissioner/hostedRoom';
import { hostedWorkflow } from './nativeApi';
import type { ThinScreenRoute } from '../shell/ThinPlayerSearch';
import './problemReport.css';

export function ProblemReport({ league, route, onClose, port = hostedWorkflow }: {
  league: string; route: ThinScreenRoute; onClose: () => void; port?: HostedPort;
}) {
  const [text, setText] = useState('');
  const [release, setRelease] = useState<number>();
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    void port({ command: 'native.directory.read', mode: 'read', arguments: { league } })
      .then(response => {
        if (!active) return;
        const appRelease = response.appRelease;
        if (typeof appRelease !== 'number' || !Number.isSafeInteger(appRelease) || appRelease < 1)
          throw new Error('The app release is unavailable. Refresh and try again.');
        setRelease(appRelease);
        setError('');
      }).catch(cause => {
        if (active) setError(cause instanceof Error ? cause.message : String(cause));
      }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [league, port]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!text.trim()) {
      setError('Write a short description before sending.');
      return;
    }
    if (Array.from(text).length > 2000) {
      setError('Keep the report to 2,000 characters or fewer.');
      return;
    }
    if (!release) {
      setError('The app release is unavailable. Refresh and try again.');
      return;
    }
    setPending(true);
    setError('');
    try {
      await port({ command: ProblemReportCreate, mode: 'execute', arguments: {
        league, text, route: `${route.node}/${route.workspace}`, appRelease: release,
        viewportWidth: window.innerWidth,
      } });
      setSent(true);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setPending(false);
    }
  }

  return <div className="problem-report-backdrop" role="presentation">
    <section className="problem-report-dialog" role="dialog" aria-modal="true"
      aria-labelledby="problem-report-title" data-testid="problem-report">
      <header>
        <h2 id="problem-report-title">Report a problem</h2>
        <HQAct onAction={onClose} label="Close report form">Close</HQAct>
      </header>
      {sent ? <p role="status">Sent — thanks</p> : <form onSubmit={event => { void submit(event); }}>
        <label htmlFor="problem-report-text">What happened?</label>
        <textarea id="problem-report-text" value={text} maxLength={4000} rows={6}
          onChange={event => setText(Array.from(event.currentTarget.value).slice(0, 2000).join(''))}
          aria-describedby="problem-report-help" />
        <p id="problem-report-help">Up to 2,000 characters. Screen route, app release, viewport
          width and user-agent are attached automatically. No screenshots or console logs are sent.</p>
        {loading && <p role="status">Reading app release…</p>}
        {error && <FailureAlert
          message={error}
          league={league}
          route={`${route.node}/${route.workspace}`}
          appRelease={release}
          allowSend={false}
        />}
        <footer>
          <HQAct onAction={onClose} disabled={pending}>Cancel</HQAct>
          <HQAct submit disabled={pending || loading || !release}>
            {pending ? 'Sending…' : 'Send report'}
          </HQAct>
        </footer>
      </form>}
    </section>
  </div>;
}
