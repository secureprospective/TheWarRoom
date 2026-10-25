import { useState } from 'react';
import { nativeAction, type NativeView } from './nativeApi';
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
      setError(cause instanceof Error ? cause.message : 'The update could not be opened. Try again.');
      setPending(false);
    }
  }
  return <main className="twr-app native-account" aria-busy={pending}>
    <h1>{app ? 'The War Room needs an update' : 'Your league server needs an update'}</h1>
    <p>{app ? 'Your league server no longer accepts this copy of The War Room.'
      : 'This copy of The War Room is newer than your league server. Ask your commissioner to update the server first.'}</p>
    <p>Your league stays on the server. Updating this app does not change or remove league data.</p>
    {app && <p>We will check the update, then open the new copy. This window closes only after the new copy starts.</p>}
    <div className="native-actions"><button type="button" disabled={pending} onClick={() => void run()}>
      {pending ? 'Checking and opening the update…' : app ? 'Update and reopen' : 'Try again'}
    </button></div>
    {pending && <p role="status">{app ? 'Keep this window open while the update is checked.' : 'Returning to sign in…'}</p>}
    {error && <p role="alert">{error}</p>}
  </main>;
}
