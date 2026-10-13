import { revealWorkSection, scrollWorkIntoView } from '../shell/workScroll';
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
  const { open, finished, unfold } = splitIndexRows(rows, s.endpointIds);
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
      {open.length > 0 && (
        <ul>{open.map((row) => <IndexRow key={row.id} row={row} selected={s.endpointIds} />)}</ul>
      )}
      {rows.length > 0 && !open.length && <p>Every endpoint here has landed live.</p>}
      {finished.length > 0 && (
        <details className="endpoint-finished" open={unfold}>
          <summary>{finished.length} finished</summary>
          <ul>{finished.map((row) => <IndexRow key={row.id} row={row} selected={s.endpointIds} />)}</ul>
        </details>
      )}
      {!rows.length && <p className="not-wired">No individual endpoints placed here.</p>}
    </section>
  );
}

type Row = ReturnType<typeof endpointsAt>[number];

// Finished rows step aside (Christopher, 10-13): the index lists what is still to land and keeps the landed
// ones one click away. The fold opens when the endpoint just opened from the command bar is a landed one.
export function splitIndexRows(rows: readonly Row[], selected: readonly string[]) {
  const finished = rows.filter((row) => row.status === 'landed');
  return {
    open: rows.filter((row) => row.status !== 'landed'),
    finished,
    unfold: finished.some((row) => selected.includes(row.id)),
  };
}

function IndexRow({ row, selected }: { row: Row; selected: readonly string[] }) {
  return (
    <li className={gravityClasses[row.gravity]} data-selected={selected.includes(row.id)} tabIndex={-1}>
      <div className="endpoint-heading">
        <b>{row.id} · {row.name}</b>
        <span>{row.kind} · {row.gravity} · ring {row.ring}</span>
      </div>
      {row.gm_question && row.gm_question !== '—' && <p>{row.gm_question}</p>}
      <small>{statusLabel(row.status)} · ring {row.ring}</small>
    </li>
  );
}

const lazySurfaceWait = 5000;

export function revealEndpoint(index: HTMLElement, ids: readonly string[], fallback = true) {
  const work = index.closest<HTMLElement>('.wbody');
  const surface = Array.from(work?.querySelectorAll<HTMLElement>('[data-endpoint]') ?? [])
    .find(element => ids.includes(element.dataset.endpoint ?? '') && !element.closest('.endpoint-index'));
  if (surface) {
    let alignment = surface;
    const controls = surface.previousElementSibling as HTMLElement | null;
    if (controls?.classList.contains('market-controls') && work) {
      const parent = surface.parentElement;
      const top = parent?.getBoundingClientRect().top;
      const gap = surface.getBoundingClientRect().top - (top ?? 0);
      if (parent && gap > 0 && gap < work.clientHeight / 2) alignment = parent;
    }
    revealWorkSection(surface, alignment);
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
