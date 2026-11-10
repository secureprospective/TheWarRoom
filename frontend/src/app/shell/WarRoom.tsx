import EntityInspector from './EntityInspector';
import { HQAct as WarRoomAct } from './HQAct';
import { createPortal } from 'react-dom';
import { RailLeaguePicker } from './leaguePicker';
import { useEffect, useRef, useState } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { useLeagueClock } from '../clock/useLeagueClock';
import { countdown } from '../clock/urgency';
import { FranchisePicker } from './FranchisePicker';
import { WarRoomDates, WarRoomHeader, WarRoomTrayMessage } from './WarRoomHeader';
import { readWarRoomHeader, nativeWarRoomHeader, useWarRoomRead, type WarRoomNative } from './warroomReads';
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
import { readWarRoomLens, setWarRoomDirection, type Direction, type WarRoomLens } from './warroomLens';
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
  const state = commands.use();
  const workspace = state.workspace.war;
  const routeTab = workspace === 'draft-room' ? 'draft' : workspace === 'trade-desk' ? 'desk' : undefined;
  const [localTab, setLocalTab] = useState<WarRoomTab>('desk');
  const tab = routeTab ?? localTab;
  const setTab = (tab: WarRoomTab) => {
    setLocalTab(tab);
    void commands.dispatch('nav.open', { node: 'war', workspace: tab === 'draft' ? 'draft-room' :
      tab === 'desk' ? 'trade-desk' : 'valuations-and-pool' });
  };
  const [offer, setOffer] = useState<string>();
  const [drawer, setDrawer] = useState<'pocket' | 'offers'>();
  const week = native ? native.week ?? 0 : clock?.value.week ?? 0;
  const [deal, changeDeal] = useState<WarRoomDeal>({ partner: '', give: [], get: [] });
  const input = state.warTrade;
  useEffect(() => {
    if (input && input.mine === mine && input.league === native?.league)
      changeDeal(d => d.give.includes(input.player) ? d : { ...d, give: [...d.give, input.player] });
  }, [input, mine, native?.league]);
  const [selected, select] = useState<WarRoomAsset>();
  const [prospect, setProspect] = useState<WarRoomProspect>();
  const context = { snapshot, mine, week, native };
  const [recorded, setRecorded] = useState(0);
  const [directing, setDirecting] = useState<{ pending?: boolean; error?: string }>({});
  const catalog = useWarRoomRead(JSON.stringify([native?.league, mine, recorded]), () =>
    readWarRoomCatalog(context));
  const lens = useWarRoomRead(JSON.stringify([native?.league, mine, deal, catalog.reading?.revision,
    recorded]), () => readWarRoomLens(context, deal));
  // A new deal re-reads the lens; the last answer stays on the desk until the next one lands.
  const lastLens = useRef<WarRoomLens>();
  if (lens.reading) lastLens.current = lens.reading;
  const lensReading = lens.error ? undefined : lens.reading ?? lastLens.current;
  const revision = catalog.reading?.revision;
  const choose = native && mine && native.owner !== false && revision !== undefined ?
    (direction: Direction) => {
      setDirecting({ pending: true });
      setWarRoomDirection(context, direction, revision).then(() => { setDirecting({});
        setRecorded(n => n + 1); }, cause => setDirecting({ error: String(cause) }));
    } : undefined;
  const header = useWarRoomRead(JSON.stringify([native?.league, mine, week]), () =>
    readWarRoomHeader(context));
  const headerReading = native?.reading ? nativeWarRoomHeader(native.reading) : header.reading;
  const calendar = useWarRoomRead(JSON.stringify([native?.league, mine, snapshot.league.value.season]),
    () => readWarRoomCalendar(context));
  const deadlines: WarRoomEvent[] = calendar.error ? [] : calendar.reading ?? [];
  const deadline = deadlines.find(d => /trade/i.test(d.id + d.label));
  // The calendar says which action window is open now. An app league with none open is not an
  // unavailable fact: it is a closed window, and the header says so.
  const windowLabel = !native || calendar.error ? undefined :
    calendar.reading ? deadlines.find(d => d.live)?.label ?? 'none open' : undefined;
  const navigate = (node: 'home' | 'war' | 'hq' | 'pulse' | 'control') => {
    if (native) { if (node !== 'war') native.leave(node); }
    else void commands.dispatch('nav.open', { node });
  };
  const tabs = (['desk', 'market', 'draft', 'plans'] as const).map((t, i) => <WarRoomAct role="tab" key={t}
    className={native?.embedded ? `act act-tab${tab === t ? ' on' : ''}` : undefined}
    selected={tab === t} onAction={() => setTab(t)}>{['TRADE DESK', 'MARKET', 'DRAFT ROOM',
      'PLANS'][i]}</WarRoomAct>);
  return <div className={`wr-v6${native?.embedded ? ' embedded' : ''}`} ref={root}><div className="shell">
    {!native?.embedded && <nav className="rail" aria-label="Main"><RailLeaguePicker />
      {native && !mine && <p role="status">No franchise assigned · read-only</p>}<div className="rail-head">
      <div className="lg">THE WAR ROOM
      </div><div className="ph">{headerReading?.name ?? unavailable} · <i>{week ? `Wk ${week}` :
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
        <span>/ command</span></div></nav>}
    <main className={`work${native?.embedded && native.tabs ? ' external-tabs' : ''}`}>
      {native?.embedded && native.tabs ? createPortal(tabs, native.tabs) :
      <div className="ws-tabs" role="tablist">{tabs}</div>}
      <WarRoomHeader name={snapshot.franchises.value.find(f => f.id === mine)?.name}
        reading={headerReading} windowLabel={windowLabel} lens={lensReading} choose={choose}
        pending={directing.pending} />
      {tab === 'desk' ? <WarRoomTrade context={context} catalog={catalog.reading} deal={deal}
        change={changeDeal} select={select} lens={lensReading} lensPending={lens.pending} /> : tab === 'market' ? <WarRoomMarket context={context}
          room={headerReading?.runway?.[0]?.room} select={p => select({ token: p.id, name: p.name })} /> :
          tab === 'draft' ? <WarRoomDraft context={context} select={(id, name, position) =>
            setProspect({ id, name, position })}
            /> : <WarRoomPlans context={context} openPocket={() => setDrawer('pocket')} />}
      <WarRoomDates events={deadlines.filter(d => d.live || !d.at || Date.parse(d.closes ?? d.at) >
        Date.now()).slice(0, 8)} failed={Boolean(calendar.error)} reason={calendar.error} />
      <div className="tray"><div className="pulls">{(['pocket', 'offers'] as const).map(d => <WarRoomAct
        key={d} className="pull" expanded={drawer === d} onAction={() =>
          setDrawer(drawer === d ? undefined : d)}>{d === 'pocket' ? 'POCKET' : 'NEGOTIATIONS'}</WarRoomAct>)}
      </div><WarRoomTrayMessage problems={[header.error ?? '', calendar.error ?? '', lens.error ?? '',
        directing.error ?? '', ...headerReading?.errors ?? []]} />
      <WarRoomAct className="btn sp" disabled>SAVE TO POCKET</WarRoomAct>
      <WarRoomAct className="btn" onAction={() =>
          changeDeal({ ...deal, give: [], get: [] })}>
        CLEAR</WarRoomAct><WarRoomAct className="btn solid g2" disabled>SEND</WarRoomAct></div>
      {drawer && <WarRoomDrawers context={context} drawer={drawer} close={() => setDrawer(undefined)}
        openOffer={id => { setOffer(id); setDrawer(undefined); }} />}
    </main>{offer && !native ? <aside className="insp" aria-label="Inspector">
      <WarRoomAct onAction={() => setOffer(undefined)}>Close offer</WarRoomAct>
      <EntityInspector entity={{ kind: 'offer', id: offer, franchiseId: mine }} snapshot={snapshot} />
    </aside> : <WarRoomInspector context={context} asset={selected} lens={lensReading}
      theirs={Boolean(selected && deal.partner && catalog.reading?.teams.find(t => t.id === deal.partner)
        ?.assets.some(a => a.token === selected.token))} add={selected &&
      catalog.reading?.teams.some(t => (t.id === mine || t.id === deal.partner) && t.assets.some(a =>
        a.token === selected.token)) ? () => {
        const side = catalog.reading?.teams.find(t => t.assets.some(a => a.token === selected.token))?.id
          === mine ? 'give' : 'get';
        if (!deal[side].includes(selected.token)) changeDeal({ ...deal, [side]: [...deal[side],
          selected.token] }); setTab('desk');
      } : undefined} watch={!native && selected && !selected.pick ? () =>
        changeWarRoomTarget(context, selected.token, false) : undefined} />}
  </div>{prospect && <WarRoomScouting context={context} prospect={prospect}
    close={() => setProspect(undefined)} />}</div>;
}
