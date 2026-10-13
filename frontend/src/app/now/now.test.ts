import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { endpointById, placementRoute } from '../registry';
import { parseNow } from './contract';
import { NowBody, NowView } from './Now';
import { commands } from '../commands/registry';
import { createNowState, nowState, type NowReader } from './state';
import wire from './wire.testdata.json';

const now = () => parseNow(wire.now);
const view = (id: string) => renderToStaticMarkup(createElement(NowView, { now: now(), selected: [id] }));

describe('League Pulse › Now', () => {
  it.each(wire.now.board.reports.map(r => [r.id]))('%s opens below the week and my matchup', id => {
    const markup = view(id);
    const report = now().board.reports.find(r => r.id === id)!;
    expect(markup.indexOf('<label>Week')).toBeLessThan(markup.indexOf('data-endpoint="X-03"'));
    const button = markup.indexOf(`aria-label="${report.title}" aria-pressed="true"`);
    expect(markup.indexOf('data-endpoint="X-03"')).toBeLessThan(button);
    expect(button).toBeLessThan(markup.indexOf(`data-endpoint="${id}"`));
    const endpoint = endpointById(id as Parameters<typeof endpointById>[0]);
    expect(endpoint.status).not.toBe('spec');
    expect(placementRoute(endpoint.placement)).toEqual({ node: 'pulse', workspace: 'now' });
  });
  it('shows my matchup side by side under the chosen week, with names and never ids', () => {
    const markup = view('M-005');
    expect(endpointById('X-03').status).not.toBe('spec');
    expect(markup).toContain('<option value="5" selected="">Week 5</option>');
    expect(markup).toContain('aria-label="Inspect Buffalo Bills"');
    expect(markup).toContain('aria-label="Inspect Fields, Justin"');
    expect(markup).toContain('Fields, Justin</button> · NYJ');
    expect(markup).not.toMatch(/>00\d\d</);
    expect(markup).toContain('class="report-group"');
    expect(markup).toMatch(/class="report-mine" aria-current="true"/);
  });
  it('offers weekly reports independently of the live matchup week', async () => {
    const markup = view('M-003');
    expect(markup).toContain('<label>Weekly reports');
    const reader: NowReader = { read: vi.fn(async () => now()), refresh: vi.fn(async () => {}) };
    const state = createNowState(reader, () => '0001');
    await state.choose('reports-week', '4');
    expect(reader.read).toHaveBeenLastCalledWith(0, 4, '0001');
    expect(markup.match(/<option /g)?.length).toBe(now().weeks.length + now().reportWeeks.length);
  });
  it('dispatches the weekly report picker through the command registry', async () => {
    const choose = vi.spyOn(nowState, 'choose').mockResolvedValue(undefined);
    commands.dispatch('now.reports', { action: 'reports-week', value: '4' });
    await new Promise(resolve => setTimeout(resolve, 0));
    expect(choose).toHaveBeenCalledWith('reports-week', '4');
    choose.mockRestore();
  });
  it('rejects a board Go would never send', () => {
    const bad = (change: (r: typeof wire.now) => void) => {
      const copy = structuredClone(wire.now);
      change(copy);
      return () => parseNow(copy);
    };
    expect(bad(r => { r.week = 40; })).toThrow('week outside');
    expect(bad(r => { r.matchup.id = 'M-005'; })).toThrow('X-03');
    expect(bad(r => { r.matchup.rows[0].cells.pop(); })).toThrow('column count');
  });
  it('refreshes the selected report week so future reports can name its kickoff', async () => {
    const reader: NowReader = { read: vi.fn(async () => now()), refresh: vi.fn(async () => {}) };
    const state = createNowState(reader, () => '0001');
    await state.load(true);
    await state.choose('reports-week', '9');
    expect(reader.refresh).toHaveBeenLastCalledWith(9, false);
    expect(reader.read).toHaveBeenLastCalledWith(5, 9, '0001');
  });
  it('asks Go for the chosen week, refreshing that week, and follows my franchise', async () => {
    const reader: NowReader = { read: vi.fn(async () => now()), refresh: vi.fn(async () => {}) };
    const state = createNowState(reader, () => '0001');
    await state.load(true);
    expect(reader.refresh).toHaveBeenCalledWith(0, false);
    expect(reader.read).toHaveBeenLastCalledWith(0, 0, '0001');
    await state.choose('week', '3');
    expect(reader.refresh).toHaveBeenLastCalledWith(3, false);
    expect(reader.read).toHaveBeenLastCalledWith(3, 0, '0001');
    await state.choose('refresh', '');
    expect(reader.refresh).toHaveBeenLastCalledWith(5, true);
    expect(() => state.choose('week', 'x')).toThrow('invalid week');
  });
});

it('says the league\'s weeks are still being read instead of drawing an empty week (finding 96)', () => {
  const waiting = { ...now(), waiting: 'Reading the league\'s weeks from MFL…' };
  const body = (n: typeof waiting) =>
    renderToStaticMarkup(createElement(NowBody, { now: n, pending: false, error: '', selected: [] }));
  const cold = body(waiting);
  expect(cold).toContain('Reading the league&#x27;s weeks from MFL…');
  expect(cold).not.toContain('Weekly reports');
  expect(body(now())).toContain('Weekly reports');
});
