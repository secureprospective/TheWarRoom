import { HQAct as WarRoomAct } from './HQAct';
import { useState } from 'react';
import type { WarRoomContext } from './warroomReads';
import type { WarRoomOwnBid } from './warroomMarketReads';
import { useWarRoomRead } from './warroomReads';
import { readWarRoomMarket, changeWarRoomTarget, warRoomTargetLanes, type WarRoomMarketReading,
  type WarRoomMarketPlayer } from './warroomMarketReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { money, unavailable } from './hqModel';
import { poolProvider } from '../pool/provider';
import { copyText, planForm } from '../pool/state';
import type { Plan } from '../pool/contract';
import { useWarRoomDrag } from './warroomDrag';
export const ownBid = (p: Plan): WarRoomOwnBid => ({ player: p.playerId, name: p.player,
  contract: p.contract, state: p.status || p.state, kind: 'plan', id: p.id, post: p.post, plan: p });
const STEP = 0.1;
export function WarRoomOwnBids({ reading, kind, step }: { reading?: WarRoomMarketReading;
  kind?: 'plan' | 'placed'; step?: (b: WarRoomOwnBid, by: number) => void }) {
  const bids = reading?.bids?.filter(b => !kind || b.kind === kind);
  const [copied, setCopied] = useState('');
  const copy = async (b: WarRoomOwnBid) => {
    setCopied((await copyText(b.post ?? '')) ? b.player : '');
  };
  return <>{bids ? bids.length ? bids.map((b, k) => <div className="bidb"
    key={b.player}><span className="nm">{kind === 'plan' ? `${k + 1}. ` : ''}{b.name}</span><b>{b.contract}</b>
      {step && b.plan && <span className="step2"><WarRoomAct label={`Lower ${b.name} bid`}
        onAction={() => step(b, -STEP)}>−</WarRoomAct><b>{b.plan.salary.text}</b><WarRoomAct
          label={`Raise ${b.name} bid`} onAction={() => step(b, STEP)}>+</WarRoomAct></span>}
      <span className="chip2">{b.state}</span>{b.post && <WarRoomAct className="chip"
        label={`Copy ProBoards post for ${b.name}`} onAction={() => { void copy(b); }}>
        {copied === b.player ? 'COPIED' : 'COPY PROBOARDS POST'}</WarRoomAct>}</div>) :
    kind === 'plan' ? null : <WarRoomEmpty reason="No own bids recorded" /> : <WarRoomEmpty />}
    {kind === 'plan' && <div className="add">Drag a player here to bid. Order = priority.</div>}</>;
}
export function WarRoomMarket({ context, room, select }: { context: WarRoomContext; room?: number;
  select: (p: WarRoomMarketPlayer) => void }) {
  const [refresh, setRefresh] = useState(0);
  const [filter, setFilter] = useState('All');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const reading = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, refresh]), () =>
    readWarRoomMarket(context));
  const value = reading.reading;
  const targets = new Set(value?.targets?.map(t => t.id));
  const change = async (id: string, withPlayer?: string) => {
    if (pending) return;
    setPending(true); setError('');
    try { await changeWarRoomTarget(context, id, targets.has(id), withPlayer); setRefresh(n => n + 1); }
    catch { setError(unavailable); } finally { setPending(false); }
  };
  const bid = async (p: WarRoomMarketPlayer) => {
    if (pending || value?.bids?.some(b => b.player === p.id) || own[p.id]) return;
    setPending(true); setError('');
    try {
      const plan = await poolProvider.draft(context.mine, { planId: '', playerId: p.id, playerName: p.name,
        salary: String((p.minimumCents ?? 0) / 1e8), years: 1, snipe: false, beat: '', evidence: '',
        closesOpen: false, closesDate: '', closesTime: '', conditionalDrop: '', conditionalOpen: false });
      setOwn(o => ({ ...o, [plan.playerId]: ownBid(plan) }));
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  // A saved or revised plan updates its own card; the pool is not read again.
  const [own, setOwn] = useState<Record<string, WarRoomOwnBid>>({});
  const step = async (b: WarRoomOwnBid, by: number) => {
    if (pending || !b.plan) return;
    const salary = Math.round((b.plan.salary.cents / 1e8 + by) * 100) / 100;
    if (salary <= 0) return;
    setPending(true); setError('');
    try {
      const plan = await poolProvider.revise(b.plan.id, context.mine,
        { ...planForm(b.plan), salary: String(salary) });
      setOwn(o => ({ ...o, [plan.playerId]: ownBid(plan) }));
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const bids = value && [...(value.bids ?? []).filter(b => !own[b.player]), ...Object.values(own)];
  const shown = value && { ...value, bids };
  const drag = useWarRoomDrag((id, over) => {
    const p = value?.players?.find(x => x.id === id);
    if (!p) return;
    if (over.closest('.bids')) { void bid(p); return; }
    const lane = over.closest<HTMLElement>('.lane2');
    if (lane && lane.dataset.lane === p.position && !targets.has(id)) void change(id);
  });
  const players = value?.players?.filter(p => filter !== 'Targets' || targets.has(p.id));
  return <section className="mk3 ws on" aria-label="Market"><div className="tg"><div className="ch">
    <b>MY TARGETS</b><span>ranked per need</span></div><div className="tg-l">
    {value?.targets?.length ? [...warRoomTargetLanes(context, value)].map(([position, lane]) => <div
      className="lane2" key={position} data-lane={position}><div className="lh"><b>{position}</b><span>{unavailable}</span></div>
      {lane.map(({ entry, player }, i) => <div className="tgt" key={entry.id}>
        <span className="n">{i + 1}</span><WarRoomAct className="nm" onAction={() => select(player)}>
          {player.name}</WarRoomAct><small>{value.bids?.find(b => b.player === entry.id)?.contract ??
            unavailable}</small><div><WarRoomAct disabled={pending || !i} label={`Move ${player.name} up`}
              onAction={() => { void change(entry.id, lane[i - 1].entry.id); }}>↑</WarRoomAct>
          <WarRoomAct disabled={pending || i === lane.length - 1} label={`Move ${player.name} down`}
            onAction={() => { void change(entry.id, lane[i + 1].entry.id); }}>↓</WarRoomAct></div></div>)}
    </div>) : <WarRoomEmpty reason={context.native ? 'No hosted private target read' :
      'Star a player to add a target'} />}</div></div>
    <div><div className="ch"><b>FREE AGENT POOL</b><span>in-season · drag a row to a target lane or to BIDS
    </span></div><div className="bench"><div className="filters">{['All', 'My needs', 'Targets',
      'Affordable'].map(f => <WarRoomAct key={f} className="chip" active={filter === f}
        disabled={f === 'My needs' || f === 'Affordable'} title={f === 'My needs' || f === 'Affordable' ?
          unavailable : undefined} onAction={() => setFilter(f)}>{f}</WarRoomAct>)}</div></div>
      <div className="fa-h">{['☆', 'POS', 'PLAYER', 'AGE', 'STRING', 'PTS', 'VALUE', 'MIN BID'].map((h, i) => <span key={i}>{h}</span>)}</div><div className="fa-list">
      {players?.length ? players.map(p => <div className={`fa-r${bids?.some(b => b.player === p.id) ? ' bid' : ''}`}
        key={p.id} onPointerDown={e => drag(e, p.id)}><WarRoomAct className="st"
          label={`${targets.has(p.id) ? 'Unwatch' : 'Watch'} ${p.name}`} active={targets.has(p.id)}
          disabled={pending || !!context.native || !value?.targets} onAction={() => { void change(p.id); }}>
          {targets.has(p.id) ? '★' : '☆'}</WarRoomAct><span className="ps">{p.position}</span>
        <WarRoomAct className="nm" onAction={() => select(p)}>{p.name}</WarRoomAct>
      <span>{p.age ?? unavailable}</span>
        <span>{p.string ?? "—"}</span><span>{p.points ?? "—"}</span>
        <span className="fa-value">{p.value ?? "—"}</span><span>{p.minimum ?? "—"}</span></div>) :
        <WarRoomEmpty pending={!value && !reading.error} reason={reading.error ? 'Pool could not be read' :
          value?.players === undefined ? 'Pool could not be read' : 'No free agents match this view'} />}</div>
        {error && <p className="empty" role="alert">{error}</p>}</div>
    <div className="bcol"><div className="budget"><div className="t"><span>CAP ROOM</span><b>{money(room)}
      </b><span className="lim">HARD LIMIT</span></div><div className="bar" /><div className="t">
        <span>committed {unavailable}</span><span>free {unavailable}</span></div></div>
      <div className="ch"><b>MY BIDS</b><span>{unavailable}</span></div><div className="bids">
        <div className="ch"><b>READY</b></div><WarRoomOwnBids reading={shown} kind="plan"
          step={(b, by) => { void step(b, by); }} />
        <div className="ch"><b>PLACED</b></div>
        <WarRoomOwnBids reading={value} kind="placed" /></div><div className="nested" />
      <div className="recent"><div className="ch"><b>RECENT WINS</b></div><WarRoomEmpty /></div></div>
  </section>;
}
