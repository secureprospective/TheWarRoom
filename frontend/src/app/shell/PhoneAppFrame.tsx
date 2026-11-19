import { useCallback, useEffect, useRef, useState, type FormEvent, 
  type PointerEvent, type ReactNode } from 'react';
import type { HomeBrief } from './homeReads';
import type { LeagueChoice } from './leaguePicker';
import { answerPhoneQuestion } from './phoneAsk';
import { HQAct } from './HQAct';
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
  onSelectLeague, onAuthenticator, onSignOut, onControlRoom, homeReadSource, tabs, children }: {
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
    /** Each ported tab's content; a tab without one shows its name until its port lands. */
    tabs?: Partial<Record<PhoneTab, ReactNode>>;
    children: ReactNode;
  }) {
  const phone = usePhoneViewport(enabled);
  const [sheet, setSheet] = useState<Sheet>(null);
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState<ReturnType<typeof answerPhoneQuestion>>();
  const [toast, setToast] = useState('');
  const opener = useRef<Exclude<Sheet, null> | null>(null);
  const selected = leagues.find(league => league.id === selectedLeagueId);
  const leagueName = selected?.name || 'Your league';

  const closeSheet = useCallback(() => setSheet(null), []);

  useEffect(() => {
    // Focus returns to the app-bar control that opened the sheet, once the sheet is gone.
    if (sheet || !opener.current) return;
    const from = opener.current === 'ask' ? '.phone-app__ask' : '.phone-app__league';
    opener.current = null;
    document.querySelector<HTMLElement>(from)?.focus();
  }, [sheet]);

  useEffect(() => {
    if (!toast) return;
    const timeout = window.setTimeout(() => setToast(''), 2500);
    return () => window.clearTimeout(timeout);
  }, [toast]);

  function openSheet(next: Exclude<Sheet, null>) {
    opener.current = next;
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

  function dismissFromHandle(event: PointerEvent<HTMLDivElement>) {
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
      <HQAct className="phone-app__league" expanded={sheet === 'leagues'}
        label={`${leagueName}, ${weekLabel}. Your leagues`} onAction={() => openSheet('leagues')}>
        <b>{leagueName}</b><span>{weekLabel}</span><Chevron />
      </HQAct>
      <HQAct className="phone-app__ask" expanded={sheet === 'ask'} label="Ask"
        onAction={() => openSheet('ask')}>Ask</HQAct>
    </header>
    <div className="phone-app__body">
      <section className="phone-app__slot" id="phone-panel" role="tabpanel"
        aria-label={tabLabel(activeTab)} key={activeTab}>
        {tabs?.[activeTab] ?? <h1>{tabLabel(activeTab)}</h1>}
      </section>
    </div>
    <nav className="phone-app__tabs" role="tablist" aria-label="Main">
      {TABS.map(tab => {
        const count = badges?.[tab.id];
        const showBadge = Number.isInteger(count) && count! > 0;
        const name = `${tab.label}${showBadge ? ` · ${count} notifications` : ''}`
          + `${tab.id === 'live' && live ? ' · Games live' : ''}`;
        return <HQAct className="phone-app__tab" key={tab.id} role="tab"
          selected={activeTab === tab.id} label={name} onAction={() => onSelectTab(tab.id)}>
          <TabIcon tab={tab.id} />
          <span>{tab.label}</span>
          {showBadge && <i className="phone-app__badge" aria-hidden="true">{count! > 99 ? '99+' : count}</i>}
          {tab.id === 'live' && live && <i className="phone-app__live-dot" aria-hidden="true" />}
        </HQAct>;
      })}
    </nav>
    <div className="phone-app__toast" role="status" aria-live="polite">{toast}</div>
    {homeReadSource && <div className="phone-app__data" aria-hidden="true" hidden>{homeReadSource}</div>}
    {sheet && <PhoneSheet title={sheet === 'leagues' ? 'Your leagues' : 'Ask'}
      onClose={closeSheet} onHandlePointer={dismissFromHandle}>
      {sheet === 'leagues' ? <>
        <div className="phone-app__league-list" aria-label="Your leagues">
          {leagues.length === 0 && <p role="status">No admitted leagues.</p>}
          {leagues.map(league => <HQAct className="phone-app__league-row" key={league.id}
            active={league.id === selectedLeagueId} label={league.name}
            onAction={() => chooseLeague(league.id)}>
            <span><b>{league.name}</b></span>
            {league.id === selectedLeagueId && <span className="phone-app__selected">SELECTED</span>}
          </HQAct>)}
        </div>
        {commissioner && activeTab === 'league' && onControlRoom && <HQAct
          className="phone-app__league-row" onAction={() => { setSheet(null); onControlRoom(); }}>
          Control Room</HQAct>}
        <section className="phone-app__account" role="group" aria-labelledby="phone-app-account-title">
          <h3 id="phone-app-account-title">Account</h3>
          <div className="phone-app__league-row phone-app__account-name">
            <span><small>Signed in as</small><b>{accountName || 'Signed-in account'}</b></span>
          </div>
          <HQAct className="phone-app__league-row" disabled={accountPending}
            onAction={() => { setSheet(null); onAuthenticator?.(); }}>Authenticator</HQAct>
          <HQAct className="phone-app__league-row" disabled={accountPending}
            onAction={() => { closeSheet(); onSignOut?.(); }}>Sign out</HQAct>
        </section>
      </> : <div className="phone-app__ask-content">
        <p>Answers use only readings already on Today. No question is sent or saved.</p>
        <form onSubmit={submitAsk}>
          <label htmlFor="phone-app-question">Ask about a score</label>
          <div className="phone-app__ask-form">
            <input id="phone-app-question" autoComplete="off" value={question}
              onChange={event => { setQuestion(event.target.value); setAnswer(undefined); }} />
            <HQAct submit label="Ask">Ask</HQAct>
          </div>
        </form>
        <div className="phone-app__suggestions" aria-label="Suggested questions">
          {['Am I winning?', 'Scores right now'].map(prompt => <HQAct key={prompt}
            onAction={() => askSuggestion(prompt)}>{prompt}</HQAct>)}
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
  onHandlePointer: (event: PointerEvent<HTMLDivElement>) => void;
  children: ReactNode;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const titleId = title === 'Ask' ? 'phone-app-ask-title' : 'phone-app-leagues-title';
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal?.();
    return () => element?.close?.();
  }, []);
  // The browser's modal dialog traps focus and turns Escape into a cancel event.
  return <dialog ref={dialog} className="phone-app__sheet" role="dialog" aria-modal="true"
    aria-labelledby={titleId}
    onCancel={event => { event.preventDefault(); onClose(); }}
    onPointerUp={event => { if (event.target === event.currentTarget) onClose(); }}>
    <div className="phone-app__grab" aria-hidden="true"
      onPointerDown={onHandlePointer} onPointerUp={onHandlePointer} onPointerCancel={onHandlePointer}>
      <i />
    </div>
    <header className="phone-app__sheet-head">
      <h2 id={titleId}>{title}</h2>
      <HQAct className="phone-app__close" label={`Close ${title}`} onAction={onClose}>Close</HQAct>
    </header>
    <div className="phone-app__sheet-body">{children}</div>
  </dialog>;
}

function tabLabel(tab: PhoneTab): string {
  return TABS.find(item => item.id === tab)?.label ?? 'Today';
}

function Chevron() {
  return <svg aria-hidden="true" viewBox="0 0 24 24" width="14" height="14" fill="none"
    stroke="currentColor" strokeWidth="2.2"><path d="m6 9 6 6 6-6" /></svg>;
}

function TabIcon({ tab }: { tab: PhoneTab }) {
  const paths: Record<PhoneTab, string> = {
    today: 'M4 5h16v15H4zM4 9h16M8 3v4M16 3v4M8 13h3v3H8z',
    live: 'M3 6h18v12H3zM12 6v12M3 10h3v4H3M21 10h-3v4h3',
    team: 'M9 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM3 20c0-3.3 2.7-6 6-6s6 2.7 6 6'
      + 'M16 5.5a3 3 0 0 1 0 5.5M18 14.5c2 .8 3 2.8 3 5.5',
    moves: 'M4 8h13l-3-3M20 16H7l3 3',
    league: 'M7 4h10v5a5 5 0 0 1-10 0zM7 6H4v2a3 3 0 0 0 3 3M17 6h3v2a3 3 0 0 1-3 3M12 14v4M8 20h8',
  };

  return <svg className="phone-app__tab-icon" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" strokeWidth="1.8" strokeLinecap="square" aria-hidden="true">
    <path d={paths[tab]} /></svg>;
}
