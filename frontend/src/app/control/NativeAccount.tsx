import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore,
  type FormEvent } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { surfaceActions } from '../commissioner/surfaceActions';
import { nativeAction, selectBrowserLeague, type NativeAction, type NativeView } from './nativeApi';
import { FailureAlert } from '../data/FailureAlert';
import { setFailureScreen } from '../data/errorCodes';
import { hostedSnapshot, onHostedChange, observeView } from '../data/hostedReads';
import { HQAct } from '../shell/HQAct';
import type { ThinHomeEntry } from '../shell/ThinHome';
import { PhoneAppFrame, usePhoneViewport, type PhoneTab } from '../shell/PhoneAppFrame';
import type { HomeBrief } from '../shell/homeReads';
import { TeamTab } from '../phone/TeamTab';
import type { LeagueChoice } from '../shell/leaguePicker';
import '../look/look.css';
import './nativeAccount.css';

const HostedLeague = lazy(() => import('../commissioner/HostedLeague'));
const ThinHome = lazy(() => import('../shell/ThinHome'));
const V3Shell = lazy(() => import('../v3/V3Shell'));
const TodayTab = lazy(() => import('../phone/TodayTab'));
import type { TodayBadges } from '../phone/TodayTab';
const LiveTab = lazy(() => import('../phone/LiveTab'));
const MovesTab = lazy(() => import('../phone/MovesTab'));
const LeagueTab = lazy(() => import('../phone/LeagueTab'));

type Screen = 'login' | 'authenticator';

export default function NativeAccount({ initial }: { initial: NativeView }) {
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
  const [phoneBadges, setPhoneBadges] = useState<Partial<Record<PhoneTab, number>>>({});
  const badge = useMemo(() => {
    const set = (tab: PhoneTab) => (count: number) =>
      setPhoneBadges(current => current[tab] === count ? current : { ...current, [tab]: count });
    return { today: set('today'), team: set('team'), league: set('league') };
  }, []);
  const [phoneLive, setPhoneLive] = useState(false);
  const todayBadges = useCallback((marks: TodayBadges) => {
    badge.team(marks.team);
    setPhoneBadges(current => current.moves === marks.moves && current.league === marks.league ? current
      : { ...current, moves: marks.moves, league: marks.league });
    setPhoneLive(marks.live);
  }, [badge]);
  const [phoneBrief, setPhoneBrief] = useState<{ leagueId: string; brief: HomeBrief; mine: string }>();
  const automaticEntry = useRef(false);
  const admittedLeagues = useMemo<LeagueChoice[]>(() =>
    (admission?.leagues ?? []).map(league => ({ ...league, app: true })), [admission]);
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
  const phoneViewport = usePhoneViewport(true);
  // On a phone the app frame owns the whole screen: none of this page's own chrome may show.
  const phoneFrame = phoneViewport && homeReady && screen === 'login'
    && !(activeLeague && activeEntry === 'control');
  useEffect(() => {
    if (phoneFrame) {
      const label = phoneTab[0].toUpperCase() + phoneTab.slice(1);
      setFailureScreen(`Phone · ${label}`, `phone/${phoneTab}`);
    } else if (showThinHome) {
      setFailureScreen('Home', 'home/overview');
    } else if (!activeLeague) {
      setFailureScreen(`Account · ${screen}`, `account/${screen}`);
    }
  }, [phoneFrame, phoneTab, showThinHome, activeLeague, screen]);
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
      const credentials = action !== 'enter' && action !== 'forget';
      const result = await nativeAction(action, credentials ? username : '', credentials ? password : '',
        ['login', 'totp-confirm', 'totp-step-up'].includes(action) ? code : '');
      setView(result);
      if (action === 'login') {
        setSignedIn(true); setSignedInName(username); setOpenAfterSignIn(true);
      }
      if (action === 'enter') {
        setSignedIn(Boolean(result.name));
        if (!automaticEntry.current) setActiveLeagueId(result.league);
        if (!result.name) setActiveLeagueId(undefined);
        automaticEntry.current = false;
      }
      if (action === 'forget') {
        setSignedIn(false); setSignedInName(''); setActiveLeagueId(undefined);
      }
    } catch (e) {
      setView(hostedSnapshot().view ?? initial);
      const reason = e instanceof Error ? e.message : 'Could not reach your league. Try again.';
      setError(reason);
      throw new Error(reason);
    } finally { setPassword(''); setCode(''); setPending(false); }
  }
  // Dispatch carries only action names, never passwords, codes or setup keys.
  useEffect(() => surfaceActions.register('native-account', args => {
    if (pending) return;
    if (args.action === 'view') {
      if (args.value === 'login') changeScreen('login');
    } else if (args.action === 'check') {
      if (args.value === 'submit') return run(screen === 'authenticator' ? 'totp-step-up' : 'login');
      const actions: NativeAction[] = ['totp-begin', 'totp-confirm', 'enter', 'forget'];
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
  const title = screen === 'authenticator' ? 'Authenticator (optional)' : 'Sign in';
  const visibleTabs = view.name ? [['login', 'League']] as const : [];
  const submitLabel = pending ? 'Please wait…' : screen === 'authenticator' ? 'Verify fresh code' : 'Sign in';
  const args = (action: 'view' | 'check', value: string) => ({ screen: 'native-account', action, value });
  const thinHome = <Suspense fallback={<p role="status">Opening Home…</p>}>
    <ThinHome leagues={admittedLeagues} onBriefReady={receiveHomeBrief} onOpen={(league, entry) => {
      setActiveEntry(entry); setPhoneTab(tabForEntry(entry)); setActiveLeagueId(league.id);
    }} />
  </Suspense>;
  const liveLeague = admittedLeagues.find(league => league.id === selectedPhoneLeagueId);
  const liveTab = <Suspense fallback={<p role="status">Opening Live…</p>}>
    <LiveTab league={liveLeague} /></Suspense>;
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
  </>;
  const accountForms = <>
    {screen === 'login' && view.name ? null : <form onSubmit={submit}>
      {homeReady && screen === 'authenticator' && <Act variant="button"
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
        autoComplete="current-password"
        required maxLength={1024} value={password} disabled={pending}
        onChange={e => setPassword(e.target.value)} /></label>
      <label>Six-digit authenticator code
        <input autoComplete="one-time-code" value={code} disabled={pending} maxLength={6}
          inputMode="numeric" onChange={e => setCode(e.target.value)} /></label>
      {view.seed && <p className="native-seed">Private setup key: <code>{view.seed}</code></p>}
      {view.recoveryCodes && <ol className="native-seed" aria-label="New recovery codes">
        {view.recoveryCodes.map(c => <li key={c}><code>{c}</code></li>)}</ol>}
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
    {signedIn && !phoneFrame && <div className="native-actions">
      <Act variant="button" verb="commissioner.surface" args={args('check', 'enter')}
        disabled={pending} label="Open my franchise">Open my franchise</Act>
      <Act variant="button" verb="commissioner.surface" args={args('check', 'forget')}
        disabled={pending} label="Sign out">
        Sign out</Act>
    </div>}
  </>;
  // Desktop v3: once admitted on a desktop-sized screen, the v3 frame owns the window (mock v3).
  // Account panels open in its work area; an open league stays mounted.
  if (homeReady && !phoneFrame && !phoneViewport) {
    const accountPanel = screen === 'login' ? undefined : <section className="twr-app native-account">
      <nav aria-label="Account access">
        {visibleTabs.map(([id, label]) =>
          <Act key={id} variant="button" verb="commissioner.surface"
          args={args('view', id)} label={label} disabled={pending}>{label}</Act>)}
      </nav>
      {error && <FailureAlert message={error} allowSend={Boolean(admission?.league)} />}
      {view.message && <p role="status">{view.message}</p>}
      {accountForms}</section>;
    const accountMenu = <>
      <Act variant="button" verb="commissioner.surface" args={args('check', 'enter')}
        disabled={pending} label="Open my franchise">Open my franchise</Act><hr />
      <Act variant="button" verb="commissioner.surface" args={args('check', 'forget')}
        disabled={pending} label="Sign out">
        Sign out</Act>
    </>;
    return <Suspense fallback={<p role="status">Opening The War Room…</p>}>
      <V3Shell leagues={admittedLeagues} accountName={signedInName || undefined} panel={accountPanel}
        account={accountMenu} status={error || undefined}
        active={activeLeague ? { league: activeLeague, entry: activeEntry } : undefined}
        onOpen={(league, entry) => {
          if (screen !== 'login') changeScreen('login');
          setActiveEntry(entry); setActiveLeagueId(league.id);
        }}
        onHome={() => { if (screen !== 'login') changeScreen('login'); setActiveLeagueId(undefined); }}
        onSignOut={() => {
          void Promise.resolve(commands.dispatch('commissioner.surface', args('check', 'forget')))
            .catch(() => {});
        }}>
        {activeLeague && <Suspense fallback={<p>Opening your league…</p>}>
          <HostedLeague key={`${fence.key}:${activeLeague.id}`} league={activeLeague.id}
            franchise={activeLeague.franchise ?? ''} name={activeLeague.franchiseName ?? ''}
            role={activeLeague.role ?? ''} entry={activeEntry} hq
            onHome={() => setActiveLeagueId(undefined)} />
        </Suspense>}
      </V3Shell></Suspense>;
  }
  return <main className={`twr-app native-account${showThinHome ? ' has-thin-home' : ''}`
    + `${homeReady ? ' phone-home' : ''}`} aria-busy={pending}>
    {!phoneFrame && <><header><h1>{view.name || 'Your league server'}</h1>
      {!view.name && <p>Your league server · No MFL account or key needed</p>}
      </header>
    <nav aria-label="Account access">
      {visibleTabs.map(([id, label]) =>
        <Act key={id} variant="button" verb="commissioner.surface"
        args={args('view', id)} label={label} disabled={pending} current={screen === id}>{label}</Act>)}
    </nav></>}
    {error && <FailureAlert message={error} allowSend={Boolean(admission?.league)} />}
    {view.message && !showThinHome && !phoneFrame && <p role="status">{view.message}</p>}
    <div hidden={screen !== 'login'}>
    {view.name && !homeReady && <section aria-label="Your league access">
      <h2>{view.franchiseName || 'League access'}</h2><p>{view.role}</p>
      {view.franchise && <p>You are connected to the server; no local league is created.</p>}
    </section>}
    {homeReady && <PhoneAppFrame activeTab={phoneTab} badges={phoneBadges} live={phoneLive}
      leagues={admittedLeagues} selectedLeagueId={selectedPhoneLeagueId} weekLabel={phoneWeekLabel}
      askBrief={currentPhoneBrief?.brief} askMine={currentPhoneBrief?.mine}
      accountName={signedInName || undefined} accountPending={pending}
      commissioner={admission?.role === 'Commissioner' || admission?.role === 'Co-commissioner'}
      showControlRoom={Boolean(activeLeague && activeEntry === 'control')}
      onSelectTab={setPhoneTab} onSelectLeague={id => {
        if (id !== admission?.league) setView(selectBrowserLeague(id));
      }} onControlRoom={() => {
        setActiveEntry('control');
        if (admission) setActiveLeagueId(admission.league);
      }} onAuthenticator={() => changeScreen('authenticator')}
      onSignOut={() => {
        void Promise.resolve(commands.dispatch('commissioner.surface', args('check', 'forget')))
          .catch(() => {});
      }} homeReadSource={thinHome} tabs={{
        today: <Suspense fallback={<p role="status">Opening Today…</p>}>
          <TodayTab leagues={admittedLeagues} onNow={badge.today}
            current={selectedPhoneLeagueId} onBadges={todayBadges} onOpen={(league, entry) => {
            if (league.id !== admission?.league) setView(selectBrowserLeague(league.id));
            setActiveEntry(entry); setPhoneTab(tabForEntry(entry)); setActiveLeagueId(league.id);
          }} /></Suspense>,
        team: admission?.franchise
          ? <TeamTab league={admission.league} mine={admission.franchise}
            onByes={badge.team} />
          : <p className="team-tab__note" role="status">No team of yours in this league.</p>,
        live: liveTab,
        moves: <Suspense fallback={<p role="status">Opening Moves…</p>}>
          <MovesTab league={admittedLeagues.find(league => league.id === selectedPhoneLeagueId)} />
        </Suspense>,
        league: <Suspense fallback={<p role="status">Opening League…</p>}>
          <LeagueTab league={admittedLeagues.find(league => league.id === selectedPhoneLeagueId)}
            onOpenJobs={badge.league} onTrade={() => setPhoneTab('moves')} onWatch={() => setPhoneTab('live')}
            onControlRoom={() => {
              setActiveEntry('control');
              if (admission) setActiveLeagueId(admission.league);
            }} />
        </Suspense>,
      }}>{leagueContent}</PhoneAppFrame>}
    </div>
    {accountForms}
    {!phoneFrame && <footer>
      You stay signed in on this device. The league needs an internet connection.
    </footer>}
  </main>;
}
