import { DensityScreen } from './DensityScreen';
import { LedgerFilters } from './LedgerFilters';
import { Act } from '../commands/Act';
import { leagueState, type Ledger } from './proxy';
import { UndoEntry } from './UndoEntry';
import './commissioner.css';

export function LedgerEntries({ reading, filtered = false }: { reading: Ledger; filtered?: boolean }) {
  return (
    <>
      <p>League version: {reading.revision}</p>
      {reading.entries.length === 0 && <p>{filtered ? 'No changes match these filters.'
        : 'No changes are visible to you in this league yet.'}</p>}
      <ol className="league-ledger-entries">
        {reading.entries.map(entry => (
          <li key={entry.id}>
            <strong>{entry.what}</strong>
            {entry.replacedAt && <p>Replaced by a later change on{' '}
              <time dateTime={entry.replacedAt}>{new Date(entry.replacedAt).toLocaleString()}</time></p>}
            <p>By {entry.who} ·{' '}
              <time dateTime={entry.when}>{new Date(entry.when).toLocaleString()}</time></p>
            <p>Reason: {entry.why || 'No reason recorded'}</p>
            <p>Safeguard: {entry.gravity}</p>
            <UndoEntry reading={reading} entry={entry} />
          </li>
        ))}
      </ol>
    </>
  );
}

export default function LeagueLedger() {
  const { choices, ledger, error, pending } = leagueState.use();
  return (
    <DensityScreen screen="ledger" label="League ledger" pending={pending}>
      <h4>League ledger</h4>
      <p>Changes recorded in this app. Imported league settings are separate.</p>
      <Act verb="commissioner.read" args={{ league: '' }} variant="button" disabled={pending}>
        Refresh ledger
      </Act>
      {pending && <p role="status">Reading league ledger…</p>}
      {error && <p role="alert">{error} Refresh the ledger to try again.</p>}
      {!pending && !error && choices.length === 0 &&
        <p>No app league has been created. Imported leagues do not make you their commissioner.</p>}
      {ledger && <LedgerFilters key={ledger.league} reading={ledger}>
        {(result, filtered) => <LedgerEntries reading={result} filtered={filtered} />}
      </LedgerFilters>}
    </DensityScreen>
  );
}
