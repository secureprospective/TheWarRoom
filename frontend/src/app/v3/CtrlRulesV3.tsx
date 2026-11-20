// Desktop v3 Control Room, Rules tab: the league rules (the old room's own HostedRules, with its change
// editor and preview), plus rule sections and pending changes counted from the same rules read (`c`).
import type { ReactNode } from 'react';
import { HostedRules } from '../commissioner/HostedRules';
import type { Rules } from '../commissioner/rules';
import { when } from '../control/room/room';
import type { ControlBodyProps } from './ControlRoomV3';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

export type SectionCount = { label: string; rules: number; needsRuling: number };
export type PendingChange = { label: string; value: string; when: string; entry: string };

/** Each rules section with its rule count and how many still need a ruling. */
export function sectionCounts(rules: Rules | undefined): SectionCount[] {
  return (rules?.sections ?? []).filter(s => s.rows.length > 0).map(s => ({
    label: s.label, rules: s.rows.length,
    needsRuling: s.rows.filter(r => r.state === 'Needs a ruling').length,
  }));
}

/** Every rule change waiting to take effect, in the order the read lists them. */
export function pendingChanges(rules: Rules | undefined): PendingChange[] {
  return (rules?.sections ?? []).flatMap(s => s.rows.flatMap(r => r.pending
    .map(p => ({ label: r.label, value: p.value, when: p.when, entry: p.entry }))));
}

export function CtrlRulesV3({ c, badge }: ControlBodyProps) {
  const rules = c.roomState.reading?.rules;
  const sections = sectionCounts(rules);
  const pending = pendingChanges(rules);
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.6fr) minmax(0,1fr)',
    gridTemplateRows: 'minmax(0,1fr) minmax(0,1fr)' }}>
    <Panel title="League rules" badge={badge} style={{ gridRow: '1/3' }}
      foot={rules ? `${rules.name} · revision ${rules.revision}` : 'Reading league rules…'}>
      {c.roomState.error && <p role="alert">{c.roomState.error}</p>}
      {c.roomState.stale &&
        <p role="status">Rules are read-only until refreshed from the league server.</p>}
      {c.roomState.message && <p role="status">{c.roomState.message}</p>}
      {rules ? <HostedRules rules={rules} commissioner room={c.room} />
        : <p role="status">{c.roomState.error || 'Reading league rules…'}</p>}
    </Panel>
    <Panel title="Pending changes" badge={badge}
      tag={rules ? <span className="tag o">{pending.length}</span> : null}
      foot={rules ? `next rollover: ${rules.rollover}` : undefined}>
      {!rules ? <St>Reading league rules…</St> : !pending.length ? <St>No rule change is waiting.</St>
        : pending.map(p => <div key={`${p.label}:${p.entry}`} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
          <span className="ell"><b>{p.label}</b> <span className="d">{p.value}</span></span>
          <span className="m">{when(p.when)}</span></div>)}
    </Panel>
    <Panel title="Rule sections" badge={badge} foot="counted from the same rules read">
      {!rules ? <St>Reading league rules…</St> : !sections.length ? <St>No rules to count.</St>
        : sections.map(s => <div key={s.label} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
          <span className="k">{s.label.toUpperCase()}</span>
          <span className="v">{s.rules}</span>
          {s.needsRuling > 0 ? <span className="tag o">{s.needsRuling} need a ruling</span> : <span />}
        </div>)}
    </Panel>
  </div></div>;
}
