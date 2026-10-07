import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement, Profiler, useEffect, useRef } from 'react';
import { createRoot } from 'react-dom/client';
import { flushSync } from 'react-dom';
import { renderToStaticMarkup } from 'react-dom/server';
import fixture from '../data/fixtures/clock.json';
import { parseClock } from '../data/parse';
import { ClockStrip } from './ClockStrip';
import { CalendarPanel } from './CalendarPanel';
import { connectClock } from './ticker';
import { clockDom } from './testDom';
import { useLeagueClock } from './useLeagueClock';
import { selectProvider } from '../data/provider';

const reading = parseClock(fixture);
const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));

afterEach(() => {
  vi.useRealTimers();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});

describe('honest clock surfaces', () => {
  it('shows the real offseason, empty calendar and seven unknown windows without dates', () => {
    const strip = renderToStaticMarkup(createElement(ClockStrip, { reading }));
    const panel = renderToStaticMarkup(createElement(CalendarPanel, { reading }));
    expect(strip).toContain('Offseason · no deadlines');
    expect(strip).toContain('title="Offseason · no deadlines on the commissioner calendar"');
    expect(strip).toContain('target-calendar');
    expect(panel).toContain('No deadlines on the commissioner calendar.');
    expect(panel.match(/ · unknown · gap closure/g)).toHaveLength(7);
    expect(panel.match(/title="league rules not yet captured/g)).toHaveLength(7);
    expect(panel).toContain('fixture');
    for (const html of [strip, panel]) {
      expect(html).not.toMatch(/\d{4}-\d{2}-\d{2}|<time|data-deadline|data-countdown/);
    }
  });
  it('shows loading and failed readings, not a guessed phase or fixture fallback', () => {
    const failed = {
      ...reading,
      provenance: {
        ...reading.provenance,
        kind: 'live' as const,
        freshness: { state: 'fail' as const, fetchedAt: '', note: 'clock offline' },
      },
    };
    for (const component of [ClockStrip, CalendarPanel]) {
      expect(renderToStaticMarkup(createElement(component))).toContain('Clock loading…');
      const html = renderToStaticMarkup(createElement(component, { reading: failed }));
      expect(html).toContain('Clock unavailable');
      expect(html).toContain('signal-red');
      expect(html).toContain('clock offline');
      expect(html).not.toContain('Offseason');
      expect(html).not.toContain('unknown · gap closure');
    }
  });
  it.each([
    [true, true, 'pinned'],
    [true, false, 'pinned'],
    [false, true, 'promoted'],
    [false, false, undefined],
  ] as const)('shows only the applicable marker for pinned=%s promoted=%s', (pinned, promoted, marker) => {
    const synthetic = {
      ...reading,
      value: {
        ...reading.value,
        deadlines: [{
          id: 'trade', label: 'TRADE_DEADLINE', at: '2026-10-06T13:00:00Z',
          urgency: 'U2' as const, pinned, promoted,
        }],
      },
    };
    const html = renderToStaticMarkup(createElement(CalendarPanel, { reading: synthetic }));
    expect(html).toContain('Trade deadline');
    for (const label of ['pinned', 'promoted']) {
      if (label === marker) expect(html).toContain(` · ${label}`);
      else expect(html).not.toContain(` · ${label}`);
    }
    expect(html).toContain(new Intl.DateTimeFormat(undefined, {
      dateStyle: 'medium', timeStyle: 'short',
    }).format(new Date('2026-10-06T13:00:00Z')));
  });
  it('mounts both surfaces while snapshot stays pending and cleans up on unmount', async () => {
    const { root, element, document } = clockDom([]);
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    const snapshot = vi.fn().mockImplementation(() => new Promise(() => undefined));
    const clock = vi.fn().mockResolvedValue({
      ...fixture, provenance: { ...fixture.provenance, kind: 'live' },
    });
    vi.stubGlobal('document', document);
    vi.stubGlobal('window', {
      document, HTMLIFrameElement: class {},
      go: { main: { App: { TargetSnapshot: snapshot, TargetClock: clock } } },
    });
    function Harness() {
      const ref = useRef<HTMLDivElement>(null);
      const clockReading = useLeagueClock(ref);
      useEffect(() => { void selectProvider().snapshot(); }, []);
      return createElement('div', { ref },
        createElement(ClockStrip, { reading: clockReading }),
        createElement(CalendarPanel, { reading: clockReading }),
      );
    }
    const mounted = createRoot(element);
    await act(async () => { mounted.render(createElement(Harness)); });
    expect(snapshot).toHaveBeenCalledTimes(1);
    expect(clock).toHaveBeenCalledTimes(1);
    expect(root.textContent).toContain('Offseason · no deadlines');
    expect(root.textContent.match(/ · unknown · gap closure/g)).toHaveLength(7);
    expect(document.listeners.size).toBe(1);
    await act(async () => { mounted.unmount(); });
    expect(document.listeners.size).toBe(0);
  });
  it('keeps a mounted React render-count probe at one across ten actual DOM ticks', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-10-06T12:00:00Z'));
    const { root, element, document } = clockDom([]);
    vi.stubGlobal('document', document);
    vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
    const synthetic = {
      ...reading,
      value: {
        ...reading.value,
        deadlines: [{
          id: 'trade', label: 'TRADE_DEADLINE', at: '2026-10-06T12:01:00Z',
          urgency: 'U3' as const, pinned: true, promoted: true,
        }],
      },
    };
    let renders = 0;
    let commits = 0;
    function Probe() {
      renders += 1;
      return createElement(ClockStrip, { reading: synthetic });
    }
    const mounted = createRoot(element);
    flushSync(() => mounted.render(createElement(Profiler, {
      id: 'clock-strip', onRender: () => { commits += 1; },
    }, createElement(Probe))));
    const stop = connectClock(element, synthetic.value.deadlines);
    expect(root.textContent).toContain('1m');
    for (let tick = 0; tick < 10; tick += 1) {
      vi.advanceTimersByTime(1000);
    }
    expect(root.textContent).toContain('under 1m');
    expect(root.querySelector('[data-deadline]')?.getAttribute('data-urgency')).toBe('U3');
    expect(renders).toBe(1);
    expect(commits).toBe(1);
    stop();
    flushSync(() => mounted.unmount());
    expect(vi.getTimerCount()).toBe(0);
  });
});
