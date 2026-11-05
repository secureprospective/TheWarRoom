import { loadMarket } from '../market/load';
import { MovesMount } from './MovesMount';
import { ClockStrip } from '../clock/ClockStrip';

import { useLeagueClock } from '../clock/useLeagueClock';

import type { Placement } from '../registry';
import { FranchiseHQMount } from './FranchiseHQMount';
import { lazy, Suspense, useEffect, useState, useRef } from 'react';
import type { Snapshot } from '../data/contract';
import { selectProvider } from '../data/provider';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { DENSITIES } from '../cards/Card';
import { nodes, nodeKeys, routeFor } from './nodes';
import { NavIcon } from './NavIcon';
import '../look/look.css';
import './shell.css';
import { connectDensity } from './density';
import { useWorkScrollReset } from './workScroll';
import { LeagueChoices, LeagueContext, LeaguePicker, useLeagues, type LeagueState } from './leaguePicker';
import AppLeagueMount from './AppLeagueMount';

const ReportDrawer = lazy(() => import('../registry/ReportDrawer'));
const StatusStrip = lazy(() => import('./StatusStrip'));
const EntityInspector = lazy(() => import('./EntityInspector'));
const Contracts = lazy(() => import('../contracts/Contracts'));
const LazyEndpointIndex = lazy(() =>
  import('../registry/EndpointIndex').then((m) => ({ default: m.EndpointIndex })),
);
function EndpointIndex({ placement }: { placement: Placement }) {
  return (
    <Suspense fallback={null}>
      <LazyEndpointIndex placement={placement} />
    </Suspense>
  );
}

const CalendarPanel = lazy(() =>
  import('../clock/CalendarPanel').then((m) => ({ default: m.CalendarPanel })),
);
const LeagueRoom = lazy(() => import('../control/LeagueRoom').then((m) => ({ default: m.LeagueRoom })));
const AppSettings = lazy(() => import('./AppSettings').then((m) => ({ default: m.AppSettings })));
const CommandBar = lazy(() => import('../commands/CommandBar').then((m) => ({ default: m.CommandBar })));
const HomePulseMount = lazy(() => import('./HomePulseMount'));
const TradeDesk = lazy(() => import('./TradeDesk'));
const WarRoom = lazy(() => import('./WarRoom'));
const Market = lazy(loadMarket);
const Race = lazy(() => import('../race/Race'));
const loadInspector = () => import('./PlayerInspector');
const PlayerInspector = lazy(() => loadInspector().then((m) => ({ default: m.PlayerInspector })));

// Workspaces that read the league snapshot: before it arrives they are loading, not unwired.
const snapshotWorkspaces = ['hq/lineup-and-roster', 'hq/contracts-and-cap', 'hq/my-moves',
  'trade/trade-desk'];

export function OwnerFooter({ snapshot }: { snapshot?: Snapshot }) {
  const { franchiseId } = commands.use();
  const { myFranchise } = commands.useMFLKey();
  return <div className="me">
    <b>{snapshot?.franchises.value.find((f) => f.id === (myFranchise || franchiseId))?.name}</b>
    <span>GM</span>
  </div>;
}

export function TargetApp() {
  const leagues = useLeagues();
  return <LeagueContext.Provider value={leagues}>
    <TargetAppContent leagues={leagues} /></LeagueContext.Provider>;
}

function TargetAppContent({ leagues }: { leagues: LeagueState }) {
  const s = commands.use();
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const workspaceBody = useRef<HTMLDivElement>(null);
  useEffect(() => root.current ? connectDensity(commands, root.current) : undefined,
    [s.node, leagues.selectedId]);
  useWorkScrollReset(workspaceBody, s.scrollRevision);
  const [nativeTabs, setNativeTabs] = useState<HTMLDivElement | null>(null);
  const [franchiseFooter, setFranchiseFooter] = useState<HTMLDivElement | null>(null);
  // An app league's Control Room puts its branch leaves under its rail item (control-room-mock-v1).
  const [controlTree, setControlTree] = useState<HTMLDivElement | null>(null);
  const [snapshot, setSnapshot] = useState<Snapshot>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    let active = true;
    selectProvider()
      .snapshot()
      .then(
        (value) => {
          if (active) {
            setSnapshot(value);
            void loadInspector().catch((cause) =>
              console.error('Inspector prefetch failed:', cause),
            );
            try {
              commands.loadSnapshot(value);
            } catch (cause) {
              setError(`Personal settings failed: ${String(cause)}`);
            }
          }
        },
        (cause) => {
          if (active) setError(`Snapshot failed: ${String(cause)}`);
        },
      );
    return () => {
      active = false;
    };
  }, []);
  const node = nodes[s.node];
  const workspaces: readonly { slug: string; label: string; ring: string }[] = node.workspaces;
  const workspace = workspaces.find((w) => w.slug === s.workspace[s.node])!;
  const placement = `${node.label} › ${workspace.label}` as Placement;
  const selectedLeague = leagues.selected;
  const appLeagueView = selectedLeague?.app
    ? s.node === 'hq' ? 'hq' : s.node === 'pulse' ? 'pulse' : s.node === 'home' ? 'home'
      : s.node === 'control' ? 'control'
      : s.node === 'war' ? 'war' : undefined
    : undefined;
  if (selectedLeague && !selectedLeague.app && s.node === 'war' && snapshot)
    return <Suspense
    fallback={<div className="twr-app"><LeaguePicker leagues={leagues.leagues} selected={selectedLeague}
      onSelect={leagues.select} /><p role="status">Opening War Room…</p></div>}>
      <WarRoom snapshot={snapshot} /></Suspense>;
  if (selectedLeague && !selectedLeague.app && s.node === 'hq' && snapshot) return <>
    <FranchiseHQMount snapshot={snapshot} />
    <div className="twr-app"><Suspense fallback={undefined}><CommandBar snapshot={snapshot}
      /></Suspense></div>
  </>;
  return (
    <div className="twr-app target-frame" ref={root} data-inspector={s.inspector}
      data-native-workspace={Boolean(appLeagueView) || undefined}>
      <aside className="rail">
        <LeaguePicker
          leagues={leagues.leagues}
          selected={selectedLeague}
          onSelect={leagues.select}
          loading={leagues.loading}
          error={leagues.error}
        />
        <Act verb="commandbar.open" args={{}} variant="command">
          <span>Type a command</span>
          <kbd>/</kbd>
        </Act>
        <nav className="nav" aria-label="Workspaces">
          {nodeKeys.map((key) => (
            <Act
              key={key}
              verb="nav.open"
              args={{ node: key }}
              variant="rail"
              active={s.node === key}
            >
              <NavIcon node={key} />
              {nodes[key].label}
            </Act>
          )).flatMap((item, i) => nodeKeys[i] === 'control' && appLeagueView === 'control'
            ? [item, <div key="control-tree" className="rtree" ref={setControlTree} />] : [item])}
        </nav>
        <div className="rail-tools">
          <Act variant="button"
            verb="calendar.summon"
            args={{}}
            expanded={s.summoned !== null}
            controls="target-calendar"
          >
            Calendar
          </Act>
          <Act variant="text" verb="surface.open" args={{ place: 'Report drawer' }}>
            Reports
          </Act>
          <Act variant="text" verb="surface.open" args={{ place: 'Status strip' }}>
            Status
          </Act>
        </div>
        {selectedLeague?.app ? <div className="me" ref={setFranchiseFooter} /> :
          <OwnerFooter snapshot={snapshot} />}
      </aside>
      <section className="ws" aria-label={node.label}>
        <header className="ws-head">
          <h3>{node.label}</h3>
          <div className="wtabs" ref={appLeagueView === 'war' ? setNativeTabs : undefined}
            role={appLeagueView === 'war' ? 'tablist' : undefined}>
            {appLeagueView !== 'war' && appLeagueView !== 'control' && workspaces.map((w) => (
              <Act
                key={w.slug}
                verb="nav.open"
                args={routeFor(s.node, w.slug)!}
                variant="tab"
                active={workspace.slug === w.slug}
              >
                {w.label}
              </Act>
            ))}
          </div>
          <div className="strip">
            <span className="dens" aria-label="Density">
              {DENSITIES.map((density, index) => (
                <Act
                  key={density}
                  verb="density.set"
                  args={{ density }}
                  variant="density"
                  densityKey={density}
                  label={`${density} density`}
                >
                  {['N', 'T', 'M'][index]}
                </Act>
              ))}
            </span>
          </div>
        </header>
        <div className="workspace-status" aria-label="Status strip">
          {!selectedLeague?.app && <><Suspense fallback={null}><StatusStrip /></Suspense>
          <ClockStrip reading={clock} /></>}
        </div>
        <div className="wbody" ref={workspaceBody}>
          {!selectedLeague?.app && s.notice && (
            <p role="status" className="not-wired">
              {s.notice}
            </p>
          )}
          {s.node === 'home' && workspace.slug === 'digests' && selectedLeague && !selectedLeague.app &&
            <LeagueChoices leagues={leagues.leagues} selectedId={leagues.selectedId}
              onSelect={leagues.select} />}
          {appLeagueView && selectedLeague ? <AppLeagueMount key={selectedLeague.id} league={selectedLeague}
            view={appLeagueView} footer={franchiseFooter} tabs={nativeTabs} tree={controlTree}
            hqTab={s.node === 'hq' ? s.workspace.hq : undefined}
            pulseTab={s.node === 'pulse' ? s.workspace.pulse : undefined}
            leave={(node) => commands.dispatch('nav.open', { node: node ?? 'home' })} /> :
          leagues.loading && !selectedLeague ? <p role="status">Reading the league…</p> :
          s.node === 'control' && workspace.slug === 'app' ? (
            <Suspense fallback={null}>
              <AppSettings />
            </Suspense>
          ) : s.node === 'control' && workspace.slug === 'league' ? (
            <Suspense fallback={<p role="status">Opening League…</p>}>
              <LeagueRoom />
            </Suspense>
          ) :
          selectedLeague?.app || !selectedLeague ? (
            s.node === 'home' ? <LeagueChoices leagues={leagues.leagues} selectedId={leagues.selectedId}
              onSelect={leagues.select} /> : <section className="shell-unwired"><h4>{workspace.label}</h4>
              <p className="not-wired">Not wired · {workspace.ring}</p></section>
          ) : (s.node === 'home' && workspace.slug === 'alert-tray') ||
          (s.node === 'pulse' && workspace.slug === 'now') ? (
            <Suspense fallback={<p role="status">Opening workspace…</p>}>
              <HomePulseMount snapshot={snapshot} reading={clock} leagues={leagues.leagues}
                selectedLeague={leagues.selectedId} onLeague={leagues.select} />
            </Suspense>
          ) : s.node === 'hq' && workspace.slug === 'lineup-and-roster' && snapshot ? (
            <FranchiseHQMount snapshot={snapshot} />
          ) : s.node === 'hq' && workspace.slug === 'contracts-and-cap' && snapshot ? (
            <Suspense fallback={<p role="status">Opening Contracts and cap…</p>}>
              <Contracts snapshot={snapshot} />
            </Suspense>
          ) : s.node === 'hq' && workspace.slug === 'my-moves' && snapshot ? (
            <MovesMount snapshot={snapshot} />
          ) : s.node === 'trade' && workspace.slug === 'trade-desk' && snapshot ? (
            <Suspense fallback={<p role="status">Opening Trade desk…</p>}>
              <TradeDesk snapshot={snapshot} />
            </Suspense>
          ) : ['valuations-and-pool', 'draft-room'].includes(
              workspace.slug,
            ) ? (
            <Suspense fallback={null}>
              <Market snapshot={snapshot} />
            </Suspense>
          ) : s.node === 'pulse' && workspace.slug === 'race' ? (
            <Suspense fallback={<p role="status">Opening Race…</p>}>
              <Race />
            </Suspense>
          ) : !snapshot && snapshotWorkspaces.includes(`${s.node}/${workspace.slug}`) ? (
            <p role="status">
              {error ? `${workspace.label} unavailable · ${error}` : 'Reading the league…'}
            </p>
          ) : (
            <section className="shell-unwired">
              <h4>{workspace.label}</h4>
              <p className="not-wired">Not wired · {workspace.ring}</p>
            </section>
          )}
          {!selectedLeague?.app && <EndpointIndex placement={placement} />}
          {error && <p role="alert">{error}</p>}
        </div>
      </section>
      {!appLeagueView && <aside className="insp" aria-label="Inspector"
        aria-hidden={s.inspector === 'closed'}>
        <div className="insp-head">
          Inspector
          <span className="x">
            <Act
              verb={s.inspector === 'expanded' ? 'inspector.collapse' : 'inspector.expand'}
              args={{}} variant="button"
              expanded={s.inspector === 'expanded'}
            >
              {s.inspector === 'expanded' ? 'collapse ⇥' : 'expand ⇤'}
            </Act>
            <Act verb="inspector.close" args={{}} variant="icon">
              Close
            </Act>
          </span>
        </div>
        <div className="insp-body">
          {!selectedLeague?.app && s.entity && snapshot ? (
            <Suspense fallback={null}><EntityInspector entity={s.entity} snapshot={snapshot} /></Suspense>
          ) : !selectedLeague?.app && s.asset ? (
            <Suspense fallback={null}>
              <Market kind={s.asset} snapshot={snapshot} />
            </Suspense>
          ) : !selectedLeague?.app && snapshot && s.subject ? (
            <Suspense fallback={null}>
              <PlayerInspector snapshot={snapshot} subject={s.subject} />
            </Suspense>
          ) : (
            <p className="not-wired">
              Every player, franchise, pick, offer and segment opens here.
            </p>
          )}
          <EndpointIndex placement="Inspector" />
        </div>
      </aside>}
      {!appLeagueView && s.inspector === 'closed' && (
        <aside className="inspector-closed">
          <Act variant="button" verb="inspector.toggle" args={{}} label="Open inspector">
            Open
          </Act>
        </aside>
      )}
      {!selectedLeague?.app && <Suspense fallback={null}>
        <CommandBar snapshot={selectedLeague?.app ? undefined : snapshot} />
      </Suspense>}
      <aside
        id="target-calendar"
        data-open={s.summoned === 'calendar'}
        aria-hidden={s.summoned !== 'calendar'}
        className="shell-overlay calendar-panel"
        aria-label="Calendar"
      >
        <header>
          Calendar
          <Act variant="button" verb="calendar.summon" args={{}}>
            Close
          </Act>
        </header>
        <Suspense fallback={null}>
          {!selectedLeague?.app && <CalendarPanel reading={clock} />}
        </Suspense>
        <EndpointIndex placement="Calendar" />
      </aside>
      {!selectedLeague?.app &&
        (s.endpointSurface === 'Report drawer' || s.endpointSurface === 'Status strip') && (
        <aside className="shell-overlay registry-panel" data-open="true" aria-label={s.endpointSurface}>
          <header>
            {s.endpointSurface}
            <Act variant="button" verb="surface.close" args={{}}>
              Close
            </Act>
          </header>
          {s.endpointSurface === 'Status strip' && <Suspense fallback={null}>
            <StatusStrip /><ClockStrip reading={clock} /></Suspense>}
          {s.endpointSurface === 'Report drawer' ? <Suspense fallback={<p>Reading reports…</p>}>
            <ReportDrawer />
          </Suspense> : <EndpointIndex placement={s.endpointSurface} />}
        </aside>
      )}
    </div>
  );
}
