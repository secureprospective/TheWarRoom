import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { draftEditor } from './draftEditor';
import type { Change } from './forms';
import { leagueState } from './proxy';
import { ruleSource } from './rulesReport';
import { RuleInput } from './RuleInput';
import { draftFrom, editDraft, toNatural, type Draft } from './ruleDraft';
import type { RuleRow, Rules } from './rules';

type Effective = Change['effective'];
const timing: { value: Effective; label: string }[] = [
  { value: 'next_rollover', label: 'From the next rollover' }, { value: 'now', label: 'Now' },
];
const start = (row?: RuleRow): Draft | undefined =>
  row && draftFrom(row.schema, row.current, row.unit === 'percent');

function CurrentValue({ row }: { row: RuleRow }) {
  return (
    <dl className="rule-current">
      <dt>In force now</dt><dd>{row.value}</dd>
      <dt>Where it comes from</dt><dd>{ruleSource(row)}</dd>
      {row.pending.map(p => <div key={p.entry}><dt>Waiting</dt><dd>{p.value}: {p.when}</dd></div>)}
    </dl>
  );
}

// RuleChangeForm starts one rule change; the safeguard shown is the one Go reports for that rule and timing.
export function RuleChangeForm({ rules, initialField = '' }: { rules: Rules; initialField?: string }) {
  const rows = rules.sections.flatMap(s => s.rows);
  const [field, setField] = useState(initialField);
  const row = rows.find(r => r.field === field);
  const [draft, setDraft] = useState<Draft | undefined>(() =>
    start(rows.find(r => r.field === initialField)));
  const [effective, setEffective] = useState<Effective>('next_rollover');
  const [reason, setReason] = useState('');
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === 'rule' ? outcome.receipt.entry : '';
  useEffect(() => {
    if (!landed) return;
    setReason('');
    setDraft(start(rows.find(r => r.field === field)));
  }, [landed]);
  useEffect(() => {
    if (!row) return undefined;
    draftEditor.handler = edit => setDraft(d => d && editDraft(row.schema, d, edit));
    return () => { draftEditor.handler = undefined; };
  }, [row]);
  const pick = (next: string) => {
    setField(next);
    setDraft(start(rows.find(r => r.field === next)));
  };
  const typed = row && draft ? toNatural(row.schema, draft, row.label, row.unit === 'percent') : undefined;
  const change: Change | null = row && typed?.ok ? {
    league: rules.league, command: 'commish.rules.set', args: { field: row.field, value: typed.value },
    reason, expected: rules.revision, effective, visibility: 'league',
  } : null;
  return (
    <section className="league-form" aria-label="Change a rule">
      <h5>Change a rule</h5>
      <label>Rule
        <select value={field} onChange={e => pick(e.target.value)}>
          <option value="">Choose a rule…</option>
          {rules.sections.map(section => (
            <optgroup key={section.label} label={section.label}>
              {section.rows.map(r => <option key={r.field} value={r.field}>{r.label}</option>)}
            </optgroup>
          ))}
        </select>
      </label>
      {row && draft && (
        <>
          <CurrentValue row={row} />
          <RuleInput schema={row.schema} draft={draft} onChange={setDraft}
            label={`New value for ${row.label}`} percent={row.unit === 'percent'} />
          <label>When it takes effect
            <select value={effective} onChange={e => setEffective(e.target.value as Effective)}>
              {timing.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
            </select>
          </label>
          {effective === 'next_rollover' && <p className="change-note">{rules.rollover}</p>}
        </>
      )}
      <ChangeBar form="rule" change={change} reason={reason} onReason={setReason}
        safeguard={row ? (effective === 'now' ? row.safeguards.now : row.safeguards.nextRollover) : 'hold'}
        confirmLabel="Record this rule change"
        blocked={!row ? 'Choose a rule to change.' : typed && !typed.ok ? typed.error : ''} />
    </section>
  );
}
