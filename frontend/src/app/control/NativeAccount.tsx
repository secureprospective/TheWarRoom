import { useEffect, useState, type FormEvent } from 'react';
import { Act } from '../commands/Act';
import { surfaceActions } from '../commissioner/surfaceActions';
import { nativeAction, type NativeAction, type NativeView } from './nativeApi';
import NativeHelpers from './NativeHelpers';
import '../look/look.css';
import './nativeAccount.css';

type Screen = 'login' | 'join' | 'authenticator' | 'recovery';
const tabs = [
  ['login', 'Sign in'], ['join', 'Join with a code'],
  ['authenticator', 'Authenticator'], ['recovery', 'Account recovery'],
] as const;

export default function NativeAccount({ initial }: { initial: NativeView }) {
  const [view, setView] = useState(initial);
  const [screen, setScreen] = useState<Screen>('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [signedIn, setSignedIn] = useState(Boolean(initial.message));
  const changeScreen = (next: Screen) => {
    setScreen(next); setPassword(''); setCode(''); setError('');
    setView(current => ({ ...current, seed: '', message: '' }));
  };
  async function run(action: NativeAction) {
    if (pending) return;
    setPending(true); setError('');
    // Remove the old private reading before fresh admission.
    setView(current => ({ ...initial, seed: current.seed }));
    try {
      const credentials = !['enter', 'forget', 'revoke-device', 'revoke-all'].includes(action);
      const result = await nativeAction(action, credentials ? username : '', credentials ? password : '',
        ['recover', 'enroll', 'totp-confirm', 'totp-step-up'].includes(action) ? code : '');
      setView(result);
      if (action === 'login') setSignedIn(true);
      if (['forget', 'revoke-device', 'revoke-all'].includes(action)) setSignedIn(false);
      if (action === 'enroll' || action === 'recover') setScreen('login');
      if (action === 'recover') setSignedIn(false);
    } catch (e) {
      setView(initial);
      setError(e instanceof Error ? e.message : 'Could not reach your league. Try again.');
    } finally { setPassword(''); setCode(''); setPending(false); }
  }
  // Dispatch carries only action names, never passwords, codes or setup keys.
  useEffect(() => surfaceActions.register('native-account', args => {
    if (pending) return;
    if (args.action === 'view') {
      const tab = tabs.find(([id]) => id === args.value);
      if (tab) changeScreen(tab[0]);
    } else if (args.action === 'check') {
      const actions: NativeAction[] = [
        'totp-begin', 'totp-confirm', 'enter', 'forget', 'revoke-device', 'revoke-all',
      ];
      const action = actions.find(value => value === args.value);
      if (action) void run(action);
    }
  }));
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void run(screen === 'recovery' ? 'recover' : screen === 'join' ? 'enroll'
      : screen === 'authenticator' ? 'totp-step-up' : 'login');
  };
  const title = screen === 'join' ? 'Create your account'
    : screen === 'authenticator' ? 'Protect privileged access' : 'Sign in';
  const submitLabel = pending ? 'Please wait…' : screen === 'join' ? 'Create account'
    : screen === 'authenticator' ? 'Verify fresh code' : 'Sign in to server';
  const args = (action: 'view' | 'check', value: string) => ({ screen: 'native-account', action, value });
  return <main className="twr-app native-account" aria-busy={pending}>
    <header><h1>{view.name || 'Your native league'}</h1>
      <p>Hosted league · No MFL account or key needed</p>
      <p className="native-address">{initial.server} · {initial.league}</p></header>
    <nav aria-label="Account access">
      {tabs.map(([id, label]) => <Act key={id} variant="button" verb="commissioner.surface"
        args={args('view', id)} label={label} disabled={pending} current={screen === id}>{label}</Act>)}
    </nav>
    {error && <p role="alert">{error}</p>}
    {view.message && <p role="status">{view.message}</p>}
    {view.name && <section aria-label="Your league access">
      <h2>{view.franchiseName || 'League access'}</h2><p>{view.role}</p>
      {view.franchise && <p>Your assigned franchise: {view.franchise}.
        You are connected to the server; no local league is created.</p>}
    </section>}
    {signedIn && view.name && view.seats.length > 0 &&
      <NativeHelpers server={initial.server}
      league={initial.league} seats={view.seats} franchise={view.franchise} />}
    {screen === 'recovery' ? <section aria-labelledby="recovery-title">
      <h2 id="recovery-title">Get back into your account</h2>
      <h3>Lost your phone or password?</h3>
      <p>Keep your username and any of your eight printed one-time recovery codes private.
        Do not post passwords, setup keys or recovery codes in league chat.</p>
      <p>Use one unused printed code to choose a new password. This signs out all your devices
        and removes your old authenticator. Then sign in and set up your authenticator again if required.</p>
      <form onSubmit={submit}>
        <label>Username<input autoComplete="username" required minLength={3} maxLength={64}
          value={username} disabled={pending} onChange={e => setUsername(e.target.value)} /></label>
        <label>Unused recovery code<input autoComplete="off" required minLength={43} maxLength={43}
          value={code} disabled={pending} onChange={e => setCode(e.target.value)} /></label>
        <label>New password<input type="password" autoComplete="new-password" required
          maxLength={1024} value={password} disabled={pending}
          onChange={e => setPassword(e.target.value)} /></label>
        <Act variant="button" verb="commissioner.surface" args={args('check', 'submit')}
          submit disabled={pending} label="Reset password">{pending ? 'Please wait…' : 'Reset password'}</Act>
      </form>
      <p>No unused code? Contact the server operator for audited console recovery with the offline
        recovery key and its key holder. A commissioner cannot reset your account by reassigning a seat.</p>
      <h3>Code expired or already used?</h3>
      <p>Enrollment codes last 48 hours and work once for the intended username. Ask your commissioner
        to personally hand you a new code. If you already joined, sign in instead.</p>
      <h3>Signed in but no franchise?</h3>
      <p>Account enrollment does not assign a seat. Ask your commissioner to check your account’s
        franchise assignment. Do not create another account.</p>
      <Act variant="button" verb="commissioner.surface" args={args('view', 'login')}
        label="Back to sign in">Back to sign in</Act>
    </section> : <form onSubmit={submit}>
      <h2>{title}</h2>
      {screen === 'authenticator' && <p>Commissioners and deputies must use an authenticator.
        Enter your password for each step. Add the setup key as a time-based account, confirm a code,
        then wait for a fresh code to verify access for five minutes.</p>}
      <label>Username<input autoComplete="username" required minLength={3} maxLength={64} value={username}
        disabled={pending} onChange={e => setUsername(e.target.value)} /></label>
      <label>Password<input type="password"
        autoComplete={screen === 'join' ? 'new-password' : 'current-password'}
        required maxLength={1024} value={password} disabled={pending}
        onChange={e => setPassword(e.target.value)} /></label>
      {screen !== 'login' && <label>
        {screen === 'join' ? 'Personally handed enrollment code' : 'Six-digit authenticator code'}
        <input autoComplete={screen === 'join' ? 'off' : 'one-time-code'} value={code} disabled={pending}
          required={screen === 'join'} maxLength={screen === 'join' ? 43 : 6}
          inputMode={screen === 'authenticator' ? 'numeric' : 'text'}
          onChange={e => setCode(e.target.value)} /></label>}
      {view.seed && <p className="native-seed">Private setup key: <code>{view.seed}</code></p>}
      <div className="native-actions">
        {screen === 'authenticator' && <>
          <Act variant="button" verb="commissioner.surface" args={args('check', 'totp-begin')}
            disabled={pending || !username || !password} label="Get setup key">Get setup key</Act>
          <Act variant="button" verb="commissioner.surface" args={args('check', 'totp-confirm')}
            disabled={pending || code.length !== 6 || !password} label="Confirm setup">Confirm setup</Act>
        </>}
        <Act variant="button" verb="commissioner.surface" args={args('check', 'submit')}
          submit disabled={pending} label={submitLabel}>{submitLabel}</Act>
      </div>
    </form>}
    {signedIn && <div className="native-actions">
      <Act variant="button" verb="commissioner.surface" args={args('check', 'enter')}
        disabled={pending} label="Open my franchise">Open my franchise</Act>
      <Act variant="button" verb="commissioner.surface" args={args('check', 'forget')}
        disabled={pending} label="Forget saved sign-in">Forget saved sign-in</Act>
      <Act variant="button" verb="commissioner.surface" args={args('check', 'revoke-device')}
        disabled={pending} label="Revoke this device">Revoke this device</Act>
      <Act variant="button" verb="commissioner.surface" args={args('check', 'revoke-all')}
        disabled={pending} label="Revoke all devices">Revoke all devices</Act>
    </div>}
    <footer>Device sessions use your OS keychain when available, otherwise Go memory only.
      Revoke all devices signs you out everywhere. Authenticator verification is never saved.
      Offline execution is unavailable.</footer>
  </main>;
}
