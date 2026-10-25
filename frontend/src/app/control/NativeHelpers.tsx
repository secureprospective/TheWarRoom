import { useEffect, useState } from 'react';
import { copyHelperConnection } from './api';
import { Act } from '../commands/Act';
import { surfaceActions } from '../commissioner/surfaceActions';
import { helperAction, readConfirmRequest, type HelperAction, type NativeSeat } from './nativeApi';

// AI helpers act for a commissioner through the twr command. Go makes and revokes their
// connection, and signs the person's say-so for one exact bigger change.
export default function NativeHelpers({ server, league, seats, franchise }: {
  server: string; league: string; seats: NativeSeat[]; franchise: string;
}) {
  const [level, setLevel] = useState<'suggest' | 'record'>('suggest');
  const [seat, setSeat] = useState(franchise || seats[0]?.franchise || '');
  const [connection, setConnection] = useState('');
  const [helperId, setHelperId] = useState('');
  const [request, setRequest] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [preview, setPreview] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const change = readConfirmRequest(request);
  async function run(action: HelperAction) {
    if (pending) return;
    if (action === 'helper-confirm' && !preview) return;
    setPreview('');
    setPending(true); setError(''); setMessage(''); setConnection(''); setConfirmation('');
    try {
      const fields = action === 'helper-issue' ? { helper: level, franchise: seat }
        : action === 'helper-revoke' ? { code: helperId.trim() } : { change: request };
      const result = await helperAction(action, fields);
      setMessage(result.message);
      if (action === 'helper-check') setPreview(result.helperPreview || '');
      if (action === 'helper-issue') { setConnection(result.connection); setHelperId(result.helperId); }
      if (action === 'helper-confirm') { setConfirmation(result.confirmation); setRequest(''); }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not reach your league. Try again.');
    } finally { setPending(false); }
  }
  async function copyConnection() {
    if (pending || !connection) return;
    setError(''); setMessage('');
    try {
      if (!await copyHelperConnection(connection)) throw new Error('Clipboard unavailable');
      setMessage('Connection line copied.');
    } catch {
      setError('Could not copy. Select the connection line and copy it yourself.');
    }
  }
  useEffect(() => surfaceActions.register('native-helpers', args => {
    if (args.action === 'check' && args.value === 'copy-connection') { void copyConnection(); return; }
    const actions: HelperAction[] = ['helper-issue', 'helper-revoke', 'helper-check', 'helper-confirm'];
    const action = actions.find(value => value === args.value);
    if (args.action === 'check' && action) void run(action);
  }));
  const args = (value: HelperAction | 'copy-connection') =>
    ({ screen: 'native-helpers', action: 'check' as const, value });
  return <section aria-labelledby="helpers-title" aria-busy={pending}>
    <h2 id="helpers-title">AI helpers</h2>
    <p>An AI helper can read this league and suggest changes for you through the twr command.
      Bigger changes always wait for you to confirm them here. Verify a fresh authenticator code first.</p>
    {error && <p role="alert">{error}</p>}
    {message && <p role="status">{message}</p>}
    <h3>Connect a helper</h3>
    <label>What it may do<select value={level} disabled={pending}
      onChange={e => setLevel(e.target.value === 'record' ? 'record' : 'suggest')}>
      <option value="suggest">Look and suggest changes</option>
      <option value="record">Also record changes (you still confirm bigger ones)</option>
    </select></label>
    <label>Acts for<select value={seat} disabled={pending} onChange={e => setSeat(e.target.value)}>
      {seats.map(s => <option key={s.franchise} value={s.franchise}>{s.name}</option>)}
    </select></label>
    <div className="native-actions">
      <Act variant="button" verb="commissioner.surface" args={args('helper-issue')}
        disabled={pending || !seat} label="Connect a helper">Connect a helper</Act>
    </div>
    {connection && <div className="native-seed">
      <p>On the helper's computer, run this and paste the line below when asked:</p>
      <p><code>twr connect --address {server} --league {league}</code></p>
      <p>Connection line (shown once): <code className="native-seed">{connection}</code></p>
      <div className="native-actions">
        <Act variant="button" verb="commissioner.surface" args={args('copy-connection')}
          disabled={pending} label="Copy connection line">Copy</Act>
      </div>
    </div>}
    <h3>Confirm a helper's change</h3>
    <label>Confirm request from your helper<textarea value={request} disabled={pending} rows={3}
      onChange={e => {
        setRequest(e.target.value); setPreview(''); setConfirmation(''); setError('');
      }} /></label>
    {preview && <p className="native-preview">{preview}</p>}
    {request && !change && <p role="alert">That is not a whole confirm request. It starts with twrc1.</p>}
    <div className="native-actions">
      <Act variant="button" verb="commissioner.surface" args={args('helper-check')}
        disabled={pending || !change} label="Check this change">Check this change</Act>
      <Act variant="button" verb="commissioner.surface" args={args('helper-confirm')}
        disabled={pending || !preview} label="Confirm this change">Confirm this change</Act>
    </div>
    {confirmation && <p className="native-seed">Code for your helper: <code>{confirmation}</code></p>}
    <h3>Disconnect a helper</h3>
    <label>Helper ID<input autoComplete="off" value={helperId} disabled={pending} maxLength={43}
      onChange={e => setHelperId(e.target.value)} /></label>
    <div className="native-actions">
      <Act variant="button" verb="commissioner.surface" args={args('helper-revoke')}
        disabled={pending || helperId.trim().length !== 43} label="Disconnect helper">Disconnect helper</Act>
    </div>
  </section>;
}
