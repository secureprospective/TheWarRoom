import type { WarRoomHeaderReading } from './warroomReads';
import { money, unavailable } from './hqModel';
export function WarRoomEmpty({ reason }: { reason?: string }) {
  return <p className="empty" title={reason}>{unavailable}</p>;
}
export function WarRoomHeader({ name, reading }: { name?: string; reading?: WarRoomHeaderReading }) {
  return <section className="table" aria-label="Franchise context">
    <div className="who"><b>{name ?? unavailable}</b><div>{reading?.record ?? unavailable} · {
      reading?.standing ?? unavailable} · window {unavailable}</div></div>
    <div className="mini"><span className="h">DIRECTION</span><div className="post">
      {['CONTEND', 'RETOOL', 'REBUILD'].map(p => <button key={p} disabled title={unavailable}
        aria-pressed={false}>{p}</button>)}</div><small>{unavailable}</small></div>
    <div style={{ display: 'flex', gap: 28, alignItems: 'center' }}>
      <div className="mini"><span className="h">{reading?.starterN === undefined ? 'BEST' :
        `Best ${reading.starterN} (any position)`}</span><div><b>{reading?.best ?? unavailable}</b>
        <small>{reading?.rank ?? unavailable}</small></div></div>
      <div className="mini"><span className="h">VALUE BY AGE</span><WarRoomEmpty /></div></div>
    <div className="mini"><span className="h">NEEDS · NEXT SEASON</span><WarRoomEmpty /></div>
    <div className="mini"><span className="h">CAP RUNWAY</span><div className="runw">
      {reading?.runway?.length ? reading.runway.map(y => <div className="y" key={y.season}>
        <span>{y.season}</span><span className="bar">{y.max && y.room !== undefined ? <i
          style={{ width: `${Math.min(100, Math.max(0, (y.max - y.room) / y.max * 100))}%` }} /> : null}
        </span><span>{money(y.room)}</span></div>) : <WarRoomEmpty />}</div></div>
  </section>;
}
