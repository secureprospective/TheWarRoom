import type { ReactNode } from 'react';
import { HQAct } from './HQAct';

export function HQWhatIfDrawer({ close, detail, port }: {
  close: () => void; detail: string; port: { input: ReactNode; preview: ReactNode } }) {
  return <section className="bdrawer open" aria-label="Roster drawer">
    <div className="bd-head"><h2>What-if</h2><span className="sub">{detail}</span>
      <HQAct className="x" onAction={close}>ESC</HQAct></div>
    <div className="bd-body"><div className="bd-main">{port.input}</div>
      <div className="bd-side">{port.preview}</div></div></section>;
}
