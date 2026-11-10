import type { ReactNode } from 'react';
import { HQAct as PulseAct } from './HQAct';

/** PulseDrawer is the bottom drawer every Pulse pull opens: a title, what it covers, its own tabs
 *  and a body. */
export function PulseDrawer({ open, onClose, title, sub, tabs, children }: { open: boolean;
  onClose: () => void; title: string; sub: string; tabs?: ReactNode; children: ReactNode }) {
  return <div className={`bdrawer${open ? ' open' : ''}`}>
    <div className="bd-head">
      <h2>{title}</h2>
      <span className="sub">{sub}</span>
      {tabs && <div className="dtab">{tabs}</div>}
      <PulseAct className="x" onAction={onClose}>ESC ✕</PulseAct>
    </div>
    <div className="dbody">{children}</div>
  </div>;
}
