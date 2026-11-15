import { useEffect, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { GovernedGlyph } from '../governed/GovernedGlyph';
import type { NativeRequest } from '../commands/native';
import { newKey } from './forms';
import { draftEditor } from './draftEditor';
import { RuleInput } from './RuleInput';
import { draftFrom, editDraft, toNatural } from './ruleDraft';
import type { RuleRow, Rules } from './rules';
import type { HostedLeagueRoom } from './hostedRoom';

const hostedRule = (field: string) =>
  /^(lineup|roster|ir|taxi|draft|bids|waivers|trades|standings|rollover)\./.test(field);

export function HostedRules({ rules, commissioner, room }: {
  rules: Rules; commissioner: boolean; room: HostedLeagueRoom;
}) {
  const [field, setField] = useState('');
  const state = room.use();
  const row = rules.sections.flatMap(s => s.rows).find(r => r.field === field);
  return <>
    <h3>Rules in force</h3>
    <p>Your commissioner controls these rules. Ask them for a ruling if a value is missing.</p>
    {rules.sections.filter(s => s.rows.some(r => hostedRule(r.field))).map(s =>
      <section key={s.label}>
        <h4>{s.label}</h4>
        <dl>{s.rows.filter(r => hostedRule(r.field)).map(r => <div key={r.field}>
          <dt>{r.label}{r.governed && <GovernedGlyph governed={r.governed} subject={r.label} />}</dt>
          <dd>{r.value}{commissioner && <HQAct disabled={!room.canChange()}
            onAction={() => { room.cancel(); setField(r.field); }}>Change {r.label}</HQAct>}</dd>
        </div>)}</dl>
      </section>)}
    {commissioner && row && <RuleEditor key={`${row.field}:${rules.revision}`} row={row}
      rules={rules} room={room} disabled={!room.canChange() || Boolean(state.review)}
      close={() => { room.cancel(); setField(''); }} />}
  </>;
}

function RuleEditor({ row, rules, room, disabled, close }: {
  row: RuleRow; rules: Rules; room: HostedLeagueRoom; disabled: boolean; close: () => void;
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
    <h4>Change {row.label}</h4>
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
    <HQAct disabled={room.peek().pending} onAction={close}>Close rule editor</HQAct>
  </section>;
}
