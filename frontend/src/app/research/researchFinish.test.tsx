import { readFileSync } from 'node:fs';
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { KeyDocument, descendants } from '../shell/mflKeyTestDom';
import { commands } from '../commands/registry';
import * as registry from '../commands/registry';
import { PlayerResearchView } from './PlayerInspector';
import { parseResearch, type PlayerReading } from './contract';
import { createResearchState, type ResearchReader } from './state';
import wire from './wire.testdata.json';

const fixture = parseResearch(wire);
const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map(key => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
let root: Root | undefined;
afterEach(async () => {
  if (root) await act(async () => root?.unmount());
  root = undefined;
  vi.restoreAllMocks(); vi.useRealTimers();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
describe('Research live-look fixes', () => {
  it('finishes once, names missing reports, and clears old progress on a complete refresh', async () => {
    const partial = { ...fixture, board: { ...fixture.board,
      reports: fixture.board.reports.map((report, index) => ({ ...report,
        status: index === 0 ? 'stale' as const : report.status })) } };
    const reader: ResearchReader = { read: async () => partial,
      refresh: async () => { throw new Error('playerScores: context deadline exceeded'); },
      news: async () => undefined, open: async () => undefined };
    const state = createResearchState(reader, () => '0001');
    await state.load(true);
    expect(state.peek().status).toMatch(/^Refresh finished\. Missing or out of date: /);
    expect(state.peek().status).not.toMatch(/playerScores|context deadline/);
    const complete = { ...fixture, board: { ...fixture.board,
      reports: fixture.board.reports.map(report => ({ ...report, status: 'ok' as const })) } };
    reader.read = async () => complete;
    reader.refresh = async () => undefined;
    await state.load(true);
    expect(state.peek().status).toBe('Refresh complete.');
    expect(state.peek().error).toBe('');
  });
  it('renders held reports and missing feed times under both Inspector shapes', () => {
    const player: PlayerReading = { ...fixture.players[0], report: {
      ...fixture.news, columns: ['Reading', 'Held fact', 'Feed time'], rows: [
        { cells: ['NFL depth', 'not held', 'not held · feed time not held'], range: null },
        { cells: ['News · Oct 5, 2026', fixture.articles[0].headline, 'held · fetched today'],
          range: null },
      ], summary: [], status: 'ok',
    }, articles: fixture.articles, howLabel: fixture.board.howLabel };
    for (const identity of [true, false]) {
      const html = renderToStaticMarkup(createElement(PlayerResearchView, { player, identity }));
      expect(html).toContain('not held · feed time not held');
      expect(html).toContain('Open article on MFL');
      expect(html.includes('<h4>Gibbs, Jahmyr</h4>')).toBe(identity);
      expect(html).not.toContain('Open this player from your roster');
    }
    const shell = readFileSync(new URL('../shell/PlayerInspector.tsx', import.meta.url), 'utf8');
    expect(shell).toContain('<ResearchInspector id={subject.id} />');
    expect(shell).toContain('<ResearchInspector id={subject.id} identity={false} />');
    expect(shell).toContain('<PlayerLists id={subject.id} />');
    expect(shell).toContain('<ContractPanel id={subject.id} />');
  });
  it('opens Inspector-held articles without a Research board and rejects forged articles', async () => {
    const player: PlayerReading = { ...fixture.players[0], report: fixture.news,
      articles: fixture.articles, howLabel: fixture.board.howLabel };
    const open = vi.fn(async () => undefined);
    const reader: ResearchReader = { read: async () => fixture, refresh: async () => undefined,
      news: async () => undefined, player: async () => player, open };
    const state = createResearchState(reader, () => '');
    await state.choose('open', '16162:rotowire640841');
    expect(open).toHaveBeenCalledWith('16162', 'rotowire640841');
    await expect(state.choose('open', '16162:forged')).rejects.toThrow('not held');
    expect(open).toHaveBeenCalledOnce();
  });
  it('shows the waiting and week-reading stages without losing the archived board', async () => {
    vi.useFakeTimers();
    let finish: (() => void) | undefined;
    let status = 'Waiting for the MFL refresh to finish';
    const reader: ResearchReader = { read: async () => fixture,
      refresh: () => new Promise(resolve => { finish = resolve; }),
      status: async () => status, news: async () => undefined, open: async () => undefined };
    const state = createResearchState(reader, () => '0001');
    await state.load();
    const pending = state.load(true);
    expect(state.peek().research).toBe(fixture);
    expect(state.peek().status).toBe(status);
    status = 'Reading week 3 of 5';
    vi.advanceTimersByTime(250);
    await Promise.resolve();
    expect(state.peek().status).toBe(status);
    finish?.(); await pending;
    expect(state.peek().pending).toBe(false);
    expect(state.peek().status).toBe('Refresh complete.');
    expect(state.peek().status).not.toContain('Reading week');
    expect(vi.getTimerCount()).toBe(0);
  });
});

describe('One paged news list', () => {
  it('does not render Inspector headlines a second time among the held facts', () => {
    const article = fixture.articles[0];
    const player: PlayerReading = { ...fixture.players[0], report: {
      ...fixture.news, columns: ['Reading', 'Held fact', 'Feed time'], rows: [
        { cells: ['News · Oct 5, 2026', article.headline, 'held · fetched today'], range: null },
      ], summary: [], status: 'ok',
    }, articles: [article], howLabel: fixture.board.howLabel };
    const html = renderToStaticMarkup(createElement(PlayerResearchView, { player, identity: true }));
    expect(html.match(/aria-label="Open article on MFL/g)).toHaveLength(1);
    expect(html).not.toContain('<dd>RB Jahmyr Gibbs:');
  });
});

it('Inspector news uses a whole Posted header and a local absolute minute from Go', () => {
  const player: PlayerReading = { ...fixture.players[0], report: {
    ...fixture.news, rows: [{ cells: ['News · Oct 13, 2026, 12:22 AM', 'Headline',
      'held · fetched Oct 13, 2026, 12:48 AM'], range: null }],
  }, articles: fixture.articles, howLabel: fixture.board.howLabel };
  const html = renderToStaticMarkup(createElement(PlayerResearchView, { player, identity: false }));
  expect(html).toContain('Posted');
  expect(html.match(/<thead>[\s\S]*?<\/thead>/)?.[0]).not.toContain('Published');
  expect(html).toContain('Oct 13, 2026, 12:22 AM');
  const css = readFileSync(new URL('../shell/shell.css', import.meta.url), 'utf8');
  expect(css).toContain('.twr-app .insp .draft-report th .act-button {');
  expect(css).toContain('overflow-wrap: normal; word-break: normal;');
});

it('starter grid uses intrinsic columns and scrolls rather than overlapping names', () => {
  const css = readFileSync('src/app/research/research.css', 'utf8');
  expect(css).toContain('width: max-content; min-width: 100%; table-layout: auto;');
  expect(css).toContain('white-space: nowrap; overflow-wrap: normal;');
});
