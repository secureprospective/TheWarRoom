import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { commands, createCommands } from '../commands/registry';
import { FixtureProvider, LiveProvider } from '../data/provider';
import type { AlertReading, ClockReading, PulseReading, PulseSide, Sourced } from '../data/contract';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { parsePulse } from '../data/parsePulse';
import { parseAlerts } from '../data/parseAlerts';
import { clockDom, TestElement } from '../clock/testDom';
import AlertTray from './AlertTray';
import PulseNow from './PulseNow';
import SeasonalCard from './SeasonalCard';

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
function side(franchiseId: string, name: string, score: number): PulseSide {
  return {
    franchiseId, name, score, secondsRemaining: 3600, playing: 1, yetToPlay: 20,
    starters: [{
      id: `1${franchiseId.slice(1)}`, name: `Star, ${name}`, position: 'QB', team: 'KC', score: 12.34,
      secondsRemaining: 1800,
    }],
  };
}
function pulse(): PulseReading {
  return {
    week: 5, provenance,
    matchups: [
      { home: side('0003', 'Third', 50), away: side('0004', 'Fourth', 40) },
      { home: side('0007', 'Rivals', 61.25), away: side('0025', 'Mine', 70) },
    ],
  };
}
function alertReading(): AlertReading {
  return {
    alerts: [
      { kind: 'lineup', urgency: 'U2', title: 'Check your lineup', detail: 'Go: needs another WR',
        node: 'hq', workspace: 'lineup-and-roster', at: '2026-10-08T23:15:00Z', provenance },
      { kind: 'trades', urgency: 'U1', title: '1 offer to you', detail: 'Review pending trade offers',
        node: 'trade', workspace: 'trade-desk-and-offers', at: '2026-10-09T18:00:00Z', provenance },
    ],
    unavailable: [
      { kind: 'ir', note: 'NFL injury status not yet read (MFL injuries export)' },
      { kind: 'trades', note: 'MFL timed out' },
    ],
  };
}
async function setup(
  screen: 'alerts' | 'pulse', reading: { alerts?: AlertReading; pulse?: PulseReading } = {},
) {
  const snapshot = heldSnapshot();
  const provider = new LiveProvider();
  vi.spyOn(provider, 'lineup').mockResolvedValue(heldLineup());
  vi.spyOn(provider, 'trades').mockResolvedValue({ offers: [], provenance });
  const alerts = vi.spyOn(provider, 'alerts').mockResolvedValue(reading.alerts ?? alertReading());
  const pulseNow = vi.spyOn(provider, 'pulseNow').mockResolvedValue(reading.pulse ?? pulse());
  vi.spyOn(provider, 'moves').mockResolvedValue([]);
  vi.spyOn(provider, 'onMovesChange').mockImplementation(() => () => {});
  vi.spyOn(provider, 'onSeasonChange').mockImplementation(() => () => {});
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: '0025' });
  await c.alerts.refresh('0025');
  for (const key of ['use', 'usePulse'] as const) vi.spyOn(commands, key).mockImplementation(c[key]);
  for (const store of ['alerts', 'pulse'] as const) {
    for (const key of ['refresh', 'peek', 'subscribe', 'onSeasonChange', 'onMovesChange'] as const) {
      vi.spyOn(commands[store], key).mockImplementation(c[store][key]);
    }
  }
  const dom = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
  const root = createRoot(dom.element);
  roots.push(root);
  await act(async () => {
    root.render(createElement(screen === 'alerts' ? AlertTray : PulseNow, { snapshot }));
    await flush();
  });
  const element = dom.element as unknown as TestElement;
  return {
    c, root, element, alerts, pulseNow, text: () => element.textContent, buttons: () => buttons(element),
  };
}

describe('Home and League Pulse boundary', () => {
  it('parses the Go readings and rejects malformed fields at their paths', () => {
    expect(parsePulse(JSON.parse(JSON.stringify(pulse())))).toEqual(pulse());
    expect(parseAlerts(JSON.parse(JSON.stringify(alertReading())))).toEqual(alertReading());
    const badPulse = JSON.parse(JSON.stringify(pulse()));
    badPulse.matchups[0].home.starters = null;
    expect(() => parsePulse(badPulse)).toThrow('matchups[0].home.starters');
    const badAlert = JSON.parse(JSON.stringify(alertReading()));
    badAlert.alerts[0].urgency = 'U9';
    expect(() => parseAlerts(badAlert)).toThrow('alerts[0].urgency');
  });
  it('says what needs the desktop app in the fixture build', async () => {
    const fixture = new FixtureProvider();
    await expect(fixture.pulseNow()).rejects.toThrow('League Pulse needs the desktop app');
    await expect(fixture.alerts('0025')).rejects.toThrow('Alerts need the desktop app');
  });
});

describe('Alert tray', () => {
  it('shows alerts in Go order with routes, then what it cannot read yet', async () => {
    const t = await setup('alerts');
    const text = t.text();
    expect(text.indexOf('Check your lineup')).toBeLessThan(text.indexOf('1 offer to you'));
    expect(text).toContain('Go: needs another WR');
    expect(text).toContain('lock in');
    expect(t.buttons()).toEqual(['Open Franchise HQ', 'Open Trade desk']);
    expect(text).toContain('IR · NFL injury status not yet read (MFL injuries export)');
    expect(text).toContain('Trades · MFL timed out');
    expect(text.toLowerCase()).not.toContain('waiver');
  });
  it('says when nothing needs him', async () => {
    const t = await setup('alerts', { alerts: { alerts: [], unavailable: [] } });
    expect(t.text()).toContain('Nothing needs you right now');
  });
});

describe('League Pulse Now', () => {
  it('puts his matchup first and open, the rest collapsed, and toggles', async () => {
    const t = await setup('pulse');
    const text = t.text();
    expect(text).toContain('Week 5');
    expect(text.indexOf('Mine')).toBeLessThan(text.indexOf('Third'));
    expect(text).toContain('61.3');
    expect(text).toContain('1 playing · 20 yet to play');
    expect(text).toContain('Star, Mine (QB, KC)');
    expect(text).not.toContain('Star, Third');
    expect(t.buttons()).toEqual(['Hide starters', 'Show starters']);
    await act(async () => { t.c.dispatch('pulse.toggle', { matchup: '5:0003:0004', expanded: false }); });
    expect(t.text()).toContain('Star, Third (QB, KC)');
  });
  it('keeps the matchups while a re-read is in flight and reports a failed feed', async () => {
    const t = await setup('pulse');
    t.pulseNow.mockReturnValueOnce(new Promise(() => {}));
    await act(async () => { void t.c.pulse.refresh('now'); await flush(); });
    expect(t.text()).toContain('Star, Mine');
    const failed = await setup('pulse', { pulse: { week: 5, matchups: [], provenance: {
      ...provenance, freshness: { ...provenance.freshness, state: 'fail', note: 'MFL timed out' },
    } } });
    expect(failed.text()).toContain('Live scoring unavailable · MFL timed out');
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
