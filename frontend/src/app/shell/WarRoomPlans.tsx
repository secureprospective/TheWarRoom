import { HQAct as WarRoomAct } from './HQAct';
import { WarRoomEmpty } from './WarRoomHeader';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import { readLookingBack } from './warroomPlansReads';
const CHIP: Record<string, string> = { accepted: 's-h', declined: 's-o', vetoed: 's-o' };
export function WarRoomPlans({ context, openPocket }: { context: WarRoomContext; openPocket: () => void }) {
  const back = useWarRoomRead(JSON.stringify(['plans-back', context.native?.league, context.mine]), () =>
    readLookingBack(context));
  const b = back.reading;
  return <section id="ws-plans" className="three ws on" aria-label="Plans"><div><div className="ch">
    <b>PLANS</b><span>only you see these</span></div><WarRoomEmpty /><WarRoomAct className="pl" disabled>
      <b>+ New plan</b>
      <small>Not available yet</small>
      </WarRoomAct>
      <div className="pneeds">
      <div className="ch">
        <b>MY NEEDS</b><span>ranked</span></div><WarRoomEmpty /></div></div>
    <div><div className="ch"><b>PLANS</b><span>check steps off as you think them through · drag between events
      </span></div><div className="cols"><WarRoomEmpty /></div></div>
    <div className="sum3"><div className="ch"><b>WHERE THINGS STAND</b><span>live from the other tabs</span>
      </div>{['TRADE THREADS', 'POCKET', 'MARKET', 'DRAFT', 'CAP IF THIS WORKS'].map(label => <div
        className="sblk" key={label}><div className="sh">{label}</div><WarRoomEmpty /></div>)}
      <div className="ch"><b>CONFLICTS</b></div><WarRoomEmpty />
      <div className="ch" style={{ borderTop: '2px solid var(--black)' }}><b>LOOKING BACK</b>
        <span>how past talks ended</span></div>
      <div className="sblk" aria-label="Past offers"><div className="sh">PAST OFFERS</div>
        {b?.offers.length ? b.offers.map(o => <div className="srow" key={o.id}><span>{o.partner} · {o.date}</span>
          <small className={CHIP[o.status] ?? 's-t'}>{o.status.toUpperCase()}</small></div>)
          : <WarRoomEmpty pending={back.pending} reason={back.error ?? b?.offersNote ?? 'No finished offers yet'} />}</div>
      <div className="sblk" aria-label="Counterparties"><div className="sh">COUNTERPARTIES</div>
        {b?.partners.length ? b.partners.map(p => <div className="srow" key={p.name}><span>{p.name}</span>
          <b className="mono">{p.talks} talk{p.talks === 1 ? '' : 's'} · {p.deals} deal{p.deals === 1 ? '' : 's'}</b>
          </div>) : <WarRoomEmpty pending={back.pending} reason={back.error ?? b?.offersNote ?? 'No trade talks yet'} />}</div>
      <div className="sblk" aria-label="Saved proposals"><div className="sh">SAVED PROPOSALS
        <WarRoomAct onAction={openPocket}>POCKET ▸</WarRoomAct></div>
        {b?.saved !== undefined ? <div className="srow"><span>{b.saved} built, not sent</span>
          <small className="s-t">{b.saved}</small></div>
          : <WarRoomEmpty pending={back.pending} reason={back.error ?? b?.savedNote} />}</div>
      </div></section>;
}
