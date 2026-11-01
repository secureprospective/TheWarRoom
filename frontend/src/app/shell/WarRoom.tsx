import { HQAct as WarRoomAct } from './HQAct';
import { RailLeaguePicker } from './leaguePicker';
import { useRef, useState } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { useLeagueClock } from '../clock/useLeagueClock';
import { countdown } from '../clock/urgency';
import { FranchisePicker } from './FranchisePicker';
import { WarRoomHeader, WarRoomEmpty } from './WarRoomHeader';
import { readWarRoomHeader, useWarRoomRead, type WarRoomNative } from './warroomReads';
import { unavailable } from './hqModel';
import { WarRoomScouting, type WarRoomProspect } from './WarRoomScouting';
import { WarRoomInspector } from './WarRoomInspector';
import { WarRoomPlans } from './WarRoomPlans';
import { WarRoomDrawers } from './WarRoomDrawers';
import { readWarRoomCalendar, type WarRoomEvent } from './warroomCalendar';
import { changeWarRoomTarget } from './warroomMarketReads';
import { WarRoomMarket } from './WarRoomMarket';
import { WarRoomDraft } from './WarRoomDraft';
import { WarRoomTrade } from './WarRoomTrade';
import { readWarRoomCatalog, type WarRoomDeal, type WarRoomAsset } from './warroomTradeReads';
import './warroom.css';
export type WarRoomTab = 'desk' | 'market' | 'draft' | 'plans';
export default function WarRoom({ snapshot, native }: { snapshot: Snapshot; native?: WarRoomNative }) {
  const state = commands.use();
  const key = commands.useMFLKey();
  const mine = native?.mine ?? (key.myFranchise || state.franchiseId);
  if (!mine && !native) return <FranchisePicker snapshot={snapshot} />;
  return <WarRoomWorkspace key={`${native?.league ?? 'mirror'}:${mine}`} snapshot={snapshot}
    mine={mine ?? ''} native={native} />;
}
function WarRoomWorkspace({ snapshot, mine, native }: { snapshot: Snapshot; mine: string;
  native?: WarRoomNative }) {
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const [tab, setTab] = useState<WarRoomTab>('desk');
  const [drawer, setDrawer] = useState<'pocket' | 'offers'>();
  const week = native ? native.week ?? 0 : clock?.value.week ?? 0;
  const [deal, changeDeal] = useState<WarRoomDeal>({ partner: '', give: [], get: [] });
  const [selected, select] = useState<WarRoomAsset>();
  const [prospect, setProspect] = useState<WarRoomProspect>();
  const context = { snapshot, mine, week, native };
  const catalog = useWarRoomRead(JSON.stringify([native?.league, mine]), () => readWarRoomCatalog(context));
  const header = useWarRoomRead(JSON.stringify([native?.league, mine, week]), () =>
    readWarRoomHeader(context));
  const calendar = useWarRoomRead(JSON.stringify([native?.league, mine, snapshot.league.value.season]),
    () => readWarRoomCalendar(context));
  const deadlines: WarRoomEvent[] = native ? calendar.reading ?? [] :
    clock?.provenance.freshness.state === 'fail' ? [] :
    clock?.value.deadlines ?? [];
  const deadline = deadlines.find(d => /trade/i.test(d.id + d.label));
  const navigate = (node: 'home' | 'war' | 'hq' | 'pulse' | 'control') => {
    if (native) { if (node !== 'war') native.leave(node); }
    else void commands.dispatch('nav.open', { node });
  };
  return <div className="wr-v6" ref={root}><div className="shell">
    <nav className="rail" aria-label="Main"><RailLeaguePicker />
      {native && !mine && <p role="status">No franchise assigned · read-only</p>}<div className="rail-head"><div className="lg">THE WAR ROOM
      </div><div className="ph">{header.reading?.name ?? unavailable} · <i>{week ? `Wk ${week}` :
        unavailable}</i></div></div><div className="rail-body"><div className="gl">MY SHORTCUTS</div>
      <WarRoomAct className="leaf" onAction={() => navigate('hq')}>
      <span className="tr">·</span>Lineup and roster
      </WarRoomAct><div className="gl">LEAGUE</div>
      {(['home', 'war', 'hq', 'pulse', 'control'] as const).map((node, i) => <div
        className={`node ${node === 'war' ? 'open' : ''}`} key={node}>
        <WarRoomAct className="nb" onAction={() => navigate(node)}><span className="k">{i + 1}</span>
          {['Home', 'War Room', 'Franchise HQ', 'League Pulse', 'Control Room'][i]}
          <span className="car">›</span></WarRoomAct>{node === 'war' && <>
          <WarRoomAct className="leaf now" disabled><span className="tag">NOW</span>{deadline ?
            `Deadline ${countdown(deadline.at, Date.now())}` : unavailable}</WarRoomAct>
          {(['desk', 'market', 'draft', 'plans'] as const).map((t, j) => <WarRoomAct key={t}
            className={`leaf ${tab === t ? 'sel' : ''}`} onAction={() => setTab(t)}>
            <span className="tr">{j === 3 ? '└' : '├'}</span>{['Trade desk', 'Market', 'Draft room',
              'Plans'][j]}</WarRoomAct>)}</>}</div>)}</div>
      <div className="rail-foot">
      <WarRoomAct className="btn" disabled title={unavailable}>CUSTOMIZE</WarRoomAct>
        <span>/ command</span></div></nav>
    <main className="work"><div className="ws-tabs" role="tablist">
      {(['desk', 'market', 'draft', 'plans'] as const).map((t, i) => <WarRoomAct role="tab" key={t}
        selected={tab === t} onAction={() => setTab(t)}>{['TRADE DESK', 'MARKET', 'DRAFT ROOM',
          'PLANS'][i]}</WarRoomAct>)}</div>
      <WarRoomHeader name={snapshot.franchises.value.find(f => f.id === mine)?.name}
        reading={header.reading} />
      {tab === 'desk' ? <WarRoomTrade context={context} catalog={catalog.reading} deal={deal}
        change={changeDeal} select={select} /> : tab === 'market' ? <WarRoomMarket context={context}
          room={header.reading?.runway?.[0]?.room} select={p => select({ token: p.id, name: p.name })} /> :
          tab === 'draft' ? <WarRoomDraft context={context} select={(id, name, position) =>
            setProspect({ id, name, position })}
            /> : <WarRoomPlans />}
      <div className="cal" aria-label="Calendar">{deadlines.length ? deadlines.map(d => <div
        className="ev" key={d.id}>
      <span className="w">{d.at ? new Date(d.at).toLocaleDateString() : unavailable}</span>
        <b>{d.label}</b><span className="c">{countdown(d.at, Date.now())}</span></div>) :
        <WarRoomEmpty />}</div>
      <div className="tray"><div className="pulls">{(['pocket', 'offers'] as const).map(d => <WarRoomAct
        key={d} className="pull" expanded={drawer === d} onAction={() =>
          setDrawer(drawer === d ? undefined : d)}>{d === 'pocket' ? 'POCKET' : 'NEGOTIATIONS'}</WarRoomAct>)}
      </div><span className="msg" title={header.error || header.reading?.errors.length ?
        unavailable : undefined}>{unavailable}
      </span>
      <WarRoomAct className="btn sp" disabled>SAVE TO POCKET</WarRoomAct>
      <WarRoomAct className="btn" onAction={() =>
          changeDeal({ ...deal, give: [], get: [] })}>
        CLEAR</WarRoomAct><WarRoomAct className="btn solid g2" disabled>SEND</WarRoomAct></div>
      {drawer && <WarRoomDrawers context={context} drawer={drawer} close={() => setDrawer(undefined)} />}
    </main><WarRoomInspector context={context} asset={selected} add={selected &&
      catalog.reading?.teams.some(t => (t.id === mine || t.id === deal.partner) && t.assets.some(a =>
        a.token === selected.token)) ? () => {
        const side = catalog.reading?.teams.find(t => t.assets.some(a => a.token === selected.token))?.id
          === mine ? 'give' : 'get';
        if (!deal[side].includes(selected.token)) changeDeal({ ...deal, [side]: [...deal[side],
          selected.token] }); setTab('desk');
      } : undefined} watch={!native && selected && !selected.pick ? () =>
        changeWarRoomTarget(context, selected.token, false) : undefined} />
  </div>{prospect && <WarRoomScouting context={context} prospect={prospect}
    close={() => setProspect(undefined)} />}</div>;
}
