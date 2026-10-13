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

it('all-position rows precede closed position disclosures and selected-position sections', () => {
    const reading: SurfaceReading = { ...unavailable(''), status: 'ok', value: {
        summary: '100 judged picks', sections: [{ id: 'L-15', title: 'All positions', note: '',
            rows: [{ key: '', label: 'All-position return', detail: '', count: 100, range: null, mark: '' }],
            breakdowns: [{ title: 'Quarterbacks', count: 25, rows: [] }],
        }, ...['L-17', 'L-18', 'L-21'].map(id => ({ id, title: id, note: '', rows: [] }))],
    } };
    const html = renderToStaticMarkup(createElement(DraftReading, { reading }));
    expect(html.indexOf('All-position return')).toBeLessThan(html.indexOf('Quarterbacks'));
    expect(html).toContain('<summary>Quarterbacks · 25 judged picks</summary>');
    expect(html).not.toMatch(/<details[^>]*\bopen[=> ]/);
    for (const id of ['L-17', 'L-18', 'L-21']) {
        expect(html).toContain(`data-endpoint="${id}" tabindex="-1"`);
        expect(html).toContain(`<summary>${id}</summary>`);
    }
});

it('parses counted position breakdowns and rejects untyped record counts', async () => {
    const { parseMarket } = await import('./provider');
    const wire = {
        ...unavailable(''), status: 'ok', cutoff: '2026-10-06', howCounted: 'Frozen picks',
        inputSHA: 'a'.repeat(64), provenance: { source: 'Lab', kind: 'archive',
            freshness: { state: 'stale', fetchedAt: '2026-10-06T00:00:00Z', note: 'Frozen' } },
        value: { summary: 'All positions', sections: [{ id: 'L-15', title: 'All positions',
            note: '', rows: [], breakdowns: [{ title: 'Quarterbacks', count: 25, rows: [] }] }] },
    };
    expect(parseMarket(wire).value?.sections[0].breakdowns?.[0].count).toBe(25);
    expect(() => parseMarket({ ...wire, value: { ...wire.value, sections: [{
        ...wire.value.sections[0], breakdowns: [{ title: 'Quarterbacks', count: '25', rows: [] }],
    }] } })).toThrow();
});
