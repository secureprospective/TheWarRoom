import type { HQCap, HQSettings } from './hqModel';
import { capacity, money, unavailable } from './hqModel';

export function CapStrip({ cap }: { cap: HQCap }) {
  const width = (value?: number) => cap.max && value !== undefined
    ? `${Math.max(0, Math.min(100, value / cap.max * 100))}%` : '0%';
  return <div className="cap">
    <div className="cap-top"><span>CAP · {cap.season || unavailable}</span>
      <span><b>{money(cap.room)}</b> ROOM OF {money(cap.max)}</span></div>
    <div className="cap-bar"><span className="seg-c" style={{ width: width(cap.committed) }} />
      <span className="seg-d" style={{ width: width(cap.dead) }} /><span className="seg-r" style={{ flex: 1
        }} /></div>
    <div className="cap-key"><span><i style={{ background: 'var(--edge-hi)' }}
      />Committed {money(cap.committed)}</span>
      <span><i style={{ background: 'var(--ink3)' }} />Dead {money(cap.dead)}</span><span><i
        />Room {money(cap.room)}</span></div>
  </div>;
}
export function HQHeader({ name, record, standing, active, settings, cap, check, lock }: {
  name: string; record?: string; standing?: string; active: number; settings: HQSettings;
  cap: HQCap; check: string; lock?: { id: string; at?: string; text: string };
}) {
  return <section className="plate">
    <div className="who"><b>{name}</b><div>{record || unavailable} · {standing || unavailable}
      · {capacity(active, settings.active)} active</div></div>
    <CapStrip cap={cap} />
    <div className="check"><span className="h">LINEUP CHECK</span><span className="v">{check}</span>
      <span className="t" data-deadline={lock?.id}>{lock ? <>Locks in <span
        data-countdown>{lock.text}</span></> : unavailable}</span></div>
  </section>;
}
