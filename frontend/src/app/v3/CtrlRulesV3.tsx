// Desktop v3 Control Room, Rules tab, after TheWarRoom-mock-v3-FINAL.html (screens.js control.rules):
// Settings board grouped into rule boards with board filter chips, source tags, governed glyphs, and the
// shared rule-change editor tray, alongside Pending changes and Rule sections from the same rules read.
import { useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { GovernedGlyph } from '../governed/GovernedGlyph';
import { HostedRuleEditorTray, RULE_BOARDS, SECTION_BOARD, hostedRule, ruleSource, useHostedRules,
  type RuleBoard } from '../commissioner/HostedRules';
import type { RuleRow, Rules } from '../commissioner/rules';
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
export type BoardRow = { row: RuleRow; section: string };
export type RuleBoardGroup = { board: RuleBoard; rows: BoardRow[] };

/** Maps a rule section label to its Settings board group. */
export function boardForSection(label: string): RuleBoard {
  return SECTION_BOARD[label] ?? 'Other';
}

/** Rules grouped into the Settings board order, skipping empty boards. */
export function ruleBoards(rules: Rules | undefined): RuleBoardGroup[] {
  const sections = rules?.sections ?? [];
  return RULE_BOARDS.map(board => ({
    board,
    rows: sections.flatMap(section => boardForSection(section.label) === board
      ? section.rows.map(row => ({ row, section: section.label })) : []),
  })).filter(group => group.rows.length > 0);
}

/** Filters grouped rule boards to the chosen board chip, or returns all groups for 'All'. */
export function filterRuleBoards(groups: RuleBoardGroup[], board: RuleBoard | 'All'): RuleBoardGroup[] {
  return board === 'All' ? groups : groups.filter(group => group.board === board);
}

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
  const [board, setBoard] = useState<RuleBoard | 'All'>('All');
  const rules = c.roomState.reading?.rules;
  const { selection, open, close, screen, background, state, row } = useHostedRules(rules, c.room);
  const groups = ruleBoards(rules);
  const visible = filterRuleBoards(groups, board);
  const totalRules = groups.reduce((sum, g) => sum + g.rows.length, 0);
  const sections = sectionCounts(rules);
  const pending = pendingChanges(rules);
  const cols = 'minmax(0,1fr) auto auto auto auto';

  return <>
    <div ref={background} className="wbody scr scr-ctrl"><div className="lay" style={{
      gridTemplateColumns: 'minmax(0,1.6fr) minmax(0,1fr)',
      gridTemplateRows: 'minmax(0,1fr) minmax(0,1fr)' }}>
      <Panel title="Settings board" badge={badge} style={{ gridRow: '1/3' }}
        tag={rules ? <span className="tag o">{totalRules}</span> : null}
        foot={rules ? `${rules.name} · revision ${rules.revision}` : 'Reading league rules…'}>
        {groups.length > 0 && <div className="fbar" aria-label="Rule boards">
          <HQAct className="chip" selected={board === 'All'} onAction={() => setBoard('All')}>
            All <span className="m">{totalRules}</span></HQAct>
          {groups.map(g => <HQAct key={g.board} className="chip" selected={board === g.board}
            onAction={() => setBoard(g.board)}>
            {g.board} <span className="m">{g.rows.length}</span></HQAct>)}
        </div>}
        {c.roomState.error && <St><span role="alert">{c.roomState.error}</span></St>}
        {c.roomState.stale && <St>
          <span role="status">Rules are read-only until refreshed from the league server.</span></St>}
        {c.roomState.message && <St><span role="status">{c.roomState.message}</span></St>}
        {!rules && !c.roomState.error && <St><span role="status">Reading league rules…</span></St>}
        {rules && visible.length === 0 && <St>No rules in this board.</St>}
        {visible.map(group => <div key={group.board} data-board={group.board}>
          <div className="sec">{group.board.toUpperCase()} · {group.rows.length}</div>
          {group.rows.map(({ row: r, section }) => <div key={r.field} className="row"
            style={{ gridTemplateColumns: cols }}>
            <span className="ell">
              <b>{r.label}</b>
              {r.governed && <> <GovernedGlyph governed={r.governed} subject={r.label} /></>}
              {' '}<span className="d">{section}</span>
              {r.pending.map(p => <span key={p.entry} className="m">
                {' '}· Takes effect: {p.when}</span>)}
            </span>
            <span className="v">{r.value}</span>
            <span className="tag o">{ruleSource(r)}</span>
            {r.state === 'Needs a ruling' ? <span className="tag am">{r.state}</span> : <span />}
            <span className="acts">
              {hostedRule(r.field) && <HQAct className="btn sm" disabled={!c.room.canChange()}
                onAction={() => open(r.field, document.activeElement as HTMLElement)}>
                Change {r.label}</HQAct>}
            </span>
          </div>)}
        </div>)}
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
    </div></div>
    {rules && row && selection && <HostedRuleEditorTray rules={rules} row={row} room={c.room}
      opener={selection.opener} screen={screen} background={background.current} onClose={close}
      review={Boolean(state.review)} />}
  </>;
}
