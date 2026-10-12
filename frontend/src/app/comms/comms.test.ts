import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { CommsView } from './Comms';
import { parseComms } from './contract';
import { createCommsState } from './state';
import { sortedRows, reportSort } from '../reports/sortState';
import { endpointById } from '../registry';
import wire from './wire.testdata.json';

const comms = () => parseComms(wire);
const markup = (panel: boolean) => renderToStaticMarkup(
  createElement(CommsView, { comms: comms(), panel, limit: 50 }),
);
describe('League communications', () => {
  it('reads Go wire sentences in both views without a second wording', () => {
    const view = comms();
    expect(view.events[0].sentence).toContain('Minnesota Vikings placed Edgerrin Cooper (LB, GBP) on IR');
    expect(view.events[0].date).toBe('Oct 7, 2026, 1:55 AM');
    const home = markup(false);
    const panel = markup(true);
    expect(home).toContain('data-endpoint="M-032"');
    expect(home).toContain('Read the league feed');
    expect(panel).toContain('No posts on MFL&#x27;s message board or polls this season.');
    expect(panel.indexOf('data-endpoint="C-05"')).toBeLessThan(panel.indexOf('data-endpoint="M-120"'));
    expect(panel).toContain('Open the league board');
    expect(panel).toContain('data-endpoint="M-123"');
    expect(panel).not.toContain('data-endpoint="M-032"');
    for (const event of view.events) {
      const escaped = event.sentence.replaceAll('&', '&amp;').replaceAll("'", '&#x27;');
      expect(home.split(escaped)).toHaveLength(3);
      expect(panel.split(escaped)).toHaveLength(2);
    }
  });
  it('limits the panel to the latest 50 and exposes Show more, while Digests holds every row', () => {
    const view = comms();
    view.events = Array.from({ length: 60 }, (_, i) => ({ sentence: `Fixture move ${i}.`, date: 'Oct 11' }));
    const render = (panel: boolean, limit: number) => renderToStaticMarkup(
      createElement(CommsView, { comms: view, panel, limit }),
    );
    expect(render(true, 50)).toContain('Fixture move 49.');
    expect(render(true, 50)).not.toContain('Fixture move 50.');
    expect(render(true, 50)).toContain('Show more');
    expect(render(true, 100)).toContain('Fixture move 59.');
    expect(render(true, 100)).not.toContain('Show more');
    expect(render(false, 50)).toContain('Fixture move 59.');
  });
  it('sorts the shared report by numeric dates in either direction', () => {
    reportSort.set({ id: 'M-032', column: 0 });
    const rows = comms().transactions.rows;
    expect(sortedRows(rows, { column: 0, desc: false })[0]).toBe(rows[2]);
    expect(sortedRows(rows, { column: 0, desc: true })[0]).toBe(rows[0]);
    expect(() => reportSort.set({ id: 'M-032', column: 2 })).toThrow('invalid column');
  });
  it('labels missing boards and does not turn a failed feed into an empty surface', () => {
    const view = comms();
    view.boardNote = 'MFL message board not available.';
    view.boardPosts[0].status = 'failed';
    view.boardPosts[0].note = 'Authentication failed; refresh from MFL.';
    view.feed.status = 'failed'; view.feed.note = 'League feed not available; refresh from MFL.';
    const html = renderToStaticMarkup(createElement(CommsView, { comms: view, panel: true, limit: 50 }));
    expect(html).toContain('Authentication failed');
    expect(html).toContain('League feed not available');
    expect(html).not.toContain('No posts');
  });
  it('keeps the surface lazy and the registry dispositions exact', () => {
    const source = readFileSync('src/app/shell/TargetApp.tsx', 'utf8');
    expect(source).toContain("const Comms = lazy(() => import('../comms/Comms'))");
    expect(source).toContain("workspace.slug === 'digests'");
    expect(source.indexOf('<Comms panel />'))
      .toBeLessThan(source.indexOf('<EndpointIndex placement="Comms"'));
    for (const id of ['C-05', 'M-032', 'M-120', 'M-123'] as const)
      expect(endpointById(id).status).toBe('built');
    for (const id of ['M-121', 'M-122', 'M-124'] as const)
      expect(endpointById(id).merged_into).toEqual(['M-120']);
    for (const id of ['M-125', 'M-126', 'M-127'] as const) expect(endpointById(id).ring).toBe(4);
  });
  it('checks IPC wire fields, reports and filters before presentation', () => {
    expect(() => parseComms({ ...wire, team: '9999' })).toThrow('filter outside');
    expect(() => parseComms({ ...wire, kind: 'vote' })).toThrow('filter outside');
    expect(() => parseComms({ ...wire, invented: true })).toThrow();
    expect(() => parseComms({ ...wire, feed: { ...wire.feed, id: 'M-001' } })).toThrow('unexpected reports');
  });
  it('dispatches filters, pagination, refresh and browser hand-offs through the reader', async () => {
    const read = vi.fn(async (team: string, kind: string) => ({ ...comms(), team, kind }));
    const refresh = vi.fn(async () => {});
    const open = vi.fn(async () => {});
    const state = createCommsState({ read, refresh, open });
    await state.load();
    expect(read).toHaveBeenLastCalledWith('', '');
    await state.choose('team', '0001');
    await state.choose('kind', 'ir');
    expect(read).toHaveBeenLastCalledWith('0001', 'ir');
    await state.choose('more', '');
    expect(state.peek().limit).toBe(100);
    await state.choose('refresh', '');
    expect(refresh).toHaveBeenCalledWith(true);
    await state.choose('open', 'league-board');
    expect(open).toHaveBeenCalledWith('league-board');
    expect(state.peek().pending).toBe(false);
    expect(state.peek().error).toBe('');
  });
  it('keeps the last good copy and labels refresh failures', async () => {
    let fail = false;
    const state = createCommsState({
      read: async () => comms(),
      refresh: async () => { if (fail) throw new Error('fixture offline'); },
      open: async () => {},
    });
    await state.load(); fail = true;
    await state.load(true);
    expect(state.peek().comms).toBeDefined();
    expect(state.peek().error).toContain('fixture offline');
    expect(state.peek().pending).toBe(false);
  });
});
