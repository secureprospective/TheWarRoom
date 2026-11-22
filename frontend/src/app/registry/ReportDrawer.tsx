import { useState } from 'react';
import { Act } from '../commands/Act';
import { nodeKeys, nodes } from '../shell/nodes';
import { endpointsAt, surfacePlaces, type Placement } from './index';
import './registry.css';

const researchNames: Record<string, string> = {
  'M-038': 'My top performers', 'M-037': 'Player stats', 'M-040': 'Position leaders',
  'M-057': 'NFL injuries', 'M-051': 'Injuries by NFL team', 'M-048': 'NFL depth charts',
  'M-052': 'MFL-wide activity', 'M-053': 'Points allowed by position',
  'M-054': 'Points allowed per NFL team', 'M-060': 'My players’ news',
};
function reportName(id: string, name: string) {
  return researchNames[id] ?? name.replace(/\s+[OR]=\w+/gi, '');
}
export function reportGroups(query: string) {
  const places = [
    ...nodeKeys.flatMap(node => nodes[node].workspaces.map(w => `${nodes[node].label} › ${w.label}`)),
    ...surfacePlaces,
  ] as Placement[];
  const needle = query.trim().toLowerCase();
  return places.map(place => ({ place, rows: endpointsAt(place).filter(row =>
    /\bRead\b/.test(row.kind) && row.status === 'landed' &&
    [reportName(row.id, row.name), row.gm_question, row.id, place]
      .some(text => text.toLowerCase().includes(needle)),
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
                <Act variant="card" verb="endpoint.open" args={{ id: row.id }}
                  label={`Open ${reportName(row.id, row.name)}`}>
                  <strong>{reportName(row.id, row.name)}</strong>
                  {row.gm_question && row.gm_question !== '—' && <span>{row.gm_question}</span>}
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
