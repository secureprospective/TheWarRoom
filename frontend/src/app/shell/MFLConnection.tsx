import { useLayoutEffect, useSyncExternalStore } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import type { MFLKeyState } from '../commands/mflKey';
import './mflKey.css';

function statusLine(status: MFLKeyState): string {
  if (status.pending) return status.activity ?? 'Checking the keyring…';
  if (status.error) return status.error;
  switch (status.state) {
    case 'absent':
      return 'Not connected · paste your API key from MFL: Help › Developer\'s API';
    case 'connected': {
      const verified = status.verifiedAt
        ? ` · verified ${new Date(status.verifiedAt).toLocaleString()}` : '';
      return `Connected · league ${status.league} · ${status.season}${verified}`;
    }
    case 'unavailable': return `Keyring unavailable · ${status.detail ?? ''}`;
    case 'rejected': return `Key not accepted · ${status.detail ?? ''}`;
    case 'unreachable': return `Could not reach MFL · ${status.detail ?? ''}`;
  }
}

export default function MFLConnection() {
  const status = commands.useMFLKey();
  const field = commands.mflField;
  const empty = useSyncExternalStore(field.subscribe, field.empty, field.empty);
  const fixture = commands.providerKind === 'fixture';
  useLayoutEffect(() => { void commands.loadMFLKey(); }, []);
  return (
    <section className="mfl-connection" aria-labelledby="mfl-connection-heading">
      <h4 id="mfl-connection-heading">MFL connection</h4>
      <p className="mfl-key-status" role="status" aria-live="polite">{statusLine(status)}</p>
      {fixture && <p className="not-wired">Connecting MFL needs the desktop app</p>}
      <label htmlFor="mfl-key">MFL API key</label>
      <input
        id="mfl-key"
        ref={field.mount}
        type="password"
        autoComplete="off"
        spellCheck={false}
        disabled={fixture}
        aria-describedby="mfl-key-help"
        onKeyDown={(event) => {
          if (event.key !== 'Enter') return;
          event.preventDefault();
          commands.dispatch('mflkey.connect', {});
        }}
      />
      <p id="mfl-key-help">Stored in your OS keyring; never shown again.</p>
      <div className="mfl-key-actions">
        <Act verb="mflkey.connect" args={{}} disabled={fixture || status.pending || empty} />
        {(fixture || status.state === 'connected') && (
          <Act verb="mflkey.forget" args={{}} disabled={Boolean(commands.mflReason(true))} />
        )}
      </div>
    </section>
  );
}
