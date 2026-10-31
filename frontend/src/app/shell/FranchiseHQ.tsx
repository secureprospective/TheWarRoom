import { useMemo, useRef } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { FranchisePicker } from './FranchisePicker';
import { franchiseRoster } from './roster';
import { LineupRoster } from './LineupRoster';
import { HQBoard } from './HQBoard';
import { readHQMarket, readRosterFacts } from './hqRosterFacts';
import { byeWeeks, projections, standingFromReport } from './hqPlayerData';
import { hqEvents } from './hqReads';
import { useLeagueClock } from '../clock/useLeagueClock';
import { countdown } from '../clock/urgency';
import { HQHeader } from './HQHeader';
import { capFromReads, lineupCheck, settingsFromReport, unavailable } from './hqModel';
import { readHQRace, readHQSettings, useHQRead } from './hqReads';
import { contractsProvider } from '../contracts/provider';
import { useContractRead } from '../contracts/Contracts';
import { readRosterReports } from '../personal/RosterReports';
import { targetNow } from '../now/api';
import { parseNow } from '../now/contract';
import './hq.css';

const readHQNow = async (week: number, reportWeek: number, mine: string) => parseNow(await targetNow(week, reportWeek, mine));

export function FranchiseHQ({ snapshot }: { snapshot: Snapshot }) {
  const s = commands.use();
  if (!s.franchiseId) return <FranchisePicker snapshot={snapshot} />;
  return <HQFranchise key={s.franchiseId} snapshot={snapshot} mine={s.franchiseId} />;
}
function HQFranchise({ snapshot, mine }: { snapshot: Snapshot; mine: string }) {
  const s = commands.use();
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const week = clock?.value.week ?? commands.lineups.peek(mine).reading?.week ?? 0;
  const settings = useHQRead('control.TargetLeagueSettings', [], readHQSettings);
  const provider = useMemo(contractsProvider, []);
  const contracts = useContractRead(mine, () => provider.contracts({ mine, team: mine, position: '', sort: 'salary', desc: true }), provider);
  const race = useHQRead('race.TargetRace', [week, mine, mine, ''], readHQRace);
  const now = useHQRead('now.TargetNow', [week, 0, mine], readHQNow);
  const reports = useContractRead(mine, () => readRosterReports(mine), hqEvents);
  const market = useHQRead('market.TargetMarketReports', [], readHQMarket);
  const ids = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.map(p => p.id) ?? [];
  const facts = useContractRead(JSON.stringify([mine, ids]), () => readRosterFacts(ids), hqEvents);
  const rules = settingsFromReport(settings.value);
  const roster = franchiseRoster(snapshot, mine);
  const franchise = snapshot.franchises.value.find(f => f.id === mine)!;
  const lock = clock?.value.deadlines.find(d => /lineup|lock/i.test(d.id + d.label));
  const tab = s.workspace.hq;
  const navigate = (workspace: 'lineup-and-roster' | 'contracts-and-cap' | 'my-moves') => { void commands.dispatch('nav.open', { node: 'hq', workspace }); };
  return <div className="hq-v10 shell" ref={root}>
    <nav className="rail" aria-label="Main">
      <div className="rail-head"><div className="lg">THE WAR ROOM</div><div className="ph">{unavailable} · <i>{week ? `Wk ${week}` : unavailable}</i></div></div>
      <div className="rail-body"><div className="gl">MY SHORTCUTS</div>
        <button className="leaf" onClick={() => navigate('lineup-and-roster')}>Lineup and roster</button>
        <button className="leaf" onClick={() => { void commands.dispatch('nav.open', { node: 'war', workspace: 'valuations-and-pool' }); }}>Free-agent pool</button>
        <button className="leaf" disabled title={unavailable}>Offers</button>
        <div className="gl">LEAGUE</div>
        {(['Home', 'War Room', 'Franchise HQ', 'Trade Floor', 'League Pulse', 'Control Room'] as const).map((name, index) => <div className={`node ${index === 2 ? 'open' : ''}`} key={name}>
          <button className="nb" onClick={() => { void commands.dispatch('nav.open', { node: (['home', 'war', 'hq', 'trade', 'pulse', 'control'] as const)[index] }); }}>
            <span className="k">{index + 1}</span>{name}<span className="car">›</span></button>
          {index === 2 && <><button className="leaf now" disabled><span className="tag">NOW</span>{lock?.at ? `Locks ${new Date(lock.at).toLocaleString()}` : unavailable}</button>
            {(['lineup-and-roster', 'contracts-and-cap', 'my-moves'] as const).map((ws, i) => <button key={ws} className={`leaf ${tab === ws ? 'sel' : ''}`} onClick={() => navigate(ws)}>
              <span className="tr">{i === 2 ? '└' : '├'}</span>{['Lineup and roster', 'Contracts and cap', 'My moves'][i]}</button>)}</>}
        </div>)}
      </div><div className="rail-foot"><button className="btn" disabled title={unavailable}>CUSTOMIZE</button><span>/ command</span></div>
    </nav>
    <main className="work">
      <div className="ws-tabs" role="tablist">
        <button role="tab" aria-selected={tab === 'lineup-and-roster'} onClick={() => navigate('lineup-and-roster')}>LINEUP AND ROSTER</button>
        <button role="tab" aria-selected={tab === 'contracts-and-cap'} onClick={() => navigate('contracts-and-cap')}>CONTRACTS AND CAP</button>
        <button className="queue" onClick={() => navigate('my-moves')} aria-label="My moves queue"><span className="ql">MY MOVES</span><span className="qn dim">{unavailable}</span></button>
        <button className="cmd" onClick={() => { void commands.dispatch('commandbar.open', {}); }}><kbd>/</kbd> lineup.swap out=… in=…</button>
      </div>
      <LineupRoster snapshot={snapshot} roster={roster} franchiseId={mine} selected={s.subject} cap={null}
        render={editor => <>
          <HQHeader name={franchise.name ?? mine} active={roster.groups[0].players.length}
            settings={rules} cap={capFromReads(snapshot, mine, contracts.reading)}
            check={editor.checkError ? unavailable : lineupCheck(editor.reading, editor.error)}
            lock={lock && { ...lock, text: countdown(lock.at, Date.now()) }}
            {...standingFromReport(race.value?.standings)} />
          <HQBoard snapshot={snapshot} mine={mine} reading={editor.error ? undefined : editor.reading}
            settings={rules} values={projections(snapshot, market.value, editor.reading?.week)} facts={facts.reading}
            byes={byeWeeks(snapshot, reports.reading)} now={now.value} changed={editor.changes} canEdit={!editor.reason}
            select={id => { void commands.dispatch('inspector.open', { subject: { kind: 'player', id, franchiseId: mine } }); }}
            place={(id, out) => { if (editor.reading?.starters.some(p => p.id === id)) return;
              if (!editor.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
              if (out && out !== id) commands.dispatch('lineup.toggle', { playerId: out });
              if (out !== id) commands.dispatch('lineup.toggle', { playerId: id }); }}
            bench={id => { if (!editor.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
              commands.dispatch('lineup.toggle', { playerId: id }); }} />
          <div className="tray"><div className="pulls"><button className="pull" disabled>TAXI <b>{roster.groups[2].players.length} / {rules.taxi ?? unavailable}</b></button><button className="pull" disabled>IR <b>{roster.groups[1].players.length} / {rules.ir ?? unavailable}</b></button><button className="pull" disabled>WHAT-IF</button></div>
            <span className="msg" role="status">{editor.checkError || s.notice || (editor.changes.length ? `${editor.changes.length} players changed from saved` : 'No changes. Swaps stay a plan until you save.')}</span>
            <button className="btn sp" disabled={!editor.draft} onClick={() => { void commands.dispatch('lineup.cancel', {}); }}>DISCARD</button>
            <button className="btn g2" disabled={Boolean(editor.saveReason)} title={editor.saveReason} onClick={() => { void commands.dispatch('lineup.draft', {}); }}>CHECK AND SAVE PLAN</button>
            <button className="btn solid" disabled title={unavailable}>OPEN MFL ↗</button>
          </div>
        </>} />
    </main>
    <aside className="insp" aria-label="Inspector"><div className="ihead">{unavailable}</div><div /><div className="idrawers" /><div className="iact"><span className="h">ACT ON THIS PLAYER</span></div><div className="ifoot">{unavailable}</div></aside>
  </div>;
}
