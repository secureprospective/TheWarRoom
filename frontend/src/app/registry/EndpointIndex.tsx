import { useEffect, useRef } from 'react';
import { endpointsAt, endpointById, type Placement } from './index';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { gravityClasses } from '../look/channels';
import { assertShortList } from '../shell/shortList';
import './registry.css';

export function EndpointIndex({ placement }: { placement: Placement }) {
  const s = commands.use();
  const root = useRef<HTMLElement>(null);
  const rows = endpointsAt(placement);
  assertShortList(rows.length);
  useEffect(() => {
    const selected = root.current?.querySelector<HTMLElement>('[data-selected="true"]');
    if (selected) {
      selected.focus({ preventScroll: true });
      selected.scrollIntoView({ block: 'nearest' });
    }
  }, [s.endpointIds, placement]);
  const merged = s.endpointIds.map(endpointById).filter((e) => e.merged_place === placement);
  return (
    <section ref={root} className="endpoint-index" aria-label={`${placement} endpoint index`}>
      <h4>Endpoint index</h4>
      {s.endpointIds.length > 1 && (
        <div className="endpoint-targets" aria-label="Merged endpoint targets">
          {s.endpointIds.map((id) => (
            <Act key={id} verb="endpoint.open" args={{ id }}>{id} · {endpointById(id).name}</Act>
          ))}
        </div>
      )}
      {merged.map((e) => (
        <p key={e.id} data-selected="true" tabIndex={-1}>
          {e.id} · {e.name} → {placement} · not wired · ring {e.ring}
        </p>
      ))}
      <ul>
        {rows.map((row) => (
          <li
            key={row.id}
            className={gravityClasses[row.gravity]}
            data-selected={s.endpointIds.includes(row.id)}
            tabIndex={-1}
          >
            <div className="endpoint-heading">
              <b>{row.id} · {row.name}</b>
              <span>{row.kind} · {row.gravity} · ring {row.ring}</span>
            </div>
            {row.gm_question && row.gm_question !== '—' && <p>{row.gm_question}</p>}
            <small>not wired · ring {row.ring}</small>
          </li>
        ))}
      </ul>
      {!rows.length && <p className="not-wired">No individual endpoints placed here.</p>}
    </section>
  );
}
