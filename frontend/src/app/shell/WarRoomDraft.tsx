import { useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { parseDraft, type Draft } from '../draftroom/draft';
import { provider as reports } from '../draftreports/api';
import type { Board } from '../reports/contract';
import type { WarRoomContext } from './warroomReads';
import { useWarRoomRead } from './warroomReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { unavailable } from './hqModel';
export async function readWarRoomDraft(c: WarRoomContext): Promise<{ draft?: Draft; board?: Board }> {
  if (!c.native) return { board: await reports.read() };
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'draft.read', mode: 'read',
    arguments: { league: c.native.league, franchise: c.mine, season: 0,
      at: '0001-01-01T00:00:00Z' } });
  const draft = parseDraft(data);
  if (draft.league !== c.native.league) throw new Error('Draft belongs to another league');
  return { draft };
}
export function WarRoomDraft({ context, select }: { context: WarRoomContext;
  select: (id: string, name: string, position?: string) => void }) {
  const reading = useWarRoomRead(JSON.stringify([context.native?.league, context.mine]), () =>
    readWarRoomDraft(context));
  const draft = reading.reading?.draft;
  const [round, setRound] = useState('All');
  const [mine, setMine] = useState(false);
  const name = (id: string) => draft?.owners.find(o => o.id === id)?.name ?? id;
  return <section className="dr5 ws on" aria-label="Draft room"><div className="clockbar">
    <div className="cb-info"><span className="lbl">DRAFT</span><b>{draft?.words ?? unavailable}</b>
      <span>{draft?.start ? new Date(draft.start).toLocaleString() : unavailable}</span></div>
    <div className="cb-picks"><span className="lbl">MY PICKS</span>{draft?.order.flatMap(r => r.picks)
      .filter(p => p.owner === context.mine).map(p => <b key={`${p.round}:${p.original}`}>{p.round}.{p.slot}
        {' · '}{name(p.original)}</b>) ?? unavailable}</div>
    <div className="cb-turn"><span className="lbl">ON CLOCK</span><b>{draft?.onClock ?
      `${draft.onClock.round}.${draft.onClock.slot} · ${name(draft.onClock.owner)}` : unavailable}</b>
      <span>{draft?.onClock?.deadline ? new Date(draft.onClock.deadline).toLocaleString() : unavailable}
      </span></div><div className="cb-act"><button className="btn" disabled title={unavailable}>
        ▸ RUN MOCK DRAFT</button></div></div>
    <div className="ord"><div className="ch"><b>DRAFT ORDER</b><span>{draft?.picks.length ?? unavailable}
      </span></div><div className="rtabs">{['All', ...draft?.order.map(r => String(r.round)) ?? []].map(r =>
        <button key={r} aria-pressed={round === r} onClick={() => setRound(r)}>{r === 'All' ? r :
          `R${r}`}</button>)}<button aria-pressed={mine} onClick={() => setMine(!mine)}>MINE</button></div>
      <div className="olist">{draft ? draft.order.filter(r => round === 'All' || String(r.round) ===
        round).map(r => <div key={r.round}><div className="rh">ROUND {r.round}</div>{r.picks.filter(p =>
          !mine || p.owner === context.mine).map(p => {
          const picked = draft.picks.find(pick => pick.round === p.round && pick.original === p.original);
          const player = picked?.player ? draft.players[picked.player] : undefined;
          return <div className={`orow ${p.owner === context.mine ? 'mine' : ''}`}
            key={`${p.round}:${p.original}`}><span className="n">{p.round}.{p.slot}</span>
            <span>{name(p.owner)}{p.owner !== p.original ? ` ↪ ${name(p.original)}` : ''}</span>
            <button onClick={() => player && select(player.id, player.name, player.position)}>
              {player?.name ?? unavailable}</button></div>;
        })}</div>) : <WarRoomEmpty reason={reading.error} />}</div><div className="legend2">
        <span>▌ mine</span><span>↪ traded</span><span>PROJ = from standings</span></div></div>
    <div className="board5"><div className="ch"><b>MY BOARD</b><span>drafted players leave the board</span>
      <button className="btn tgl" disabled title={unavailable}>MY NOTES ONLY</button></div>
      <div className="tiers">{draft?.available.length ? <div className="tier"><div className="hd">
        {unavailable}</div>{draft.available.map(p => <button className="pros" key={p.id}
          onClick={() => select(p.id, p.name, p.position)}><span className="ps">{p.position}</span>
          <span className="nm">{p.name}</span><span className="adp">{unavailable}</span></button>)}
      </div> : reading.reading?.board ? reading.reading.board.reports.map(r => <div className="tier"
        key={r.id}><div className="hd">{r.title}</div>{r.rows.map((row, i) => <div className="pros"
          key={i}><span className="nm">{row.cells.join(' · ')}</span></div>)}</div>) : <WarRoomEmpty />}
      </div></div><div className="dq"><div className="ch"><b>MY QUEUE</b>
        <span>auto-pick follows this order</span></div><div className="qz"><WarRoomEmpty /></div>
        <div className="legend2"><span>POS need</span><span>N my notes</span><span>click = report</span>
        </div></div>
  </section>;
}
