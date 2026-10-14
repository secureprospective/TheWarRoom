import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import type { Change } from './forms';
import { leagueState, type Ledger, type LedgerEntry } from './proxy';

const privateNote = 'This change is private, so its correction is private too.';

// UndoEntry offers a correction for one ledger entry, only where Go said this person may make it.
// The correction is a new entry; the original stays in the ledger.
export function UndoEntry({ reading, entry }: { reading: Ledger; entry: LedgerEntry }) {
  const [reason, setReason] = useState('');
  const form = `undo:${entry.id}`;
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === form;
  useEffect(() => { if (landed) setReason(''); }, [landed]);
  if (entry.undo === '') return null;
  const change: Change = {
    league: reading.league, command: 'commish.correction.revert', args: { entry: entry.id }, reason,
    expected: reading.revision, effective: 'now', visibility: entry.undo,
  };
  return (
    <details className="ledger-undo">
      <summary>Undo this change</summary>
      <ChangeBar form={form} change={change} safeguard="reason" reason={reason} onReason={setReason}
        confirmLabel="Undo this change"
        note={entry.undo === 'private' ? privateNote : undefined} />
    </details>
  );
}
