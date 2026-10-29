import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import wire from './wire.testdata.json';
import { parseResearch } from './contract';
import { createResearchState, type ResearchReader } from './state';
import { NewsView, ResearchView } from './Research';
import { reportSort, sortedRows } from '../reports/sortState';

const fixture = parseResearch(wire);
function reader(): ResearchReader {
  return { read: vi.fn(async () => fixture), refresh: vi.fn(async () => undefined),
    news: vi.fn(async () => undefined), open: vi.fn(async () => undefined) };
}
describe('Player research', () => {
  it('decodes Go wire and rejects new fields and malformed identity at the boundary', () => {
    expect(fixture.board.reports.map(r => r.id)).toEqual([
      'M-038', 'M-037', 'M-040', 'M-057', 'M-051', 'M-048', 'M-052', 'M-053', 'M-054', 'M-060',
    ]);
    expect(fixture.news.id).toBe('M-058');
    expect(fixture.articles[0].article).toBe('rotowire640841');
    expect(() => parseResearch({ ...wire, unexpected: 1 })).toThrow('unexpected field');
    expect(() => parseResearch({ ...wire, players: [{ ...wire.players[0], id: 'bad' }] }))
      .toThrow('canonical player');
    expect(fixture.board.reports.find(r => r.id === 'M-038')?.rows[0].mine).toBe(true);
  });
  it('renders the search first, report choices, source, counting disclosure and controls', () => {
    const html = renderToStaticMarkup(createElement(ResearchView, {
      research: fixture, selected: ['M-052'],
    }));
    expect(html.indexOf('data-endpoint="M-062"')).toBeLessThan(html.indexOf('<h4>Player research'));
    expect(html).toContain('Gibbs, Jahmyr');
    expect(html).toContain('open a player in the Inspector');
    expect(html).toContain('Activity');
    expect(html).toContain('MFL-wide, not this league');
    expect(html).toContain('How this is counted');
    expect(html).not.toContain('confidence');
  });
  it('makes group, depth-grid and game-injury choices reachable and names article hand-offs', () => {
    for (const [id, label] of [['M-040', 'Position group'], ['M-048', 'Starter grid'],
      ['M-057', 'This week’s games']]) {
      const html = renderToStaticMarkup(createElement(ResearchView, { research: fixture, selected: [id] }));
      expect(html).toContain(label);
    }
    const html = renderToStaticMarkup(createElement(NewsView, { research: fixture }));
    expect(html).toContain('data-endpoint="M-058"');
    expect(html).toContain('data-endpoint="M-059"');
    expect(html).toContain('Oct 5, 2026');
    expect(html).not.toContain('6 days');
    expect(html).toContain('Open article on MFL');
    expect(html).toContain('Source: MFL player news');
  });
  it('dispatches refresh, filters, daily scoped news and only held article hand-offs', async () => {
    const r = reader();
    const state = createResearchState(r, () => '0001');
    await state.load();
    expect(r.refresh).not.toHaveBeenCalled();
    await state.choose('group', 'IDP');
    expect(r.read).toHaveBeenLastCalledWith(expect.objectContaining({ group: 'IDP', mine: '0001' }));
    await state.choose('refresh', '');
    expect(r.refresh).toHaveBeenCalledWith(true);
    await state.choose('news', 'mine');
    expect(r.news).toHaveBeenLastCalledWith('0001');
    await state.choose('news', 'league');
    expect(r.news).toHaveBeenLastCalledWith('');
    await state.choose('open', '16162:rotowire640841');
    expect(r.open).toHaveBeenCalledWith('16162', 'rotowire640841');
    await expect(state.choose('open', '16162:forged')).rejects.toThrow('not held');
    await expect(state.choose('invalid', '')).rejects.toThrow('invalid choice');
  });
  it('keeps the old report on a failed refresh and rejects out-of-order replies', async () => {
    const r = reader();
    const state = createResearchState(r, () => '0001');
    await state.load();
    r.refresh = vi.fn(async () => { throw new Error('fixture unavailable'); });
    await state.choose('refresh', '');
    expect(state.peek().research).toBe(fixture);
    expect(state.peek().error).toBe('');
    expect(state.peek().status).toContain('Refresh finished');
    expect(state.peek().status).not.toContain('fixture unavailable');
    r.refresh = vi.fn(async () => undefined);
    await state.choose('refresh', '');
    expect(state.peek().error).toBe('');
    let first: ((value: typeof fixture) => void) | undefined;
    let second: ((value: typeof fixture) => void) | undefined;
    r.read = vi.fn().mockImplementationOnce(() => new Promise(resolve => { first = resolve; }))
      .mockImplementationOnce(() => new Promise(resolve => { second = resolve; }));
    const a = state.choose('team', 'BUF');
    const b = state.choose('team', 'DET');
    const newer = { ...fixture, scope: { ...fixture.scope, team: 'DET' } };
    second?.(newer); await b;
    first?.(fixture); await a;
    expect(state.peek().research).toBe(newer);
    expect(state.peek().pending).toBe(false);
  });
  it('keeps a refresh status failure after the read succeeds and stops polling', async () => {
    vi.useFakeTimers();
    try {
      const r = reader();
      let finish!: () => void;
      r.refresh = vi.fn(() => new Promise<void>(resolve => { finish = resolve; }));
      r.status = vi.fn(async () => { throw new Error('status unavailable'); });
      const state = createResearchState(r, () => '0001');
      const pending = state.choose('refresh', '');
      expect(state.peek().status).toBe('Waiting for the MFL refresh to finish');
      await vi.advanceTimersByTimeAsync(250);
      expect(state.peek().error).toBe('');
      finish();
      await pending;
      expect(state.peek()).toMatchObject({ research: fixture, pending: false });
      expect(state.peek().error).toBe('');
      await vi.advanceTimersByTimeAsync(1000);
      expect(r.status).toHaveBeenCalledTimes(1);
    } finally { vi.useRealTimers(); }
  });
  it('sorts the actual Go points and names instead of formatted numeric text', () => {
    const stats = fixture.board.reports.find(r => r.id === 'M-037')!;
    const asc = sortedRows(stats.rows, { column: 3, desc: false });
    expect(asc[0].sort![3]).toBeLessThanOrEqual(asc[1].sort![3]);
    expect(() => reportSort.set({ id: 'M-037', column: 8 })).not.toThrow();
    expect(() => reportSort.set({ id: 'M-037', column: 23 })).toThrow('invalid column');
  });
});
