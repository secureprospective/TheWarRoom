import { afterEach, describe, expect, it, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { connectClock } from '../clock/ticker';
import { clockDom } from '../clock/testDom';
import { entriesOf, parseControls, searchControls } from './controls';
import { countdown, localTime, parseJobs } from './jobs';
import { JobsStrip } from './JobsStrip';
import { ControlSearch } from './ControlSearch';
import { ControlRow } from './ControlRow';
import { parseOpen, scopedRules } from './landing';
import { parseRules } from './rules';
import controlsWire from './controls.wire.testdata.json';
import jobsWire from './jobs.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';

const entries = entriesOf(parseControls(controlsWire));
const html = (node: ReturnType<typeof createElement>) => renderToStaticMarkup(node);
afterEach(() => { vi.restoreAllMocks(); });

describe('League landing wire and search', () => {
  it('finds all 56 controls by their exact question, including those not built', () => {
    expect(entries).toHaveLength(56);
    for (const entry of entries) {
      expect(entry.question.length).toBeGreaterThan(10);
      expect(searchControls(entries, entry.question).map(h => h.entry.id)).toContain(entry.id);
    }
    expect(searchControls(entries, '  WHO GETS PRIORITY  ').map(h => h.entry.id)).toContain('CM-012');
    expect(searchControls(entries, '')).toEqual([]);
    expect(searchControls(entries, 'purple elephant')).toEqual([]);
  });
  it('finds every rule by its own label and opens its group with that field selected', () => {
    for (const entry of entries) for (const field of entry.fields) {
      const hit = searchControls(entries, field.label).find(h => h.entry.id === entry.id);
      expect(hit?.via).toEqual(field);
      const opened = parseOpen(`${entry.id} ${field.field}`, entries);
      expect(opened?.control.group).toBe(entry.group);
      expect(opened?.field).toBe(field.field);
    }
    expect(parseOpen('missing', entries)).toBeNull();
    const calendar = entries.filter(e => e.group === 'calendar');
    const owned = new Set(calendar.flatMap(e => e.fields.map(f => f.field)));
    const scoped = scopedRules(parseRules(rulesWire), owned);
    expect(scoped.sections.flatMap(s => s.rows).every(r => owned.has(r.field))).toBe(true);
    expect(scoped.sections.flatMap(s => s.rows).length).toBeGreaterThan(0);
  });
  it('shows unbuilt questions honestly rather than claiming they work, without internal codes', () => {
    for (const entry of entries.filter(e => !e.built)) {
      const row = html(createElement(ControlRow, {
        entry, rules: new Map(), mayChange: false, refer: () => undefined,
      }));
      expect(row).toContain('Not built yet');
      expect(row).not.toContain('Available now');
      expect(row).not.toMatch(/CM-\d|G[123]\b/);
      expect(row).toContain(entry.question.replaceAll('&', '&amp;').replaceAll("'", '&#x27;'));
    }
    const search = html(createElement(ControlSearch, { entries }));
    expect(search).toContain('Search the league controls');
    expect(search).not.toMatch(/CM-\d|G[123]\b/);
  });
  it('keeps the native landing lazy and first, and all MFL references below it', () => {
    const source = readFileSync(new URL('../control/LeagueRoom.tsx', import.meta.url), 'utf8');
    expect(source).toContain("lazy(() => import('../commissioner/LeagueLanding'))");
    expect(source).toContain('<LeagueLanding />');
    expect(source.indexOf('<LeagueLanding')).toBeLessThan(source.indexOf('s.settings &&'));
    for (const part of ['MFL settings, by-laws and accounting', 's.settings', 's.bylaws', 's.accounting',
      'Open MFL accounting', 'bylaw-search']) expect(source).toContain(part);
    expect(source).not.toContain('<LeagueDesk');
  });
});

describe('League jobs strip', () => {
  it('renders the Go wire in live-first order, blocked reasons and truthful unbuilt states', () => {
    for (const view of Object.values(jobsWire)) {
      const reading = parseJobs(view);
      const strip = html(createElement(JobsStrip, {
        jobs: reading, clock: () => Date.parse('2027-05-02T12:00Z'),
      }));
      const states = [...strip.matchAll(/data-state="([^"]+)"/g)].map(m => m[1]);
      expect(states).toEqual(reading.jobs.map(j => j.state));
      const rank = { live: 0, upcoming: 1, blocked: 2, unbuilt: 3 };
      expect(states.map(s => rank[s as keyof typeof rank]))
        .toEqual([...states.map(s => rank[s as keyof typeof rank])].sort());
      for (const job of reading.jobs) {
        expect(strip).toContain(job.name);
        if (job.state === 'blocked') expect(strip).toContain(job.words);
        if (job.state === 'unbuilt') expect(job.words).toBe('Not built yet');
      }
      expect(strip.replace(/<[^>]+>/g, '')).not.toMatch(/CM-\d|G[123]\b/);
    }
  });
  it('shows countdown units and the closing instant in viewer-local time, never forced UTC', () => {
    const zone = vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions');
    const resolved = new Intl.DateTimeFormat().resolvedOptions();
    zone.mockReturnValue({ ...resolved, timeZone: 'America/Los_Angeles' });
    const iso = '2027-05-03T04:00:00Z';
    const expected = new Date(iso).toLocaleString(undefined, {
      weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
      timeZoneName: 'short', timeZone: 'America/Los_Angeles',
    });
    expect(localTime(iso)).toBe(expected);
    const strip = html(createElement(JobsStrip, {
      jobs: parseJobs(jobsWire.live), clock: () => Date.parse('2027-05-02T12:00:00Z'),
    }));
    expect(strip).toContain('16 hours 0 minutes');
    expect(strip).toContain(expected);
    expect(countdown(iso, Date.parse('2027-05-01T03:00:00Z'))).toBe('2 days 1 hour');
    expect(countdown(iso, Date.parse('2027-05-03T03:59:00Z'))).toBe('1 minute');
    expect(countdown(iso, Date.parse('2027-05-03T05:00:00Z'))).toBe('Deadline passed');
  });
  it('reuses the DOM ticker for plain countdowns, updates and cancels on unmount', () => {
    vi.useFakeTimers();
    const now = Date.parse('2027-05-03T03:58:00Z');
    vi.setSystemTime(now);
    const { root, element } = clockDom(['contract']);
    const stop = connectClock(element, [{ id: 'contract', label: 'Contract season',
      at: '2027-05-03T04:00:00Z', urgency: 'U3', pinned: false, promoted: false,
    }], Date.now, (at, time) => countdown(at!, time));
    try {
      expect(root.textContent).toBe('2 minutes');
      vi.advanceTimersByTime(60_000);
      expect(root.textContent).toBe('1 minute');
      vi.advanceTimersByTime(60_000);
      expect(root.textContent).toBe('Deadline passed');
    } finally {
      stop();
      expect(vi.getTimerCount()).toBe(0);
      vi.useRealTimers();
    }
  });
  it('rejects malformed jobs and controls rather than guessing states', () => {
    expect(() => parseJobs({ ...jobsWire.live, jobs: [{ ...jobsWire.live.jobs[0], state: 'working' }] }))
      .toThrow();
    expect(() => parseControls({ groups: [{ ...controlsWire.groups[0], controls: [{ built: 'yes' }] }] }))
      .toThrow();
  });
});
