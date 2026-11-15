import { lazy, Suspense, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { readAppLeaguePeople, type LeagueChoice } from './leaguePicker';
import { localWorkflow } from './localWorkflow';
import { browserStorage, FRANCHISE_KEY } from './franchise';
import type { People } from '../commissioner/people';
import { commands } from '../commands/registry';
import type { HQTab } from './HQShell';
import type { PulseTab } from './NativeLeaguePulse';
import type { Node } from './nodes';
import { readRailNow, usePulseRead, type RailNow } from './pulseReads';
import { usePulseLive } from './hqLive';

const loaders = {
  hq: () => import('./NativeFranchiseHQ'), war: () => import('./WarRoomNative'),
  pulse: () => import('./NativeLeaguePulse'), home: () => import('./HomeBrief'),
  control: () => import('../control/room/ControlRoom'),
};
const NativeFranchiseHQ = lazy(loaders.hq);
const WarRoomNative = lazy(loaders.war);
const NativeLeaguePulse = lazy(loaders.pulse);
const HomeBrief = lazy(loaders.home);
const ControlRoom = lazy(loaders.control);
const views = ['home', 'hq', 'pulse', 'war', 'control'] as const;

/** RailWeek reads what the rail says on every screen: the week being played, how many NFL games
 *  are live, and where my matchup stands. It draws nothing. */
function RailWeek({ league, mine, publish }: { league: string; mine: string;
  publish: (rail: { now?: RailNow; live?: number }) => void }) {
  const { reading } = usePulseRead<RailNow>(JSON.stringify(['rail', league, mine]),
    () => readRailNow(league, mine, localWorkflow));
  const live = usePulseLive('').view?.games.filter(g => g.status === 'in').length;
  useEffect(() => { publish({ now: reading, live }); }, [reading, live, publish]);
  useEffect(() => () => publish({}), [publish]);
  return null;
}

export default function AppLeagueMount({ league, view, leave, footer, hqTab, pulseTab, tabs, tree, onRail }: {
  league: LeagueChoice;
  view: 'hq' | 'war' | 'pulse' | 'home' | 'control';
  leave: (node?: Node) => void;
  footer?: HTMLElement | null;
  hqTab?: HQTab; pulseTab?: PulseTab; tabs?: HTMLElement | null;
  /** Control Room's rail slot under its nav item. */
  tree?: HTMLElement | null;
  /** Tells the shell rail the week, the live count and my matchup's score. */
  onRail?: (rail: { now?: RailNow; live?: number }) => void;
}) {
  const [people, setPeople] = useState<People>();
  const [mine, setMine] = useState('');
  const [error, setError] = useState<string>();
  const key = `${FRANCHISE_KEY}:${league.id}`;
  const scope = `${league.id}:${mine}`;
  // Keep one instance per visited rail screen, only for the current league and franchise.
  const [opened, setOpened] = useState({ scope, views: [view] });
  if (opened.scope !== scope) setOpened({ scope, views: [view] });
  else if (!opened.views.includes(view)) setOpened({ scope, views: [...opened.views, view] });
  useEffect(() => {
    let active = true;
    void loaders[view]().catch(cause => {
      if (active) setError(`Could not load the workspace: ${String(cause)}`);
    });
    return () => { active = false; };
  }, [view]);
  useEffect(() => {
    let active = true;
    setPeople(undefined);
    setMine('');
    setError(undefined);
    readAppLeaguePeople(league.id).then(
      (reading) => {
        if (!active) return;
        const saved = browserStorage()?.getItem(key);
        const selected = reading.seats.some(s => s.franchise === saved) ? saved! : reading.yourFranchise;
        setMine(reading.seats.some(s => s.franchise === selected) ? selected : '');
        setPeople(reading);
      },
      (cause) => {
        if (active) setError(cause instanceof Error ? cause.message : String(cause));
      },
    );
    return () => { active = false; };
  }, [league.id, key]);
  // The Control Room runs the league, not one franchise: it needs neither a franchise nor the seat list.
  if (view !== 'control' && error) return <p role="alert">Could not open {league.name}: {error}</p>;
  if (view !== 'control' && !people) return <p role="status">Opening {league.name}…</p>;
  const seats = people?.seats ?? [];
  const chooser = <label>View a franchise
    <select value={mine} onChange={event => {
      const id = event.target.value;
      if (!seats.some(s => s.franchise === id)) return;
      try {
        browserStorage()?.setItem(key, id);
        setMine(id);
      } catch (cause) {
        setError(`Franchise selection failed: ${String(cause)}`);
      }
    }}>
      <option value="">Choose a franchise</option>
      {seats.map(s => <option key={s.franchise} value={s.franchise}>{s.name}</option>)}
    </select>
  </label>;
  return <>
    {onRail && <RailWeek league={league.id} mine={mine} publish={onRail} />}
    {view !== 'control' && (footer ? createPortal(chooser, footer) : chooser)}
    {view !== 'control' && !mine &&
      <p role="status">Choose a franchise to view its roster, contracts and War Room.</p>}
    {views.filter(screen => opened.views.includes(screen) && (screen === 'control' || mine)).map(screen =>
      <div key={`${scope}:${screen}`} hidden={view !== screen} style={{ height: '100%' }}>
        <Suspense fallback={<p role="status">Opening {league.name}…</p>}>
          {screen === 'control' ? <ControlRoom league={league.id}
            tree={view === screen ? tree : null} footer={view === screen ? footer : null} /> :
          screen === 'hq' ? <NativeFranchiseHQ embedded
            league={league.id} mine={mine} name={league.name} port={localWorkflow} leave={leave}
            canManage={people?.yourFranchise === mine} tab={hqTab}
            navigate={tab => commands.dispatch('nav.open', { node: 'hq', workspace: tab })} /> :
          screen === 'home' ? <HomeBrief league={league.id} mine={mine}
            port={localWorkflow} owner={people?.yourFranchise === mine} /> :
          screen === 'pulse' ? <NativeLeaguePulse embedded league={league.id}
            mine={mine} port={localWorkflow} leave={leave} tab={pulseTab} /> :
          <WarRoomNative embedded league={league.id} mine={mine}
            port={localWorkflow} leave={leave} tabs={view === screen ? tabs : null}
            owner={people?.yourFranchise === mine} />}
        </Suspense>
      </div>)}
  </>;
}
