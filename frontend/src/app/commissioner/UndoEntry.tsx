import { TrayLauncher } from './TrayLauncher';
import type { LedgerEntry } from './proxy';

// UndoEntry is the way into a correction for one ledger entry, offered only where Go said this person may
// make it. The correction itself is made in the side tray.
export function UndoEntry({ entry }: { entry: LedgerEntry }) {
  if (entry.undo === '') return null;
  return <TrayLauncher tray={`undo ${entry.id}`} text="Undo this change"
    label={`Undo this change: ${entry.what}`} />;
}
