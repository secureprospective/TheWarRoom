import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ledgerState, type Ledger } from './proxy';

export function LedgerEntries({ reading }: { reading: Ledger }) {
  return (
    <>
      <p>League version: {reading.revision}</p>
      {reading.entries.length === 0 && <p>No changes are visible to you in this league yet.</p>}
      <ol className="league-ledger-entries">
        {reading.entries.map(entry => (
          <li key={entry.id}>
            <strong>{entry.what}</strong>
            <p>By {entry.who} ·{' '}
              <time dateTime={entry.when}>{new Date(entry.when).toLocaleString()}</time></p>
            <p>Reason: {entry.why || 'No reason recorded'}</p>
            <p>Safeguard: {entry.gravity}</p>
          </li>
        ))}
      </ol>
    </>
  );
}

export default function LeagueLedger() {
  const { choices, selected, reading, error, pending } = ledgerState.use();
  useEffect(() => { void ledgerState.choose(); }, []);
  return (
    <section aria-label="League ledger" aria-busy={pending}>
      <h4>League ledger</h4>
      <p>Changes recorded in this app. Imported league settings are separate.</p>
      {choices.length > 0 && <label>App league{' '}
        <select value={selected || choices[0]?.id}
          onChange={e => commands.dispatch('commissioner.read', { league: e.target.value })}>
          {choices.map(choice => <option key={choice.id} value={choice.id}>{choice.name}</option>)}
        </select>
      </label>}
      <Act verb="commissioner.read" args={{ league: '' }} variant="button" disabled={pending}>
        Refresh ledger
      </Act>
      {pending && <p role="status">Reading league ledger…</p>}
      {error && <p role="alert">{error} Refresh the ledger to try again.</p>}
      {!pending && !error && choices.length === 0 &&
        <p>No app league has been created. Imported leagues do not make you their commissioner.</p>}
      {!pending && reading && <LedgerEntries reading={reading} />}
    </section>
  );
}
