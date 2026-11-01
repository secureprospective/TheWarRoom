import { HQAct as WarRoomAct } from './HQAct';
import type { WarRoomHeaderReading } from './warroomReads';
import type { WarRoomEvent } from './warroomCalendar';
import { countdown } from '../clock/urgency';
import { money, placeLine, unavailable } from './hqModel';
export function CompactAbsent() {
  return <span title={unavailable} aria-label={unavailable}>—</span>;
}
export function WarRoomEmpty({ reason }: { reason?: string }) {
  return <p className="empty" title={reason ? unavailable : undefined}>{unavailable}</p>;
}
export function WarRoomHeader({ name, reading, windowLabel }: { name?: string;
  reading?: WarRoomHeaderReading; windowLabel?: string }) {
  return <section className="franchise-context" aria-label="Franchise context">
    <div className="who"><b>{name ?? unavailable}</b><div title={reading?.standingsNote}>{placeLine(
      reading?.record, reading?.standing, reading?.standingsNote)} · window {windowLabel || unavailable}</div>
    </div>
    <div className="mini"><span className="h">DIRECTION</span><div className="post">
      {['Win now', 'Retool', 'Rebuild'].map(p => <WarRoomAct key={p} disabled title={unavailable}
        active={false}>{p}</WarRoomAct>)}</div><small>Not set</small></div>
    <div style={{ display: 'flex', gap: 28, alignItems: 'center' }}>
      <div className="mini"><span className="h">{reading?.starterN === undefined ? 'BEST' :
        `Best ${reading.starterN} (any position)`}</span><div><b title={reading?.valueNote}>
          {reading?.valuesPreparing ? <span role="status">Values are being prepared</span> :
            reading?.best ?? <CompactAbsent />}</b>
        <small>{!reading?.valuesPreparing && (reading?.rank ?? <CompactAbsent />)}</small></div></div>
      <div className="mini"><span className="h">VALUE BY AGE</span>
        {reading?.valuesPreparing ? <p className="empty">Values are being prepared</p> :
          reading?.age?.some(b => b.players > 0) ? <dl className="value-ages"
          aria-label="Dynasty value by age">
          {reading.age.map(b => <div key={b.from}><dt>{b.open ? `${b.from}+` : b.from === 0 ?
            `≤${b.to}` : `${b.from}–${b.to}`}</dt><dd title={`${b.players} players`}>{b.value.toFixed(1)}</dd>
          </div>)}</dl> : <WarRoomEmpty />}</div></div>
    {/* Nothing computes next-season needs yet; say so rather than implying the read failed. */}
    <div className="mini"><span className="h">NEEDS · NEXT SEASON</span>
      <p className="empty" title="No engine works out next-season needs yet.">Not built yet</p></div>
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
    failed ? <WarRoomEmpty reason={reason} /> : <p className="empty">No dates yet</p>}</div>;
}
// The desk strip speaks only when a read failed. Working reads leave it quiet rather than filling it
// with words that look like an error.
export function WarRoomTrayMessage({ problems }: { problems: string[] }) {
  const held = problems.filter(Boolean);
  return <span className="msg" title={held.join(' · ')}>{held.length ? unavailable : ''}</span>;
}
