// Desktop v3 War Room, Plans, after TheWarRoom-mock-v3-FINAL.html (screens.js war.plans): looking back
// over finished trade talks and the league calendar. The mock's private plans and notes have no server
// read yet, so they are not drawn.
import type { ReactNode } from 'react';
import { useWarRoomRead, type WarRoomContext } from '../shell/warroomReads';
import { readLookingBack } from '../shell/warroomPlansReads';
import type { WarRoomEvent } from '../shell/warroomCalendar';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
const day = (at?: string) => at ? new Date(at).toLocaleString('en-US', { weekday: 'short', month: 'short',
  day: 'numeric', hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' }) + ' CT' : 'no date set';

/** Open windows first, then what is still ahead, soonest first; past events drop off. */
export function calendarRows(events: WarRoomEvent[], now: number): WarRoomEvent[] {
  const ahead = (e: WarRoomEvent) => e.live || !e.at || Date.parse(e.at) > now;
  return events.filter(ahead).sort((a, b) => Number(Boolean(b.live)) - Number(Boolean(a.live))
    || (a.at ? Date.parse(a.at) : Infinity) - (b.at ? Date.parse(b.at) : Infinity));
}

export function WarPlansV3({ context, badge, events, eventsError }: { context: WarRoomContext;
  badge: ReactNode; events: WarRoomEvent[]; eventsError?: string }) {
  const back = useWarRoomRead(JSON.stringify(['plans-back', context.native?.league, context.mine]), () =>
    readLookingBack(context));
  const b = back.reading;
  const cal = calendarRows(events, Date.now());
  return <div className="wbody scr scr-war"><div className="lay" style={{
    gridTemplateColumns: 'repeat(2,minmax(0,1fr))', gridTemplateRows: 'minmax(0,1fr)' }}>
    <section className="pc panel sz-L"><div className="ph">{badge}<b className="pt">Looking back</b>
      <span className="sp" /></div><div className="pb">
      <div className="gl"><span>PAST OFFERS</span></div>
      {b?.offers.length ? b.offers.map(o => <div key={o.id} className="row"
        style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
        <span className="ell">{o.partner}</span><span className="m">{o.date}</span>
        <span className="tag">{o.status.toUpperCase()}</span></div>)
        : <St>{back.pending ? 'Reading past offers…'
          : back.error ?? b?.offersNote ?? 'No finished offers yet.'}</St>}
      <div className="gl"><span>PARTNERS</span></div>
      {b?.partners.length ? b.partners.map(p => <div key={p.name} className="row"
        style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
        <span className="ell">{p.name}</span><span className="m">{p.talks} talk{p.talks === 1 ? '' : 's'}
          {' · '}{p.deals} deal{p.deals === 1 ? '' : 's'}</span></div>)
        : <St>{back.pending ? 'Reading…' : back.error ?? b?.offersNote ?? 'No trade talks yet.'}</St>}
      <div className="gl"><span>SAVED PROPOSALS</span></div>
      <St>{b?.saved !== undefined ? `${b.saved} built, not sent`
        : back.error ?? b?.savedNote ?? 'Reading…'}</St>
    </div><div className="pf"><span className="ell">past offers and partners · private to you</span></div>
    </section>
    <section className="pc panel sz-L"><div className="ph">{badge}<b className="pt">Calendar</b>
      <span className="sp" /></div><div className="pb">
      {eventsError ? <St>The league calendar could not be read. {eventsError}</St>
        : !cal.length ? <St>Nothing on the league calendar ahead.</St>
          : cal.map(e => <div key={e.id} className="row"
            style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
            <span className="ell">{e.label}</span>
            <span className="m">{e.live && e.closes ? `closes ${day(e.closes)}` : day(e.at)}</span>
            {e.live ? <span className="tag o">OPEN</span> : <span />}</div>)}
    </div><div className="pf"><span className="ell">the league's own dates · open windows first</span></div>
    </section>
  </div></div>;
}
