import { useEffect, useState } from 'react';
import { nativeAction, type NativeView } from './nativeApi';
import { Act } from '../commands/Act';
import { surfaceActions } from '../commissioner/surfaceActions';
import '../look/look.css';
import './nativeAccount.css';

export default function UpdateRequired({ initial, onRetry }: {
  initial: NativeView; onRetry: (view: NativeView) => void;
}) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const app = initial.update === 'app';
  async function run() {
    if (pending) return;
    setPending(true); setError('');
    try {
      const result = await nativeAction(app ? 'update' : 'retry');
      if (!app) onRetry(result);
    } catch (cause) {
      const reason = cause instanceof Error ? cause.message : 'The update could not be opened. Try again.';
      setError(reason);
      setPending(false);
      throw new Error(reason);
    }
  }
  useEffect(() => surfaceActions.register('desktop-update', args => {
    if (args.action === 'check') return run();
  }));
  const label = pending ? 'Checking and opening the update…' : app ? 'Update and reopen' : 'Try again';
  return <main className="twr-app native-account" aria-busy={pending}>
    <h1>{app ? 'The War Room needs an update' : 'Your league server needs an update'}</h1>
    <p>{app ? 'Your league server no longer accepts this copy of The War Room.'
      : <>This copy of The War Room is newer than your league server.
        Ask your commissioner to update the server first.</>}</p>
    <p>Your league stays on the server. Updating this app does not change or remove league data.</p>
    {app && <p>We will check the update, then open the new copy.
      This window closes only after the new copy starts.</p>}
    <div className="native-actions"><Act verb="commissioner.surface" variant="button" disabled={pending}
      args={{ screen: 'desktop-update', action: 'check', value: '' }} label={label}>{label}</Act></div>
    {pending && <p role="status">{app ? 'Keep this window open while the update is checked.'
      : 'Returning to sign in…'}</p>}
    {error && <p role="alert">{error}</p>}
  </main>;
}
