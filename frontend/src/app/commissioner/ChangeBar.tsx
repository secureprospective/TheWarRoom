import { useEffect } from 'react';
import { Act } from '../commands/Act';
import type { Safeguard } from './rules';
import { bodyOf, noReasonWords, safeguardWords, useRequestKey, type Change } from './forms';
import { HoldToFire } from './HoldToFire';
import { leagueState } from './proxy';

type Props = {
  form: string; change: Change | null; safeguard: Safeguard | null; reason: string;
  onReason(text: string): void;
  request?: string; confirmLabel: string; blocked?: string; note?: string;
};

// ChangeBar is the end of every commissioner form, and it is drawn inside the side tray only: the reason,
// the safeguard Go applies, the confirm and whatever Go answered. Nothing here decides who may act or how
// firmly to confirm; the safeguard is Go's, and until Go has named one there is nothing to confirm with.
export function ChangeBar({ form, change, safeguard, reason, onReason, confirmLabel,
  blocked, note, request }: Props) {
  const outcome = leagueState.use().outcome;
  // A receipt or refusal belongs to the form that was confirmed, not to the next one opened with its name.
  useEffect(() => {
    if (leagueState.peek().outcome.status !== 'sending') leagueState.forget();
  }, []);
  const typed = reason.trim();
  const sent = change && { ...change, reason: typed };
  const key = useRequestKey(sent);
  const body = sent ? bodyOf(sent, request ?? key) : '';
  const sending = outcome.status === 'sending' && outcome.form === form;
  const need = safeguard === 'reason' && typed === '' ? noReasonWords : '';
  const why = blocked || need || (sending ? 'Sending the change…' : '');
  const ready = body !== '' && why === '' && safeguard !== null;
  return (
    <div className="change-bar">
      <label className="change-reason">
        Reason{safeguard && (safeguard === 'reason' ? ' (required)' : ' (optional)')}
        <textarea value={reason} rows={2} onChange={e => onReason(e.target.value)} />
      </label>
      {safeguard && <p className="change-safeguard">{safeguardWords[safeguard]}</p>}
      {note && <p className="change-note">{note}</p>}
      {safeguard === 'hold'
        ? <HoldToFire label={confirmLabel} disabled={!ready} body={body}
          answer={outcome.status !== 'idle' && outcome.form === form ? outcome.status : ''} />
        : <Act verb="commissioner.execute" args={{ body }} variant="button" disabled={!ready}>
          {confirmLabel}
        </Act>}
      {why && !ready && <p className="change-why" role="status">{why}</p>}
      {outcome.status === 'done' && outcome.form === form &&
        <p role="status" className="change-done">{outcome.receipt.message}</p>}
      {outcome.status === 'refused' && outcome.form === form &&
        <p role="alert" className="change-refused">{outcome.message}</p>}
    </div>
  );
}
