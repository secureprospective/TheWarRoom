import { useRef } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';

// One drag grammar (mock v6): pointer drag, a ghost row follows the cursor, drop zones light up.
// WebKit drops HTML5 drag-and-drop in the desktop shell, so rows drag on pointer events instead.
const zones = '.bids,.lane2';
export function useWarRoomDrag(drop: (id: string, over: Element) => void) {
  const onDrop = useRef(drop);
  onDrop.current = drop;
  return (e: ReactPointerEvent<HTMLElement>, id: string) => {
    if (e.button || (e.target as Element).closest('.st')) return;
    e.preventDefault();
    const row = e.currentTarget, x0 = e.clientX, y0 = e.clientY;
    let ghost: HTMLElement | undefined;
    const lit = () => document.querySelectorAll(zones).forEach(z => z.classList.remove('over'));
    const move = (m: PointerEvent) => {
      if (!ghost && Math.hypot(m.clientX - x0, m.clientY - y0) < 4) return;
      if (!ghost) {
        ghost = row.cloneNode(true) as HTMLElement;
        ghost.className = 'fa-r ghost';
        (row.closest('.wr-v6') ?? document.body).appendChild(ghost);
        document.body.classList.add('dragging');
      }
      ghost.style.transform = `translate(${m.clientX + 10}px,${m.clientY - 14}px)`;
      lit();
      document.elementFromPoint(m.clientX, m.clientY)?.closest(zones)?.classList.add('over');
    };
    const up = (u: PointerEvent) => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      lit();
      if (!ghost) return;
      ghost.remove();
      document.body.classList.remove('dragging');
      const over = document.elementFromPoint(u.clientX, u.clientY);
      if (over) onDrop.current(id, over);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };
}
