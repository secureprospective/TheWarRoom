import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { leagueState } from '../../commissioner/proxy';
import { commands } from '../../commands/registry';
import { draftRoom } from '../../draftroom/state';
import { nativeBindings } from '../../commissioner/api';
import { readShellStatus } from '../api';
import { readRoom, type RoomData, type RoomPort } from './roomReads';
import { routeCommand, unbuiltWords, type Route } from './route';
import { EditorTray, type TrayContent } from './EditorTray';
import { runSearch, type Hit } from './search';
import { NotYet, RoomContext, type Room, type Tab, type Card, tabs } from './room';
import { JobsView } from './JobsView';
import { SituationsView } from './SituationsView';
import { RulesView } from './RulesView';
import { SeasonView } from './SeasonView';
import './controlRoom.css';

const views: Record<Tab, () => ReactNode> = {
  jobs: () => <JobsView />, sit: () => <SituationsView />, rules: () => <RulesView />,
  season: () => <SeasonView />, fix: () => <NotYet what="this branch is wired in a later commit" />, picks: () => <NotYet what="this branch is wired in a later commit" />,
  people: () => <NotYet what="this branch is wired in a later commit" />, league: () => <NotYet what="this branch is wired in a later commit" />, server: () => <NotYet what="this branch is wired in a later commit" />,
};

const desktopPort = (): RoomPort => ({ ...nativeBindings(), TargetShellStatus: readShellStatus });

/** ControlRoom is Control Room for an app league, laid out as docs/ui/mocks/control-room-mock-v1.html:
 *  nine branches, a find box, the editor tray with the command line, the inspector and the centre card.
 *  Every value comes from an existing read; every button runs an existing command through its form. */
export default function ControlRoom({ league, port = desktopPort, initial, start = 'jobs' }: {
  league: string; port?: () => RoomPort; initial?: RoomData; start?: Tab;
}) {
  const state = leagueState.use();
  const [data, setData] = useState<RoomData>(initial ?? { errors: {} });
  const [tab, setTab] = useState<Tab>(start);
  const [inspector, setInspector] = useState<ReactNode>(null);
  const [card, setCard] = useState<Card | null>(null);
  const [tray, setTray] = useState<TrayContent | null>(null);
  const [toastWords, setToast] = useState('');
  const [out, setOut] = useState('Every screen action is also a command.');
  const [query, setQuery] = useState('');
  const [hits, setHits] = useState<Hit[] | null>(null);
  const pending = useRef<(() => void) | null>(null);
  const find = useRef<HTMLInputElement>(null);
  const toastTimer = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => { void leagueState.choose(league); }, [league]);
  const revision = state.ledger?.revision ?? 0;
  useEffect(() => {
    if (initial) return;
    let active = true;
    let reader: RoomPort;
    try { reader = port(); } catch (cause) {
      setData({ errors: { directory: cause instanceof Error ? cause.message : String(cause) } });
      return;
    }
    void readRoom(league, reader).then(d => { if (active) setData(d); });
    return () => { active = false; };
  }, [league, port, initial, revision]);

  const toast = useCallback((words: string) => {
    setToast(words);
    clearTimeout(toastTimer.current);
    toastTimer.current = setTimeout(() => setToast(''), 2600);
  }, []);
  const go = useCallback((next: Tab, then?: () => void) => {
    setTab(next); setTray(null); setCard(null); setInspector(null);
    pending.current = then ?? null;
  }, []);
  useEffect(() => {
    const then = pending.current;
    pending.current = null;
    then?.();
  }, [tab]);
  const open = useCallback((route: Route) => {
    if (route.kind === 'unbuilt') { toast(unbuiltWords(route.command)); return; }
    if (route.kind === 'draft') {
      draftRoom.choose(false);
      commands.dispatch('nav.open', { node: 'trade', workspace: 'draft-room' });
      return;
    }
    setCard(null);
    setTray({ kind: 'route', route });
  }, [toast]);
  const room: Room = useMemo(() => ({
    league, data, state, tab, go, toast,
    inspect: setInspector,
    card: setCard,
    run: (command: string) => {
      setOut(`› ${command}`);
      // A rule's name opens the rule-change form: settings change only through the one rule-change flow.
      if (state.rules?.sections.some(s => s.rows.some(r => r.field === command))) {
        setCard(null); setTray({ kind: 'rule', field: command }); return;
      }
      open(routeCommand(command));
    },
    rule: (field: string) => { setCard(null); setTray({ kind: 'rule', field }); },
    undo: (entry: string) => { setCard(null); setTray({ kind: 'undo', entry }); },
    closeTray: () => setTray(null),
  }), [league, data, state, tab, go, toast, open]);

  const choose = (hit: Hit) => { setHits(null); setQuery(''); hit.open(room); };
  const search = (q: string) => { setQuery(q); setHits(q.trim() ? runSearch(room, q) : null); };
  const command = (line: string) => {
    const [name = '', ...rest] = line.trim().split(/\s+/);
    if (!name) return;
    if (name === 'help') { setOut('Try: a rule name (cap.amount) · waivers.process · find owner left'); return; }
    if (name === 'find') { search(rest.join(' ')); find.current?.focus(); return; }
    const leaf = tabs.find(([k, l]) => k === name || l.toLowerCase() === name.toLowerCase());
    if (leaf) { go(leaf[0]); setOut(`› opened ${leaf[1]}`); return; }
    const rule = state.rules?.sections.flatMap(s => s.rows).find(r => r.field === name);
    if (rule) { go('rules', () => room.rule(rule.field)); setOut(`› opened ${rule.label}`); return; }
    const route = routeCommand(name);
    if (route.kind === 'unbuilt') { setOut('Not a command. Type help.'); return; }
    room.run(name);
  };
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      const typing = /INPUT|TEXTAREA|SELECT/.test((document.activeElement as HTMLElement | null)?.tagName ?? '');
      if (e.key === '/' && !typing) { e.preventDefault(); find.current?.focus(); }
      if (e.key === 'Escape') { setCard(null); setTray(null); }
    };
    document.addEventListener('keydown', key);
    return () => document.removeEventListener('keydown', key);
  }, []);
  const live = state.jobs?.jobs.filter(j => j.state === 'live').length ?? 0;

  return (
    <RoomContext.Provider value={room}>
      <div className="crx" data-testid="control-room">
        <main className="work">
          <div className="ws-tabs" role="tablist" aria-label="Control Room">
            {tabs.map(([k, l]) => <button key={k} role="tab" aria-selected={k === tab} onClick={() => go(k)}>
              {l}{k === 'jobs' && live > 0 && <> <span className="cr-st live">{live}</span></>}
            </button>)}
          </div>
          <div className="cr-find">
            <input ref={find} value={query} aria-label="Find anything"
              placeholder="Find anything: a rule, a job, a fix, a team, a situation ... (press /)"
              autoComplete="off" onChange={e => search(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter' && hits?.[0]) choose(hits[0]);
                if (e.key === 'Escape') { setHits(null); e.currentTarget.blur(); }
              }} />
            <span className="hint">searches names, questions and current values</span>
            <div className={`cr-res${hits ? ' open' : ''}`}>
              {hits && (hits.length ? hits.map((h, i) => <button key={`${h.type}:${h.label}:${i}`}
                className={i === 0 ? 'k' : ''} onClick={() => choose(h)}>
                <span className="ty">{h.type}</span><span className="lb"><b>{h.label}</b><span>{h.words}</span></span>
              </button>) : <button><span className="ty">—</span><span className="lb"><b>Nothing matches</b>
                <span>Try fewer words, or a situation: “owner left”, “game cancelled”.</span></span></button>)}
            </div>
          </div>
          {state.error && <p role="alert" className="cr-note">{state.error}</p>}
          <section className="cr-main on" aria-label={tabs.find(t => t[0] === tab)![1]}>{views[tab]()}</section>
          <div className="cr-tray">
            {tray && <EditorTray content={tray} />}
            <div className="cr-cmd">
              <span className="pr">COMMAND ›</span>
              <input aria-label="Command" placeholder="type a command, e.g.  cap.amount 130M   ·   waivers.process   ·   find owner left"
                onKeyDown={e => { if (e.key === 'Enter') { command(e.currentTarget.value); e.currentTarget.value = ''; } }} />
              <span className="out" role="status">{out}</span>
            </div>
          </div>
          <div className={`cr-toast${toastWords ? ' on' : ''}`} role="status">{toastWords}</div>
          <div className={`cr-scrim${card ? ' open' : ''}`} onClick={() => setCard(null)} />
          {card && <div className={`cr-card open${card.wide ? ' wide' : ''}`} role="dialog" aria-modal="true"
            aria-label={card.title}>
            <div className="cr-ch"><div><h2>{card.title}</h2><div className="sub">{card.sub}</div></div>
              <button className="x" onClick={() => setCard(null)}>ESC ✕</button></div>
            <div className="cr-cb">{card.body}</div>
          </div>}
        </main>
        <aside className="insp" aria-label="Inspector"><div className="cr-ins">{inspector}</div></aside>
      </div>
    </RoomContext.Provider>
  );
}
