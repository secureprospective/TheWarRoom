import { useState } from 'react';
import { Act } from '../commands/Act';
import { searchControls, type Entry } from './controls';

const RESULTS = 12;

// ControlSearch finds a control by its plain question, or by the name of a rule it holds. A result
// opens that control's group and moves to it.
export function ControlSearch({ entries }: { entries: Entry[] }) {
  const [query, setQuery] = useState('');
  const hits = searchControls(entries, query);
  return (
    <section className="control-search" aria-label="Search the league controls">
      <label htmlFor="control-search">Search the league controls</label>
      <input className="search-input" id="control-search" type="search" value={query}
        placeholder="Ask what you want to change, such as “who gets waiver priority”"
        onChange={e => setQuery(e.target.value)} />
      {query.trim() !== '' && hits.length === 0 && <p>No control matches “{query.trim()}”.</p>}
      <ul className="control-hits">
        {hits.slice(0, RESULTS).map(({ entry, via }) => (
          <li key={entry.id}>
            <Act verb="commissioner.surface" variant="card"
              args={{ screen: 'landing', action: 'open', value: `${entry.id} ${via?.field ?? ''}`.trim() }}
              label={`${entry.name}. ${entry.question}${via ? ` Rule: ${via.label}.` : ''}${
                entry.built ? '' : ' Not built yet.'}`}>
              <strong>{entry.name}</strong>
              <span className="job-line">{entry.question}</span>
              {via && <span className="job-line">Rule: {via.label}</span>}
              <span className="job-line">
                {entry.built ? `Available now in ${entry.groupLabel}` : 'Not built yet'}
              </span>
            </Act>
          </li>
        ))}
      </ul>
      {hits.length > RESULTS && <p>{hits.length - RESULTS} more: type more of the question to narrow it.</p>}
    </section>
  );
}
