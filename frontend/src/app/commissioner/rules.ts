import { array, choice, integer, object, text } from '../data/parseValues';

// The Go rules reading (TargetCommissionerRules). The rules screen imports this lazily.
export const ruleKinds = ['count', 'cents', 'ten_thousandths', 'nanoseconds', 'boolean', 'date', 'string',
  'record', 'list', 'map'] as const;
export type RuleKind = (typeof ruleKinds)[number];
export type RuleChoice = { value: string; label: string };
export type RuleSchema = {
  kind: RuleKind; minimum?: number; maximum?: number; choices: RuleChoice[];
  fields: { name: string; label: string; schema: RuleSchema }[]; key?: RuleSchema; element?: RuleSchema;
};
export type RulePending = { entry: string; value: string; when: string };
export type RuleRow = {
  field: string; label: string; value: string; state: 'Set' | 'Needs a ruling'; citation: string;
  schema: RuleSchema; competitive: boolean; pending: RulePending[];
};
export type Rules = {
  league: string; name: string; revision: number; rollover: string;
  sections: { label: string; rows: RuleRow[] }[];
};

function bound(v: unknown, path: string): number | undefined {
  return v === undefined ? undefined : integer(v, path, Number.MIN_SAFE_INTEGER);
}
function flag(v: unknown, path: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${path}: expected true or false`);
  return v;
}
export function parseSchema(v: unknown, path: string): RuleSchema {
  const r = object(v, path, ['kind', 'minimum', 'maximum', 'choices', 'fields', 'key', 'element']);
  const schema: RuleSchema = {
    kind: choice(r.kind, `${path}.kind`, ruleKinds),
    choices: array(r.choices, `${path}.choices`, (c, p) => {
      const o = object(c, p, ['value', 'label']);
      return { value: text(o.value, p), label: text(o.label, p) };
    }),
    fields: array(r.fields, `${path}.fields`, (f, p) => {
      const o = object(f, p, ['name', 'label', 'schema']);
      return { name: text(o.name, p), label: text(o.label, p), schema: parseSchema(o.schema, `${p}.schema`) };
    }),
  };
  const minimum = bound(r.minimum, `${path}.minimum`);
  const maximum = bound(r.maximum, `${path}.maximum`);
  if (minimum !== undefined) schema.minimum = minimum;
  if (maximum !== undefined) schema.maximum = maximum;
  if (r.key !== undefined) schema.key = parseSchema(r.key, `${path}.key`);
  if (r.element !== undefined) schema.element = parseSchema(r.element, `${path}.element`);
  return schema;
}
function parseRow(v: unknown, p: string): RuleRow {
  const keys = ['field', 'label', 'value', 'state', 'citation', 'schema', 'competitive', 'pending'];
  const r = object(v, p, keys);
  return {
    field: text(r.field, p), label: text(r.label, p), value: text(r.value, p),
    state: choice(r.state, `${p}.state`, ['Set', 'Needs a ruling'] as const), citation: text(r.citation, p),
    schema: parseSchema(r.schema, `${p}.schema`), competitive: flag(r.competitive, `${p}.competitive`),
    pending: array(r.pending, `${p}.pending`, (w, q) => {
      const o = object(w, q, ['entry', 'value', 'when']);
      return { entry: text(o.entry, q), value: text(o.value, q), when: text(o.when, q) };
    }),
  };
}
export function parseRules(v: unknown): Rules {
  const r = object(v, 'League rules', ['league', 'name', 'revision', 'rollover', 'sections']);
  return {
    league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'League version'), rollover: text(r.rollover, 'Next rollover'),
    sections: array(r.sections, 'Rule sections', (s, p) => {
      const o = object(s, p, ['label', 'rows']);
      return { label: text(o.label, p), rows: array(o.rows, `${p}.rows`, parseRow) };
    }),
  };
}
