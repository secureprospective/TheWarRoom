import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore,
  type FormEvent } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { surfaceActions } from '../commissioner/surfaceActions';
import { nativeAction, type NativeAction, type NativeView } from './nativeApi';
import { isWebTransport } from './transport';
import { hostedSnapshot, onHostedChange, observeView } from '../data/hostedReads';
import NativeHelpers from './NativeHelpers';
import { HQAct } from '../shell/HQAct';
import type { ThinHomeEntry } from '../shell/ThinHome';
import { PhoneAppFrame, type PhoneTab } from '../shell/PhoneAppFrame';
import type { HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import '../look/look.css';
import './nativeAccount.css';

const HostedLeague = lazy(() => import('../commissioner/HostedLeague'));
const ThinHome = lazy(() => import('../shell/ThinHome'));

type Screen = 'login' | 'join' | 'authenticator' | 'recovery';
const tabs = [
  ['login', 'Sign in'], ['join', 'Join with a code'],
  ['authenticator', 'Authenticator'], ['recovery', 'Account recovery'],
] as const;
// Once the league is open, signing in, joining and recovery are done; only the league and the
// optional authenticator stay.
const openTabs = [['login', 'League'], ['authenticator', 'Authenticator']] as const;

export default function NativeAccount({ initial }: { initial: NativeView }) {
  const browser = isWebTransport();
  const [view, setView] = useState(initial);
  const [observed, setObserved] = useState(false);
  useEffect(() => { observeView(initial); setObserved(true); }, [initial]);
  const fence = useSyncExternalStore(onHostedChange, hostedSnapshot);
  const admission = fence.view;
  useEffect(() => onHostedChange(() => {
    const latest = hostedSnapshot().view;
    if (latest) setView(latest);
  }), []);
  const [screen, setScreen] = useState<Screen>('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [signedIn, setSignedIn] = useState(Boolean(initial.message || initial.name));
  const [signedInName, setSignedInName] = useState('');
  const [openAfterSignIn, setOpenAfterSignIn] = useState(Boolean(initial.message || initial.name));
  const [activeLeagueId, setActiveLeagueId] = useState<string>();
  const [activeEntry, setActiveEntry] = useState<ThinHomeEntry>('hq');
  const [phoneTab, setPhoneTab] = useState<PhoneTab>('today');
  const [phoneBrief, setPhoneBrief] = useState<{ leagueId: string; brief: HomeBrief; mine: string }>();
  const automaticEntry = useRef(false);
  const admittedLeagues = useMemo<LeagueChoice[]>(() => admission ? [{
    id: admission.league, name: admission.name, app: true, franchise: admission.franchise,
    franchiseName: admission.franchiseName, role: admission.role,
  }] : [], [admission]);
  const homeReady = observed && !fence.blocked && signedIn && Boolean(admission)
    && Boolean(admission?.franchise || admission?.role === 'Commissioner'
      || admission?.role === 'Co-commissioner');
  const receiveHomeBrief = useCallback((leagueId: string, mine: string, brief: HomeBrief) => {
    const latest = hostedSnapshot();
    if (latest.blocked || latest.key !== fence.key || leagueId !== admission?.league) return;
    setPhoneBrief({ leagueId, brief, mine });
  }, [admission?.league, fence.key]);
  const tabForEntry = (entry: ThinHomeEntry): PhoneTab => entry === 'pulse' ? 'live'
    : entry === 'hq' || entry === 'lineup' ? 'team'
      : entry === 'war' || entry === 'bids' || entry === 'trades' ? 'moves' : 'league';
  useEffect(() => {
    if (fence.blocked || !signedIn) {
      setActiveLeagueId(undefined); setPhoneTab('today'); setPhoneBrief(undefined);
    }
    if (fence.blocked) setSignedInName('');
  }, [fence.blocked, signedIn]);
  useEffect(() => { setPhoneBrief(undefined); }, [fence.blocked, fence.key, admission?.league]);
  const activeLeague = admittedLeagues.find(league => league.id === activeLeagueId);
  const showThinHome = homeReady && screen === 'login' && !activeLeague;
  const selectedPhoneLeagueId = admission?.league ?? '';
  const currentPhoneBrief = phoneBrief?.leagueId === selectedPhoneLeagueId ? phoneBrief : undefined;
  const phoneWeek = currentPhoneBrief?.brief.pulse?.week;
  const phoneWeekLabel = phoneWeek !== undefined ? `WK ${phoneWeek}`
    : currentPhoneBrief?.brief.pulse?.weekNote?.trim() || 'Week unavailable';
  const changeScreen = (next: Screen) => {
    setScreen(next); setPassword(''); setCode(''); setError('');
    setView(current => ({ ...current, seed: '', message: current.name ? current.message : '' }));
  };
  async function run(action: NativeAction) {
    if (pending) return;
    setPending(true); setError('');
    try {
      const credentials = !['enter', 'forget', 'revoke-device', 'revoke-all'].includes(action);
      const result = await nativeAction(action, credentials ? username : '', credentials ? password : '',
        ['recover', 'enroll', 'totp-confirm', 'totp-step-up'].includes(action)
          || (browser && action === 'login') ? code : '');
      setView(result);
      if (action === 'login') {
        setSignedIn(true); setSignedInName(username); setOpenAfterSignIn(true);
      }
      if (action === 'enter') {
        setSignedIn(browser ? Boolean(result.name) : true);
        if (!automaticEntry.current) setActiveLeagueId(result.league);
        if (browser && !result.name) setActiveLeagueId(undefined);
        automaticEntry.current = false;
      }
      if (['forget', 'revoke-device', 'revoke-all'].includes(action)) {
        setSignedIn(false); setSignedInName(''); setActiveLeagueId(undefined);
      }
      if (action === 'enroll' || action === 'recover') setScreen('login');
      if (action === 'recover') setSignedIn(false);
    } catch (e) {
      setView(browser ? (hostedSnapshot().view ?? initial) : initial);
      const reason = e instanceof Error ? e.message : 'Could not reach your league. Try again.';
      setError(reason);
      throw new Error(reason);
    } finally { setPassword(''); setCode(''); setPending(false); }
  }
  // Dispatch carries only action names, never passwords, codes or setup keys.
  useEffect(() => surfaceActions.register('native-account', args => {
    if (pending) return;
    if (args.action === 'view') {
      const tab = tabs.find(([id]) => id === args.value && (!browser || id === 'login'));
      if (tab) changeScreen(tab[0]);
    } else if (args.action === 'check') {
      if (args.value === 'submit') return run(screen === 'recovery' ? 'recover'
        : screen === 'join' ? 'enroll' : screen === 'authenticator' ? 'totp-step-up' : 'login');
      const actions: NativeAction[] = [
        'totp-begin', 'totp-confirm', 'recovery-codes', 'enter', 'forget', 'revoke-device', 'revoke-all',
      ];
      const action = actions.find(value => value === args.value);
      if (action) return run(action);
    }
  }));
  // Sign-in still rechecks admission before Home; a refusal shows its reason while saved access stays.
  useEffect(() => {
    if (!openAfterSignIn || pending) return;
    setOpenAfterSignIn(false);
    automaticEntry.current = true;
    void Promise.resolve(commands.dispatch('commissioner.surface', {
      screen: 'native-account', action: 'check', value: 'enter',
    })).catch(() => { automaticEntry.current = false; });
  }, [openAfterSignIn, pending]);
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void commands.dispatch('commissioner.surface', {
      screen: 'native-account', action: 'check', value: 'submit',
    });
  };
  const title = screen === 'join' ? 'Create your account'
    : screen === 'authenticator' ? 'Authenticator (optional)' : 'Sign in';
  const visibleTabs = browser
    ? view.name ? [['login', 'League']] as const : [['login', 'Sign in']] as const
    : view.name ? openTabs : tabs;
  const submitLabel = pending ? 'Please wait…' : screen === 'join' ? 'Create account'
    : screen === 'authenticator' ? 'Verify fresh code' : browser ? 'Sign in' : 'Sign in to server';
  const args = (action: 'view' | 'check', value: string) => ({ screen: 'native-account', action, value });
  const thinHome = <Suspense fallback={<p role="status">Opening Home…</p>}>
    <ThinHome leagues={admittedLeagues} onBriefReady={receiveHomeBrief} onOpen={(league, entry) => {
      setActiveEntry(entry); setPhoneTab(tabForEntry(entry)); setActiveLeagueId(league.id);
    }} />
  </Suspense>;
  const leagueContent = <>
    {showThinHome && thinHome}
    {homeReady && activeLeague && <section className="native-hosted-league" aria-label="League view">
      <HQAct className="native-home-return" onAction={() => setActiveLeagueId(undefined)}>← Home</HQAct>
      <Suspense fallback={<p>Opening your league…</p>}>
        <HostedLeague key={`${fence.key}:${activeLeague.id}`} league={activeLeague.id}
          franchise={activeLeague.franchise ?? ''} name={activeLeague.franchiseName ?? ''}
          role={activeLeague.role ?? ''} entry={activeEntry} hq
          onHome={() => setActiveLeagueId(undefined)} />
      </Suspense>
    </section>}
    {!browser && homeReady && activeLeague && admission && admission.seats.length > 0 &&
      <NativeHelpers server={initial.server} league={admission.league} seats={admission.seats}
        franchise={admission.franchise} />}
  </>;
  return <main className={`twr-app native-account${showThinHome ? ' has-thin-home' : ''}`
    + `${browser && homeReady ? ' phone-home' : ''}`} aria-busy={pending}>
    <header><h1>{view.name || 'Your league server'}</h1>
      {!view.name && <p>Your league server · No MFL account or key needed</p>}
      {!browser && <p className="native-address">{initial.server}</p>}</header>
    <nav aria-label="Account access">
      {visibleTabs.map(([id, label]) =>
        <Act key={id} variant="button" verb="commissioner.surface"
        args={args('view', id)} label={label} disabled={pending} current={screen === id}>{label}</Act>)}
    </nav>
    {error && <p role="alert">{error}</p>}
    {view.message && !showThinHome && <p role="status">{view.message}</p>}
    <div hidden={screen !== 'login'}>
    {view.name && !homeReady && <section aria-label="Your league access">
      <h2>{view.franchiseName || 'League access'}</h2><p>{view.role}</p>
      {view.franchise && <p>You are connected to the server; no local league is created.</p>}
    </section>}
    {homeReady && <PhoneAppFrame enabled={browser} activeTab={phoneTab}
      leagues={admittedLeagues} selectedLeagueId={selectedPhoneLeagueId} weekLabel={phoneWeekLabel}
      askBrief={currentPhoneBrief?.brief} askMine={currentPhoneBrief?.mine}
      accountName={signedInName || undefined} accountPending={pending}
      commissioner={admission?.role === 'Commissioner' || admission?.role === 'Co-commissioner'}
      showControlRoom={Boolean(activeLeague && activeEntry === 'control')}
      onSelectTab={setPhoneTab} onControlRoom={() => {
        setActiveEntry('control');
        if (admission) setActiveLeagueId(admission.league);
      }} onAuthenticator={() => changeScreen('authenticator')}
      onSignOut={() => {
        void Promise.resolve(commands.dispatch('commissioner.surface', args('check', 'forget')))
          .catch(() => {});
      }} homeReadSource={thinHome}>{leagueContent}</PhoneAppFrame>}
    </div>
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
    </section> : screen === 'login' && view.name ? null : <form onSubmit={submit}>
      {browser && homeReady && screen === 'authenticator' && <Act variant="button"
        verb="commissioner.surface" args={args('view', 'login')} label="Back to league">
        ← Back to league</Act>}
      <h2>{title}</h2>
      {screen === 'authenticator' && <p>An authenticator is optional. Once it is on,
        commissioners and deputies verify a fresh code to open the league. Enter your password
        for each step. Add the setup key as a time-based account, confirm a code, then wait for
        a fresh code to verify access for five minutes.</p>}
      <label>Username<input autoComplete="username" required minLength={3} maxLength={64} value={username}
        disabled={pending} onChange={e => setUsername(e.target.value)} /></label>
      <label>Password<input type="password"
        autoComplete={screen === 'join' ? 'new-password' : 'current-password'}
        required maxLength={1024} value={password} disabled={pending}
        onChange={e => setPassword(e.target.value)} /></label>
      {(screen !== 'login' || browser) && <label>
        {screen === 'join' ? 'Personally handed enrollment code' : 'Six-digit authenticator code'}
        <input autoComplete={screen === 'join' ? 'off' : 'one-time-code'} value={code} disabled={pending}
          required={screen === 'join'} maxLength={screen === 'join' ? 43 : 6}
          inputMode={screen === 'authenticator' || (browser && screen === 'login') ? 'numeric' : 'text'}
          onChange={e => setCode(e.target.value)} /></label>}
      {view.seed && <p className="native-seed">Private setup key: <code>{view.seed}</code></p>}
      {view.recoveryCodes && <ol className="native-seed" aria-label="New recovery codes">
        {view.recoveryCodes.map(c => <li key={c}><code>{c}</code></li>)}</ol>}
      <div className="native-actions">
        {screen === 'authenticator' && <>
          <Act variant="button" verb="commissioner.surface" args={args('check', 'totp-begin')}
            disabled={pending || !username || !password} label="Get setup key">Get setup key</Act>
          <Act variant="button" verb="commissioner.surface" args={args('check', 'totp-confirm')}
            disabled={pending || code.length !== 6 || !password} label="Confirm setup">Confirm setup</Act>
          {!browser && <Act variant="button" verb="commissioner.surface"
            args={args('check', 'recovery-codes')}
            disabled={pending || !username || !password} label="Print new recovery codes">
            Print new recovery codes</Act>}
        </>}
        <Act variant="button" verb="commissioner.surface" args={args('check', 'submit')}
          submit disabled={pending} label={submitLabel}>{submitLabel}</Act>
      </div>
    </form>}
    {signedIn && <div className="native-actions">
      <Act variant="button" verb="commissioner.surface" args={args('check', 'enter')}
        disabled={pending} label="Open my franchise">Open my franchise</Act>
      <Act variant="button" verb="commissioner.surface" args={args('check', 'forget')}
        disabled={pending} label={browser ? 'Sign out' : 'Forget saved sign-in'}>
        {browser ? 'Sign out' : 'Forget saved sign-in'}</Act>
      {!browser && <>
        <Act variant="button" verb="commissioner.surface" args={args('check', 'revoke-device')}
          disabled={pending} label="Revoke this device">Revoke this device</Act>
        <Act variant="button" verb="commissioner.surface" args={args('check', 'revoke-all')}
          disabled={pending} label="Revoke all devices">Revoke all devices</Act>
      </>}
    </div>}
    <footer>{browser
      ? 'Your sign-in stays in this browser as an HttpOnly cookie. The league needs an internet connection.'
      : 'This computer keeps your sign-in in its keychain when it has one, '
        + 'otherwise until you close the app. Revoke all devices signs you out everywhere. '
        + 'Authenticator codes are never saved. The league needs an internet connection.'}</footer>
  </main>;
}
