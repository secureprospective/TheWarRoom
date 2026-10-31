import { WarRoomEmpty } from './WarRoomHeader';
export function WarRoomPlans() {
  return <section className="three ws on" aria-label="Plans"><div><div className="ch"><b>PLANS</b>
    <span>only you see these</span></div><WarRoomEmpty /><button className="pl" disabled>
      <b>+ New plan</b><small>Not available yet</small></button></div>
    <div><div className="ch"><b>PLANS</b><span>check steps off as you think them through · drag between events
      </span></div><div className="cols"><WarRoomEmpty /></div></div>
    <div><div className="ch"><b>WHERE THINGS STAND</b><span>live from the other tabs</span></div>
      <WarRoomEmpty /><div className="ch"><b>CONFLICTS</b></div><WarRoomEmpty /></div></section>;
}
