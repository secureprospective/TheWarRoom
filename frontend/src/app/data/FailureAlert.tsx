import { useEffect, useMemo, useRef, useState } from 'react';
import type { HostedPort } from '../commissioner/hostedRoom';
import { hostedWorkflow } from '../control/nativeApi';
import { HQAct } from '../shell/HQAct';
import {
  formatFailureDetails,
  getFailureMeta,
  lookupErrorCode,
  newClientErrorCode,
  recordAppRelease,
  validErrorCode,
} from './errorCodes';
import { plainFailure } from './plainFailure';
import './failureAlert.css';

const PROBLEM_REPORT_CREATE = 'feedback.report.create' as const;

export type FailureAlertProps = {
  message: string;
  prefix?: string;
  code?: string;
  screen?: string;
  route?: string;
  league?: string;
  leagueName?: string;
  appRelease?: number;
  className?: string;
  allowSend?: boolean;
  onReload?: () => void;
  port?: HostedPort;
  now?: () => Date;
};

async function writeClipboard(text: string): Promise<boolean> {
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through to legacy copy path for webviews without clipboard permissions.
    }
  }
  if (typeof document === 'undefined') return false;
  const field = document.createElement('textarea');
  field.value = text;
  field.setAttribute('readonly', 'true');
  field.style.position = 'fixed';
  field.style.opacity = '0';
  document.body.appendChild(field);
  field.select();
  let ok = false;
  try {
    ok = typeof document.execCommand === 'function' && document.execCommand('copy');
  } catch {
    ok = false;
  }
  field.remove();
  return ok;
}

function viewportWidth(): number {
  const width = typeof window !== 'undefined' ? Math.round(window.innerWidth) : 1024;
  return Math.min(20000, Math.max(1, width || 1024));
}

export function FailureAlert({
  message,
  prefix = '',
  code: explicitCode,
  screen,
  route,
  league,
  leagueName,
  appRelease,
  className,
  allowSend = true,
  onReload,
  port = hostedWorkflow,
  now,
}: FailureAlertProps) {
  const cleanMessage = plainFailure(message);
  const resolvedCode = useMemo(() => {
    if (explicitCode && validErrorCode(explicitCode)) return explicitCode;
    return lookupErrorCode(cleanMessage) ?? newClientErrorCode();
  }, [explicitCode, cleanMessage]);
  const occurredAt = useMemo(
    () => (now ? now() : new Date()),
    [cleanMessage, resolvedCode, now],
  );
  const [copied, setCopied] = useState(false);
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [notice, setNotice] = useState('');
  const copyTimer = useRef<number | undefined>();

  useEffect(() => {
    setSent(false);
    setNotice('');
  }, [cleanMessage, resolvedCode]);

  useEffect(() => () => {
    if (copyTimer.current !== undefined) window.clearTimeout(copyTimer.current);
  }, []);

  function buildDetails(releaseOverride?: number): {
    text: string;
    leagueId: string;
    reportRoute: string;
    release?: number;
  } {
    const meta = getFailureMeta({
      screen,
      route,
      league,
      leagueName,
      appRelease: releaseOverride ?? appRelease,
    });
    const screenText = meta.screenLabel.includes(meta.route)
      ? meta.screenLabel : `${meta.screenLabel} (${meta.route})`;
    const leagueText = meta.leagueName && meta.league
      ? `${meta.leagueName} (${meta.league})` : meta.league || meta.leagueName;
    const text = formatFailureDetails({
      code: resolvedCode,
      message: cleanMessage,
      at: occurredAt,
      screen: screenText,
      appVersion: meta.appVersion,
      league: leagueText,
    });
    return { text, leagueId: meta.league, reportRoute: meta.route, release: meta.appRelease };
  }

  async function handleCopy(): Promise<void> {
    const { text } = buildDetails();
    const ok = await writeClipboard(text);
    if (!ok) {
      setNotice('Could not copy automatically. Select the code above.');
      return;
    }
    setNotice('');
    setCopied(true);
    if (copyTimer.current !== undefined) window.clearTimeout(copyTimer.current);
    copyTimer.current = window.setTimeout(() => setCopied(false), 2000);
  }

  async function handleSend(): Promise<void> {
    if (sending || sent) return;
    const initial = buildDetails();
    if (!initial.leagueId.startsWith('native.') || initial.leagueId === 'native.') {
      setNotice('Sign in to a league first, or tap Copy to share it with your commissioner.');
      return;
    }
    setSending(true);
    setNotice('');
    try {
      let release = initial.release;
      if (!release) {
        const probe = await port({
          command: 'native.directory.read',
          mode: 'read',
          arguments: { league: initial.leagueId },
        });
        if (typeof probe.appRelease === 'number' && Number.isSafeInteger(probe.appRelease)
          && probe.appRelease >= 1) {
          release = probe.appRelease;
          recordAppRelease(release);
        }
      }
      if (!release) {
        setNotice('Could not send right now. Tap Copy to share it with your commissioner.');
        return;
      }
      const details = buildDetails(release);
      await port({
        command: PROBLEM_REPORT_CREATE,
        mode: 'execute',
        arguments: {
          league: details.leagueId,
          text: details.text,
          route: details.reportRoute,
          appRelease: release,
          viewportWidth: viewportWidth(),
        },
      });
      setSent(true);
    } catch (cause) {
      const reason = cause instanceof Error ? plainFailure(cause.message) : '';
      const hint = reason.includes('10 reports')
        ? reason : 'Could not send right now. Tap Copy to share it with your commissioner.';
      setNotice(hint);
    } finally {
      setSending(false);
    }
  }

  return <div className="twr-failure" data-testid="failure-alert">
    <p role="alert" className={className}>{prefix}{cleanMessage}</p>
    <div className="twr-failure__bar">
      <code className="twr-failure__code" data-testid="failure-code">{resolvedCode}</code>
      <HQAct className="twr-failure__btn" onAction={handleCopy}>
        {copied ? 'Copied' : 'Copy'}
      </HQAct>
      {allowSend && <HQAct
        className="twr-failure__btn twr-failure__send"
        disabled={sending || sent}
        onAction={handleSend}
      >
        {sent ? 'Sent — thanks' : sending ? 'Sending…' : 'Send to commissioner'}
      </HQAct>}
      {onReload && <HQAct className="twr-failure__btn" onAction={onReload}>
        Reload
      </HQAct>}
    </div>
    {notice && <p className="twr-failure__note" role="status">{notice}</p>}
  </div>;
}
