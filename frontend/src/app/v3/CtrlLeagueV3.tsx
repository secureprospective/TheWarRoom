// Desktop v3 Control Room, League tab (screens.js control.league). The tab's real reads and actions
// (league notices and notes, the post-a-notice/note form with review, the ledger and its filters)
// live in the old body's self-contained component, so it is rendered here inside a v3 panel.
import type { ReactNode } from 'react';
import { HostedControlRoomLeague } from '../commissioner/HostedControlRoomLeague';
import type { ControlBodyProps } from './ControlRoomV3';

function Panel({ title, badge, children, foot }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode }) {
  return <section className="pc panel sz-L" style={{ gridRow: '1/2' }}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" /></div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

export function CtrlLeagueV3({ c, league, port, badge }: ControlBodyProps) {
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1fr)', gridTemplateRows: 'minmax(0,1fr)' }}>
    <Panel title="League" badge={badge}
      foot="in-app only · never emailed">
      <section className="crx hosted-control-room"><div className="cr-main on hosted-control-content">
        <HostedControlRoomLeague key={league} league={league} room={c.room} port={port} />
      </div></section>
    </Panel>
  </div></div>;
}
