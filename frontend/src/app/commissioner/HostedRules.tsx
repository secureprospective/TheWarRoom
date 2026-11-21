import { useEffect, useId, useRef, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { GovernedGlyph } from '../governed/GovernedGlyph';
import type { NativeRequest } from '../commands/native';
import { newKey } from './forms';
import { draftEditor } from './draftEditor';
import { RuleInput } from './RuleInput';
import { draftFrom, editDraft, toNatural } from './ruleDraft';
import type { RuleRow, Rules } from './rules';
import type { HostedLeagueRoom } from './hostedRoom';
import { surfaceActions } from './surfaceActions';
import { Review } from './HostedReview';
import SideTray from './SideTray';
import './commissioner.css';

export const hostedRule = (field: string) =>
  /^(lineup|roster|ir|taxi|draft|bids|waivers|trades|standings|rollover)\./.test(field);

export const RULE_BOARDS = ['Structure', 'Rosters', 'Scoring', 'Money', 'Moves', 'Draft', 'Other'] as const;
export type RuleBoard = (typeof RULE_BOARDS)[number];

export const SECTION_BOARD: Record<string, RuleBoard> = {
  'League shape': 'Structure', 'Player pools': 'Structure',
  Roster: 'Rosters', Lineup: 'Rosters', IR: 'Rosters', 'Taxi squad': 'Rosters',
  Scoring: 'Scoring', Standings: 'Scoring',
  'Salary cap': 'Money', Contracts: 'Money',
  Waivers: 'Moves', 'Free-agent bids': 'Moves', Trades: 'Moves',
  Schedule: 'Structure', Playoffs: 'Structure', Calendar: 'Structure',
  Draft: 'Draft',
};

export function ruleSource(row: RuleRow): string {
  if (row.governed) return row.governed.who === 'commissioner' ? 'YOU' : row.governed.who;
  if (/import/i.test(row.citation)) return 'IMPORTED';
  return row.citation || 'Starting value';
}

/** Shared state and side-tray registration for viewing and editing hosted league rules. */
export function useHostedRules(rules: Rules | undefined, room: HostedLeagueRoom) {
  const [selection, setSelection] = useState<{ field: string; opener: HTMLElement }>();
  const screen = useId();
  const background = useRef<HTMLDivElement>(null);
  const close = () => { room.cancel(); setSelection(undefined); };
  useEffect(() => surfaceActions.register(screen, args => {
    if (args.action === 'close') close();
  }), [screen, room]);
  const state = room.use();
  const row = rules?.sections.flatMap(s => s.rows).find(r => r.field === selection?.field);
  const open = (field: string, opener: HTMLElement) => {
    room.cancel();
    setSelection({ field, opener });
  };
  return { selection, open, close, screen, background, state, row };
}

/** Embedded rule-change editor and review tray, shared by the old room and v3 Rules tab. */
export function HostedRuleEditorTray({
  rules, row, room, opener, screen, background, onClose, review,
}: {
  rules: Rules; row: RuleRow; room: HostedLeagueRoom; opener: HTMLElement;
  screen: string; background: HTMLElement | null; onClose: () => void; review: boolean;
}) {
  return <SideTray title={`Change ${row.label}`} screen={screen}
    opener={opener} background={background} onClose={onClose}
    closeLabel="Close rule editor">
    <RuleEditor key={`${row.field}:${rules.revision}`} row={row}
      rules={rules} room={room} disabled={!room.canChange() || review} />
    {review && <Review room={room} />}
  </SideTray>;
}

export function HostedRules({ rules, commissioner, room }: {
  rules: Rules; commissioner: boolean; room: HostedLeagueRoom;
}) {
  const { selection, open, close, screen, background, state, row } = useHostedRules(rules, room);
  return <>
    <div ref={background}>
      <h3>League rules</h3>
      <p>Settings come from the league server. Sources and timing come from recorded rule data.</p>
      {!commissioner && <p>Your commissioner controls these rules. Ask them for a ruling if a value is
        missing.</p>}
      {RULE_BOARDS.map(board => {
        const rows = rules.sections.flatMap(section => section.rows
          .filter(() => (SECTION_BOARD[section.label] ?? 'Other') === board)
          .map(r => ({ row: r, section: section.label })));
        if (!rows.length) return null;
        return <section key={board} aria-label={`${board} rules`}>
          <h4>{board}</h4>
          <dl>{rows.map(({ row: r, section }) => <div key={r.field}>
            <dt>{r.label}{r.governed && <GovernedGlyph governed={r.governed} subject={r.label} />}</dt>
            <dd>{r.value} <span className="hosted-rule-source">{ruleSource(r)}</span>
              {r.pending.map(pending => <span className="hosted-rule-pending" key={pending.entry}>
                Takes effect: {pending.when}</span>)}
              {commissioner && hostedRule(r.field) && <HQAct disabled={!room.canChange()}
                onAction={() => open(r.field, document.activeElement as HTMLElement)}>
                Change {r.label}</HQAct>}</dd>
            <small>{section}</small>
          </div>)}</dl>
        </section>;
      })}
    </div>
    {commissioner && row && selection && <HostedRuleEditorTray rules={rules} row={row} room={room}
      opener={selection.opener} screen={screen} background={background.current} onClose={close}
      review={Boolean(state.review)} />}
  </>;
}

function RuleEditor({ row, rules, room, disabled }: {
  row: RuleRow; rules: Rules; room: HostedLeagueRoom; disabled: boolean;
}) {
  const percent = row.unit === 'percent';
  const [draft, setDraft] = useState(() => draftFrom(row.schema, row.current, percent));
  const [effective, setEffective] = useState<'now' | 'next_rollover'>(row.nowOnly ? 'now' : 'next_rollover');
  const [reason, setReason] = useState('');
  useEffect(() => {
    draftEditor.handler = edit => setDraft(d => editDraft(row.schema, d, edit));
    return () => { draftEditor.handler = undefined; };
  }, [row]);
  const typed = toNatural(row.schema, draft, row.label, percent);
  const preview = () => {
    if (!typed.ok || !reason.trim()) return;
    return room.review({ command: 'commish.rules.set', mode: 'preview', arguments: {
      league: rules.league, command: 'commish.rules.set', args: { field: row.field, value: typed.value },
      reason: reason.trim(), expected: rules.revision, request: newKey(), effective, visibility: 'league',
    } } as NativeRequest, `Change ${row.label}`);
  };
  return <section className="league-form" aria-label={`Change ${row.label}`}>
    <p>In force now: {row.value}</p>
    <fieldset disabled={disabled}>
      <RuleInput schema={row.schema} draft={draft} onChange={setDraft}
        label={`New value for ${row.label}`} percent={percent} />
      <label>When it takes effect<select value={effective}
        onChange={e => setEffective(e.target.value as typeof effective)}>
        {!row.nowOnly && <option value="next_rollover">From the next rollover</option>}
        <option value="now">Now</option>
      </select></label>
      {effective === 'next_rollover' && <p>{rules.rollover}</p>}
      <label>Reason for the rule change<textarea value={reason}
        onChange={e => setReason(e.target.value)} /></label>
    </fieldset>
    {!typed.ok && <p role="alert">{typed.error}</p>}
    <HQAct disabled={disabled || !typed.ok || !reason.trim()} onAction={preview}>Preview rule change</HQAct>
  </section>;
}
