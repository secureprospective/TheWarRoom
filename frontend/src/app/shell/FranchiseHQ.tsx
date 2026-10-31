import { useMemo, useRef } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { FranchisePicker } from './FranchisePicker';
import { franchiseRoster } from './roster';
import { LineupRoster } from './LineupRoster';
import { useLineup } from './lineups';
import { useLeagueClock } from '../clock/useLeagueClock';
import { countdown } from '../clock/urgency';
import { HQHeader } from './HQHeader';
import { capFromReads, lineupCheck, settingsFromReport, unavailable } from './hqModel';
import { readHQRace, readHQSettings, useHQRead } from './hqReads';
import { contractsProvider } from '../contracts/provider';
import { useContractRead } from '../contracts/Contracts';
import './hq.css';

export function FranchiseHQ({ snapshot }: { snapshot: Snapshot }) {
  const s = commands.use();
  if (!s.franchiseId) return <FranchisePicker snapshot={snapshot} />;
  return <HQFranchise key={s.franchiseId} snapshot={snapshot} mine={s.franchiseId} />;
}
function HQFranchise({ snapshot, mine }: { snapshot: Snapshot; mine: string }) {
  const s = commands.use();
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const { reading, error } = useLineup(commands.lineups, mine, snapshot);
  const settings = useHQRead('control.TargetLeagueSettings', [], readHQSettings);
  const provider = useMemo(contractsProvider, []);
  const contracts = useContractRead(mine, () => provider.contracts({ mine, team: mine, position: '', sort: 'salary', desc: true }), provider);
  const race = useHQRead('race.TargetRace', [reading?.week ?? 0, mine, mine, ''], readHQRace);
  const roster = franchiseRoster(snapshot, mine);
  const franchise = snapshot.franchises.value.find(f => f.id === mine)!;
  const lock = clock?.value.deadlines.find(d => /lineup|lock/i.test(d.id + d.label));
  const tab = s.workspace.hq;
  const navigate = (workspace: 'lineup-and-roster' | 'contracts-and-cap' | 'my-moves') => { void commands.dispatch('nav.open', { node: 'hq', workspace }); };
  return <div className="hq-v10 shell" ref={root}>
    <nav className="rail" aria-label="Main">
      <div className="rail-head"><div className="lg">THE WAR ROOM</div><div className="ph">{unavailable} · <i>{reading ? `Wk ${reading.week}` : unavailable}</i></div></div>
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
      <HQHeader name={franchise.name ?? mine} active={roster.groups[0].players.length}
        settings={settingsFromReport(settings.value)} cap={capFromReads(snapshot, mine, contracts.reading)}
        check={lineupCheck(reading, error)} lock={lock && { ...lock, text: countdown(lock.at, Date.now()) }}
        record={race.value?.standings.rows.find(r => r.mine)?.cells[2]} />
      <section style={{ overflow: 'auto' }}><LineupRoster snapshot={snapshot} roster={roster} franchiseId={mine} selected={s.subject} cap={null} /></section>
      <div className="tray"><div className="pulls"><button className="pull" disabled>TAXI</button><button className="pull" disabled>IR</button><button className="pull" disabled>WHAT-IF</button></div><span className="msg">No changes. Swaps stay a plan until you save.</span></div>
    </main>
    <aside className="insp" aria-label="Inspector"><div className="ihead">{unavailable}</div><div /><div className="idrawers" /><div className="iact"><span className="h">ACT ON THIS PLAYER</span></div><div className="ifoot">{unavailable}</div></aside>
  </div>;
}
