import { useCallback, useEffect, useRef, useState, type FormEvent, type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent, type ReactNode } from 'react';
import type { HomeBrief } from './homeReads';
import type { LeagueChoice } from './leaguePicker';
import { answerPhoneQuestion } from './phoneAsk';
import './PhoneAppFrame.css';

export const PHONE_QUERY = '(max-width: 699px)';
export type PhoneTab = 'today' | 'live' | 'team' | 'moves' | 'league';
type Sheet = 'leagues' | 'ask' | null;
type PhoneBadges = Partial<Record<PhoneTab, number>>;
const TABS: { id: PhoneTab; label: string }[] = [
  { id: 'today', label: 'Today' }, { id: 'live', label: 'Live' },
  { id: 'team', label: 'Team' }, { id: 'moves', label: 'Moves' },
  { id: 'league', label: 'League' },
];

export function usePhoneViewport(enabled: boolean): boolean {
  const [phone, setPhone] = useState(() => enabled && phoneViewport());
  useEffect(() => {
    if (!enabled) return;
    if (typeof window.matchMedia !== 'function') {
      if (typeof window.addEventListener !== 'function') return;
      const update = () => setPhone(window.innerWidth < 700);
      window.addEventListener('resize', update);
      return () => window.removeEventListener('resize', update);
    }
    const query = window.matchMedia(PHONE_QUERY);
    const update = (event: MediaQueryListEvent) => setPhone(event.matches);
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }, [enabled]);
  return phone;
}

function phoneViewport(): boolean {
  if (typeof window === 'undefined') return false;
  return typeof window.matchMedia === 'function'
    ? window.matchMedia(PHONE_QUERY).matches : window.innerWidth < 700;
}

export function PhoneAppFrame({ enabled = true, activeTab, leagues, selectedLeagueId,
  weekLabel = 'Week unavailable', badges, live = false, askBrief, askMine = '', accountName,
  accountPending = false, commissioner = false, showControlRoom = false, onSelectTab,
  onSelectLeague, onAuthenticator, onSignOut, onControlRoom, homeReadSource, children }: {
    enabled?: boolean;
    activeTab: PhoneTab;
    leagues: LeagueChoice[];
    selectedLeagueId: string;
    weekLabel?: string;
    badges?: PhoneBadges;
    live?: boolean;
    askBrief?: HomeBrief;
    askMine?: string;
    accountName?: string;
    accountPending?: boolean;
    commissioner?: boolean;
    showControlRoom?: boolean;
    onSelectTab: (tab: PhoneTab) => void;
    onSelectLeague?: (leagueId: string) => void;
    onAuthenticator?: () => void;
    onSignOut?: () => void;
    onControlRoom?: () => void;
    /** Keep Home's reads available to Ask and the week chip without filling the placeholder tab body. */
    homeReadSource?: ReactNode;
    children: ReactNode;
  }) {
  const phone = usePhoneViewport(enabled);
  const [sheet, setSheet] = useState<Sheet>(null);
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState<ReturnType<typeof answerPhoneQuestion>>();
  const [toast, setToast] = useState('');
  const opener = useRef<HTMLButtonElement | null>(null);
  const selected = leagues.find(league => league.id === selectedLeagueId);
  const leagueName = selected?.name || 'Your league';

  const closeSheet = useCallback(() => {
    setSheet(null);
    window.setTimeout(() => opener.current?.focus(), 0);
  }, []);

  useEffect(() => {
    if (!sheet) return;
    const escape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      closeSheet();
    };
    document.addEventListener('keydown', escape);
    return () => document.removeEventListener('keydown', escape);
  }, [closeSheet, sheet]);

  useEffect(() => {
    if (!toast) return;
    const timeout = window.setTimeout(() => setToast(''), 2500);
    return () => window.clearTimeout(timeout);
  }, [toast]);

  function openSheet(next: Exclude<Sheet, null>, button: HTMLButtonElement) {
    opener.current = button;
    setSheet(next);
  }

  function chooseLeague(id: string) {
    const choice = leagues.find(league => league.id === id);
    if (!choice) return;
    if (id !== selectedLeagueId) {
      onSelectLeague?.(id);
      setToast(`Switched to ${choice.name}`);
    }
    closeSheet();
  }

  function submitAsk(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const prompt = question.trim();
    if (!prompt) return;
    setQuestion(prompt);
    setAnswer(answerPhoneQuestion(prompt, askBrief, askMine));
  }

  function askSuggestion(prompt: string) {
    setQuestion(prompt);
    setAnswer(answerPhoneQuestion(prompt, askBrief, askMine));
  }

  function dismissFromHandle(event: PointerEvent<HTMLButtonElement>) {
    if (event.type === 'pointerdown') {
      event.currentTarget.dataset.startY = String(event.clientY);
      event.currentTarget.setPointerCapture?.(event.pointerId);
      return;
    }
    const start = Number(event.currentTarget.dataset.startY);
    delete event.currentTarget.dataset.startY;
    if (event.type === 'pointerup' && Number.isFinite(start) && event.clientY - start > 70)
      closeSheet();
  }

  if (!enabled || !phone || showControlRoom) return <>{children}</>;

  return <section className="phone-app" aria-label="The War Room phone app">
    <header className="phone-app__bar">
      <button className="phone-app__league" type="button" aria-haspopup="dialog"
        aria-expanded={sheet === 'leagues'} onClick={event => openSheet('leagues', event.currentTarget)}>
        <b>{leagueName}</b><span>{weekLabel}</span><Chevron />
      </button>
      <button className="phone-app__ask" type="button" aria-haspopup="dialog"
        aria-expanded={sheet === 'ask'} onClick={event => openSheet('ask', event.currentTarget)}>Ask</button>
    </header>
    <div className="phone-app__body">
      <section className="phone-app__slot" id="phone-panel" role="tabpanel"
        aria-labelledby={`phone-tab-${activeTab}`} key={activeTab}>
        <h1>{tabLabel(activeTab)}</h1>
      </section>
    </div>
    <nav className="phone-app__tabs" role="tablist" aria-label="Main">
      {TABS.map(tab => {
        const count = badges?.[tab.id];
        const showBadge = Number.isInteger(count) && count! > 0;
        const name = `${tab.label}${showBadge ? ` · ${count} notifications` : ''}`
          + `${tab.id === 'live' && live ? ' · Games live' : ''}`;
        return <button className="phone-app__tab" id={`phone-tab-${tab.id}`} key={tab.id} type="button"
          role="tab" aria-controls="phone-panel" aria-selected={activeTab === tab.id}
          tabIndex={activeTab === tab.id ? 0 : -1} aria-label={name}
          onClick={() => onSelectTab(tab.id)}
          onKeyDown={event => movePhoneTab(event, tab.id, onSelectTab)}>
          <TabIcon tab={tab.id} />
          <span>{tab.label}</span>
          {showBadge && <i className="phone-app__badge" aria-label={`${count} notifications`}
            title={`${count} notifications`}>{count! > 99 ? '99+' : count}</i>}
          {tab.id === 'live' && live && <i className="phone-app__live-dot" aria-label="Games live" />}
        </button>;
      })}
    </nav>
    <div className="phone-app__toast" role="status" aria-live="polite">{toast}</div>
    {homeReadSource && <div className="phone-app__data" aria-hidden="true" hidden>{homeReadSource}</div>}
    {sheet && <PhoneSheet title={sheet === 'leagues' ? 'Your leagues' : 'Ask'}
      onClose={closeSheet} onHandlePointer={dismissFromHandle}>
      {sheet === 'leagues' ? <>
        <div className="phone-app__league-list" aria-label="Your leagues">
          {leagues.length === 0 && <p role="status">No admitted leagues.</p>}
          {leagues.map(league => <button className="phone-app__league-row" key={league.id}
            type="button" aria-current={league.id === selectedLeagueId ? 'true' : undefined}
            onClick={() => chooseLeague(league.id)}>
            <span><b>{league.name}</b></span>
            {league.id === selectedLeagueId && <span className="phone-app__selected">SELECTED</span>}
          </button>)}
        </div>
        {commissioner && activeTab === 'league' && onControlRoom && <button
          className="phone-app__league-row" type="button" onClick={() => {
            setSheet(null); onControlRoom();
          }}>Control Room</button>}
        <section className="phone-app__account" role="group" aria-labelledby="phone-app-account-title">
          <h3 id="phone-app-account-title">Account</h3>
          <div className="phone-app__league-row phone-app__account-name">
            <span><small>Signed in as</small><b>{accountName || 'Signed-in account'}</b></span>
          </div>
          <button className="phone-app__league-row" type="button" disabled={accountPending}
            onClick={() => { setSheet(null); onAuthenticator?.(); }}>Authenticator</button>
          <button className="phone-app__league-row" type="button" disabled={accountPending}
            onClick={() => { closeSheet(); onSignOut?.(); }}>Sign out</button>
        </section>
      </> : <div className="phone-app__ask-content">
        <p>Answers use only readings already on Today. No question is sent or saved.</p>
        <form onSubmit={submitAsk}>
          <label htmlFor="phone-app-question">Ask about a score</label>
          <div className="phone-app__ask-form">
            <input id="phone-app-question" autoComplete="off" value={question}
              onChange={event => { setQuestion(event.target.value); setAnswer(undefined); }} />
            <button type="submit">Ask</button>
          </div>
        </form>
        <div className="phone-app__suggestions" aria-label="Suggested questions">
          {['Am I winning?', 'Scores right now'].map(prompt => <button key={prompt} type="button"
            onClick={() => askSuggestion(prompt)}>{prompt}</button>)}
        </div>
        {answer && <div className="phone-app__answer" role="status" aria-live="polite">
          <b>{answer.label}</b><p>{answer.text}</p>
        </div>}
      </div>}
    </PhoneSheet>}
  </section>;
}

function PhoneSheet({ title, onClose, onHandlePointer, children }: {
  title: string;
  onClose: () => void;
  onHandlePointer: (event: PointerEvent<HTMLButtonElement>) => void;
  children: ReactNode;
}) {
  const closeButton = useRef<HTMLButtonElement>(null);
  const titleId = title === 'Ask' ? 'phone-app-ask-title' : 'phone-app-leagues-title';
  useEffect(() => { closeButton.current?.focus(); }, []);
  return <div className="phone-app__overlay">
    <button className="phone-app__scrim" type="button" aria-label={`Close ${title}`}
      onClick={onClose} />
    <section className="phone-app__sheet" role="dialog" aria-modal="true" aria-labelledby={titleId}
      onKeyDown={trapTab}>
      <button className="phone-app__grab" type="button" aria-label="Swipe down to close"
        onPointerDown={onHandlePointer} onPointerUp={onHandlePointer} onPointerCancel={onHandlePointer}>
        <i />
      </button>
      <header className="phone-app__sheet-head">
        <h2 id={titleId}>{title}</h2>
        <button ref={closeButton} className="phone-app__close" type="button" onClick={onClose}>Close</button>
      </header>
      <div className="phone-app__sheet-body">{children}</div>
    </section>
  </div>;
}

function movePhoneTab(event: ReactKeyboardEvent<HTMLButtonElement>, current: PhoneTab,
  onSelectTab: (tab: PhoneTab) => void) {
  const at = TABS.findIndex(tab => tab.id === current);
  const next = event.key === 'ArrowRight' ? TABS[(at + 1) % TABS.length]
    : event.key === 'ArrowLeft' ? TABS[(at + TABS.length - 1) % TABS.length]
      : event.key === 'Home' ? TABS[0] : event.key === 'End' ? TABS[TABS.length - 1] : undefined;
  if (!next) return;
  event.preventDefault();
  onSelectTab(next.id);
  event.currentTarget.parentElement?.querySelector<HTMLButtonElement>(`#phone-tab-${next.id}`)?.focus();
}

function trapTab(event: ReactKeyboardEvent<HTMLElement>) {
  if (event.key !== 'Tab') return;
  const focusable = [...event.currentTarget.querySelectorAll<HTMLElement>(
    'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [href], [tabindex]:not([tabindex="-1"])',
  )];
  const first = focusable[0], last = focusable[focusable.length - 1];
  if (!first || !last) return;
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault(); last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault(); first.focus();
  }
}

function tabLabel(tab: PhoneTab): string {
  return TABS.find(item => item.id === tab)?.label ?? 'Today';
}

function Chevron() {
  return <svg aria-hidden="true" viewBox="0 0 24 24" width="14" height="14" fill="none"
    stroke="currentColor" strokeWidth="2.2"><path d="m6 9 6 6 6-6" /></svg>;
}

function TabIcon({ tab }: { tab: PhoneTab }) {
  const paths: Record<PhoneTab, ReactNode> = {
    today: <><rect x="4" y="4" width="6" height="6" /><rect x="14" y="4" width="6" height="6" />
      <rect x="4" y="14" width="6" height="6" /><rect x="14" y="14" width="6" height="6" /></>,
    live: <><path d="M3 12h4l3-7 4 14 3-7h4" /></>,
    team: <><circle cx="9" cy="8" r="3" /><path d="M3 20v-2a6 6 0 0 1 12 0v2M16 5a3 3 0 0 1 0 6m2 3a4 4 0 0 1 3 4v2" /></>,
    moves: <><path d="M4 7h15m-4-4 4 4-4 4M20 17H5m4-4-4 4 4 4" /></>,
    league: <><path d="M4 5h16v14H4zM8 9h8M8 13h8M8 16h4" /></>,
  };
  return <svg className="phone-app__tab-icon" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" strokeWidth="1.7" strokeLinecap="square" aria-hidden="true">{paths[tab]}</svg>;
}
