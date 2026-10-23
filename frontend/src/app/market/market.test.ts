import { describe, it, expect, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import Market, { canRetry } from './Market';
import { marketState } from './state';
import { DraftReading } from './Draft';
import { ReadingView, RangeBar } from './ReadingView';
import { createMarketState, filtered } from './state';
import { parseMarket, unavailable } from './provider';
import type { MarketProvider, MarketScope, SurfaceReading } from './contract';
import { createCommands } from '../commands/registry';
import { searchCandidates } from '../commands/search';
import { endpointRoutes } from '../registry/routes';
import { endpoints } from '../registry';
const scope: MarketScope = { from: 2017, to: 2026, phases: null, assets: null, byAge: false };
function reading(id = 'L-02'): SurfaceReading {
    return {
        status: 'ok',
        absenceReason: null,
        count: 250,
        threshold: /^L-(1[5-9]|2[0-2])$/.test(id) ? 40 : 80,
        baseline: 'Median cost per delivered win',
        cutoff: '2026-10-06T01:42:57+00:00',
        method: 'ridge fit',
        inputSHA: 'a'.repeat(64),
        buildStamp: 'dev',
        howCounted: 'Frozen historical trades with checked forecast wins.',
        provenance: {
            source: 'compiled history',
            kind: 'archive',
            freshness: { state: 'stale', fetchedAt: '2026-10-06T15:57:20+00:00', note: 'Frozen archive' },
        },
        value: {
            summary: 'A historical finding supplied by Go',
            sections: [
                {
                    id,
                    title: 'Historical evidence',
                    note: 'Range supplied by Go',
                    rows: [
                        {
                            key: 'LB',
                            label: 'Linebackers',
                            count: 25,
                            detail: 'Clear in both halves',
                            mark: '',
                            range: { value: 0.4, lo: 0.2, hi: 0.8, text: '40.0%; likely 20.0% to 80.0%' },
                        },
                    ],
                },
            ],
        },
    };
}
const surfaceIDs = [
    'L-01',
    'L-02',
    'L-03',
    'L-04',
    'L-05',
    'L-09',
    'L-10',
    'L-11',
    'L-12',
    'L-13',
    'L-14',
    'L-35',
    ...Array.from({ length: 8 }, (_, i) => `L-${i + 15}`),
];
describe('Every Market analytic surface', () => {
    for (const id of surfaceIDs) {
        it(`${id} renders Go ok payload and disclosure without a grade`, () => {
            const html = renderToStaticMarkup(createElement(ReadingView, {
              reading: reading(id), title: id, inspect: true }));
            expect(html).toContain(`data-endpoint="${id}"`);
            expect(html).toContain('40.0%; likely 20.0% to 80.0%');
            expect(html).toContain('<svg');
            expect(html).toContain('How this is counted');
            expect(html).toContain('not today’s live market');
            expect(html).toContain(`250 / ${reading(id).threshold}`);
            expect(html).toContain('Inspect Linebackers');
            expect(html).not.toMatch(/\b(solid|rough|grade)\b/);
        });
        it(`${id} shows thin reason and count, not an invented figure`, () => {
            const thin = {
                ...reading(id),
                status: 'thin' as const,
                count: 7,
                absenceReason: `Fewer than ${reading(id).threshold} records`,
                value: { summary: 'Widen this scope', sections: [] },
            };
            const html = renderToStaticMarkup(createElement(ReadingView, { reading: thin, title: id }));
            expect(html).toContain('Too few to judge');
            expect(html).toContain(`Fewer than ${reading(id).threshold} records`);
            expect(html).toContain(`7 records / ${reading(id).threshold} required`);
            expect(html).not.toContain('<svg');
            expect(html).toContain('How this is counted');
        });
        it(`${id} shows not-wired with no fixture numbers`, () => {
            const html = renderToStaticMarkup(createElement(ReadingView, {
              reading: unavailable('Desktop required', 'not_wired'), title: id }));
            expect(html).toContain('Not wired');
            expect(html).not.toContain('<svg');
        });
        it(`${id} shows failed reason without displaying a stale payload`, () => {
            const failed = {
                ...reading(id),
                status: 'failed' as const,
                absenceReason: 'fingerprint mismatch',
            };
            const html = renderToStaticMarkup(createElement(ReadingView, { reading: failed, title: id }));
            expect(html).toContain('role="alert"');
            expect(html).toContain('fingerprint mismatch');
            expect(html).not.toContain('A historical finding');
            expect(html).not.toContain('<svg');
        });
    }
    it('shows not wired plainly, never falls back to fixture numbers', () => {
        const html = renderToStaticMarkup(createElement(ReadingView, {
            reading: unavailable('Desktop required', 'not_wired'),
            title: 'Market',
        }));
        expect(html).toContain('Not wired');
        expect(html).toContain('Desktop required');
        expect(html).not.toContain('<svg');
    });
    it('draws the supplied likely range and estimate with a baseline, not a tier', () => {
        const html = renderToStaticMarkup(createElement(RangeBar, {
          range: reading().value!.sections[0].rows[0].range! }));
        expect(html).toContain('class="reference"');
        expect(html).toContain('class="range"');
        expect(html).toContain('<circle');
        expect(html).toContain('aria-label="40.0%; likely 20.0% to 80.0%"');
        expect(html).not.toMatch(/\b(solid|rough|thin)\b/);
    });
});
it('validates IPC shell, finite ranges, missing disclosure and no unknown grade', () => {
    expect(parseMarket(reading())).toEqual(reading());
    for (const bad of [
        { ...reading(), count: -1 },
        { ...reading(), status: 'rough' },
        { ...reading(), inputSHA: '' },
        { ...reading(), howCounted: '' },
        { ...reading(), grade: 'solid' },
        { ...reading(), status: 'thin', absenceReason: null },
    ])
        expect(() => parseMarket(bad)).toThrow();
    const badRange = reading();
    badRange.value!.sections[0].rows[0].range!.lo = Infinity;
    expect(() => parseMarket(badRange)).toThrow('finite');
    const reversed = reading();
    reversed.value!.sections[0].rows[0].range!.lo = 10;
    expect(() => parseMarket(reversed)).toThrow('reversed');
    expect(parseMarket(unavailable('Failed'))).toEqual(unavailable('Failed'));
});
function provider(): MarketProvider {
    return {
        lists: vi.fn(async () => reading()),
        clock: vi.fn(async () => reading('L-14')),
        stretches: vi.fn(async () => reading('L-35')),
        profile: vi.fn(async () => reading('L-09')),
    };
}
it('re-queries Go on filters while retaining displayed readings', async () => {
    const p = provider();
    const s = createMarketState(p);
    await s.refresh();
    for (const change of [
        { field: 'phase' as const, value: 'winter' },
        { field: 'asset' as const, value: 'LB' },
        { field: 'age' as const, value: 'split' },
        { field: 'window' as const, value: 'season' },
    ]) {
        const old = s.peek().lists;
        const pending = s.change(change);
        expect(s.peek().lists).toBe(old);
        expect(s.peek().pending).toBe(true);
        await pending;
        expect(p.lists).toHaveBeenLastCalledWith(s.peek().scope);
        expect(p.clock).toHaveBeenLastCalledWith(s.peek().scope);
        expect(p.stretches).toHaveBeenLastCalledWith(s.peek().scope);
    }
    expect(s.peek().scope).toMatchObject({
        phases: ['early', 'midseason', 'deadline'],
        assets: ['LB'],
        byAge: true,
    });
    await s.profile('LB:prime');
    expect(p.profile).toHaveBeenLastCalledWith(s.peek().scope, 'LB:prime');
});
it('toggles selections back to everything and rejects unknown stretches/assets', () => {
    let s = filtered(scope, { field: 'phase', value: 'winter' });
    s = filtered(s, { field: 'phase', value: 'winter' });
    expect(s.phases).toBeNull();
    s = filtered(s, { field: 'asset', value: 'LB' });
    s = filtered(s, { field: 'asset', value: 'DL' });
    expect(s.assets).toEqual(['DL', 'LB']);
    expect(filtered(s, { field: 'reset', value: '' })).toEqual(scope);
    expect(() => filtered(scope, { field: 'phase', value: 'bogus' })).toThrow();
    expect(() => filtered(scope, { field: 'asset', value: 'bogus' })).toThrow();
});
it('retains profiles and ignores stale Go replies', async () => {
    const p = provider();
    const s = createMarketState(p);
    await s.refresh();
    await s.profile('LB');
    const old = s.peek().profiles.LB;
    let finish!: (r: SurfaceReading) => void;
    p.profile = vi.fn(() => new Promise<SurfaceReading>((resolve) => {
        finish = resolve;
    }));
    await s.change({ field: 'phase', value: 'early' });
    const work = s.profile('LB');
    expect(s.peek().profiles.LB).toBe(old);
    await s.change({ field: 'phase', value: 'deadline' });
    finish({ ...reading(), count: 999 });
    await work;
    expect(s.peek().profiles.LB.count).not.toBe(999);
});
it('ignores out-of-order list responses and turns rejected requests into explicit failures', async () => {
    const p = provider();
    let finish!: (r: SurfaceReading) => void;
    p.lists = vi
        .fn()
        .mockImplementationOnce(() => new Promise((resolve) => {
        finish = resolve;
    }))
        .mockResolvedValueOnce({ ...reading(), count: 123 });
    const s = createMarketState(p);
    const old = s.refresh();
    await s.change({ field: 'phase', value: 'early' });
    finish({ ...reading(), count: 999 });
    await old;
    expect(s.peek().lists?.count).toBe(123);
    p.lists = vi.fn(async () => {
        throw new Error('offline');
    });
    await s.refresh();
    expect(s.peek().lists?.status).toBe('failed');
    expect(s.peek().lists?.absenceReason).toContain('offline');
});
it('reaches all 16 registry ids by menu and endpoint command, with an Inspector asset action', () => {
    const c = createCommands(() => undefined);
    const ids = [
        ...Array.from({ length: 14 }, (_, i) => `L-${String(i + 1).padStart(2, '0')}`),
        'L-33',
        'L-35',
    ];
    const rows = endpoints.filter((r) => ids.includes(r.id));
    const routes = endpointRoutes(rows, searchCandidates(c));
    expect(rows).toHaveLength(16);
    for (const row of rows) {
        expect(row.status).not.toBe('spec');
        expect(routes.get(row.id)).toEqual(new Set(['nav', 'command']));
        c.dispatch('endpoint.open', { id: row.id });
        expect(c.read()).toMatchObject({ node: 'war', workspace: { war: 'market' } });
    }
    c.dispatch('nav.open', { node: 'war', workspace: 'market' });
    c.dispatch('market', { field: 'inspect', value: 'LB' });
    expect(c.read()).toMatchObject({ asset: 'LB', inspector: 'rest' });
});
it('keeps Market lazy and prefetched, with verb-only controls', () => {
    const load = readFileSync('src/app/market/load.ts', 'utf8');
    const shell = readFileSync('src/app/shell/TargetApp.tsx', 'utf8');
    const registry = readFileSync('src/app/commands/registry.ts', 'utf8');
    const market = readFileSync('src/app/market/Market.tsx', 'utf8');
    expect(load).toContain("import('./Market')");
    expect(shell).toContain('lazy(loadMarket)');
    expect(shell).toMatch(/<Suspense fallback=\{null\}>\s*<Market/);
    expect(registry).toContain('m.warm()');
    expect(market).not.toMatch(/onClick=|\.dispatch\(|Math\.|reduce\(/);
    expect(market).toContain('verb="market"');
});
it('Part A: retries only failures; window and stretch labels are unique', async () => {
    expect(canRetry(reading(), { ...reading(), status: 'thin', absenceReason: 'Too few' })).toBe(false);
    expect(canRetry(unavailable('Desktop required', 'not_wired'))).toBe(false);
    expect(canRetry(reading(), unavailable('Read failed'))).toBe(true);
    const old = marketState.peek().lists;
    await marketState.refresh();
    const hook = vi.spyOn(marketState, "use").mockImplementation(() => marketState.peek());
    const html = renderToStaticMarkup(createElement(Market));
    hook.mockRestore();
    expect(html).toContain('Rookie draft through kickoff');
    const labels = [...html.matchAll(/<button[^>]*>([^<]+)<\/button>/g)].map(m => m[1]);
    const time = labels.filter(s => ['Offseason', 'Rookie draft through kickoff', 'In season',
       'After the season', 'Before NFL draft', 'NFL to rookie draft', 'Rookie draft',
      'Draft to kickoff', 'Weeks 1–4', 'Midseason', 'Deadline run-in'].includes(s));
    expect(time).toHaveLength(11);
    expect(new Set(time).size).toBe(11);
    expect(html).not.toContain('Retry reading');
    expect(old).toBeUndefined();
});
it('draft pick card keeps plays-like, hit and retention history in the asset Inspector', () => {
    const r = reading('L-19');
    r.value!.sections.push({ ...r.value!.sections[0], id: 'L-20' }, { ...r.value!.sections[0],
      id: 'L-21' }, { ...r.value!.sections[0], id: 'L-22' });
    const html = renderToStaticMarkup(createElement(DraftReading, { reading: r, kind: 'pick1:next' }));
    expect(html).toContain('Draft pick card');
    for (const id of ['L-19', 'L-20', 'L-21'])
        expect(html).toContain(`data-endpoint="${id}"`);
    expect(html).not.toContain('data-endpoint="L-22"');
    expect(html).toContain('How this is counted');
});
it('all eight Draft endpoints route through Trade Floor Draft room and have menu and command doors', () => {
    const c = createCommands(() => undefined), rows = endpoints.filter(r => /^L-(1[5-9]|2[0-2])$/.test(r.id));
    const routes = endpointRoutes(rows, searchCandidates(c));
    expect(rows).toHaveLength(8);
    for (const row of rows) {
        expect(row.status).not.toBe('spec');
        expect(routes.get(row.id)).toEqual(new Set(['nav', 'command']));
        c.dispatch('endpoint.open', { id: row.id });
        expect(c.read()).toMatchObject({ node: 'trade', workspace: { trade: 'draft-room' },
          endpointIds: [row.id] });
    }
    c.dispatch('market', { field: 'inspect', value: 'pick1:next' });
    expect(c.read()).toMatchObject({ asset: 'pick1:next', inspector: 'rest' });
    const source = readFileSync('src/app/market/Market.tsx', 'utf8');
    expect(source).toContain("import('./Draft')");
    expect(source).toContain("args={{ id: 'L-15' }}");
    expect(source).toContain('<Draft kind={kind}');
});
