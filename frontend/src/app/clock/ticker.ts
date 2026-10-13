import type { Deadline } from '../data/contract';
import { countdown, grade } from './urgency';

export function nextDelay(deadlines: readonly Deadline[], now: number): number | undefined {
  let delay: number | undefined;
  for (const deadline of deadlines) {
    if (deadline.at === undefined) continue;
    const remaining = Date.parse(deadline.at) - now;
    if (remaining <= 0) continue;
    if (remaining < 3_600_000) return 1000;
    const flip = (remaining % 60_000 || 60_000) + 25;
    if (delay === undefined || flip < delay) delay = flip;
  }
  return delay;
}

// One ticker paints every countdown under the root. Surfaces that mount after the reading arrives
// (the Status overlay, a lazily loaded Calendar panel) are found on each paint, and painted at once
// when they mount, so no countdown is left showing its placeholder until the next tick.
export function connectClock(
  root: HTMLElement,
  deadlines: readonly Deadline[],
  now: () => number = Date.now,
): () => void {
  const byId = new Map(deadlines.map((deadline) => [deadline.id, deadline]));
  const document = root.ownerDocument;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const clear = () => {
    if (timer !== undefined) clearTimeout(timer);
    timer = undefined;
  };
  const paint = (time: number) => {
    for (const node of Array.from(root.querySelectorAll<HTMLElement>('[data-deadline]'))) {
      const deadline = byId.get(node.getAttribute('data-deadline')!);
      if (!deadline) continue;
      node.setAttribute('data-urgency', grade(deadline.at, time));
      const text = node.querySelector('[data-countdown]');
      const label = countdown(deadline.at, time);
      if (text && text.textContent !== label) text.textContent = label;
    }
  };
  const tick = () => {
    clear();
    if (document.hidden) return;
    const time = now();
    paint(time);
    const delay = nextDelay(deadlines, time);
    if (delay !== undefined) timer = setTimeout(tick, delay);
  };
  // Painting only writes text, so it adds no element and cannot wake the observer again.
  const mounted = typeof MutationObserver === 'undefined' ? undefined : new MutationObserver((records) => {
    if (records.some((r) => Array.from(r.addedNodes).some((n) => n.nodeType === 1))) paint(now());
  });
  mounted?.observe(root, { childList: true, subtree: true });
  tick();
  document.addEventListener('visibilitychange', tick);
  return () => {
    clear();
    mounted?.disconnect();
    document.removeEventListener('visibilitychange', tick);
  };
}
