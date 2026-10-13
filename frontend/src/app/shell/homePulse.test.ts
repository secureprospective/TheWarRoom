import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { commands, createCommands } from '../commands/registry';
import { FixtureProvider, LiveProvider } from '../data/provider';
import type { AlertReading, ClockReading, Sourced } from '../data/contract';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { parseAlerts } from '../data/parseAlerts';
import { clockDom, TestElement } from '../clock/testDom';
import AlertTray from './AlertTray';
import SeasonalCard from './SeasonalCard';
import * as registry from '../commands/registry';

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => { for (const root of roots.splice(0)) root.unmount(); });
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
async function flush() {
  for (let i = 0; i < 4; i += 1) await new Promise((resolve) => setTimeout(resolve, 0));
}
function buttons(element: TestElement): string[] {
  return element.children.flatMap((child): string[] => {
    if (!(child instanceof TestElement)) return [];
    return child.tagName === 'BUTTON' ? [child.textContent] : buttons(child);
  });
}
const provenance = heldLineup().provenance;
function alertReading(): AlertReading {
  return {
    alerts: [
      { kind: 'lineup', urgency: 'U2', title: 'Check your lineup', detail: 'Go: needs another WR',
        node: 'hq', workspace: 'lineup-and-roster', at: '2026-10-08T23:15:00Z', provenance },
      { kind: 'trades', urgency: 'U1', title: '1 offer to you', detail: 'Review pending trade offers',
        node: 'trade', workspace: 'trade-desk', at: '2026-10-09T18:00:00Z', provenance },
    ],
    unavailable: [
      { kind: 'ir', note: 'NFL injury status not yet read (MFL injuries export)' },
      { kind: 'trades', note: 'MFL timed out' },
    ],
  };
}
async function setup(reading: { alerts?: AlertReading; clock?: Sourced<ClockReading> } = {}) {
  const snapshot = heldSnapshot();
  const provider = new LiveProvider();
  vi.spyOn(provider, 'lineup').mockResolvedValue(heldLineup());
  vi.spyOn(provider, 'trades').mockResolvedValue({ offers: [], provenance });
  const alerts = vi.spyOn(provider, 'alerts').mockResolvedValue(reading.alerts ?? alertReading());
  vi.spyOn(provider, 'moves').mockResolvedValue([]);
  vi.spyOn(provider, 'onMovesChange').mockImplementation(() => () => {});
  vi.spyOn(provider, 'onSeasonChange').mockImplementation(() => () => {});
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: '0025' });
  await c.alerts.refresh('0025');
  vi.spyOn(commands, 'use').mockImplementation(c.use);
  for (const key of ['refresh', 'peek', 'subscribe', 'onSeasonChange', 'onMovesChange'] as const) {
    vi.spyOn(commands.alerts, key).mockImplementation(c.alerts[key]);
  }
  const dom = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
  const root = createRoot(dom.element);
  roots.push(root);
  await act(async () => {
    root.render(createElement(AlertTray, { snapshot, reading: reading.clock }));
    await flush();
  });
  const element = dom.element as unknown as TestElement;
  return {
    c, root, element, alerts, text: () => element.textContent, buttons: () => buttons(element),
  };
}

describe('Home and League Pulse boundary', () => {
  it('parses the Go readings and rejects malformed fields at their paths', () => {
    expect(parseAlerts(JSON.parse(JSON.stringify(alertReading())))).toEqual(alertReading());
    const badAlert = JSON.parse(JSON.stringify(alertReading()));
    badAlert.alerts[0].urgency = 'U9';
    expect(() => parseAlerts(badAlert)).toThrow('alerts[0].urgency');
  });
  it('says what needs the desktop app in the fixture build', async () => {
    const fixture = new FixtureProvider();
    await expect(fixture.alerts('0025')).rejects.toThrow('Alerts need the desktop app');
  });
});

describe('Alert tray', () => {
  it('shows alerts in Go order with routes, then what it cannot read yet', async () => {
    const t = await setup();
    const text = t.text();
    expect(text.indexOf('Check your lineup')).toBeLessThan(text.indexOf('1 offer to you'));
    expect(text).toContain('Go: needs another WR');
    expect(text).toContain('lock in');
    expect(t.buttons()).toEqual(['Open Franchise HQ', 'Open Trade desk']);
    expect(text).toContain('IR · NFL injury status not yet read (MFL injuries export)');
    expect(text).toContain('Trades · MFL timed out');
    expect(text.toLowerCase()).not.toContain('waiver');
  });
  it('pins clock deadlines above alerts with an action and ticking countdown', async () => {
    const now = vi.spyOn(Date, 'now').mockReturnValue(Date.parse('2026-10-08T22:00:00Z'));
    const clock: Sourced<ClockReading> = { provenance, value: {
      season: 2026, week: 5, phase: 'REGULAR_SEASON', windows: [], deadlines: [
        { id: 'lineup-w5', label: 'LINEUP_LOCK', at: '2026-10-08T23:15:00Z', urgency: 'U2',
          pinned: true, promoted: true },
        { id: 'not-pinned', label: 'Not yet', urgency: 'U1', pinned: false, promoted: false },
      ],
    } };
    const t = await setup({ clock, alerts: { alerts: [], unavailable: [] } });
    expect(t.text()).toContain('Lineup lock');
    expect(t.text()).toContain('1h 15m');
    expect(t.text()).not.toContain('Nothing needs you');
    expect(t.text()).not.toContain('Not yet');
    expect(t.buttons()).toEqual(['Act · Lineup lock']);
    now.mockReturnValue(Date.parse('2026-10-08T22:01:00Z'));
    const document = t.element.ownerDocument;
    document.visibility(false);
    expect(t.text()).toContain('1h 14m');
    vi.spyOn(registry, 'dispatch').mockImplementation(t.c.dispatch);
    const button = t.element.querySelectorAll('[aria-label]').find(e =>
      e.tagName === 'BUTTON' && e.getAttribute('aria-label') === 'Act · Lineup lock')!;
    const key = Object.keys(button).find(name => name.startsWith('__reactProps$'))!;
    const props = (button as unknown as Record<string, { onClick(): void }>)[key];
    await act(async () => props.onClick());
    expect(t.c.read().node).toBe('hq');
    expect(t.c.read().workspace.hq).toBe('lineup-and-roster');
  });
  it('says when nothing needs him', async () => {
    const t = await setup({ alerts: { alerts: [], unavailable: [] } });
    expect(t.text()).toContain('Nothing needs you right now');
  });
});

describe('Seasonal card', () => {
  it('shows season, phase, week and the next deadline', async () => {
    const dom = clockDom([]);
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    vi.stubGlobal('document', dom.document);
    vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
    const root = createRoot(dom.element);
    roots.push(root);
    const reading: Sourced<ClockReading> = { provenance, value: {
      season: 2026, week: 5, phase: 'REGULAR_SEASON', windows: [], deadlines: [{
        id: 'lineup-w5', label: 'LINEUP_LOCK', at: '2026-10-08T23:15:00Z', urgency: 'U2',
        pinned: true, promoted: true,
      }],
    } };
    await act(async () => { root.render(createElement(SeasonalCard, { reading })); });
    const text = (dom.element as unknown as TestElement).textContent;
    expect(text).toContain('Season 2026 · Regular season');
    expect(text).toContain('NFL week 5');
    expect(text).toContain('Lineup lock');
  });
});
