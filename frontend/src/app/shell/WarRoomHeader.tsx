import { HQAct as WarRoomAct } from './HQAct';
import type { WarRoomHeaderReading } from './warroomReads';
import type { WarRoomEvent } from './warroomCalendar';
import { countdown } from '../clock/urgency';
import { absent, money, placeLine, unavailable } from './hqModel';
import { directionLabel, directions, type Direction, type WarRoomLens } from './warroomLens';
export function CompactAbsent({ pending }: { pending?: boolean } = {}) {
  return <span title={absent(pending)} aria-label={absent(pending)}>—</span>;
}
export function WarRoomEmpty({ reason, pending }: { reason?: string; pending?: boolean }) {
  return <p className="empty" role={pending ? 'status' : undefined}
    >{pending ? absent(true) : reason || unavailable}</p>;
}
export function WarRoomHeader({ name, reading, windowLabel, lens, choose, pending }: { name?: string;
  reading?: WarRoomHeaderReading; windowLabel?: string; lens?: WarRoomLens;
  choose?: (d: Direction) => void; pending?: boolean }) {
  const you = lens && !lens.preparing ? lens.you : undefined;
  const ages = reading?.age ?? [];
  const peak = Math.max(0, ...ages.map(b => b.value));
  const band = (i: number) => ages[i].open ? `${ages[i].from}+` : i === 0 && ages.length > 1 ?
    `≤${ages[1].from - 1}` : `${ages[i].from}`;
  return <section className="franchise-context" aria-label="Franchise context">
    <div className="who"><b>{name ?? unavailable}</b><div title={reading?.standingsNote}>{placeLine(
      reading?.record, reading?.standing, reading?.standingsNote)}{!windowLabel ? '' :
      windowLabel === 'none open' ? ' · no window open' : ` · open now: ${windowLabel}`}</div>
    </div>
    <div className="mini"><span className="h">DIRECTION{you && !you.set &&
      <small>our read: {directionLabel[you.direction].toLowerCase()} · pick yours</small>}</span>
      <div className="post">{directions.map(d => <WarRoomAct key={d} disabled={!choose || !you || pending}
        title="Your direction is shown to the whole league. Your lens follows it."
        active={Boolean(you?.set && you.direction === d)} onAction={() => choose?.(d)}>
        {directionLabel[d].toUpperCase()}</WarRoomAct>)}</div></div>
    <div style={{ display: 'flex', gap: 28, alignItems: 'center' }}>
      <div className="mini"><span className="h">{reading?.starterN === undefined ? 'BEST' :
        `Best ${reading.starterN} (any position)`}</span><div><b title={reading?.valueNote}>
          {reading?.valuesPreparing ? <span role="status">Values are being prepared</span> :
            reading?.best ?? <CompactAbsent />}</b>
        <small>{!reading?.valuesPreparing && (reading?.rank ?? <CompactAbsent />)}</small></div></div>
      <div className="mini"><span className="h">VALUE BY AGE</span>
        {reading?.valuesPreparing ? <p className="empty">Values are being prepared</p> :
          ages.some(b => b.players > 0) ? <div role="img" aria-label={`Long-term value by age: ${
            ages.map((b, i) => `${band(i)} ${b.value.toFixed(1)}`).join(', ')}`}>
          <div className="agec">{ages.map((b, i) => <i key={b.from} className={b.value >= peak * 0.8 ?
            'peak' : ''} style={{ height: `${Math.max(4, b.value / peak * 100)}%` }}
            title={`Age ${band(i)}: ${b.value.toFixed(1)} long-term value · ${b.players} players`} />)}</div>
          <div className="agek">{ages.map((b, i) => <span key={b.from}>{band(i)}</span>)}</div></div> :
          <WarRoomEmpty />}</div></div>
    <div className="mini"><span className="h">NEEDS · NEXT SEASON</span>
      {you?.needs.length ? <div className="needs">{you.needs.map(n => <span key={n.position}
        className={`nd ${n.state === 'ok' ? '' : 'hole'}`} title={`${n.have} starter-grade ${n.position
        } signed for next season · the lineup starts ${n.starts}`}>{n.position}<small>{n.state}</small>
      </span>)}</div> : <p className="empty">{lens?.preparing ? 'Values are being prepared' :
        absent(!lens)}</p>}</div>
    <div className="mini"><span className="h">CAP RUNWAY</span><div className="runw">
      {reading?.runway?.length ? reading.runway.slice(0, 3).map(y => <div className="y fit" key={y.season}>
        <span>{y.season}</span><span className="bar">{y.max && y.room !== undefined ? <i
          style={{ width: `${Math.min(100, Math.max(0, (y.max - y.room) / y.max * 100))}%` }} /> : null}
        </span><span>{money(y.room)}</span></div>) : <WarRoomEmpty />}</div></div>
  </section>;
}

// A league with no dated window is not a failed read: it has no dates yet. The unavailable words are
// kept for a calendar that could not be read, with the reason in the hover.
export function WarRoomDates({ events, failed, reason }: { events: WarRoomEvent[]; failed?: boolean;
  reason?: string }) {
  return <div className="cal" aria-label="Calendar">{events.length ? events.map(d => <div className="ev"
    key={d.id}><span className="w">{d.at ? new Date(d.at).toLocaleDateString() : unavailable}</span>
    <b>{d.label}</b><span className="c">{countdown(d.at, Date.now())}</span></div>) :
    failed ? <p className="empty" title={reason}>{unavailable}</p> :
      <p className="empty">No dates yet</p>}</div>;
}
// The desk strip speaks only when a read failed. Working reads leave it quiet rather than filling it
// with words that look like an error.
export function WarRoomTrayMessage({ problems }: { problems: string[] }) {
  const held = problems.filter(Boolean);
  return <span className="msg" title={held.join(' · ')}>{held.length ? unavailable : ''}</span>;
}
