import type { RuleSchema } from './rules';

// A rule value as someone is typing it, and the plain JSON Go accepts. Numbers are typed in the units a
// GM reads (dollars, percent, hours); exact integer arithmetic turns them back into cents, ten-thousandths
// and nanoseconds. Go still validates everything; this only refuses what cannot be read as a value.
export const timeUnits = [
  { value: 'seconds', label: 'seconds', ns: 1e9 }, { value: 'minutes', label: 'minutes', ns: 6e10 },
  { value: 'hours', label: 'hours', ns: 3.6e12 }, { value: 'days', label: 'days', ns: 8.64e13 },
] as const;
export type TimeUnit = (typeof timeUnits)[number]['value'];
export type Draft =
  | { t: 'text'; v: string }
  | { t: 'flag'; v: boolean }
  | { t: 'time'; v: string; unit: TimeUnit }
  | { t: 'rec'; f: Record<string, Draft> }
  | { t: 'list'; items: Draft[]; unset?: boolean }
  | { t: 'map'; items: { k: string; v: Draft }[]; unset?: boolean };
export type Typed = { ok: true; value: unknown } | { ok: false; error: string };
type Natural = unknown;

const places = (percent: boolean) => (percent ? 2 : 4);
export function blankDraft(schema: RuleSchema): Draft {
  switch (schema.kind) {
    case 'boolean': return { t: 'flag', v: false };
    case 'nanoseconds': return { t: 'time', v: '', unit: 'hours' };
    case 'record': return {
      t: 'rec', f: Object.fromEntries(schema.fields.map(f => [f.name, blankDraft(f.schema)])),
    };
    case 'list': return { t: 'list', items: [], unset: true };
    case 'map': return { t: 'map', items: [], unset: true };
    default: return { t: 'text', v: '' };
  }
}

// scaled writes n / 10^digits without binary rounding, dropping trailing zeros.
export function scaled(n: number, digits: number): string {
  const sign = n < 0 ? '-' : '';
  const size = 10 ** digits;
  const whole = Math.trunc(Math.abs(n) / size);
  const fraction = String(Math.abs(n) % size).padStart(digits, '0').replace(/0+$/, '');
  return sign + whole + (fraction ? `.${fraction}` : '');
}
// unscale reads a typed decimal as an integer count of 10^-digits; null when it is not that.
function unscale(text: string, digits: number): number | null {
  const m = /^(-?)(\d+)(?:\.(\d*))?$/.exec(text.trim());
  if (!m || (m[3] ?? '').length > digits) return null;
  const n = Number(m[2] + (m[3] ?? '').padEnd(digits, '0'));
  return Number.isSafeInteger(n) ? (m[1] ? -n : n) : null;
}
function timeDraft(ns: number): Draft {
  const unit = [...timeUnits].reverse().find(u => ns % u.ns === 0) ?? timeUnits[0];
  return { t: 'time', v: scaled(Math.round((ns / unit.ns) * 1e4), 4), unit: unit.value };
}
const local = (d: Date) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);

export function draftFrom(schema: RuleSchema, natural: Natural, percent = false): Draft {
  if (natural === null || natural === undefined) return blankDraft(schema);
  const number = typeof natural === 'number' ? natural : 0;
  switch (schema.kind) {
    case 'boolean': return { t: 'flag', v: natural === true };
    case 'cents': return { t: 'text', v: scaled(number, 2) };
    case 'ten_thousandths': return { t: 'text', v: scaled(number, percent ? 2 : 4) };
    case 'nanoseconds': return timeDraft(number);
    case 'date': return { t: 'text', v: typeof natural === 'string' ? local(new Date(natural)) : '' };
    case 'record': return {
      t: 'rec', f: Object.fromEntries(schema.fields.map(f =>
        [f.name, draftFrom(f.schema, (natural as Record<string, Natural>)[f.name])])),
    };
    case 'list': return {
      t: 'list',
      items: (Array.isArray(natural) ? natural : []).map(n => draftFrom(schema.element ?? schema, n)),
    };
    case 'map': return {
      t: 'map', items: Object.entries(natural as Record<string, Natural>)
        .map(([k, v]) => ({ k, v: draftFrom(schema.element ?? schema, v) })),
    };
    default: return { t: 'text', v: String(natural) };
  }
}

const bad = (error: string): Typed => ({ ok: false, error });
const good = (value: unknown): Typed => ({ ok: true, value });
function inBounds(n: number, schema: RuleSchema, label: string, show: (n: number) => string): Typed {
  const { minimum, maximum } = schema;
  if (minimum !== undefined && n < minimum) return bad(`${label}: must be at least ${show(minimum)}`);
  if (maximum !== undefined && n > maximum) return bad(`${label}: must be at most ${show(maximum)}`);
  return good(n);
}
function scalar(schema: RuleSchema, draft: Draft, label: string, percent: boolean, top: boolean): Typed {
  if (draft.t === 'flag') return good(draft.v);
  const blank = (draft.t === 'text' || draft.t === 'time') && draft.v.trim() === '';
  // Inside a record a blank number is left out, so that part of the rule stays as unset as it was.
  if (blank && !top && (schema.kind !== 'string' || schema.choices.length > 0)) return good(undefined);
  if (draft.t === 'time') {
    const amount = unscale(draft.v, 4);
    const perUnit = (timeUnits.find(u => u.value === draft.unit)?.ns ?? 1e4) / 1e4;
    const ns = amount === null ? null : amount * perUnit;
    if (ns === null || !Number.isSafeInteger(ns)) return bad(`${label}: enter a length of time`);
    return inBounds(ns, schema, label, n => `${scaled(n, 9)} seconds`);
  }
  if (draft.t !== 'text') return bad(`${label}: enter a value`);
  const kind = schema.kind;
  if (kind === 'count') {
    const n = unscale(draft.v, 0);
    return n === null ? bad(`${label}: enter a whole number`) : inBounds(n, schema, label, String);
  }
  if (kind === 'cents') {
    const n = unscale(draft.v, 2);
    return n === null ? bad(`${label}: enter an amount in dollars`)
      : inBounds(n, schema, label, c => `$${scaled(c, 2)}`);
  }
  if (kind === 'ten_thousandths') {
    const n = unscale(draft.v, places(percent));
    return n === null ? bad(`${label}: enter a number`)
      : inBounds(n, schema, label, c => scaled(c, places(percent)) + (percent ? '%' : ''));
  }
  if (kind === 'date') {
    const at = new Date(draft.v);
    if (draft.v === '') return good(undefined);
    return Number.isNaN(at.getTime()) ? bad(`${label}: choose a date and time`) : good(at.toISOString());
  }
  return good(draft.v);
}

// toNatural reads a draft against its schema. top is true for the rule's own value, which may not be blank.
export function toNatural(
  schema: RuleSchema, draft: Draft, label: string, percent = false, top = true,
): Typed {
  if (draft.t === 'rec') {
    const out: Record<string, unknown> = {};
    for (const field of schema.fields) {
      const typed = toNatural(field.schema, draft.f[field.name] ?? blankDraft(field.schema),
        `${label} › ${field.label}`, false, false);
      if (!typed.ok) return typed;
      if (typed.value !== undefined) out[field.name] = typed.value;
    }
    if (Object.keys(out).length > 0) return good(out);
    return top ? bad(`${label}: enter a value`) : good(undefined);
  }
  if (draft.t === 'list') {
    const items: unknown[] = [];
    for (const [i, item] of draft.items.entries()) {
      const typed = toNatural(schema.element ?? schema, item, `${label} #${i + 1}`, false, false);
      if (!typed.ok) return typed;
      if (typed.value === undefined) return bad(`${label} #${i + 1}: fill this in or remove it`);
      items.push(typed.value);
    }
    return !top && draft.unset && items.length === 0 ? good(undefined) : good(items);
  }
  if (draft.t === 'map') {
    const typed = toMap(schema, draft.items, label);
    return !top && draft.unset && draft.items.length === 0 ? good(undefined) : typed;
  }
  const typed = scalar(schema, draft, label, percent, top);
  if (typed.ok && typed.value === undefined && top) return bad(`${label}: choose a date and time`);
  if (typed.ok && schema.kind === 'string' && draft.t === 'text' && draft.v.trim() === '' && top)
    return bad(`${label}: enter a value`);
  return typed;
}
function toMap(schema: RuleSchema, entries: { k: string; v: Draft }[], label: string): Typed {
  const out: Record<string, unknown> = {};
  for (const { k, v } of entries) {
    const key = schema.key?.kind === 'count' ? String(unscale(k, 0) ?? '') : k.trim();
    if (key === '') return bad(`${label}: name every entry`);
    if (key in out) return bad(`${label}: two entries are named ${key}`);
    const typed = toNatural(schema.element ?? schema, v, `${label} › ${key}`, false, false);
    if (!typed.ok) return typed;
    if (typed.value === undefined) return bad(`${label} › ${key}: fill this in or remove it`);
    out[key] = typed.value;
  }
  return good(out);
}

// Add and remove act on a list or map somewhere inside the draft. A path walks field names (records) and
// positions (lists and maps) from the rule's own value down to the list or map being changed.
export type Step = string | number;
export type Edit = { action: 'add' | 'remove'; path: Step[]; index: number };
function change(schema: RuleSchema, draft: Draft, edit: Edit, path: Step[]): Draft {
  const [head, ...rest] = path;
  if (head === undefined) return grow(schema, draft, edit);
  if (draft.t === 'rec') {
    const field = schema.fields.find(f => f.name === head);
    if (!field || draft.f[head] === undefined) return draft;
    return { t: 'rec', f: { ...draft.f, [head]: change(field.schema, draft.f[head], edit, rest) } };
  }
  const element = schema.element ?? schema;
  if (draft.t === 'list' && typeof head === 'number' && draft.items[head] !== undefined)
    return { ...draft, items: draft.items.map((d, i) => (i === head ? change(element, d, edit, rest) : d)) };
  if (draft.t === 'map' && typeof head === 'number' && draft.items[head] !== undefined) {
    const into = (e: { k: string; v: Draft }, i: number) =>
      (i === head ? { ...e, v: change(element, e.v, edit, rest) } : e);
    return { ...draft, items: draft.items.map(into) };
  }
  return draft;
}
function grow(schema: RuleSchema, draft: Draft, edit: Edit): Draft {
  const element = schema.element ?? schema;
  if (draft.t === 'list') {
    return { t: 'list', items: edit.action === 'add' ? [...draft.items, blankDraft(element)]
      : draft.items.filter((_, i) => i !== edit.index) };
  }
  if (draft.t === 'map') {
    return { t: 'map', items: edit.action === 'add' ? [...draft.items, { k: '', v: blankDraft(element) }]
      : draft.items.filter((_, i) => i !== edit.index) };
  }
  return draft;
}
export const editDraft = (schema: RuleSchema, draft: Draft, edit: Edit): Draft =>
  change(schema, draft, edit, edit.path);
