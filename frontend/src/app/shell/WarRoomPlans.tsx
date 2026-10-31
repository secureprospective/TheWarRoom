import { HQAct as WarRoomAct } from './HQAct';
import { WarRoomEmpty } from './WarRoomHeader';
export function WarRoomPlans() {
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
      <div className="ch"><b>CONFLICTS</b></div><WarRoomEmpty /></div></section>;
}
