import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { endpointById, placementRoute } from '../registry';
import { parseNow } from './contract';
import { NowView } from './Now';
import { createNowState, type NowReader } from './state';
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
  it('asks Go for the chosen week, refreshing that week, and follows my franchise', async () => {
    const reader: NowReader = { read: vi.fn(async () => now()), refresh: vi.fn(async () => {}) };
    const state = createNowState(reader, () => '0001');
    await state.load(true);
    expect(reader.refresh).toHaveBeenCalledWith(0, false);
    expect(reader.read).toHaveBeenLastCalledWith(0, '0001');
    await state.choose('week', '3');
    expect(reader.refresh).toHaveBeenLastCalledWith(3, false);
    expect(reader.read).toHaveBeenLastCalledWith(3, '0001');
    await state.choose('refresh', '');
    expect(reader.refresh).toHaveBeenLastCalledWith(5, true);
    expect(() => state.choose('week', 'x')).toThrow('invalid week');
  });
});
