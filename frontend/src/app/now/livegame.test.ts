import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { Field, fieldPositions, gameTime, LiveGameView, PlaySpine, orderedSpine } from './LiveGame';
import { parseLive } from './liveContract';
import { createLiveState, startLivePolling } from './liveState';
import wire from './live.testdata.json';
import pregameWire from './live-pregame.testdata.json';

const view = () => parseLive(wire);
const render = (v: ReturnType<typeof view>) => renderToStaticMarkup(createElement(LiveGameView, { view: v }));
describe('Personal live game view', () => {
  it('places scrimmage and first-down lines in both possession directions', () => {
    const state = { possession: '22', down: 1, distance: 10, yardLine: 35, toEndzone: 65 };
    expect(fieldPositions(state, true)).toEqual({ line: 35, first: 45, direction: 1 });
    expect(fieldPositions({ ...state, possession: '8', yardLine: 65 }, false))
      .toEqual({ line: 65, first: 55, direction: -1 });
    expect(fieldPositions({ ...state, yardLine: 98, toEndzone: 2 }, true).first).toBe(100);
    expect(fieldPositions({ ...state, yardLine: 2, toEndzone: 2 }, false).first).toBe(0);
    const game = { ...view().followed!, state };
    const html = renderToStaticMarkup(createElement(Field, { game }));
    expect(html).toContain('data-line="scrimmage" x1="205" x2="205"');
    expect(html).toContain('data-line="first-down" x1="255" x2="255"');
    const away = { ...game, state: { ...state, possession: '8', yardLine: 65 } };
    expect(renderToStaticMarkup(createElement(Field, { game: away })))
      .toContain('data-line="scrimmage" x1="355" x2="355"');
    expect(renderToStaticMarkup(createElement(Field, { game: { ...game, state: null } })))
      .toContain('Field position not supplied');
  });
  it('shows local kickoff without a pregame box score; renders Final truthfully', () => {
    const v = view(); const g = v.followed!;
    expect(gameTime(g)).toBe('Final');
    expect(render(v)).toContain('Final');
    const pre = parseLive(pregameWire);
    const html = render(pre);
    expect(gameTime(pre.followed!)).toBe(`Kickoff ${new Date(pre.followed!.kickoff).toLocaleString()}`);
    expect(html).toContain('Not started.');
    expect(html).not.toContain('data-line="scrimmage"');
    expect(html).toContain('personal use only');
    expect(html).toContain('Feeds are not being fetched');
  });
  it('orders the spine newest first and marks mine/opponent without name inference', () => {
    const v = view(); const newer = structuredClone(v.spine[1]);
    newer.play.id = 'later'; newer.play.sequence++;
    const entries = [v.spine[0], newer, v.spine[1]];
    expect(orderedSpine(entries)[0].play.id).toBe('later');
    const html = renderToStaticMarkup(createElement(PlaySpine, { entries, unmatched: 1 }));
    expect(html.indexOf('Teagan Quitoriano')).toBeLessThan(html.indexOf('Jacoby Brissett'));
    expect(html).toContain('data-side="mine"'); expect(html).toContain('data-side="theirs"');
    expect(html).toContain('Mine'); expect(html).toContain('Opponent');
    expect(html).toContain('1 participant appearances unmatched');
    expect(html).toContain('photo unavailable'); expect(html).not.toContain('<img');
    expect(html).toContain('Q1 · 5:06');
    expect(html).toContain('TOUCHDOWN');
  });
  it('polls only while mounted and visible, and removes its timer and visibility listener', async () => {
    vi.useFakeTimers();
    const listeners = new Map<string, () => Promise<void>>();
    const doc = { hidden: false,
      addEventListener: (key: string, fn: () => Promise<void>) => listeners.set(key, fn),
      removeEventListener: (key: string) => listeners.delete(key) };
    const oldDocument = globalThis.document;
    vi.stubGlobal('document', doc);
    try {
      const load = vi.fn(async () => {}); const update = vi.fn(async () => {});
      const stop = startLivePolling('0001', load, update);
      vi.advanceTimersByTime(0); await Promise.resolve();
      expect(load).toHaveBeenCalledTimes(1);
      vi.advanceTimersByTime(15000); await Promise.resolve();
      expect(load).toHaveBeenCalledTimes(2);
      doc.hidden = true; vi.advanceTimersByTime(30000); await Promise.resolve();
      expect(load).toHaveBeenCalledTimes(2);
      doc.hidden = false; await listeners.get('visibilitychange')!();
      expect(load).toHaveBeenCalledTimes(3);
      stop(); expect(listeners.size).toBe(0);
      vi.advanceTimersByTime(30000); await Promise.resolve();
      expect(load).toHaveBeenCalledTimes(3);
      expect(update).toHaveBeenCalledTimes(3);
    } finally { vi.useRealTimers(); vi.stubGlobal('document', oldDocument); }
  });
  it('rejects malformed IPC fields and uncached photo addresses', () => {
    const broken = structuredClone(wire); broken.spine[0].player.photo.src = 'https://external.invalid/photo';
    expect(() => parseLive(broken)).toThrow('cached photo');
    const bad = structuredClone(wire); bad.games[0].home.score = -1;
    expect(() => parseLive(bad)).toThrow('integer');
  });
  it('keeps the view on read failure and dispatches follow selection as a typed id', async () => {
    const read = vi.fn(async () => view()); const state = createLiveState(read);
    state.follow('DET-ARI'); await state.load(true, '0001');
    expect(read).toHaveBeenLastCalledWith(true, 'DET-ARI', '0001');
    read.mockRejectedValueOnce(new Error('all sources down')); await state.load(false, '0001');
    expect(state.peek().view?.source).toContain('ESPN');
    expect(state.peek().error).toContain('all sources down');
    expect(() => state.follow('../evil')).toThrow('invalid id');
    const html = render(view()); expect(html).toContain('aria-label="Follow DET at ARI"');
    expect(html.indexOf('aria-label="Followed game"')).toBeLessThan(html.indexOf('class="live-game"'));
  });
});
