import { useEffect, useState } from 'react';
import { nativeBindings } from './actReadings';
import { newKey } from './forms';
import { startingJob } from './jobs';
import { HoldToFire } from './HoldToFire';
import { PreviewReview } from './PreviewReview';
import type { Preview } from './preview';
import { leagueState, readPreview, refusalText } from './proxy';
import SideTray from './SideTray';

// Reading the starting assets preview takes seconds on a full league, so the reason is not re-read on every
// keystroke: Go is asked again once typing pauses, and the last preview stays on screen meanwhile. Confirming
// still waits for the preview of the exact request that will be sent.
export const seedPreviewPause = 400;

export function SeedForm({ league }: { league: string }) {
  const start = startingJob(leagueState.use().jobs);
  const label = start?.name ?? 'Record starting assets';
  const [reason, setReason] = useState('');
  const [target] = useState(() => ({ league, expected: leagueState.peek().ledger?.revision ?? 0 }));
  const [request, setRequest] = useState(newKey);
  const [review, setReview] = useState<{ body: string; preview: Preview }>();
  const [error, setError] = useState('');
  const [receipt, setReceipt] = useState('');
  const [sending, setSending] = useState(false);
  const body = JSON.stringify({ ...target, request, reason: reason.trim() });
  const first = review === undefined;
  useEffect(() => {
    let active = true;
    setError('');
    const read = () => void readPreview(body).then(p => {
      const stale = p.target.league !== target.league || p.target.version !== target.expected;
      if (p.blockers.length === 0 && stale)
        throw new Error('The preview does not match this league version. Reopen the change.');
      if (active) setReview({ body, preview: p });
    }).catch(cause => { if (active) setError(refusalText(cause)); });
    const timer = first ? undefined : setTimeout(read, seedPreviewPause);
    if (first) read();
    return () => { active = false; clearTimeout(timer); };
  }, [body]);
  const preview = review?.body === body ? review.preview : undefined;
  const record = async () => {
    if (!ready) return;
    setSending(true);
    try {
      const result = await nativeBindings().TargetNativeSeed(JSON.parse(body));
      setReceipt(result.message);
      await leagueState.choose(league);
    } catch (cause) { setError(refusalText(cause)); }
    finally { setSending(false); }
  };
  const ready = !!preview && preview.blockers.length === 0 && reason.trim() !== '' && !sending && !receipt;
  return <section className="league-form" aria-label={label}>
    <p>{start?.words ?? "Record the held franchises, rostered players and draft picks as this league’s starting assets."}</p>
    <p>No MFL changes are made.</p>
    <PreviewReview preview={review?.preview} confirm={<>
      <label>Reason (required)<textarea value={reason} rows={2} onChange={e => {
        setReason(e.target.value); setRequest(newKey());
      }} /></label>
      <HoldToFire label={label} body={body} disabled={!ready}
        onConfirm={() => void record()} answer={sending ? 'sending' : receipt} />
      {review && !preview && !error && !receipt && <p role="status">Checking the change again…</p>}
      {error && <p role="alert">{error}</p>}
      {receipt && <p role="status">{receipt}</p>}
    </>}>
      {!review && !error && <p role="status">Reading the starting assets preview…</p>}
    </PreviewReview>
  </section>;
}
export default function SeedTray({ league, opener, background, onClose }: {
  league: string; opener: HTMLElement; background: HTMLElement | null; onClose(): void;
}) {
  const start = startingJob(leagueState.use().jobs);
  if (!start) return null;
  return <SideTray title={start.name} screen="landing" opener={opener} background={background}
    onClose={onClose} closeLabel="Close starting assets"><SeedForm league={league} /></SideTray>;
}
