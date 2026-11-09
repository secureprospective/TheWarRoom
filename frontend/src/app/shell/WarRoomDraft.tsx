import { readWarRoomDraft } from './warroomDraftReads';
import { HQAct as WarRoomAct } from './HQAct';
import { lazy, Suspense, useState } from 'react';
import type { WarRoomContext } from './warroomReads';
import { useWarRoomRead } from './warroomReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { unavailable } from './hqModel';
const DraftRoom = lazy(() => import('../draftroom/DraftRoom'));
export function WarRoomDraft({ context, select }: { context: WarRoomContext;
  select: (id: string, name: string, position?: string) => void }) {
  if (context.native?.embedded) return <section className="native-draft ws on" aria-label="Draft room">
    <div className="ch"><b>DRAFT ROOM</b><span>league draft and actions</span></div>
    <Suspense fallback={<p role="status">Reading the draft…</p>}>
      <DraftRoom league={context.native.league} />
    </Suspense>
  </section>;
  return <DraftReading context={context} select={select} />;
}
function DraftReading({ context, select }: { context: WarRoomContext;
  select: (id: string, name: string, position?: string) => void }) {
  const reading = useWarRoomRead(JSON.stringify([context.native?.league, context.mine]), () =>
    readWarRoomDraft(context));
  const draft = reading.reading?.draft;
  const mirror = reading.reading?.board;
  const results = mirror?.reports.find(r => r.id === 'M-183') ??
    mirror?.reports.find(r => r.id === 'M-063');
  const resultsRows = results && ['ok', 'stale'].includes(results.status) ? results.rows : [];
  const teamName = context.snapshot.franchises.value.find(f => f.id === context.mine)?.name;
  const draftWords = draft?.words ?? mirror?.reports.find(r => r.id === 'M-176')?.rows
    .map(r => r.cells.join(' · ')).join(' · ');
  const [round, setRound] = useState('All');
  const [mine, setMine] = useState(false);
  const name = (id: string) => draft?.owners.find(o => o.id === id)?.name ?? id;
  return <section className="dr5 ws on" aria-label="Draft room"><div className="clockbar">
    <div className="cb-k"><span className="lbl">DRAFT</span><b>{draftWords || unavailable}</b>
      <span>{draft?.start ? new Date(draft.start).toLocaleString() : unavailable}</span></div>
    <div className="mpk"><span className="lbl">MY PICKS</span>{draft?.order.flatMap(r => r.picks)
      .filter(p => p.owner === context.mine).map(p => <b key={`${p.round}:${p.original}`}>{p.round}.{p.slot}
        {' · '}{name(p.original)}</b>) ?? unavailable}</div>
    <div className="cb-k"><span className="lbl">ON CLOCK</span><b>{draft?.onClock ?
      `${draft.onClock.round}.${draft.onClock.slot} · ${name(draft.onClock.owner)}` : unavailable}</b>
      <span>{draft?.onClock?.deadline ? new Date(draft.onClock.deadline).toLocaleString() : unavailable}
      </span></div><div className="cb-act"><WarRoomAct className="btn" disabled title={unavailable}>
        ▸ RUN MOCK DRAFT</WarRoomAct></div></div>
    <div className="ord"><div className="ch"><b>DRAFT ORDER</b><span>{draft?.picks.length ?? unavailable}
      </span></div><div className="rtabs">{['All', ...draft?.order.map(r => String(r.round)) ??
        [...new Set(resultsRows.map(r => r.cells[0].split('.')[0]))]].map(r =>
        <WarRoomAct key={r} active={round === r} onAction={() => setRound(r)}>{r === 'All' ? r :
          `R${r}`}</WarRoomAct>)}<WarRoomAct active={mine} onAction={() => setMine(!mine)}>MINE</WarRoomAct>
      </div>
      <div className="olist">{draft ? draft.order.filter(r => round === 'All' || String(r.round) ===
        round).map(r => <div key={r.round}><div className="rh">ROUND {r.round}</div>{r.picks.filter(p =>
          !mine || p.owner === context.mine).map(p => {
          const picked = draft.picks.find(pick => pick.round === p.round && pick.original === p.original);
          const player = picked?.player ? draft.players[picked.player] : undefined;
          return <div className={`slt ${p.owner === context.mine ? 'me' : ''} ${
            draft.onClock?.round === p.round && draft.onClock.original === p.original ? 'otc' : ''}`}
            key={`${p.round}:${p.original}`}><span className="no">{p.round}.{p.slot}</span>
            <span className="tm">{name(p.owner)}{p.owner !== p.original ?
              ` ↪ ${name(p.original)}` : ''}</span>
            <WarRoomAct className="pick" onAction={() =>
              player && select(player.id, player.name, player.position)}>
              {player?.name ?? unavailable}</WarRoomAct></div>;
        })}</div>) : resultsRows.length ? resultsRows.filter(r => (round === 'All' ||
          r.cells[0].startsWith(`${round}.`)) && (!mine || r.cells[1] === teamName)).map((r, i) =>
          <div className={`slt ${r.cells[1] === teamName ? 'me' : ''}`} key={i}>
            <span className="no">{r.cells[0]}</span><span className="tm">{r.cells[1]}</span>
            <span className="pick">{r.cells[2]}</span></div>) :
            <WarRoomEmpty reason={reading.error} />}</div><div className="legend2">
        <span>▌ mine</span><span>↪ traded</span><span>PROJ = from standings</span></div></div>
    <div className="board5"><div className="ch"><b>MY BOARD</b><span>drafted players leave the board</span>
      <WarRoomAct className="btn tgl" disabled title={unavailable}>MY NOTES ONLY</WarRoomAct></div>
      <div className="tiers">{draft?.available.length ? <div className="tier"><div className="hd">
        {unavailable}</div>
      <div className="band">{draft.available.map(p => <WarRoomAct className="pros" key={p.id}
          onAction={() => select(p.id, p.name, p.position)}><span className="ps">{p.position}</span>
          <span className="nm">{p.name}</span><span className="adp">{unavailable}</span></WarRoomAct>)}</div>
      </div> : <WarRoomEmpty />}
      </div></div><div className="dq"><div className="ch"><b>MY QUEUE</b>
        <span>auto-pick follows this order</span></div><div className="qz"><WarRoomEmpty /></div>
        <div className="legend2"><span>POS need</span><span>N my notes</span><span>click = report</span>
        </div></div>
  </section>;
}
