import { Act } from '../../commands/Act';
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { leagueState } from '../../commissioner/proxy';
import { commands } from '../../commands/registry';
import { draftRoom } from '../../draftroom/state';
import { nativeBindings } from '../../commissioner/api';
import { getParams, readShellStatus } from '../api';
import { readRoom, type RoomData, type RoomPort } from './roomReads';
import { routeCommand, unbuiltWords, type Route } from './route';
import { EditorTray, type TrayContent } from './EditorTray';
import { runSearch, type Hit } from './search';
import { RoomContext, type Room, type Tab, type Card, tabs } from './room';
import { JobsView } from './JobsView';
import { SituationsView } from './SituationsView';
import { RulesView } from './RulesView';
import { SeasonView } from './SeasonView';
import { FixView } from './FixView';
import { PicksView } from './PicksView';
import { PeopleView } from './PeopleView';
import { LeagueView } from './LeagueView';
import { ServerView } from './ServerView';
import { useRoomSurface } from './useRoomSurface';
import { openFix, openPickAct, openSituation, openStep } from './openers';
import './controlRoom.css';

const views: Record<Tab, () => ReactNode> = {
  jobs: () => <JobsView />,
  sit: () => <SituationsView />,
  rules: () => <RulesView />,
  season: () => <SeasonView />,
  fix: () => <FixView />,
  picks: () => <PicksView />,
  people: () => <PeopleView />,
  league: () => <LeagueView />,
  server: () => <ServerView />,
};

/** The mock's rail leaf words (sentence case), where they differ from the branch tab. */
const leafLabel: Partial<Record<Tab, string>> = {
  jobs: 'Jobs',
  sit: 'Situations',
  rules: 'Rules',
  season: 'Season desk',
  fix: 'Fix',
  picks: 'Picks',
  people: 'People',
  league: 'League',
  server: 'Server',
};

const desktopPort = (): RoomPort => ({
  ...nativeBindings(),
  TargetShellStatus: readShellStatus,
  GetParams: getParams,
});

/** ControlRoom is Control Room for an app league, laid out as docs/ui/mocks/control-room-mock-v1.html:
 *  nine branches, a find box, the editor tray with the command line, the inspector and the centre card.
 *  Every value comes from an existing read; every button runs an existing command through its form. */
export default function ControlRoom({
  league,
  port = desktopPort,
  initial,
  start = 'jobs',
  tree,
  footer,
}: {
  league: string;
  port?: () => RoomPort;
  initial?: RoomData;
  start?: Tab;
  /** The shell's slot under the Control Room rail item, and the rail footer (the mock's tree and
    "Signed in"). */
  tree?: HTMLElement | null;
  footer?: HTMLElement | null;
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
  const [commandLine, setCommandLine] = useState('');
  const [hits, setHits] = useState<Hit[] | null>(null);
  const pending = useRef<(() => void) | null>(null);
  const find = useRef<HTMLInputElement>(null);
  const toastTimer = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    void leagueState.choose(league);
  }, [league]);
  const revision = state.ledger?.revision ?? 0;
  useEffect(() => {
    if (initial) return;
    let active = true;
    let reader: RoomPort;
    try {
      reader = port();
    } catch (cause) {
      setData({ errors: { directory: cause instanceof Error ? cause.message : String(cause) } });
      return;
    }
    const stop = new AbortController();
    void readRoom(league, reader, stop.signal).then((d) => {
      if (active) setData(d);
    });
    return () => {
      active = false;
      stop.abort();
    };
  }, [league, port, initial, revision]);

  const toast = useCallback((words: string) => {
    setToast(words);
    clearTimeout(toastTimer.current);
    toastTimer.current = setTimeout(() => setToast(''), 2600);
  }, []);
  const go = useCallback((next: Tab, then?: () => void) => {
    setTab(next);
    setTray(null);
    setCard(null);
    setInspector(null);
    pending.current = then ?? null;
  }, []);
  useEffect(() => {
    const then = pending.current;
    pending.current = null;
    then?.();
  }, [tab]);
  const open = useCallback(
    (route: Route) => {
      if (route.kind === 'unbuilt') {
        toast(unbuiltWords(route.command));
        return;
      }
      if (route.kind === 'draft') {
        draftRoom.choose(false);
        commands.dispatch('nav.open', { node: 'war', workspace: 'draft-room' });
        return;
      }
      setCard(null);
      setTray({ kind: 'route', route });
    },
    [toast],
  );
  const room: Room = useMemo(
    () => ({
      league,
      data,
      state,
      tab,
      go,
      toast,
      inspect: setInspector,
      card: setCard,
      run: (command: string, initial?: Record<string, string>) => {
        setOut(`› ${command}`);
        // A rule's name opens the rule-change form: settings change only through the one rule-change flow.
        if (state.rules?.sections.some((s) => s.rows.some((r) => r.field === command))) {
          setCard(null);
          setTray({ kind: 'rule', field: command });
          return;
        }
        const route = routeCommand(command);
        open(route.kind === 'act' && initial ? { ...route, initial } : route);
      },
      rule: (field: string) => {
        setCard(null);
        setTray({ kind: 'rule', field });
      },
      undo: (entry: string) => {
        setCard(null);
        setTray({ kind: 'undo', entry });
      },
      closeTray: () => setTray(null),
    }),
    [league, data, state, tab, go, toast, open],
  );

  const choose = (hit: Hit) => {
    setHits(null);
    setQuery('');
    hit.open(room);
  };
  const search = (q: string) => {
    setQuery(q);
    setHits(q.trim() ? runSearch(room, q) : null);
  };
  const command = (line: string) => {
    const [name = '', ...rest] = line.trim().split(/\s+/);
    if (!name) return;
    if (name === 'help') {
      setOut('Try: a rule name (cap.amount) · waivers.process · find owner left');
      return;
    }
    if (name === 'find') {
      search(rest.join(' '));
      find.current?.focus();
      return;
    }
    const leaf = tabs.find(([k, l]) => k === name || l.toLowerCase() === name.toLowerCase());
    if (leaf) {
      go(leaf[0]);
      setOut(`› opened ${leaf[1]}`);
      return;
    }
    const rule = state.rules?.sections.flatMap((s) => s.rows).find((r) => r.field === name);
    if (rule) {
      go('rules', () => room.rule(rule.field));
      setOut(`› opened ${rule.label}`);
      return;
    }
    const route = routeCommand(name);
    if (route.kind === 'unbuilt') {
      setOut('Not a command. Type help.');
      return;
    }
    room.run(name);
  };
  useRoomSurface('control-room', ({ action, value }) => {
    if (action === 'view') {
      const tab = tabs.find(([key]) => key === value);
      if (tab) go(tab[0]);
    } else if (action === 'close') {
      if (value === 'tray') room.closeTray();
      else setCard(null);
    } else if (action === 'clear') room.undo(value);
    else if (action === 'check') room.rule(value);
    else if (action === 'pick') openStep(room, value);
    else if (action === 'find' && hits?.[Number(value)]) choose(hits[Number(value)]!);
    else if (action === 'tray') {
      command(commandLine);
      setCommandLine('');
    } else if (action === 'open') {
      if (value.startsWith('fix:')) openFix(room, Number(value.slice(4)));
      else if (value.startsWith('pick:')) openPickAct(room, Number(value.slice(5)));
      else if (value.startsWith('situation:')) openSituation(room, Number(value.slice(10)));
      else room.run(value);
    }
  });
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      const typing = /INPUT|TEXTAREA|SELECT/.test(
        (document.activeElement as HTMLElement | null)?.tagName ?? '',
      );
      if (e.key === '/' && !typing) {
        e.preventDefault();
        find.current?.focus();
      }
      if (e.key === 'Escape') {
        if (document.activeElement === find.current) {
          setHits(null);
          find.current?.blur();
        }
        setCard(null);
        setTray(null);
      }
    };
    document.addEventListener('keydown', key);
    return () => document.removeEventListener('keydown', key);
  }, []);
  const live = state.jobs?.jobs.filter((j) => j.state === 'live').length ?? 0;

  const leaves = tabs.map(([k, l], i) => (
    <Act
      key={k}
      className={`leaf${k === tab ? ' sel' : ''}`}
      current={k === tab ? 'page' : undefined}
      verb="commissioner.surface"
      variant="surface"
      label={l}
      args={{ screen: 'control-room', action: 'view', value: k }}
      attributes={{ 'data-tab': String(k) }}
    >
      <span className="tr">{i === tabs.length - 1 ? '└' : '├'}</span>
      {leafLabel[k] ?? l}
    </Act>
  ));
  return (
    <RoomContext.Provider value={room}>
      {tree && createPortal(<div className="crx-tree">{leaves}</div>, tree)}
      {footer &&
        createPortal(
          <span className="crx-who">
            Signed in: {(state.people?.yourRole ?? '').replace(/^./, (c) => c.toUpperCase()) || '—'}
            {state.people?.yourRole?.toLowerCase() === 'visitor' && <>{' · '}
              <Act variant="button" verb="commissioner.surface"
                args={{ screen: 'control-room', action: 'open', value: 'backup.now' }}>
                Sign in to claim your seat
              </Act>
            </>}
          </span>,
          footer,
        )}
      <div className="crx" data-testid="control-room">
        <main className="work">
          <div className="ws-tabs" role="tablist" aria-label="Control Room">
            {tabs.map(([k, l]) => (
              <Act
                key={k}
                role="tab"
                selected={k === tab}
                verb="commissioner.surface"
                variant="surface"
                label={l}
                args={{ screen: 'control-room', action: 'view', value: k }}
              >
                {l}
                {k === 'jobs' && live > 0 && (
                  <>
                    {' '}
                    <span className="cr-st live">{live}</span>
                  </>
                )}
              </Act>
            ))}
          </div>
          <form
            className="cr-find"
            onSubmit={(e) => {
              e.preventDefault();
              if (hits?.[0])
                commands.dispatch('commissioner.surface', {
                  screen: 'control-room',
                  action: 'find',
                  value: '0',
                });
            }}
          >
            <input
              ref={find}
              value={query}
              aria-label="Find anything"
              placeholder="Find anything: a rule, a job, a fix, a team, a situation ... (press /)"
              autoComplete="off"
              onChange={(e) => search(e.target.value)}
            />
            <span className="hint">searches names, questions and current values</span>
            <div className={`cr-res${hits ? ' open' : ''}`}>
              {hits &&
                (hits.length ? (
                  hits.map((h, i) => (
                    <Act
                      key={`${h.type}:${h.label}:${i}`}
                      className={i === 0 ? 'k' : ''}
                      verb="commissioner.surface"
                      variant="surface"
                      label={h.label}
                      args={{ screen: 'control-room', action: 'find', value: String(i) }}
                    >
                      <span className="ty">{h.type}</span>
                      <span className="lb">
                        <b>{h.label}</b>
                        <span>{h.words}</span>
                      </span>
                    </Act>
                  ))
                ) : (
                  <Act
                    verb="commissioner.surface"
                    variant="surface"
                    label={'Nothing matches'}
                    args={{ screen: 'control-room', action: 'find', value: '' }}
                  >
                    <span className="ty">—</span>
                    <span className="lb">
                      <b>Nothing matches</b>
                      <span>Try fewer words, or a situation: “owner left”, “game cancelled”.</span>
                    </span>
                  </Act>
                ))}
            </div>
          </form>
          {state.error && (
            <p role="alert" className="cr-note">
              {state.error}
            </p>
          )}
          <section className="cr-main on" aria-label={tabs.find((t) => t[0] === tab)![1]}>
            {views[tab]()}
          </section>
          <div className="cr-tray">
            {tray && <EditorTray content={tray} />}
            <form
              className="cr-cmd"
              onSubmit={(e) => {
                e.preventDefault();
                commands.dispatch('commissioner.surface', {
                  screen: 'control-room',
                  action: 'tray',
                  value: '',
                });
              }}
            >
              <span className="pr">COMMAND ›</span>
              <input
                aria-label="Command"
                placeholder="Type a command, or help"
                value={commandLine}
                onChange={(e) => setCommandLine(e.target.value)}
              />
              <span className="out" role="status">
                {out}
              </span>
            </form>
          </div>
          <div className={`cr-toast${toastWords ? ' on' : ''}`} role="status">
            {toastWords}
          </div>
          <Act
            className={`cr-scrim${card ? ' open' : ''}`}
            verb="commissioner.surface"
            variant="surface"
            label="Close card"
            args={{ screen: 'control-room', action: 'close', value: 'card' }}
          >
            <span aria-hidden="true" />
          </Act>
          {card && (
            <div
              className={`cr-card open${card.wide ? ' wide' : ''}`}
              role="dialog"
              aria-modal="true"
              aria-label={card.title}
            >
              <div className="cr-ch">
                <div>
                  <h2>{card.title}</h2>
                  <div className="sub">{card.sub}</div>
                </div>
                <Act
                  className="x"
                  verb="commissioner.surface"
                  variant="surface"
                  label={'Close card'}
                  args={{ screen: 'control-room', action: 'close', value: 'card' }}
                >
                  ESC ✕
                </Act>
              </div>
              <div className="cr-cb">{card.body}</div>
            </div>
          )}
        </main>
        <aside className="insp" aria-label="Inspector">
          <div className="cr-ins">{inspector}</div>
        </aside>
      </div>
    </RoomContext.Provider>
  );
}
