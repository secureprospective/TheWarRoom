import { useState } from 'react';
import { commands } from '../commands/registry';
import { HQAct } from './HQAct';
import { nodes, nodeKeys, type Node } from './nodes';
import type { LeagueState } from './leaguePicker';
import type { RailNow } from './pulseReads';
import './rail.css';

// The leaves each open node shows (rail-mock-v1, option 3). War Room and Control Room hang their
// own leaves in the slots below; Home has none.
const LEAVES: Partial<Record<Node, readonly { slug: string; label: string }[]>> = {
  hq: nodes.hq.workspaces,
  pulse: [{ slug: 'now', label: 'Live' }, { slug: 'race', label: 'Race' }],
};

/** liveWords is the Pulse leaf's line: where my matchup stands in the week being played. */
export function liveWords(now?: RailNow): string | undefined {
  if (now?.mine === undefined || now.theirs === undefined) return undefined;
  const word = now.mine === now.theirs ? 'Level' : now.decided
    ? (now.mine > now.theirs ? 'Won' : 'Lost') : now.mine > now.theirs ? 'Leading' : 'Trailing';
  return `${word} ${now.mine.toFixed(1)}–${now.theirs.toFixed(1)}`;
}

/** ShellRail is the one left rail every app-league screen sits beside. */
export function ShellRail({ leagues, node, workspace, now, live, setTree, setFooter }: {
  leagues: LeagueState; node: Node; workspace: string; now?: RailNow; live?: number;
  setTree: (slot: HTMLDivElement | null) => void; setFooter: (slot: HTMLDivElement | null) => void;
}) {
  const [switching, setSwitching] = useState(false);
  const go = (to: Node, slug?: string) =>
    commands.dispatch('nav.open', (slug ? { node: to, workspace: slug } : { node: to }) as never);
  const league = leagues.selected;
  const playing = [now?.week ? `Wk ${now.week}` : '', live ? `${live} live` : ''].filter(Boolean).join(' · ');
  const words = liveWords(now);
  return <aside className="rail mock">
    <div className="rail-head">
      <div className="lg">THE WAR ROOM</div>
      <div className="ph">
        <HQAct className="lgn" expanded={switching} title="Switch league"
          label={`Switch league · ${league?.name ?? ''}`} onAction={() => setSwitching(!switching)}>
          {league?.name ?? 'Choose a league'}</HQAct>
        {playing && <> · <i>{playing}</i></>}
      </div>
      {switching && <ul className="lgs" aria-label="Leagues">
        {leagues.leagues.map(l => <li key={l.id}>
          <HQAct className={l.id === league?.id ? 'sel' : undefined}
            label={`${l.name} · ${l.app ? l.id : 'Synced from MFL'}`}
            onAction={() => { leagues.select(l.id); setSwitching(false); }}>
            {l.name}<small>{l.teams} teams{l.app ? '' : ' · synced from MFL'}</small></HQAct>
        </li>)}
      </ul>}
      {leagues.error && <p role="alert">{leagues.error}</p>}
    </div>
    <nav className="rail-body" aria-label="Workspaces">
      <div className="gl">MY SHORTCUTS</div>
      <HQAct className="leaf" label="Shortcut · Lineup and roster"
        onAction={() => go('hq', 'lineup-and-roster')}><span className="tr">·</span>Lineup and roster</HQAct>
      <div className="gl">LEAGUE</div>
      {nodeKeys.map((key, index) => {
        const open = node === key;
        const leaves = LEAVES[key] ?? [];
        return <div className={`node${open ? ' open' : ''}`} key={key}>
          <HQAct className="nb" label={nodes[key].label} active={open} onAction={() => go(key)}>
            <span className="k">{index + 1}</span>{nodes[key].label}
            {key === 'pulse' && Boolean(live) && <span className="mark" />}
            <span className="car">›</span></HQAct>
          {open && key === 'pulse' && words && <HQAct className="leaf now" label={`My matchup · ${words}`}
            onAction={() => go('pulse', 'now')}><span className="tag">{now?.decided ? 'FINAL' : 'LIVE'}</span>
            {words}</HQAct>}
          {open && leaves.map((leaf, i) => <HQAct key={leaf.slug} label={leaf.label}
            className={`leaf${workspace === leaf.slug ? ' sel' : ''}`} onAction={() => go(key, leaf.slug)}>
            <span className="tr">{i === leaves.length - 1 ? '└' : '├'}</span>{leaf.label}</HQAct>)}
          {open && key === 'control' && <div className="rtree" ref={setTree} />}
        </div>;
      })}
    </nav>
    <div className="me" ref={setFooter} />
    <div className="rail-foot">
      <HQAct className="btn" disabled title="Not built yet">CUSTOMIZE</HQAct><span>/ command</span>
    </div>
  </aside>;
}
