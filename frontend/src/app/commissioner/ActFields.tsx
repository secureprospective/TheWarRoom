import type { Field } from './actTable';
import type { ActValues } from './actBody';
import type { ActReadings, Option } from './actReadings';

const words: Record<string, string> = {
  division_weighted: 'More division matchups', league_rules: 'League playoff rules',
  legacy_nfl: 'Legacy NFL playoff layout',
  approve: 'Approve', veto: 'Veto', add: 'Add', remove: 'Remove', ir: 'IR', taxi: 'Taxi',
  held: 'Starting roster', draft: 'Draft', waiver: 'Waivers', trade: 'Trade',
  free_agent: 'Free agent', commissioner: 'Commissioner', all: 'Everyone', commish: 'Commissioners only',
  standings: 'Final standings, worst first', linear: 'Same order each round', snake: 'Reverse each round',
  'roster.limits': 'Roster size', 'roster.positionBounds': 'Roster position limits',
  'cap.enforcement': 'Salary cap', 'ir.eligible': 'IR eligibility', 'taxi.maximumYears': 'Taxi experience',
};
const labelFor = (value: string) => words[value] ?? value.replace(/^calendar\./, '')
  .replace(/_/g, ' ').replace(/^./, c => c.toUpperCase());
function Choices({ options, value, onChange }: {
  options: Option[]; value: string; onChange(value: string): void;
}) {
  return <select value={value} onChange={e => onChange(e.target.value)}>
    <option value="">Choose…</option>
    {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
  </select>;
}
export function ActFields({ fields, values, held, onChange }: {
  fields: Field[]; values: ActValues; held: ActReadings;
  onChange(path: string, value: ActValues[string]): void;
}) {
  return <>{fields.map(field => {
    if (field.kind === 'self') return null;
    const raw = values[field.path];
    const options = held.options[field.kind];
    if (field.kind === 'order') {
      const franchises = held.options.franchise ?? [];
      const order = Array.isArray(raw) ? raw : franchises.map(() => '');
      return <fieldset key={field.path}><legend>{field.label}</legend>
        {order.map((value, index) => <label key={index}>Place {index + 1}
          <Choices options={franchises} value={value} onChange={next =>
            onChange(field.path, order.map((v, i) => i === index ? next : v))} />
        </label>)}
      </fieldset>;
    }
    if (field.kind === 'starters') {
      const chosen = Array.isArray(raw) ? raw : [];
      return <fieldset key={field.path}><legend>{field.label}</legend>
        {(held.options.player ?? []).map(p => <label key={p.value}>
          <input type="checkbox" checked={chosen.includes(p.value)} onChange={e =>
            onChange(field.path, e.target.checked ? [...chosen, p.value]
              : chosen.filter(v => v !== p.value))} />
          {p.label}
        </label>)}
      </fieldset>;
    }
    return <label key={field.path}>{field.label}
      {field.kind === 'boolean' ? <input type="checkbox" checked={raw === true}
        onChange={e => onChange(field.path, e.target.checked)} />
        : field.kind === 'choice' || options ?
          <Choices value={String(raw ?? '')} options={options ?? (field.choices ?? [])
            .map(value => ({ value, label: labelFor(value) }))} onChange={v => onChange(field.path, v)} />
          : <input value={String(raw ?? '')}
            type={field.kind === 'instant' ? 'datetime-local' : 'text'}
            inputMode={['number', 'money', 'decimal'].includes(field.kind) ? 'decimal' : undefined}
            onChange={e => onChange(field.path, e.target.value)} />}
      {options?.length === 0 && <span>No available choices in the held league data.</span>}
    </label>;
  })}</>;
}
