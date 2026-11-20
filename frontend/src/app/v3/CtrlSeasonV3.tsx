// Desktop v3 Control Room, Season tab: the old room's season body (calendar, schedule, playoffs,
// rollover with its backup step) and the act tray its buttons open, drawn inside v3 panels.
// Reads and actions are the old body's own (HostedControlRoomSeason, useHostedControlRoom via `c`).
import { HostedCommissionerActTray } from '../commissioner/CommissionerActTray';
import { HostedControlRoomSeason } from '../commissioner/HostedControlRoomSeason';
import type { ControlBodyProps } from './ControlRoomV3';

export function CtrlSeasonV3({ c, league, port, badge }: ControlBodyProps) {
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1fr)', gridTemplateRows: 'minmax(0,1fr)' }}>
    <section className="pc panel sz-L">
      <div className="ph">{badge}<b className="pt">Season</b><span className="sp" /></div>
      <div className="pb">
        <HostedControlRoomSeason league={league} port={port} onTool={c.openSeasonTool} />
        {c.definition && <HostedCommissionerActTray definition={c.definition} league={league}
          room={c.room} port={port} />}
      </div>
      <div className="pf"><span className="ell">
        from the league server's schedule and calendar reads</span></div>
    </section>
  </div></div>;
}
