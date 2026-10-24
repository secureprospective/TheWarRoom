import { useEffect, useState, type FormEvent } from 'react';
import { Act } from '../commands/Act';
import { object, text } from '../data/parseValues';
import { ledgerBindings as bindings } from './api';
import { leagueState } from './proxy';
import { surfaceActions } from './surfaceActions';

type Seat = { hasAccount: boolean; username: string };
export function parseSeat(v: unknown): Seat {
  const r = object(v, 'Sign-in', ['hasAccount', 'username']);
  if (typeof r.hasAccount !== 'boolean') throw new Error('Sign-in.hasAccount: expected true or false');
  return { hasAccount: r.hasAccount,
    username: r.username === undefined ? '' : text(r.username, 'Username') };
}
const KEEP_TITLE = 'Keep your league if you change computers';
const RESTORED_HELP =
  'Sign in with the username and password made on the original computer to get your roles back.';
const NEW_HELP = 'Choose a username and password. If you restore a backup on another computer, '
  + 'sign in there with them to keep your roles.';
const args = { screen: 'local-sign-in', action: 'check', value: 'submit' } as const;

// LocalSignIn keeps the league with its owner when the data moves to another computer or user.
// The password never leaves this form except to Go, and the field clears after every attempt.
// onSeat hears each sign-in Go accepts.
export function LocalSignIn({ onSeat }: { onSeat?: (seat: Seat) => void } = {}) {
  const [seat, setSeat] = useState<Seat>();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    bindings().TargetCommissionerSeat().then(v => setSeat(parseSeat(v)))
      .catch(cause => setError(String(cause)));
  }, []);
  async function signIn() {
    if (pending) return;
    setPending(true); setError('');
    try {
      const signed = parseSeat(await bindings().TargetCommissionerSeatSignIn(username, password));
      setSeat(signed);
      onSeat?.(signed);
      await leagueState.choose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally { setPassword(''); setPending(false); }
  }
  useEffect(() => surfaceActions.register('local-sign-in', () => { void signIn(); }));
  const submit = (event: FormEvent) => { event.preventDefault(); void signIn(); };
  if (!seat) return error ? <p role="alert">{error}</p> : null;
  if (seat.username) return <p>This computer is signed in as {seat.username}. A backup restored on another
    computer keeps your league roles once you sign in there with the same username and password.</p>;
  const label = pending ? 'Please wait…' : seat.hasAccount ? 'Sign in' : 'Create sign-in';
  return (
    <form className="league-form" aria-label="League sign-in" onSubmit={submit}>
      <h5>{seat.hasAccount ? 'Restored from another computer?' : KEEP_TITLE}</h5>
      <p>{seat.hasAccount ? RESTORED_HELP : NEW_HELP}</p>
      {error && <p role="alert">{error}</p>}
      <label>Username<input autoComplete="username" required minLength={3} maxLength={64} value={username}
        disabled={pending} onChange={e => setUsername(e.target.value)} /></label>
      <label>Password<input type="password"
        autoComplete={seat.hasAccount ? 'current-password' : 'new-password'}
        required maxLength={1024} value={password} disabled={pending}
        onChange={e => setPassword(e.target.value)} /></label>
      <Act variant="button" verb="commissioner.surface" args={args} submit disabled={pending}
        label={label}>{label}</Act>
    </form>
  );
}
