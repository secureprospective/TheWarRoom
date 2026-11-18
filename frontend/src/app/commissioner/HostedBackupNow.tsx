import { useCallback, useEffect, useState } from 'react';
import { object, requiredText, timestamp } from '../data/parseValues';
import { HQAct } from '../shell/HQAct';
import type { NativeRequest } from '../commands/native';
import { HoldToFire } from './HoldToFire';
import { parsePreview, type Preview } from './preview';
import { PreviewReview } from './PreviewReview';
import type { HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';

type BackupResult = {
  league: string; captured: string; verified: string; where: string; message: string;
};
function parseBackupResult(value: unknown): BackupResult {
  const result = object(value, 'Backup result',
    ['league', 'captured', 'verified', 'where', 'checked', 'message']);
  if (result.checked !== true) throw new Error('The server did not confirm its backup check.');
  const captured = timestamp(result.captured, 'Backup time');
  const verified = timestamp(result.verified, 'Backup check time');
  if (!captured || !verified || Date.parse(verified) < Date.parse(captured))
    throw new Error('The server returned an incomplete backup check.');
  return {
    league: requiredText(result.league, 'Backup league'), captured, verified,
    where: requiredText(result.where, 'Backup location'),
    message: requiredText(result.message, 'Backup result'),
  };
}

export function HostedBackupNow({ league, port = hostedWorkflow, autoPreview = false, onDone }: {
  league: string; port?: HostedPort; autoPreview?: boolean; onDone?(): void;
}) {
  const [preview, setPreview] = useState<Preview>();
  const [result, setResult] = useState<BackupResult>();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const request = useCallback((mode: 'preview' | 'execute'): NativeRequest => ({
    command: 'backup.now', mode, arguments: { league },
  }), [league]);
  const previewBackup = useCallback(async () => {
    setPending(true); setError(''); setPreview(undefined);
    try {
      const response = await port(request('preview'));
      const checked = parsePreview(response.data);
      if (checked.target.league !== league || checked.blockers.length) {
        throw new Error(checked.blockers.join(' ') || 'The preview belongs to another league.');
      }
      setPreview(checked);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally { setPending(false); }
  }, [league, port, request]);
  useEffect(() => { if (autoPreview) void previewBackup(); }, [autoPreview, previewBackup]);
  const confirmBackup = async () => {
    if (!preview || pending) return;
    setPending(true); setError('');
    try {
      const response = await port(request('execute'));
      const saved = parseBackupResult(response.data);
      if (saved.league !== league) throw new Error('The server returned a backup for another league.');
      setResult(saved); setPreview(undefined); onDone?.();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally { setPending(false); }
  };
  return <section aria-label="Back up now">
    <h4>Back up now</h4>
    <p>Save and check a fresh off-site backup of this league before continuing.</p>
    <HQAct disabled={pending} onAction={() => { void previewBackup(); }}>Preview backup now</HQAct>
    {error && <p role="alert">{error}</p>}
    {preview && <PreviewReview preview={preview} confirm={<HoldToFire
      label="Confirm back up now" body={JSON.stringify({ league, command: 'backup.now' })}
      disabled={pending} answer={pending ? 'sending' : ''}
      onConfirm={() => { void confirmBackup(); }} />}>{null}</PreviewReview>}
    {result && <div aria-label="Backup result" role="status">
      <p>{result.message}</p>
      <p>Saved {result.captured}; checked {result.verified} in {result.where}.</p>
    </div>}
  </section>;
}
