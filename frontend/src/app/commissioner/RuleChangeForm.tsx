import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { draftEditor } from './draftEditor';
import { bodyOf, useRequestKey, type Change } from './forms';
import { PreviewReview } from './PreviewReview';
import type { Preview } from './preview';
import { leagueState, readPreview, refusalText } from './proxy';
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
    reason: reason.trim(), expected: rules.revision, effective, visibility: 'league',
  } : null;
  const request = useRequestKey(change);
  const body = change ? bodyOf(change, request) : '';
  const [review, setReview] = useState<{ body: string; preview?: Preview; error?: string }>();
  useEffect(() => {
    if (!body) return;
    let active = true;
    void readPreview(body).then(preview => {
      if (preview.target.league !== change?.league || preview.target.version !== change.expected)
        throw new Error('The preview does not match this league version. Reopen the change.');
      if (active) setReview({ body, preview });
    }).catch(cause => { if (active) setReview({ body, error: refusalText(cause) }); });
    return () => { active = false; };
  }, [body]);
  const current = review?.body === body ? review : undefined;
  const preview = current?.preview;
  const blocked = !row ? 'Choose a rule to change.' : typed && !typed.ok ? typed.error :
    current?.error || (!preview ? 'Reading the change preview…' : preview.blockers.join(' '));
  return (
    <section className="league-form" aria-label="Change a rule">
      {!preview && <h5>Change a rule</h5>}
      <PreviewReview preview={preview} confirm={
        <ChangeBar form="rule" change={change} reason={reason} onReason={setReason} request={request}
          safeguard={preview?.safeguard ?? 'hold'} confirmLabel="Record this rule change" blocked={blocked} />
      }>
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
      </PreviewReview>
    </section>
  );
}
