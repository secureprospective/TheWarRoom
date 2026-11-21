/** Touch-friendly press-and-hold pointer drag controller for phone lineup rows.
 *  A hold (~300 ms) activates drag; movement beyond MOVE_CANCEL_PX before the hold fires
 *  cancels the pending drag so native vertical scrolling is never blocked. */
export const HOLD_MS = 300;
export const MOVE_CANCEL_PX = 8;

export type DragState = {
  key: string;
  pointerId: number;
  x: number;
  y: number;
  over?: string;
};

type Pending = {
  key: string;
  pointerId: number;
  x: number;
  y: number;
  timer: ReturnType<typeof setTimeout>;
};

export function rowKeyAtPoint(x: number, y: number): string | undefined {
  if (typeof document === 'undefined' || typeof document.elementFromPoint !== 'function') return undefined;
  return document.elementFromPoint(x, y)?.closest<HTMLElement>('[data-row]')?.dataset.row;
}

export function createRowDrag() {
  let pending: Pending | undefined;
  let active: DragState | undefined;
  let skipClick = false;
  let skipTimer: ReturnType<typeof setTimeout> | undefined;

  const clearPending = () => {
    if (!pending) return;
    clearTimeout(pending.timer);
    pending = undefined;
  };

  const armSkipClick = () => {
    skipClick = true;
    if (skipTimer !== undefined) clearTimeout(skipTimer);
    skipTimer = setTimeout(() => { skipClick = false; skipTimer = undefined; }, 0);
  };

  return {
    start(key: string, pointerId: number, x: number, y: number, onActivate: (d: DragState) => void) {
      clearPending();
      active = undefined;
      const timer = setTimeout(() => {
        if (!pending || pending.pointerId !== pointerId || pending.key !== key) return;
        const { x: px, y: py } = pending;
        pending = undefined;
        active = { key, pointerId, x: px, y: py };
        onActivate(active);
      }, HOLD_MS);
      pending = { key, pointerId, x, y, timer };
    },

    move(
      pointerId: number, x: number, y: number,
      rowAt: (px: number, py: number) => string | undefined,
      onUpdate: (d: DragState) => void,
    ) {
      if (pending && pending.pointerId === pointerId) {
        if (Math.hypot(x - pending.x, y - pending.y) > MOVE_CANCEL_PX) clearPending();
        return;
      }
      if (!active || active.pointerId !== pointerId) return;
      active = { ...active, x, y, over: rowAt(x, y) };
      onUpdate(active);
    },

    finish(
      pointerId: number, x: number, y: number,
      rowAt: (px: number, py: number) => string | undefined,
    ): { from: string; to?: string } | undefined {
      if (pending && pending.pointerId === pointerId) {
        clearPending();
        return undefined;
      }
      if (!active || active.pointerId !== pointerId) return undefined;
      const from = active.key;
      const to = rowAt(x, y);
      active = undefined;
      armSkipClick();
      return { from, to };
    },

    cancel() {
      clearPending();
      active = undefined;
      if (skipTimer !== undefined) {
        clearTimeout(skipTimer);
        skipTimer = undefined;
      }
      skipClick = false;
    },

    consumeClick(): boolean {
      if (!skipClick) return false;
      skipClick = false;
      if (skipTimer !== undefined) {
        clearTimeout(skipTimer);
        skipTimer = undefined;
      }
      return true;
    },
  };
}
