import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { useRequestKey, type Change } from './forms';
import { PreviewReview } from './PreviewReview';
import { leagueState, type Ledger, type LedgerEntry } from './proxy';
import { useChangePreview } from './useChangePreview';

const privateNote = 'This change is private, so its correction is private too.';

// UndoForm offers a correction for one ledger entry, in the side tray, only where Go said this person may
// make it. The correction is a new entry; the original stays in the ledger.
export function UndoForm({ reading, entry }: { reading: Ledger; entry: LedgerEntry }) {
  const [reason, setReason] = useState('');
  const form = `undo:${entry.id}`;
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === form;
  useEffect(() => { if (landed) setReason(''); }, [landed]);
  const change: Change | null = entry.undo === '' ? null : {
    league: reading.league, command: 'commish.correction.revert', args: { entry: entry.id },
    reason: reason.trim(), expected: reading.revision, effective: 'now', visibility: entry.undo,
  };
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  if (entry.undo === '' && !landed)
    return <p role="status">This change can no longer be undone from here.</p>;
  return (
    <section className="league-form" aria-label="Undo this change">
      <PreviewReview preview={preview} confirm={
        <ChangeBar form={form} change={change} request={request} reason={reason} onReason={setReason}
          safeguard={safeguard} confirmLabel="Undo this change" blocked={wait}
          note={entry.undo === 'private' ? privateNote : undefined} />
      }>
        <p>{entry.what}</p>
      </PreviewReview>
    </section>
  );
}
