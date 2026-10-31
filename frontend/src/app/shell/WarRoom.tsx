import { useRef, useState } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { useLeagueClock } from '../clock/useLeagueClock';
import { countdown } from '../clock/urgency';
import { FranchisePicker } from './FranchisePicker';
import { WarRoomHeader, WarRoomEmpty } from './WarRoomHeader';
import { readWarRoomHeader, useWarRoomRead, type WarRoomNative } from './warroomReads';
import { unavailable } from './hqModel';
import './warroom.css';
export type WarRoomTab = 'desk' | 'market' | 'draft' | 'plans';
export default function WarRoom({ snapshot, native }: { snapshot: Snapshot; native?: WarRoomNative }) {
  const state = commands.use();
  const key = commands.useMFLKey();
  const mine = native?.mine ?? key.myFranchise ?? state.franchiseId;
  if (!mine) return <FranchisePicker snapshot={snapshot} />;
  return <WarRoomWorkspace key={`${native?.league ?? 'mirror'}:${mine}`} snapshot={snapshot}
    mine={mine} native={native} />;
}
function WarRoomWorkspace({ snapshot, mine, native }: { snapshot: Snapshot; mine: string;
  native?: WarRoomNative }) {
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const [tab, setTab] = useState<WarRoomTab>('desk');
  const [drawer, setDrawer] = useState<'pocket' | 'offers'>();
  const week = native ? 0 : clock?.value.week ?? 0;
  const context = { snapshot, mine, week, native };
  const header = useWarRoomRead(JSON.stringify([native?.league, mine, week]), () =>
    readWarRoomHeader(context));
  const deadlines = native ? [] : clock?.provenance.freshness.state === 'fail' ? [] :
    clock?.value.deadlines ?? [];
  const deadline = deadlines.find(d => /trade/i.test(d.id + d.label));
  const navigate = (node: 'home' | 'war' | 'hq' | 'pulse' | 'control') => {
    if (native) { if (node !== 'war') native.leave(); }
    else void commands.dispatch('nav.open', { node });
  };
  return <div className="wr-v6" ref={root}><div className="shell">
    <nav className="rail" aria-label="Main"><div className="rail-head"><div className="lg">THE WAR ROOM
      </div><div className="ph">{header.reading?.name ?? unavailable} · <i>{week ? `Wk ${week}` :
        unavailable}</i></div></div><div className="rail-body"><div className="gl">MY SHORTCUTS</div>
      <button className="leaf" onClick={() => navigate('hq')}><span className="tr">·</span>Lineup and roster
      </button><div className="gl">LEAGUE</div>
      {(['home', 'war', 'hq', 'pulse', 'control'] as const).map((node, i) => <div
        className={`node ${node === 'war' ? 'open' : ''}`} key={node}>
        <button className="nb" onClick={() => navigate(node)}><span className="k">{i + 1}</span>
          {['Home', 'War Room', 'Franchise HQ', 'League Pulse', 'Control Room'][i]}
          <span className="car">›</span></button>{node === 'war' && <>
          <button className="leaf now" disabled><span className="tag">NOW</span>{deadline ?
            `Deadline ${countdown(deadline.at, Date.now())}` : unavailable}</button>
          {(['desk', 'market', 'draft', 'plans'] as const).map((t, j) => <button key={t}
            className={`leaf ${tab === t ? 'sel' : ''}`} onClick={() => setTab(t)}>
            <span className="tr">{j === 3 ? '└' : '├'}</span>{['Trade desk', 'Market', 'Draft room',
              'Plans'][j]}</button>)}</>}</div>)}</div>
      <div className="rail-foot"><button className="btn" disabled title={unavailable}>CUSTOMIZE</button>
        <span>/ command</span></div></nav>
    <main className="work"><div className="ws-tabs" role="tablist">
      {(['desk', 'market', 'draft', 'plans'] as const).map((t, i) => <button role="tab" key={t}
        aria-selected={tab === t} onClick={() => setTab(t)}>{['TRADE DESK', 'MARKET', 'DRAFT ROOM',
          'PLANS'][i]}</button>)}<div className="cmd" style={{ marginLeft: 'auto' }}><kbd>/</kbd>
          trade.build with=… send=… get=…</div></div>
      <WarRoomHeader name={snapshot.franchises.value.find(f => f.id === mine)?.name}
        reading={header.reading} />
      <section className="desk ws on"><WarRoomEmpty /></section>
      <div className="cal" aria-label="Calendar">{deadlines.length ? deadlines.map(d => <div
        className="ev" key={d.id}><span className="w">{d.at ? new Date(d.at).toLocaleDateString() : unavailable}</span>
        <b>{d.label}</b><span className="c">{countdown(d.at, Date.now())}</span></div>) :
        <WarRoomEmpty />}</div>
      <div className="tray"><div className="pulls">{(['pocket', 'offers'] as const).map(d => <button
        key={d} className="pull" aria-expanded={drawer === d} onClick={() =>
          setDrawer(drawer === d ? undefined : d)}>{d === 'pocket' ? 'POCKET' : 'NEGOTIATIONS'}</button>)}
      </div><span className="msg" title={header.error ?? header.reading?.errors.join(' · ')}>{unavailable}
      </span><button className="btn sp" disabled>SAVE TO POCKET</button><button className="btn" disabled>
        CLEAR</button><button className="btn solid g2" disabled>SEND</button></div>
      {drawer && <div className="bdrawer open"><div className="bd-head"><h2>{drawer === 'pocket' ?
        'Pocket' : 'Negotiations'}</h2><button className="x" onClick={() => setDrawer(undefined)}>ESC
        </button></div><div className="bd-body pocket"><WarRoomEmpty /></div></div>}
    </main><aside className="insp" aria-label="Inspector"><WarRoomEmpty /></aside>
  </div></div>;
}
