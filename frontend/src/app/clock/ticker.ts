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

export function connectClock(
  root: HTMLElement,
  deadlines: readonly Deadline[],
  now: () => number = Date.now,
): () => void {
  const byId = new Map(deadlines.map((deadline) => [deadline.id, deadline]));
  const nodes = Array.from(root.querySelectorAll<HTMLElement>('[data-deadline]'));
  const document = root.ownerDocument;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const clear = () => {
    if (timer !== undefined) clearTimeout(timer);
    timer = undefined;
  };
  const tick = () => {
    clear();
    if (document.hidden) return;
    const time = now();
    for (const node of nodes) {
      const deadline = byId.get(node.getAttribute('data-deadline')!);
      if (!deadline) continue;
      node.setAttribute('data-urgency', grade(deadline.at, time));
      const text = node.querySelector('[data-countdown]');
      if (text) text.textContent = countdown(deadline.at, time);
    }
    const delay = nextDelay(deadlines, time);
    if (delay !== undefined) timer = setTimeout(tick, delay);
  };
  tick();
  document.addEventListener('visibilitychange', tick);
  return () => {
    clear();
    document.removeEventListener('visibilitychange', tick);
  };
}
