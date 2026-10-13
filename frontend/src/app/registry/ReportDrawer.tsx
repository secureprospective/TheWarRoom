import { useState } from 'react';
import { Act } from '../commands/Act';
import { nodeKeys, nodes } from '../shell/nodes';
import { endpointsAt, surfacePlaces, type Placement } from './index';
import './registry.css';

export function reportGroups(query: string) {
  const places = [
    ...nodeKeys.flatMap(node => nodes[node].workspaces.map(w => `${nodes[node].label} › ${w.label}`)),
    ...surfacePlaces,
  ] as Placement[];
  const needle = query.trim().toLowerCase();
  // Keep placed, kept rows with a Read role (including hybrids and gated reads), regardless of status.
  // Match name, GM question, id and place like command-bar substrings, without its eight-result cap.
  return places.map(place => ({ place, rows: endpointsAt(place).filter(row =>
    /\bRead\b/.test(row.kind) &&
    [row.name, row.gm_question, row.id, place].some(text => text.toLowerCase().includes(needle)),
  ) })).filter(group => group.rows.length);
}

export default function ReportDrawer() {
  const [query, setQuery] = useState('');
  const groups = reportGroups(query);
  const count = groups.reduce((sum, group) => sum + group.rows.length, 0);
  return (
    <section className="report-drawer" aria-label="Report shortcuts">
      <label className="report-filter">
        Filter reports
        <input type="search" value={query} onChange={event => setQuery(event.target.value)}
          placeholder="Name, question or place" />
      </label>
      <p role="status">{count} {count === 1 ? 'report' : 'reports'}</p>
      {groups.map(({ place, rows }) => (
        <section key={place} aria-label={place}>
          <h4>{place}</h4>
          <ul>
            {rows.map(row => (
              <li key={row.id}>
                <Act variant="card" verb="endpoint.open" args={{ id: row.id }} label={`Open ${row.name}`}>
                  <strong>{row.name}</strong>
                  {row.gm_question && row.gm_question !== '—' && <span>{row.gm_question}</span>}
                  {row.status !== 'landed' && <small>
                    {row.status === 'built' ? 'Built, not yet landed live' : 'Not wired'}
                  </small>}
                </Act>
              </li>
            ))}
          </ul>
        </section>
      ))}
      {!count && <p>No reports match. Try another name, question or place.</p>}
    </section>
  );
}
