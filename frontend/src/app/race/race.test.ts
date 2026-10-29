import { readFileSync } from 'node:fs';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { endpointById, placementRoute } from '../registry';
import { parseRace } from './contract';
import { RaceView } from './Race';
import { createRaceState, type RaceReader } from './state';
import wire from './wire.testdata.json';

const race = () => parseRace(wire.race);
const html = renderToStaticMarkup;
const view = (id: string) => html(createElement(RaceView, { race: race(), selected: [id] }));

describe('League Pulse › Race', () => {
  it.each(['M-008', 'C-02'])('offers roster counts with Starting lineup selected for %s', id => {
    const markup = view(id);
    expect(markup).toContain('<label>Roster count');
    expect(markup).toContain('<option value="lineup" selected="">Starting lineup</option>');
    expect(markup).toContain('Whole roster');
    expect(markup).toContain('Best 21 (any position)');
    const unknown = parseRace({ ...wire.race, starterN: 0 });
    expect(html(createElement(RaceView, { race: unknown, selected: [id] })))
      .toContain('Best starters (any position)');
  });
  it.each(wire.race.board.reports.map(r => [r.id]))('%s opens below the standings and its buttons', id => {
    const markup = view(id);
    const report = race().board.reports.find(r => r.id === id)!;
    expect(markup.indexOf('data-endpoint="M-002"'))
      .toBeLessThan(markup.indexOf('aria-label="League race reports"'));
    expect(markup.indexOf(`aria-label="${report.title}" aria-pressed="true"`))
      .toBeLessThan(markup.indexOf(`data-endpoint="${id}"`));
    expect(markup).toContain(report.note.replaceAll("'", '&#x27;'));
    const endpoint = endpointById(id as Parameters<typeof endpointById>[0]);
    expect(endpoint.status).not.toBe('spec');
    expect(placementRoute(endpoint.placement)).toEqual({ node: 'pulse', workspace: 'race' });
  });
  it('builds the standings row, retires Victory Points and marks only my franchise', () => {
    expect(endpointById('M-002').status).not.toBe('spec');
    expect(endpointById('M-075').disposition).toBe('retired');
    const markup = view('M-090');
    expect(markup.match(/class="report-mine" aria-current="true"/g)?.length).toBeGreaterThanOrEqual(2);
    expect(markup).toContain('Buffalo Bills');
    expect(markup).not.toMatch(/>00\d\d</);
  });
  it('offers the week only to lineup reports and the franchise only to one-team reports', () => {
    const has = (id: string) => [view(id).includes('<label>Week'), view(id).includes('<label>Franchise')];
    expect(has('M-009')).toEqual([true, false]);
    expect(has('M-010')).toEqual([true, true]);
    expect(has('M-012')).toEqual([false, true]);
    expect(has('M-077')).toEqual([false, true]);
    expect(has('M-161')).toEqual([false, false]);
    expect(view('M-010')).toContain('<option value="3" selected="">Week 3</option>');
  });
  it('draws weekly points on one shared scale and the blend as a zero-based bar', () => {
    const r = race().board.reports.find(r => r.id === 'M-071')!;
    const markup = view('M-071');
    expect(markup.match(/class="report-bars"/g)).toHaveLength(r.rows.length);
    expect(markup.match(/<rect /g)).toHaveLength(r.rows.length * r.barLabels!.length);
    for (const row of r.rows) expect(Math.max(...row.bars!)).toBeLessThanOrEqual(r.barMax!);
    expect(view('M-008')).toContain('class="market-range"');
  });
  it('rejects a board Go would never send', () => {
    const bad = (change: (r: typeof wire.race) => void) => {
      const copy = structuredClone(wire.race);
      change(copy);
      return () => parseRace(copy);
    };
    expect(bad(r => { r.starterN = -1; })).toThrow('starterN');
    expect(bad(r => { r.week = 9; })).toThrow('week outside');
    expect(bad(r => { r.team = '0099'; })).toThrow('team outside');
    expect(bad(r => { r.standings.id = 'M-011'; })).toThrow('M-002');
    type Chart = { barMax: number; rows: { bars: number[] }[] };
    const chart = (r: typeof wire.race) => r.board.reports.find(x => x.id === 'M-071') as unknown as Chart;
    expect(bad(r => { chart(r).rows[0].bars[0] = chart(r).barMax + 1; })).toThrow('chart scale');
    expect(bad(r => { chart(r).rows[0].bars.pop(); })).toThrow('bar count');
  });
  it('asks Go for the chosen week and team and follows my franchise until one is chosen', async () => {
    const reader: RaceReader = { read: vi.fn(async () => race()), refresh: vi.fn(async () => {}) };
    const state = createRaceState(reader, () => '0001');
    await state.load(true);
    expect(reader.refresh).toHaveBeenCalledWith(false);
    expect(reader.read).toHaveBeenLastCalledWith(0, '0001', '0001', 'lineup');
    await state.choose('week', '2');
    expect(reader.read).toHaveBeenLastCalledWith(2, '0001', '0001', 'lineup');
    await state.choose('team', '0017');
    expect(reader.read).toHaveBeenLastCalledWith(3, '0017', '0001', 'lineup');
    await state.choose('refresh', '');
    expect(reader.refresh).toHaveBeenLastCalledWith(true);
    expect(() => state.choose('week', 'x')).toThrow('invalid week');
    expect(() => state.choose('team', '17')).toThrow('invalid franchise');
    const before = state.peek();
    expect(() => state.choose('aggMode', 'unknown')).toThrow('invalid roster count');
    expect(state.peek()).toBe(before);
  });
});

it('keeps schedule abbreviations intact in scrollable intrinsic columns', () => {
  const css = readFileSync('src/app/draftreports/reports.css', 'utf8');
  expect(css).toContain('.draft-report[data-endpoint="M-013"] .draft-table-scroll table {\n' +
    '  width: max-content; min-width: 100%; table-layout: auto;');
  expect(css).toContain('white-space: nowrap; overflow-wrap: normal;');
});
