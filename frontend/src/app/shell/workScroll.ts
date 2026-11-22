import { useCallback, useLayoutEffect, useRef, type RefObject } from 'react';

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

// A new page starts at its top. This runs as a layout effect so it always precedes the passive effects that
// reveal a requested endpoint on that page; a passive reset would run after them and undo the reveal.
export function useWorkScrollReset(scroll: RefObject<HTMLElement | null>, revision: number) {
  useLayoutEffect(() => {
    if (scroll.current) scroll.current.scrollTop = 0;
  }, [scroll, revision]);
}

export function revealWorkSection(element: HTMLElement, alignment = element) {
  let parent: HTMLElement | null = element;
  while (parent && !parent.classList?.contains('wbody')) {
    if (parent.tagName === 'DETAILS') (parent as HTMLDetailsElement).open = true;
    parent = parent.parentElement;
  }
  scrollWorkIntoView(alignment);
  element.tabIndex = -1;
  element.focus({ preventScroll: true });
  settleWorkSection(alignment);
}

// Five seconds covers lazy reads without keeping a navigation anchor alive indefinitely.
export const workSettleWindow = 5000;
const settling = new WeakMap<HTMLElement, () => void>();

function settleWorkSection(element: HTMLElement) {
  const scroll = element.closest<HTMLElement>('.wbody');
  if (!scroll || typeof ResizeObserver === 'undefined' || !scroll.addEventListener) return;
  settling.get(scroll)?.();
  let expected = scroll.scrollTop, active = true;
  const inputs = ['wheel', 'touchmove', 'pointerdown', 'click', 'keydown', 'input'];
  const stop = () => {
    if (!active) return;
    active = false;
    clearTimeout(expire);
    resize.disconnect();
    mutation?.disconnect();
    scroll.removeEventListener('scroll', moved);
    for (const event of inputs) document.removeEventListener(event, stop, true);
    settling.delete(scroll);
  };
  const align = () => {
    if (!active) return;
    if (!element.isConnected || scroll.scrollTop !== expected) { stop(); return; }
    scrollWorkIntoView(element);
    expected = scroll.scrollTop;
  };
  const moved = () => { if (scroll.scrollTop !== expected) stop(); };
  const resize = new ResizeObserver(align);
  const observe = () => {
    if (!active) return;
    resize.disconnect();
    for (const child of Array.from(scroll.children)) resize.observe(child);
    let parent: HTMLElement | null = element;
    while (parent && parent !== scroll) { resize.observe(parent); parent = parent.parentElement; }
    align();
  };
  const mutation = typeof MutationObserver === 'undefined' ? undefined : new MutationObserver(observe);
  const expire = setTimeout(stop, workSettleWindow);
  settling.set(scroll, stop);
  for (const event of inputs) document.addEventListener(event, stop, true);
  scroll.addEventListener('scroll', moved, { passive: true });
  mutation?.observe(scroll, { childList: true, subtree: true });
  observe();
}

export type WorkAnchor = { row: HTMLElement; top: number; scroll: HTMLElement; fallback?: WorkAnchor };
export function restoreWorkAnchor(anchor?: WorkAnchor) {
  if (anchor?.row.isConnected) {
    anchor.scroll.scrollTop += anchor.row.getBoundingClientRect().top - anchor.top;
    const box = anchor.row.getBoundingClientRect(), view = anchor.scroll.getBoundingClientRect();
    if (box.bottom <= view.top || box.top >= view.top + anchor.scroll.clientHeight) {
      scrollWorkIntoView(anchor.row, 'nearest');
      anchor.top = anchor.row.getBoundingClientRect().top;
    }
  } else if (anchor?.fallback) restoreWorkAnchor(anchor.fallback);
}

// Cards hold their top; insert edits hold the touched control, never a hidden source row.
export function useWorkAnchor(root: RefObject<HTMLElement>, selector: string) {
  const anchor = useRef<WorkAnchor>();
  useLayoutEffect(() => { restoreWorkAnchor(anchor.current); });
  useLayoutEffect(() => {
    const element = root.current;
    const scroll = element?.closest?.<HTMLElement>('.wbody');
    if (!element || !scroll) return;
    const visible = (row: HTMLElement) => {
      const box = row.getBoundingClientRect(), view = scroll.getBoundingClientRect();
      return box.bottom > view.top && box.top < view.top + scroll.clientHeight;
    };
    const hold = (row: HTMLElement): WorkAnchor => ({ row, scroll, top: row.getBoundingClientRect().top });
    const capture = (event: Event) => {
      const target = event.target as HTMLElement;
      if (target.closest?.('button')?.getAttribute('aria-label') === 'Contract Acts…') {
        anchor.current = undefined;
        return;
      }
      const inside = element.contains(target);
      const row = target.closest?.<HTMLElement>(selector);
      const insert = target.closest?.<HTMLElement>('.bid-form, .contract-move');
      const previous = insert?.closest('tr')?.previousElementSibling as HTMLElement | undefined;
      const held = anchor.current;
      const remembered = held?.row.closest?.(selector) === held?.row ? held?.row : held?.fallback?.row;
      const fallback = previous ?? (remembered && element.contains(remembered) && visible(remembered)
        ? remembered : undefined);
      const control = target.closest?.<HTMLElement>('button, input, select, textarea, summary');
      const chosen = inside ? insert ? control ?? target : row ?? control :
        Array.from(element.querySelectorAll<HTMLElement>(selector)).find(visible);
      if (chosen && visible(chosen)) anchor.current = { ...hold(chosen),
        fallback: fallback && fallback !== chosen ? hold(fallback) : undefined };
      else anchor.current = undefined;
    };
    const remember = () => { anchor.current = undefined; };
    const observer = typeof ResizeObserver === 'undefined' ? undefined :
      new ResizeObserver(() => restoreWorkAnchor(anchor.current));
    observer?.observe(element);
    for (const event of ['click', 'input', 'change']) document.addEventListener(event, capture, true);
    scroll.addEventListener('wheel', remember, { passive: true });
    scroll.addEventListener('touchmove', remember, { passive: true });
    return () => {
      observer?.disconnect();
      for (const event of ['click', 'input', 'change']) document.removeEventListener(event, capture, true);
      scroll.removeEventListener('wheel', remember);
      scroll.removeEventListener('touchmove', remember);
    };
  }, [root, selector]);
  return useCallback(() => { anchor.current = undefined; }, []);
}
