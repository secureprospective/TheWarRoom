import { Act } from '../commands/Act';
import { blankDraft, timeUnits, type Draft, type Step, type TimeUnit } from './ruleDraft';
import type { RuleSchema } from './rules';

type Props = {
  schema: RuleSchema; draft: Draft; onChange(draft: Draft): void; label: string; percent?: boolean;
  // Where this value sits in the rule, so the add and remove verbs can find it.
  path?: Step[];
};
const distinct = (choices: RuleSchema['choices']) => choices.filter((c, i) =>
  choices.findIndex(o => o.value === c.value) === i);
// Go's bounds, in the units the box is typed in.
function bound(schema: RuleSchema, value: number | undefined, percent: boolean): number | undefined {
  if (value === undefined) return undefined;
  if (schema.kind === 'cents') return value / 100;
  return schema.kind === 'ten_thousandths' ? value / (percent ? 100 : 1e4) : value;
}
function limits(schema: RuleSchema, percent: boolean): string {
  const low = bound(schema, schema.minimum, percent), high = bound(schema, schema.maximum, percent);
  if (low !== undefined && high !== undefined) return `From ${low} to ${high}`;
  if (low !== undefined) return low > 0 ? `At least ${low}` : '';
  return high === undefined ? '' : `At most ${high}`;
}

function Choice({ schema, value, label, onChange }:
  { schema: RuleSchema; value: string; label: string; onChange(v: string): void }) {
  return (
    <select aria-label={label} value={value} onChange={e => onChange(e.target.value)}>
      <option value="">Choose…</option>
      {distinct(schema.choices).map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
    </select>
  );
}
type Part<T extends Draft['t']> = Props & { draft: Extract<Draft, { t: T }> };
function Text({ schema, draft, onChange, label, percent }: Part<'text'>) {
  const set = (v: string) => onChange({ t: 'text', v });
  const hint = limits(schema, !!percent);
  if (schema.kind === 'string' && schema.choices.length > 0)
    return <Choice schema={schema} value={draft.v} label={label} onChange={set} />;
  const number = schema.kind === 'count';
  return (
    <>
      {schema.kind === 'cents' && <span aria-hidden="true">$</span>}
      <input value={draft.v} onChange={e => set(e.target.value)}
        type={number ? 'number' : schema.kind === 'date' ? 'datetime-local' : 'text'}
        inputMode={schema.kind === 'cents' || schema.kind === 'ten_thousandths' ? 'decimal' : undefined}
        min={number ? bound(schema, schema.minimum, false) : undefined}
        max={number ? bound(schema, schema.maximum, false) : undefined} step={number ? 1 : undefined} />
      {percent && <span aria-hidden="true">%</span>}
      {hint && <small className="rule-hint">{hint}</small>}
    </>
  );
}
function Time({ draft, onChange, label }: Part<'time'>) {
  return (
    <>
      <input type="text" inputMode="decimal" value={draft.v}
        onChange={e => onChange({ ...draft, v: e.target.value })} />
      <select aria-label={`${label}: unit`} value={draft.unit}
        onChange={e => onChange({ ...draft, unit: e.target.value as TimeUnit })}>
        {timeUnits.map(u => <option key={u.value} value={u.value}>{u.label}</option>)}
      </select>
    </>
  );
}
function Many({ schema, draft, onChange, label, path = [] }: Part<'list' | 'map'>) {
  const element = schema.element ?? schema;
  const items: { k?: string; v: Draft }[] = draft.t === 'list' ? draft.items.map(v => ({ v })) : draft.items;
  const write = (next: typeof items) => onChange(draft.t === 'list'
    ? { t: 'list', items: next.map(i => i.v) }
    : { t: 'map', items: next.map(i => ({ k: i.k ?? '', v: i.v })) });
  const edit = (i: number, change: object) => write(items.map((o, j) => (j === i ? { ...o, ...change } : o)));
  const used = items.map(i => i.k);
  const where = JSON.stringify(path);
  return (
    <fieldset className="rule-many">
      <legend>{label}</legend>
      {items.map((item, i) => {
        const name = item.k ? `${label} › ${item.k}` : `${label} #${i + 1}`;
        const keys = schema.key?.choices ?? [];
        return (
          <div key={i} className="rule-entry">
            {draft.t === 'map' && (keys.length > 0
              ? <Choice schema={{ ...schema.key as RuleSchema,
                choices: keys.filter(c => c.value === item.k || !used.includes(c.value)) }}
                value={item.k ?? ''} label={`${label}: entry ${i + 1} name`} onChange={k => edit(i, { k })} />
              : <input aria-label={`${label}: entry ${i + 1} name`} value={item.k ?? ''}
                onChange={e => edit(i, { k: e.target.value })} />)}
            <RuleInput schema={element} draft={item.v} label={name} path={[...path, i]}
              onChange={v => edit(i, { v })} />
            <Act verb="commissioner.draft" args={{ action: 'remove', path: where, index: i }}
              variant="button" label={`Remove ${name}`}>Remove</Act>
          </div>
        );
      })}
      <Act verb="commissioner.draft" args={{ action: 'add', path: where, index: 0 }} variant="button"
        label={`Add to ${label}`}>Add</Act>
    </fieldset>
  );
}

// RuleInput renders the edit box a rule's kind calls for, straight from Go's schema.
export function RuleInput(props: Props) {
  const { schema, draft, onChange, label, path = [] } = props;
  if (draft.t === 'flag') {
    return (
      <label className="rule-flag">
        <input type="checkbox" checked={draft.v}
          onChange={e => onChange({ t: 'flag', v: e.target.checked })} />
        {label}: on
      </label>
    );
  }
  if (draft.t === 'rec') {
    return (
      <fieldset className="rule-record">
        <legend>{label}</legend>
        {schema.fields.map(f => (
          <RuleInput key={f.name} schema={f.schema} label={f.label} path={[...path, f.name]}
            draft={draft.f[f.name] ?? blankDraft(f.schema)}
            onChange={d => onChange({ t: 'rec', f: { ...draft.f, [f.name]: d } })} />
        ))}
      </fieldset>
    );
  }
  if (draft.t === 'list' || draft.t === 'map') return <Many {...props} draft={draft} />;
  return (
    <label className="rule-field">
      <span>{label}</span>
      {draft.t === 'time' ? <Time {...props} draft={draft} /> : <Text {...props} draft={draft} />}
    </label>
  );
}
