import { Act } from '../commands/Act';
import { GovernedGlyph } from '../governed/GovernedGlyph';
import type { Entry } from './controls';
import type { RuleRow } from './rules';

type Props = {
  entry: Entry; rules: ReadonlyMap<string, RuleRow>; mayChange: boolean;
  refer: (id: string, element: HTMLLIElement | null) => void;
};

// ControlRow is one commissioner control: its plain question, whether it works yet, and the rules it holds.
export function ControlRow({ entry, rules, mayChange, refer }: Props) {
  return (
    <li ref={element => refer(entry.id, element)} tabIndex={-1} className="control-row"
      data-built={entry.built}>
      <h6>{entry.name}</h6>
      <p>{entry.question}</p>
      <p className="control-status">{entry.built ? 'Available now' : 'Not built yet'}</p>
      {entry.note && <p className="change-note">{entry.note}</p>}
      {entry.fields.length > 0 && rules.size > 0 && (
        <ul className="control-rules">
          {entry.fields.map(f => {
            const row = rules.get(f.field);
            return (
              <li key={f.field}>
                <span>{f.label}: {row ? row.value : 'Not read yet'}</span>
                {row?.governed && <GovernedGlyph governed={row.governed} subject={f.label} />}
                {mayChange && row && <Act verb="commissioner.surface" variant="text"
                  args={{ screen: 'landing', action: 'open', value: `${entry.id} ${f.field}` }}
                  label={`Change ${f.label}`}>Change</Act>}
              </li>
            );
          })}
        </ul>
      )}
    </li>
  );
}
