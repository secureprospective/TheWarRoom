import { CommandBar } from '../commands/CommandBar';
import { FranchiseHQ } from './FranchiseHQ';
import { PlayerInspector } from './PlayerInspector';
import { AppSettings } from './AppSettings';
import { useEffect, useState } from 'react';
import type { Snapshot } from '../data/contract';
import { selectProvider } from '../data/provider';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { SignalChip } from '../look/Slots';
import { DENSITIES } from '../cards/Card';
import { nodes, nodeKeys, routeFor } from './nodes';
import { NavIcon } from './NavIcon';
import { snapshotSummary } from './snapshotSummary';
import '../look/look.css';
import './shell.css';

export function TargetApp() {
  const s = commands.use();
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
  const summary = snapshot && snapshotSummary(snapshot);
  const node = nodes[s.node];
  const workspaces: readonly { slug: string; label: string; ring: string }[] =
    node.workspaces;
  const workspace = workspaces.find((w) => w.slug === s.workspace[s.node])!;
  return (
    <div
      className="twr-app target-frame"
      data-density={s.density}
      data-inspector={s.inspector}
    >
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
          <Act
            verb="calendar.summon"
            args={{}}
            expanded={s.summoned !== null}
            controls="target-calendar"
          >
            Calendar
          </Act>
          <Act verb="harness.open" args={{}}>
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
            {summary ? (
              <SignalChip {...summary.provenance} />
            ) : (
              <span>{error ?? 'Snapshot loading…'}</span>
            )}
            <span>league clock · not wired</span>
            <span className="dens" aria-label="Density">
              {DENSITIES.map((density, index) => (
                <Act
                  key={density}
                  verb="density.set"
                  args={{ density }}
                  variant="density"
                  active={s.density === density}
                  label={`${density} density`}
                >
                  {['N', 'T', 'M'][index]}
                </Act>
              ))}
            </span>
          </div>
        </header>
        <div className="wbody">
          {s.notice && (
            <p role="status" className="not-wired">
              {s.notice}
            </p>
          )}
          {s.node === 'hq' &&
          (!s.franchiseId || workspace.slug === 'lineup-and-roster') &&
          snapshot ? (
            <FranchiseHQ snapshot={snapshot} />
          ) : s.node === 'control' && workspace.slug === 'app' ? (
            <AppSettings />
          ) : (
            <section className="shell-unwired">
              <h4>{workspace.label}</h4>
              <p className="not-wired">Not wired · {workspace.ring}</p>
              {s.node === 'control' && workspace.slug === 'league' && (
                <p>Commissioner or admin role required · running as GM</p>
              )}
            </section>
          )}
          {s.node === 'home' && summary && (
            <section className="snapshot-summary" aria-label="Snapshot summary">
              <h4>Snapshot summary</h4>
              <dl className="kv">
                <dt>Season</dt>
                <dd>{summary.season}</dd>
                <dt>Franchises</dt>
                <dd>{summary.franchises}</dd>
                <dt>Rostered players</dt>
                <dd>{summary.rostered}</dd>
              </dl>
              <SignalChip {...summary.provenance} />
            </section>
          )}
          {error && <p role="alert">{error}</p>}
        </div>
      </section>
      {s.inspector !== 'closed' ? (
        <aside className="insp" aria-label="Inspector">
          <div className="insp-head">
            Inspector
            <span className="x">
              <Act
                verb={
                  s.inspector === 'expanded' ? 'inspector.collapse' : 'inspector.expand'
                }
                args={{}}
                variant="text"
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
            {snapshot && s.subject ? (
              <PlayerInspector snapshot={snapshot} subject={s.subject} />
            ) : (
              <p className="not-wired">
                Every player, franchise, pick, offer and segment opens here.
              </p>
            )}
          </div>
        </aside>
      ) : (
        <aside className="inspector-closed">
          <Act verb="inspector.toggle" args={{}}>
            Open inspector
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
      {s.commandbar && snapshot && <CommandBar snapshot={snapshot} />}
      {s.comms && (
        <aside id="target-comms" className="shell-overlay comms-panel" aria-label="Comms">
          <header>
            Comms
            <Act verb="comms.toggle" args={{}}>
              Close
            </Act>
          </header>
          <p className="not-wired">Not wired · Ring 3</p>
        </aside>
      )}
      {s.summoned === 'calendar' && (
        <aside
          id="target-calendar"
          className="shell-overlay calendar-panel"
          aria-label="Calendar"
        >
          <header>
            Calendar
            <Act verb="calendar.summon" args={{}}>
              Close
            </Act>
          </header>
          <p className="not-wired">Not wired · P5</p>
        </aside>
      )}
    </div>
  );
}
