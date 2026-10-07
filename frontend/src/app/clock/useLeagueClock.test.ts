import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement, useRef } from 'react';
import { createRoot } from 'react-dom/client';
import type { ClockReading, Sourced } from '../data/contract';
import * as providers from '../data/provider';
import fixture from '../data/fixtures/clock.json';
import { parseClock } from '../data/parse';
import { ClockStrip } from './ClockStrip';
import { CalendarPanel } from './CalendarPanel';
import { clockDom } from './testDom';
import { useLeagueClock } from './useLeagueClock';

const originalGlobals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  for (const { key, descriptor } of originalGlobals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});

function pendingReading() {
  let resolve!: (reading: Sourced<ClockReading>) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<Sourced<ClockReading>>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

function weekReading(week: number) {
  return parseClock({
    ...fixture,
    value: {
      ...fixture.value,
      phase: 'REGULAR_SEASON',
      week,
      deadlines: [{
        id: `lineup-w${week}`,
        label: 'LINEUP_LOCK',
        at: new Date(Date.parse('2026-10-08T21:00:00Z') + (week - 5) * 604_800_000).toISOString(),
        urgency: 'U2',
        pinned: true,
        promoted: true,
      }],
    },
  });
}

async function mountClock(clock: () => Promise<Sourced<ClockReading>>) {
  const { root, element, document } = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', document);
  vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
  let event!: () => void;
  const unsubscribe = vi.fn();
  const onClockChange = vi.fn((listener: () => void) => {
    event = listener;
    return unsubscribe;
  });
  vi.spyOn(providers, 'selectProvider').mockReturnValue({
    mflKey: vi.fn(),
    connectMFL: vi.fn(),
    forgetMFL: vi.fn(),
    kind: 'fixture',
    reason: 'test',
    demo: vi.fn(),
    snapshot: vi.fn(),
    clock,
    onClockChange,
    onSeasonChange: () => () => {},
    lineup: vi.fn(),
  });
  let renders = 0;
  function StripProbe({ reading }: { reading?: Sourced<ClockReading> }) {
    renders += 1;
    return createElement(ClockStrip, { reading });
  }
  function Harness() {
    const ref = useRef<HTMLDivElement>(null);
    const reading = useLeagueClock(ref);
    return createElement('div', { ref },
      createElement(StripProbe, { reading }),
      createElement(CalendarPanel, { reading }),
    );
  }
  const mounted = createRoot(element);
  await act(async () => { mounted.render(createElement(Harness)); });
  return { root, mounted, event, unsubscribe, onClockChange, renders: () => renders, document };
}

it('one event reads and renders once, then unsubscribes on unmount', async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-10-07T12:00:00Z'));
  const clock = vi.fn().mockResolvedValueOnce(weekReading(5)).mockResolvedValueOnce(weekReading(6));
  const test = await mountClock(clock);
  expect(clock).toHaveBeenCalledTimes(1);
  expect(test.root.textContent).toContain('Regular season · Wk 5 · Lineup lock · 1d 9h');
  const renders = test.renders();
  await act(async () => { test.event(); });
  expect(clock).toHaveBeenCalledTimes(2);
  expect(test.renders()).toBe(renders + 1);
  expect(test.onClockChange).toHaveBeenCalledTimes(1);
  expect(test.root.textContent).toContain('Regular season · Wk 6 · Lineup lock · 8d 9h');
  expect(test.root.textContent).toContain('NFL week 6');
  expect(test.document.listeners.size).toBe(1);
  await act(async () => { test.mounted.unmount(); });
  expect(test.unsubscribe).toHaveBeenCalledTimes(1);
  expect(test.document.listeners.size).toBe(0);
  expect(vi.getTimerCount()).toBe(0);
});

it('drops older resolutions, including the initial read, and ignores late unmounted work', async () => {
  const initial = pendingReading();
  const older = pendingReading();
  const newer = pendingReading();
  const late = pendingReading();
  const clock = vi.fn()
    .mockReturnValueOnce(initial.promise)
    .mockReturnValueOnce(older.promise)
    .mockReturnValueOnce(newer.promise)
    .mockReturnValueOnce(late.promise);
  const test = await mountClock(clock);
  await act(async () => { test.event(); test.event(); });
  expect(clock).toHaveBeenCalledTimes(3);
  await act(async () => { newer.resolve(weekReading(6)); });
  expect(test.root.textContent).toContain('Regular season · Wk 6');
  const renders = test.renders();
  await act(async () => {
    older.resolve(weekReading(5));
    initial.resolve(weekReading(4));
  });
  expect(test.renders()).toBe(renders);
  expect(test.root.textContent).toContain('NFL week 6');
  await act(async () => { test.event(); });
  await act(async () => { test.mounted.unmount(); });
  await act(async () => { late.resolve(weekReading(7)); });
  expect(test.renders()).toBe(renders);
  expect(test.unsubscribe).toHaveBeenCalledTimes(1);
});

it('keeps the last reading on rejection and retries only on the next event', async () => {
  const clock = vi.fn()
    .mockResolvedValueOnce(weekReading(5))
    .mockRejectedValueOnce(new Error('refresh failed'))
    .mockResolvedValueOnce(weekReading(6));
  const test = await mountClock(clock);
  const renders = test.renders();
  await act(async () => { test.event(); });
  expect(clock).toHaveBeenCalledTimes(2);
  expect(test.renders()).toBe(renders);
  expect(test.root.textContent).toContain('Regular season · Wk 5');
  await act(async () => { test.event(); });
  expect(clock).toHaveBeenCalledTimes(3);
  expect(test.root.textContent).toContain('Regular season · Wk 6');
  await act(async () => { test.mounted.unmount(); });
});


it('keeps the initial failed reading visible until an event delivers a good reading', async () => {
  const failed = {
    ...weekReading(5),
    provenance: {
      ...weekReading(5).provenance,
      freshness: { state: 'fail' as const, fetchedAt: '', note: 'TargetClock failed: offline' },
    },
  };
  const clock = vi.fn().mockResolvedValueOnce(failed).mockResolvedValueOnce(weekReading(5));
  const test = await mountClock(clock);
  expect(test.root.textContent).toContain('Clock unavailable');
  expect(test.root.textContent).not.toContain('Regular season');
  await act(async () => { test.event(); });
  expect(test.root.textContent).toContain('Regular season · Wk 5');
  await act(async () => { test.mounted.unmount(); });
});
