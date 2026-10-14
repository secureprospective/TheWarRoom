import type { Entry } from './controls';
import type { Rules } from './rules';

// Where a click on a job or a search result sends the landing: the control's group opens and the
// control takes focus; a rule found by its own name is offered in the form straight away.
export type Reveal = { control: string; field: string; n: number };
export type LandingView = { open: string | null; reveal?: Reveal };

// The value is a control id, then optionally the rule found: "CM-024 calendar.timezone".
export function parseOpen(value: string, entries: Entry[]): { control: Entry; field: string } | null {
  const [id, field = ''] = value.split(' ');
  const control = entries.find(e => e.id === id);
  return control && (!field || control.fields.some(f => f.field === field)) ? { control, field } : null;
}

// scopedRules keeps only the rules a group's controls own, so its form offers just those.
export function scopedRules(rules: Rules, owned: ReadonlySet<string>): Rules {
  const sections = rules.sections
    .map(s => ({ ...s, rows: s.rows.filter(r => owned.has(r.field)) }))
    .filter(s => s.rows.length > 0);
  return { ...rules, sections };
}
