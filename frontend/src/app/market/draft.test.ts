import { it, expect, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { createDraftState, draftFiltered, readDraft, type DraftScope } from './draftState';
import { DraftReading, DraftControls } from './Draft';
import { unavailable } from './provider';
import type { SurfaceReading } from './contract';
const scope: DraftScope = { from: 2017, to: 2026, seasons: 3, group: 'DL', segments: [], nfl: [] };
it('Draft filters validate selections and narrow Go inputs, never calculate return math', async () => {
    const reader = vi.fn(async () => ({ ...unavailable('Too few'), status: 'thin' as const }));
    const s = createDraftState(reader);
    await s.refresh();
    for (const change of [
        { field: 'draft-segment' as const, value: 'r3' }, { field: 'draft-nfl' as const, value: 'n1' },
        { field: 'draft-position' as const, value: 'DB' }, { field: 'draft-seasons' as const, value: '4' },
        { field: 'draft-from' as const, value: 'recent' },
    ]) {
        const old = s.peek().reading;
        const pending = s.change(change);
        expect(s.peek().reading).toBe(old);
        expect(s.peek().pending).toBe(true);
        await pending;
        expect(reader).toHaveBeenLastCalledWith(s.peek().scope);
    }
    expect(s.peek().scope).toEqual({ from: 2024, to: 2026, seasons: 4, group: 'DB',
      segments: ['r3'], nfl: ['n1'] });
    await s.change({ field: 'draft-reset', value: '' });
    expect(s.peek().scope).toEqual(scope);
    expect(draftFiltered({ ...scope, segments: ['r3'] }, { field: 'draft-segment',
      value: 'r3' }).segments).toEqual([]);
    for (const change of [{ field: 'draft-nfl' as const, value: 'bogus' }, {
       field: 'draft-segment' as const, value: 'bogus' }, { field: 'draft-position' as const,
      value: 'bogus' }, { field: 'draft-seasons' as const, value: '5' }])
        expect(() => draftFiltered(scope, change)).toThrow();
});
it('Draft preserves prior readings, drops stale replies and labels rejected reads failed', async () => {
    let finish!: (r: SurfaceReading) => void;
    const reader = vi.fn().mockImplementationOnce(() => new Promise(resolve => {
      finish = resolve; })).mockResolvedValueOnce(unavailable('second'));
    const s = createDraftState(reader);
    const old = s.refresh();
    await s.change({ field: 'draft-seasons', value: '2' });
    finish(unavailable('stale'));
    await old;
    expect(s.peek().reading?.absenceReason).toBe('second');
    const failed = createDraftState(async () => { throw new Error('fingerprint mismatch'); });
    await failed.refresh();
    expect(failed.peek().reading?.status).toBe('failed');
});
it('pick cards request the historical round cohort and retry failed cards', async () => {
    const reader = vi.fn(async () => unavailable('offline'));
    const s = createDraftState(reader);
    await s.pick('pick1:next');
    expect(reader).toHaveBeenLastCalledWith({ ...scope, segments: ['r1a', 'r1b'] });
    await s.pick('pick2:later');
    expect(reader).toHaveBeenLastCalledWith({ ...scope, segments: ['r2a', 'r2b'] });
    await s.pick('pick3:next');
    expect(reader).toHaveBeenLastCalledWith({ ...scope, segments: ['r3', 'r4', 'r5'] });
    await s.change({ field: 'draft-retry', value: '' });
    await s.pick('pick1:next');
    expect(reader).toHaveBeenCalledTimes(5);
});
it('browser Draft is explicitly not wired, never substitutes archive fixtures', async () => {
    const r = await readDraft(scope);
    expect(r.status).toBe('not_wired');
    expect(r.threshold).toBe(40);
    const html = renderToStaticMarkup(createElement(DraftReading, { reading: r }));
    expect(html).toContain('Not wired');
    expect(html).not.toContain('<svg');
});
it('Draft controls have explicit accessible labels and use only typed Acts', () => {
    const html = renderToStaticMarkup(createElement(DraftControls));
    expect(html).toContain('Judge on first 3 seasons');
    expect(html).toContain('Filter draft stretch 1.01');
    expect(html).not.toContain('Retry reading');
    const code = readFileSync('src/app/market/Draft.tsx', 'utf8');
    expect(code).not.toMatch(/onClick=|Math\.|reduce\(/);
    expect(code).toContain('verb="market"');
});
