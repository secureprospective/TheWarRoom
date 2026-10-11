import { useLayoutEffect, useRef, type RefObject } from 'react';

// Native scrollIntoView can scroll the clipped shell as well as its work column.
export function scrollWorkIntoView(element: HTMLElement, block: 'start' | 'nearest' = 'start') {
  const scroll = element.closest<HTMLElement>('.wbody');
  if (!scroll) return;
  const row = element.getBoundingClientRect();
  const view = scroll.getBoundingClientRect();
  const top = view.top + scroll.clientTop;
  const bottom = top + scroll.clientHeight;
  const delta = block === 'start' ? row.top - top :
    row.top < top ? row.top - top : row.bottom > bottom ? row.bottom - bottom : 0;
  scroll.scrollTop += delta;
}

export type WorkAnchor = { row: HTMLElement; top: number; scroll: HTMLElement };
export function restoreWorkAnchor(anchor?: WorkAnchor) {
  if (anchor?.row.isConnected) {
    anchor.scroll.scrollTop += anchor.row.getBoundingClientRect().top - anchor.top;
  }
}

// Hold the chosen row through async panel growth, including an Inspector route and Close.
export function useWorkAnchor(root: RefObject<HTMLElement>, selector: string) {
  const anchor = useRef<WorkAnchor>();
  useLayoutEffect(() => { restoreWorkAnchor(anchor.current); });
  useLayoutEffect(() => {
    const element = root.current;
    const scroll = element?.closest<HTMLElement>('.wbody');
    if (!element || !scroll) return;
    const capture = (event: Event) => {
      const target = event.target as HTMLElement;
      const chosen = target.closest?.<HTMLElement>(selector);
      const row = chosen && element.contains(chosen) ? chosen :
        anchor.current?.row ?? Array.from(element.querySelectorAll<HTMLElement>(selector))
          .find(row => row.getBoundingClientRect().bottom > scroll.getBoundingClientRect().top);
      if (row) anchor.current = { row, scroll, top: row.getBoundingClientRect().top };
    };
    const remember = () => {
      if (anchor.current) anchor.current.top = anchor.current.row.getBoundingClientRect().top;
    };
    const observer = typeof ResizeObserver === 'undefined' ? undefined :
      new ResizeObserver(() => restoreWorkAnchor(anchor.current));
    observer?.observe(element);
    document.addEventListener('click', capture, true);
    scroll.addEventListener('scroll', remember);
    return () => {
      observer?.disconnect();
      document.removeEventListener('click', capture, true);
      scroll.removeEventListener('scroll', remember);
    };
  }, [root, selector]);
}
