import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, it, expect, vi } from 'vitest';
import { report } from '../reports/contract';
import { createControlState, matching, parseBylaws, type ControlReader } from './state';
import wire from './wire.testdata.json';

const reader = (over: Partial<ControlReader> = {}): ControlReader => ({
  settings: async () => report(wire.settings, 'League settings'),
  bylaws: async () => parseBylaws(wire.bylaws),
  accounting: async () => report(wire.accounting, 'Accounting'),
  news: async () => report(wire.news, "What's New"),
  openAccounting: async () => undefined,
  ...over,
});

describe('Control Room', () => {
  it('reads the Go wire for League settings, by-laws, accounting and site news unchanged', async () => {
    const c = createControlState(reader());
    await c.choose('league');
    const s = c.peek();
    expect(s.settings?.rows[0].cells)
      .toEqual(['Scoring weeks', 'Weeks 1 to 17; the regular season ends after week 13']);
    expect(s.bylaws?.[0].title).toBe('1. SALARY CAP');
    expect(s.accounting?.status).toBe('empty');
    expect(s.news).toBeUndefined();
    await c.choose('news');
    expect(c.peek().news?.rows).toHaveLength(2);
    expect(c.peek().error).toBe('');
  });
  it('searches the by-laws by every word, in titles and lines', () => {
    const sections = parseBylaws(wire.bylaws);
    expect(matching(sections, '')).toHaveLength(sections.length);
    expect(matching(sections, 'practice squad').map(s => s.title)).toEqual(['2. ROSTERS']);
    expect(matching(sections, 'SALARY nearest').map(s => s.title)).toEqual(['1. SALARY CAP']);
    expect(matching(sections, 'salary zebra')).toEqual([]);
    expect(matching(sections, 'taxi cap').map(s => s.title)).toEqual(['2. ROSTERS']);
  });
  it('rejects malformed by-laws rather than showing a guess', () => {
    expect(() => parseBylaws([{ title: 'x' }])).toThrow();
    expect(() => parseBylaws({})).toThrow();
  });
  it('shows a failed read as an error and keeps what was already read', async () => {
    const c = createControlState(reader({ news: async () => { throw new Error('MFL did not answer'); } }));
    await c.choose('league');
    await c.choose('news');
    expect(c.peek().error).toContain('MFL did not answer');
    expect(c.peek().settings).toBeDefined();
    expect(c.peek().pending).toBe(false);
  });
  it('opens MFL accounting only through the hand-off, and search does not read anything', async () => {
    const open = vi.fn(async () => undefined);
    const settings = vi.fn(async () => report(wire.settings, 'League settings'));
    const c = createControlState(reader({ openAccounting: open, settings }));
    await c.choose('search', 'cap');
    expect(c.peek().search).toBe('cap');
    expect(settings).not.toHaveBeenCalled();
    await c.choose('open-accounting');
    expect(open).toHaveBeenCalledTimes(1);
  });
  it('reads site news as articles with MFL paragraphs, newest open, one heading', async () => {
    const { NewsArticles } = await import('./WhatsNew');
    const news = report(wire.news, "What's New");
    const html = renderToStaticMarkup(createElement(NewsArticles, { news }));
    expect(html.match(/<h[1-6]>[^<]*<\/h[1-6]>/g)).toEqual(['<h4>What&#x27;s New on MFL</h4>']);
    expect(html).toContain('data-endpoint="M-141"');
    expect(html.match(/<details open="">/g)).toHaveLength(1);
    const paragraphs = news.rows[0].cells[1].split('\n');
    expect(paragraphs.length).toBeGreaterThan(1);
    expect(html.split('<p>').length - 1).toBeGreaterThanOrEqual(paragraphs.length);
  });
  it('renders the League room with every section and the MFL hand-off', async () => {
    vi.resetModules();
    vi.doMock('./state', async () => {
      const real = await vi.importActual<typeof import('./state')>('./state');
      const c = real.createControlState(reader());
      await c.choose('league');
      return { ...real, controlState: { ...c, use: () => c.peek() } };
    });
    const { LeagueRoom } = await import('./LeagueRoom');
    const html = renderToStaticMarkup(createElement(LeagueRoom));
    expect(html).toContain('data-endpoint="M-185"');
    expect(html).toContain('1. SALARY CAP');
    expect(html).toContain('Search the by-laws');
    expect(html).toContain('Open MFL accounting');
    expect(html).toContain('From MFL (reference only)');
    expect(html).toContain('Refreshing MFL updates this reference');
    expect(html).toContain('holds no entries for this league');
    expect(html).toContain('Scoring weeks');
    expect(html.indexOf('League settings')).toBeLessThan(html.indexOf('By-laws'));
    vi.doUnmock('./state');
  });
});

it('publishes the league compound read only when every part succeeds', async () => {
  let failed = false;
  const accounting = report(wire.accounting, 'Accounting');
  const c = createControlState(reader({
    settings: async () => ({ ...report(wire.settings, 'Settings'), title: failed ? 'new' : 'held' }),
    accounting: async () => { if (failed) throw new Error('missing ledger'); return accounting; },
  }));
  await c.choose('league');
  failed = true;
  await c.choose('league');
  expect(c.peek()).toMatchObject({ settings: { title: 'held' }, accounting, pending: false });
  expect(c.peek().error).toContain('missing ledger');
  await c.choose('news');
  expect(c.peek().error).toBe('');
  expect(c.peek().settings?.title).toBe('held');
});

it('preserves the reader receiver for independent news reads', async () => {
  const c = createControlState(reader({ async news(this: ControlReader) { return this.settings(); } }));
  await c.choose('news');
  expect(c.peek().news?.title).toBe(report(wire.settings, 'League settings').title);
  expect(c.peek().error).toBe('');
});
