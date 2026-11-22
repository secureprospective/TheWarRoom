// isTypingTarget reports whether the user is currently typing, in which case a global
// shortcut must NOT fire: "j" belongs to the text field, not the board. This is the single
// definition of that guard; any new global key handler imports it.
export function isTypingTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  return !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable);
}
