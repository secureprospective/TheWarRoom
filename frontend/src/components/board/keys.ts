import { useCallback, useEffect, useRef, useState } from 'react';

// isTypingTarget reports whether the user is currently typing, in which case a global
// shortcut must NOT fire — "j" belongs to the text field, not the board.
//
// This is the SINGLE definition of that guard. App.tsx had it inline for the 1/2/3/i/Esc
// shortcuts and B-5's J/K/Enter needs exactly the same rule; two copies would drift, and
// the failure mode of a drifted copy is silent and awful (a shortcut hijacking a keystroke
// mid-word in one place but not another). Any new global key handler imports this.
export function isTypingTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  return !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable);
}

// ownsEnter reports whether the focused element already has its own meaning for Enter, in
// which case a global Enter handler must keep its hands off.
//
// This is not hypothetical (GLM review, B-5): Tab to a sort-header button, press Enter, and
// a global handler that claims Enter would preventDefault the button's synthesized click —
// the column silently fails to sort AND an unrelated row opens in the Inspector. Enter on a
// focused control belongs to that control. Board rows carry role="button" and their own
// Enter handler, so they are covered by the role check and keep working.
export function ownsEnter(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el) return false;
  if (el.tagName === 'BUTTON' || el.tagName === 'A' || el.tagName === 'SELECT') return true;
  return el.getAttribute?.('role') === 'button';
}

// useBoardKeys wires Session-B's locked board navigation: J moves the cursor down, K moves
// it up, Enter opens the Inspector for the cursored row. Density (1/2/3), inspector toggle
// (i) and Escape stay in App.tsx — those are app-global, these are board-local.
//
// The CURSOR is deliberately distinct from the SELECTION. The cursor is where the keyboard
// is pointing; the selection is what the Inspector is showing. Keeping them separate is
// what lets the user travel the board with J/K without firing a GetPlayerScore fetch on
// every row they pass over — Enter is the commit. Collapsing them would make arrow-key
// browsing issue one IPC call per keystroke.
//
// ids must be the rows in DISPLAYED order (post-sort, post-filter). The cursor is tracked
// by ID rather than by index so a re-sort or a filter change keeps the cursor on the same
// player instead of teleporting it to whoever now occupies that row number.
//
// Returns setCursorID as well so the CLICK site can plant the cursor. A click is the
// clearest possible "I am here" statement, and ignoring it meant clicking row 50 and then
// pressing J jumped to row 1 instead of row 51 (GLM review, B-5).
export function useBoardKeys(ids: string[], onCommit: (id: string) => void) {
  const [cursorID, setCursorID] = useState<string | null>(null);

  // Refs keep the key handler stable across renders: the listener is bound once, and reads
  // current values at fire time. Re-binding a window listener on every data change would
  // churn on every board refresh.
  const idsRef = useRef(ids);
  idsRef.current = ids;
  const cursorRef = useRef(cursorID);
  cursorRef.current = cursorID;
  const commitRef = useRef(onCommit);
  commitRef.current = onCommit;

  const move = useCallback((step: number) => {
    const list = idsRef.current;
    if (list.length === 0) return;
    const at = cursorRef.current ? list.indexOf(cursorRef.current) : -1;
    // No cursor yet (or it scrolled out of the filtered set): J starts at the top, K at the
    // bottom — entering the list from the direction the user is travelling.
    if (at === -1) {
      setCursorID(step > 0 ? list[0] : list[list.length - 1]);
      return;
    }
    // Clamp rather than wrap. Wrapping from row 1200 back to row 1 on one extra keypress
    // is disorienting on a board this long, and there is no way to tell it happened.
    const next = Math.min(list.length - 1, Math.max(0, at + step));
    setCursorID(list[next]);
  }, []);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (isTypingTarget(e.target)) return;
      // Let the browser/app keep every modified chord (Ctrl-J, Cmd-Enter, …) — claiming
      // them would shadow real shortcuts for a plain navigation key.
      if (e.ctrlKey || e.metaKey || e.altKey) return;

      if (e.key === 'j' || e.key === 'J') {
        e.preventDefault();
        move(1);
      } else if (e.key === 'k' || e.key === 'K') {
        e.preventDefault();
        move(-1);
      } else if (e.key === 'Enter') {
        if (ownsEnter(e.target)) return; // the focused control's Enter, not ours
        const id = cursorRef.current;
        if (!id) return; // nothing cursored — let Enter through to whatever else wants it
        // Re-check the cursor against the CURRENT list before committing. The cleanup
        // effect that drops a vanished cursor runs after render, so a fast J→Enter across
        // a re-sort or filter change can otherwise commit a row that is no longer visible
        // (GLM review, B-5). Validating here closes the window instead of narrowing it.
        if (!idsRef.current.includes(id)) return;
        e.preventDefault();
        commitRef.current(id);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [move]);

  // Drop a cursor that no longer exists in the list (filtered out, or the board reloaded
  // without that player). Leaving a dangling cursor would let Enter open a row the user
  // can no longer see.
  useEffect(() => {
    if (cursorID && !ids.includes(cursorID)) setCursorID(null);
  }, [ids, cursorID]);

  return { cursorID, setCursorID };
}

// useScrollCursorIntoView keeps the cursored row on screen as J/K travel past the viewport.
// 'nearest' scrolls the minimum distance needed, so a cursor already visible does not cause
// the board to jump — the common case must be perfectly still.
export function useScrollCursorIntoView(cursorID: string | null) {
  useEffect(() => {
    if (!cursorID) return;
    const el = document.querySelector(`[data-row-id="${CSS.escape(cursorID)}"]`);
    el?.scrollIntoView({ block: 'nearest' });
  }, [cursorID]);
}
