import { scrollWorkIntoView } from '../shell/workScroll';
import { useEffect, useRef } from 'react';
import { endpointsAt, endpointById, type Placement } from './index';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { gravityClasses } from '../look/channels';
import { assertShortList } from '../shell/shortList';
import './registry.css';

function statusLabel(status: string): string {
  switch (status) {
    case 'landed': return 'landed live';
    case 'built': return 'built, not yet landed live';
    default: return 'not wired';
  }
}

export function EndpointIndex({ placement }: { placement: Placement }) {
  const s = commands.use();
  const root = useRef<HTMLElement>(null);
  const rows = endpointsAt(placement);
  assertShortList(rows.length);
  useEffect(() => {
    const index = root.current;
    if (!index) return;
    if (revealEndpoint(index, s.endpointIds)) return;
    // A lazy surface may mount after the index; watch briefly for it, never re-scrolling to the card.
    const work = index.closest('.wbody');
    const observer = typeof MutationObserver === 'undefined' ? undefined : new MutationObserver(() => {
      if (revealEndpoint(index, s.endpointIds, false)) observer?.disconnect();
    });
    if (work) observer?.observe(work, { childList: true, subtree: true });
    const expire = setTimeout(() => observer?.disconnect(), lazySurfaceWait);
    return () => { clearTimeout(expire); observer?.disconnect(); };
  }, [s.endpointIds, placement]);
  const merged = s.endpointIds.map(endpointById).filter((e) => e.merged_place === placement);
  return (
    <section ref={root} className="endpoint-index" aria-label={`${placement} endpoint index`}>
      <h4>Endpoint index</h4>
      {s.endpointIds.length > 1 && (
        <div className="endpoint-targets" aria-label="Merged endpoint targets">
          {s.endpointIds.map((id) => (
            <Act variant="text"
              key={id} verb="endpoint.open" args={{ id }}>{id} · {endpointById(id).name}</Act>
          ))}
        </div>
      )}
      {merged.map((e) => (
        <p key={e.id} data-selected="true" tabIndex={-1}>
          {e.id} · {e.name} → {placement} · {statusLabel(e.status)} · ring {e.ring}
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
            <small>{statusLabel(row.status)} · ring {row.ring}</small>
          </li>
        ))}
      </ul>
      {!rows.length && <p className="not-wired">No individual endpoints placed here.</p>}
    </section>
  );
}

const lazySurfaceWait = 5000;

export function revealEndpoint(index: HTMLElement, ids: readonly string[], fallback = true) {
  const work = index.closest<HTMLElement>('.wbody');
  const surface = Array.from(work?.querySelectorAll<HTMLElement>('[data-endpoint]') ?? [])
    .find(element => ids.includes(element.dataset.endpoint ?? '') && !element.closest('.endpoint-index'));
  if (surface) {
    scrollWorkIntoView(surface, 'start');
    const controls = surface.previousElementSibling as HTMLElement | null;
    if (controls?.classList.contains('market-controls') && work) {
      const parent = surface.parentElement;
      const top = parent?.getBoundingClientRect().top;
      const gap = surface.getBoundingClientRect().top - (top ?? 0);
      if (parent && gap > 0 && gap < work.clientHeight / 2) scrollWorkIntoView(parent, 'start');
    }
    return true;
  }
  if (!fallback) return false;
  const selected = index.querySelector<HTMLElement>('[data-selected="true"]');
  if (selected) {
    selected.focus({ preventScroll: true });
    scrollWorkIntoView(selected, 'nearest');
  }
  return false;
}
