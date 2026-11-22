import { revealWorkSection, scrollWorkIntoView } from '../shell/workScroll';
import { useEffect, useRef } from 'react';
import { endpointById, type Placement } from './index';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import './registry.css';

// The endpoint map is planning data (docs/ui/endpoint-registry.csv), not something owners read (Christopher,
// 10-19: remove the index from the app). What stays is the anchor that reveals an endpoint opened from the
// command bar, and the chooser when one id opens several places.
export function EndpointIndex({ placement }: { placement: Placement }) {
  const s = commands.use();
  const root = useRef<HTMLElement>(null);
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
  const choose = s.endpointIds.length > 1;
  return (
    <section ref={root} className="endpoint-index" aria-label={`${placement} endpoint index`}
      hidden={!choose}>
      {choose && (
        <div className="endpoint-targets" aria-label="Merged endpoint targets">
          {s.endpointIds.map((id) => (
            <Act variant="text"
              key={id} verb="endpoint.open" args={{ id }}>{id} · {endpointById(id).name}</Act>
          ))}
        </div>
      )}
    </section>
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
