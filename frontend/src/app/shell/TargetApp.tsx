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

const ReportDrawer = lazy(() => import('../registry/ReportDrawer'));
const StatusStrip = lazy(() => import('./StatusStrip'));
const EntityInspector = lazy(() => import('./EntityInspector'));
const DataSources = lazy(() => import('../datasources/DataSources'));
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
const Market = lazy(loadMarket);
const Research = lazy(() => import('../research/Research'));
const Race = lazy(() => import('../race/Race'));
const Archive = lazy(() => import('../archive/Archive'));
const Comms = lazy(() => import('../comms/Comms'));
const loadInspector = () => import('./PlayerInspector');
const PlayerInspector = lazy(() => loadInspector().then((m) => ({ default: m.PlayerInspector })));

// Workspaces that read the league snapshot: before it arrives they are loading, not unwired.
const snapshotWorkspaces = ['hq/lineup-and-roster', 'hq/contracts-and-cap', 'hq/my-moves',
  'trade/trade-desk'];

export function TargetApp() {
  const s = commands.use();
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const workspaceBody = useRef<HTMLDivElement>(null);
  useEffect(() => connectDensity(commands, root.current!), []);
  useWorkScrollReset(workspaceBody, s.scrollRevision);
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
  return (
    <div className="twr-app target-frame" ref={root} data-inspector={s.inspector} data-comms={s.comms}>
      <aside className="rail">
        <div className="league">
          <div className="crest">
            <i>LN</i>
            <div>
              <b>Legacy NFL</b>
              <small>
                {snapshot?.league.provenance.freshness.state !== 'fail' && snapshot
                  ? `${snapshot.league.value.franchiseCount} teams`
                  : 'Teams unavailable'}{' '}
                · dynasty IDP
              </small>
            </div>
          </div>
        </div>
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
          ))}
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
          <Act variant="button" verb="harness.open" args={{}}>
            Harness
          </Act>
        </div>
        <div className="me">
          <b>{snapshot?.franchises.value.find((f) => f.id === s.franchiseId)?.name}</b>
          <span>GM</span>
        </div>
      </aside>
      <section className="ws" aria-label={node.label}>
        <header className="ws-head">
          <h3>{node.label}</h3>
          <div className="wtabs">
            {workspaces.map((w) => (
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
          <Suspense fallback={null}><StatusStrip /></Suspense>
          <ClockStrip reading={clock} />
        </div>
        <div className="wbody" ref={workspaceBody}>
          {s.notice && (
            <p role="status" className="not-wired">
              {s.notice}
            </p>
          )}
          {(s.node === 'home' && ['seasonal-card', 'alert-tray'].includes(workspace.slug)) ||
          (s.node === 'pulse' && workspace.slug === 'now') ? (
            <Suspense fallback={<p role="status">Opening workspace…</p>}>
              <HomePulseMount snapshot={snapshot} reading={clock} />
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
          ) : ['market', 'counterparties', 'offers-history', 'valuations-and-pool', 'draft-room'].includes(
              workspace.slug,
            ) ? (
            <Suspense fallback={null}>
              <Market snapshot={snapshot} />
            </Suspense>
          ) : s.node === 'war' && workspace.slug === 'research' ? (
            <Suspense fallback={<p role="status">Opening Research…</p>}>
              <Research />
            </Suspense>
          ) : s.node === 'pulse' && workspace.slug === 'race' ? (
            <Suspense fallback={<p role="status">Opening Race…</p>}>
              <Race />
            </Suspense>
          ) : s.node === 'pulse' && workspace.slug === 'archive' ? (
            <Suspense fallback={<p role="status">Opening Archive…</p>}>
              <Archive />
            </Suspense>
          ) : s.node === 'home' && workspace.slug === 'digests' ? (
            <Suspense fallback={<p role="status">Opening Digests…</p>}>
              <Comms />
              <Research newsOnly />
            </Suspense>
          ) : s.node === 'control' && workspace.slug === 'data-and-sources' ? (
            <Suspense fallback={<p role="status">Opening Data and sources…</p>}>
              <DataSources />
            </Suspense>
          ) : s.node === 'control' && workspace.slug === 'app' ? (
            <Suspense fallback={null}>
              <AppSettings />
            </Suspense>
          ) : s.node === 'control' && workspace.slug === 'league' ? (
            <Suspense fallback={<p role="status">Opening League…</p>}>
              <LeagueRoom />
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
          <EndpointIndex placement={placement} />
          {error && <p role="alert">{error}</p>}
        </div>
      </section>
      <aside className="insp" aria-label="Inspector" aria-hidden={s.inspector === 'closed'}>
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
          {s.entity && snapshot ? (
            <Suspense fallback={null}><EntityInspector entity={s.entity} snapshot={snapshot} /></Suspense>
          ) : s.asset ? (
            <Suspense fallback={null}>
              <Market kind={s.asset} snapshot={snapshot} />
            </Suspense>
          ) : snapshot && s.subject ? (
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
      </aside>
      {s.inspector === 'closed' && (
        <aside className="inspector-closed">
          <Act variant="button" verb="inspector.toggle" args={{}} label="Open inspector">
            Open
          </Act>
        </aside>
      )}
      <aside className="comms">
        <Act
          verb="comms.toggle"
          args={{}}
          variant="icon"
          expanded={s.comms}
          controls="target-comms"
        >
          C
        </Act>
        <span className="vt">Comms · league feed</span>
      </aside>
      <Suspense fallback={null}>
        <CommandBar snapshot={snapshot} />
      </Suspense>
      <aside
        id="target-comms"
        data-open={s.comms}
        aria-hidden={!s.comms}
        className="shell-overlay comms-panel"
        aria-label="Comms"
      >
        <header>
          Comms
          <Act variant="button" verb="comms.toggle" args={{}}>
            Close
          </Act>
        </header>
        {s.comms && <Suspense fallback={<p role="status">Reading league communications…</p>}>
          <Comms panel />
        </Suspense>}
        <EndpointIndex placement="Comms" />
      </aside>
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
          <CalendarPanel reading={clock} />
        </Suspense>
        <EndpointIndex placement="Calendar" />
      </aside>
      {(s.endpointSurface === 'Report drawer' || s.endpointSurface === 'Status strip') && (
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
