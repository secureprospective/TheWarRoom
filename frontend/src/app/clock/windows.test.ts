import { readFileSync } from 'node:fs';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, it, expect, vi } from 'vitest';
import fixture from '../data/fixtures/clock.json';
import { parseClock } from '../data/parse';
import { FixtureProvider, LiveProvider } from '../data/provider';
import { createCommands } from '../commands/registry';
import SeasonalCard from '../shell/SeasonalCard';
import { CalendarPanel } from './CalendarPanel';
import { deadlineRoute } from './deadlineRoute';
import { localDate } from './labels';
import { localInput } from './WindowRow';
import { setWindow } from './windowState';

const at = '2026-11-02T18:00:00Z';
const moved = () => parseClock({
  ...fixture, value: { ...fixture.value, phase: 'REGULAR_SEASON',
    windows: fixture.value.windows.map(window => ['trade_deadline', 'cut_day'].includes(window.kind)
      ? { ...window, status: 'dated', at, moved: true, reason: 'Rulebook basis' } : window),
    deadlines: ['lineup-w5', 'window:trade_deadline', 'window:ufa_bidding', 'custom'].map(id => ({
      id, label: id, at, urgency: 'U1', pinned: false, promoted: false,
    })),
  },
});

describe('league window editing and destinations', () => {
  it('rejects a dated window without at and an unknown window with at', () => {
    const value = JSON.parse(JSON.stringify(fixture));
    value.value.windows[0].status = 'dated';
    expect(() => parseClock(value)).toThrow('date required exactly when window is dated');
    value.value.windows[0].status = 'unknown';
    value.value.windows[0].at = at;
    expect(() => parseClock(value)).toThrow('date required exactly when window is dated');
  });
  it('routes every deadline branch to its action workspace', () => {
    expect(deadlineRoute('lineup-w12')).toEqual({
      verb: 'nav.open', args: { node: 'hq', workspace: 'lineup-and-roster' },
    });
    expect(deadlineRoute('window:trade_deadline')).toEqual({
      verb: 'nav.open', args: { node: 'trade', workspace: 'trade-desk' },
    });
    for (const kind of ['playoff_bids_only', 'multiyear_bids_close', 'ufa_bidding']) {
      expect(deadlineRoute(`window:${kind}`)).toEqual({
        verb: 'nav.open', args: { node: 'war', workspace: 'valuations-and-pool' },
      });
    }
    expect(deadlineRoute('window:cut_day')).toEqual({ verb: 'calendar.summon', args: {} });
    expect(deadlineRoute('custom')).toEqual({ verb: 'calendar.summon', args: {} });
  });
  it('marks the current phase and makes every seasonal deadline an Act at G2', () => {
    const html = renderToStaticMarkup(createElement(SeasonalCard, { reading: moved() }));
    expect(html).toContain('twr-card gravity-g2');
    expect(html.match(/aria-current="true"/g)).toHaveLength(1);
    expect(html).toMatch(/class="act act-text on"[^>]*aria-label="Regular season · Lineup and roster"/);
    expect(html).toContain('Offseason · Contracts and cap');
    expect(html).toContain('Draft · Draft room');
    expect(html).toContain('Playoffs · Race');
    for (const label of ['lineup-w5', 'Trade deadline', 'UFA bidding', 'custom']) {
      expect(html).toContain(`aria-label="${label}"`);
    }
    expect(html).toContain('League windows and their sources');
    expect(html.match(/<button/g)).toHaveLength(9);
  });
  it('renders local dates, visible basis, commissioner marks and both calendar Acts', () => {
    const html = renderToStaticMarkup(createElement(CalendarPanel, { reading: moved() }));
    expect(html).toContain(localDate(at));
    expect(html).toContain('<p class="league-window-basis">Rulebook basis</p>');
    expect(html.match(/data-moved="true"/g)).toHaveLength(2);
    expect(html).not.toContain('set by the commissioner');
    expect(html).toContain(`type="datetime-local" value="${localInput(at)}"`);
    expect(html.match(/aria-label="Set date"/g)).toHaveLength(9);
    expect(html).toContain('Back to the rulebook');
    expect(html).toContain('Clear date');
    expect(html).toContain('Bids narrow to playoff teams');
    expect(html).toContain('Multi-year bids close');
    expect(html).toContain('No date');
  });
  it('window.set calls the provider and reports errors beside its row', async () => {
    const provider = new FixtureProvider();
    const save = vi.spyOn(provider, 'setWindow').mockRejectedValue(new Error('Date could not be saved'));
    const commands = createCommands(() => undefined, provider);
    commands.dispatch('window.set', { kind: 'cut_day', at });
    await new Promise(resolve => setTimeout(resolve, 30));
    expect(save).toHaveBeenCalledWith('cut_day', at);
    await setWindow(provider, 'draft', at);
    const html = renderToStaticMarkup(createElement(CalendarPanel, { reading: moved() }));
    expect(html).toContain('role="alert">Date could not be saved');
    expect(commands.registry['window.set'].gravity).toBe('G2');
  });
  it('fixture edits and resets its clock and emits a reread event', async () => {
    const provider = new FixtureProvider();
    const changed = vi.fn();
    const stop = provider.onClockChange(changed);
    await provider.setWindow('cut_day', at);
    expect((await provider.clock()).value.windows.find(w => w.kind === 'cut_day'))
      .toMatchObject({ moved: true, status: 'dated', at });
    await provider.setWindow('cut_day', '');
    expect((await provider.clock()).value.windows.find(w => w.kind === 'cut_day'))
      .toMatchObject({ moved: false, status: 'unknown' });
    expect(changed).toHaveBeenCalledTimes(2);
    stop();
  });
  it('live setWindow calls the Wails binding', async () => {
    const save = vi.fn().mockResolvedValue(undefined);
    const before = Object.getOwnPropertyDescriptor(globalThis, 'window');
    vi.stubGlobal('window', { go: { main: { App: { TargetSetWindow: save } } } });
    try {
      await new LiveProvider().setWindow('trade_deadline', at);
      expect(save).toHaveBeenCalledWith('trade_deadline', at);
    } finally {
      if (before) Object.defineProperty(globalThis, 'window', before);
      else Reflect.deleteProperty(globalThis, 'window');
    }
  });
});

it('gives the Calendar date fields the shell dark face, overridable by any screen', () => {
  const css = readFileSync('src/app/shell/shell.css', 'utf8');
  const rule = css.match(/\.twr-app :where\(input:not\([^)]*\), textarea\) \{([^}]+)\}/)![1];
  for (const want of ['color-scheme: dark', 'background-color: var(--surface-2)',
    'color: var(--surface-tx1)'])
    expect(rule).toContain(want);
});
