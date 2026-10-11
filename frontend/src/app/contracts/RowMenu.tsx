import { createPortal } from 'react-dom';
import { useLayoutEffect, useRef, useState, type ReactNode } from 'react';

type Bounds = { left: number; right: number; top: number; bottom: number };
// DOMRect fields are prototype getters, so a rect is read field by field, never spread.
export function menuPlacement(control: Bounds, work: Bounds, height: number, desiredWidth = 380) {
  const clamp = (y: number) => Math.max(work.top, Math.min(y, work.bottom));
  const top = clamp(control.top), bottom = clamp(control.bottom);
  const width = Math.min(desiredWidth, work.right - work.left);
  const below = Math.max(0, work.bottom - bottom);
  const above = Math.max(0, top - work.top);
  const up = height > below && above > below;
  const maxHeight = up ? above : below;
  return { left: Math.max(work.left, Math.min(control.right - width, work.right - width)),
    top: up ? Math.max(work.top, top - Math.min(height, maxHeight)) : bottom,
    width, maxHeight };
}
export function placeRowMenu(popup: HTMLElement, control: Element, work: Element, host: Element | null) {
  const bounds = work.getBoundingClientRect();
  const inspector = host?.getAttribute('data-inspector') !== 'closed'
    ? host?.querySelector('.insp')?.getBoundingClientRect().left : undefined;
  const right = Math.min(bounds.right, inspector ?? bounds.right);
  const position = menuPlacement(control.getBoundingClientRect(), {
    left: Math.max(8, bounds.left + 8), right: Math.min(innerWidth - 8, right - 8),
    top: Math.max(8, bounds.top + 8), bottom: Math.min(innerHeight - 8, bounds.bottom - 8),
  }, popup.scrollHeight);
  Object.assign(popup.style, Object.fromEntries(Object.entries(position)
    .map(([key, value]) => [key, `${value}px`])));
}
export function RowMenu({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLDetailsElement>(null);
  const [open, setOpen] = useState(false);
  const [host, setHost] = useState<Element | null>(null);
  const popupRef = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => { setHost(ref.current?.closest('.twr-app') ?? null); }, []);
  useLayoutEffect(() => {
    const details = ref.current;
    if (!open || !details) return;
    const control = details.querySelector('summary')!;
    const popup = (popupRef.current ?? details).querySelector<HTMLElement>('.contract-whatif-acts')!;
    const work = details.closest('.wbody');
    if (!work || !popup) return;
    const place = () => placeRowMenu(popup, control, work, host);
    place();
    const observer = new ResizeObserver(place); observer.observe(popup);
    window.addEventListener('resize', place); work.addEventListener('scroll', place);
    return () => { observer.disconnect(); window.removeEventListener('resize', place);
      work.removeEventListener('scroll', place); };
  }, [open, host]);
  const close = (target: EventTarget) => {
    if ((target as HTMLElement).closest('button') && ref.current) ref.current.open = false;
  };
  return <details ref={ref} className="contract-menu" onToggle={e => setOpen(e.currentTarget.open)}
    onClickCapture={e => close(e.target)}><summary>What if…</summary>
    {open && host ? createPortal(<div ref={popupRef} className="contract-menu contract-menu-overlay"
      onClickCapture={e => close(e.target)}>{children}</div>, host) : children}
  </details>;
}
