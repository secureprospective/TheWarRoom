import { afterEach, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { TargetOffers } from '../../../wailsjs/go/main/App';
import { commands, createCommands } from '../commands/registry';
import { FixtureProvider } from '../data/provider';
import { clockDom, TestElement } from '../clock/testDom';
import { initialScope, parseOffers, readOffers, type OfferReading } from './provider';
import OffersHistory, { OffersScreen } from './Offers';
import { localDate } from '../clock/labels';
import { nodes } from '../shell/nodes';
import wire from './wire.testdata.json';
vi.mock('../../../wailsjs/go/main/App', () => ({ TargetOffers: vi.fn() }));
const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map(key => ({ key,
  descriptor: Object.getOwnPropertyDescriptor(globalThis, key) }));
  
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
    await act(async () => { for (const root of roots.splice(0))
        root.unmount(); });
    vi.restoreAllMocks();
    for (const { key, descriptor } of globals) {
        if (descriptor)
            Object.defineProperty(globalThis, key, descriptor);
        else
            Reflect.deleteProperty(globalThis, key);
    }
});
const screen = (reading: OfferReading) => renderToStaticMarkup(createElement(OffersScreen,
  { reading, scope: initialScope }));
it('keeps the L-32 count/owner/cutoff header and HowCounted in all four reading states', () => {
    const r = parseOffers(wire);
    for (const status of ['ok', 'thin', 'failed', 'not_wired'] as const) {
        const text = screen({ ...r, status, absenceReason: status === 'ok' ? null : 'seeded reason' });
        for (const phrase of ['data-endpoint="L-32"', 'Arizona Cardinals',
          '0025', '470', localDate(wire.cutoff), 'How this is counted', "GM&#x27;s own offers"])
            expect(text).toContain(phrase);
        if (status === 'failed' || status === 'not_wired')
            expect(text).not.toContain('Ending not on record ');
    }
    const text = screen(r);
    expect(text.match(/offer-bar-track/g)).toHaveLength(8);
    expect(text).toContain('Ending not on record ');
    expect(text).not.toMatch(/\bsolid\b|\brough\b|break-all|Loading/);
});
it('uses Go labels and rejects malformed wires', () => {
    const r = parseOffers(wire);
    r.value!.words.heading = 'Heading from Go';
    r.value!.directions[1].label = 'Direction from Go';
    r.value!.stretches[0].countText = 'Count from Go';
    expect(screen(r)).toContain('Heading from Go');
    expect(screen(r)).toContain('Direction from Go');
    expect(screen(r)).toContain('Count from Go');
    expect(() => parseOffers({ ...wire, extra: true })).toThrow();
    expect(() => parseOffers({ ...wire, status: 'thin', absenceReason: null })).toThrow();
    const s = wire.value.stretches[0];
    expect(() => parseOffers({ ...wire, value: { ...wire.value,
      stretches: [{ ...s, outcomes: [{ ...s.outcomes[0], width: 101 }] }] } })).toThrow(/width/);
      
});
it('returns an honest failed header when IPC fails', async () => {
    vi.mocked(TargetOffers).mockRejectedValueOnce(new Error('IPC unavailable'));
    const r = await readOffers(initialScope);
    expect(r.status).toBe('failed');
    expect(screen(r)).toContain('whose offers, the count and the cutoff cannot be verified');
    expect(screen(r)).toContain('GM&#x27;s own offers');
});
function descendants(element: TestElement): TestElement[] {
    return element.children.flatMap(child => child instanceof TestElement ? [child,
      ...descendants(child)] : []);
}
async function flush() { for (let i = 0; i < 5; i++)
    await new Promise(resolve => setTimeout(resolve, 0)); }
it('the L-31 Sent button re-queries Go without dropping the disclosure or flashing figures', async () => {
    vi.mocked(TargetOffers).mockResolvedValue(wire as never);
    const dom = clockDom([]);
    vi.stubGlobal('document', dom.document);
    vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {
        } });
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    const root = createRoot(dom.element);
    roots.push(root);
    const { Suspense } = await import('react');
    await act(async () => { root.render(createElement(Suspense,
      { fallback: null }, createElement(OffersHistory))); await flush();
      });
    const element = dom.element as unknown as TestElement;
    const button = descendants(element).find(e => e.tagName === 'BUTTON' && e.textContent === 'Sent')!;
    const key = Object.keys(button).find(key => key.startsWith('__reactProps$'))!;
    const props = (button as unknown as Record<string, {
        onClick: () => void;
    }>)[key];
    await act(async () => { props.onClick(); await flush(); });
    expect(vi.mocked(TargetOffers).mock.calls.some(([s]) => s.direction === 'sent')).toBe(true);
    expect(element.textContent).toContain('470 offers visible to Arizona Cardinals');
    expect(element.textContent).not.toContain('Reading offers');
});
it('both routes reach the same lazy Offers surface, independently of the live franchise', () => {
    const c = createCommands(() => undefined, new FixtureProvider());
    for (const id of ['L-30', 'L-31', 'L-32'] as const) {
        c.dispatch('endpoint.open', { id });
        expect(c.read().node).toBe('trade');
        expect(c.read().workspace.trade).toBe('offers-history');
    }
    c.dispatch('nav.open', { node: 'trade', workspace: 'trade-desk' });
    expect(c.read().workspace.trade).toBe('trade-desk');
    const desk = readFileSync('src/app/shell/TradeDesk.tsx', 'utf8');
    expect(desk).not.toContain('OffersHistory');
    const mount = readFileSync('src/app/shell/TargetApp.tsx', 'utf8');
    expect(mount).toContain("['market', 'counterparties', 'offers-history', " +
      "'valuations-and-pool', 'draft-room']");
    const market = readFileSync('src/app/market/Market.tsx', 'utf8');
    expect(market).toContain("lazy(() => import('../offers/Offers'))");
    expect(market).toContain("s.workspace[s.node] === 'offers-history'");
    expect(nodes.trade.workspaces.map(w => w.label)).toEqual([
      'Trade desk', 'Offers history', 'Counterparties', 'Draft room']);
    expect(desk).not.toContain('if (!franchiseId) return');
    const css = readFileSync('src/app/offers/offers.css', 'utf8');
    expect(css).toContain('minmax(min(100%, 260px), 1fr)');
    expect(css).not.toMatch(/anywhere|break-all/);
    expect(commands).toBeDefined();
});

it('keeps previous figures until the latest filter response and rejects invalid controls', async () => {
  const { createOfferState } = await import('./state');
  const r = parseOffers(wire);
  let resolveSent: (r: OfferReading) => void = () => {};
  const read = vi.fn().mockResolvedValueOnce(r)
    .mockImplementationOnce(() => new Promise<OfferReading>(resolve => { resolveSent = resolve; }))
    .mockResolvedValueOnce({ ...r, count: 12 });
  const state = createOfferState(read);
  let opening: unknown;
  try { state.first(); } catch (promise) { opening = promise; }
  await opening;
  const sent = state.change({ field: 'direction', value: 'sent' });
  expect(state.peek().pending).toBe(true);
  expect(state.peek().reading?.count).toBe(470);
  expect(state.peek().reading?.value?.disclosure).toContain('Arizona Cardinals');
  await state.change({ field: 'direction', value: 'received' });
  resolveSent({ ...r, count: 99 });
  await sent;
  expect(state.peek().reading?.count).toBe(12);
  expect(state.peek().scope.direction).toBe('received');
  expect(() => state.change({ field: 'direction', value: 'bogus' })).toThrow();
  expect(() => state.change({ field: 'team', value: 'bogus' })).toThrow();
  expect(() => state.change({ field: 'window', value: 'bogus' })).toThrow();
  expect(() => state.change({ field: 'range', value: 'bogus' })).toThrow();
});

it('shares readable shell selects without an Offers override', () => {
  const css = readFileSync('src/app/shell/shell.css', 'utf8');
  const select = css.match(/\.twr-app select \{([^}]+)\}/)![1];
  for (const rule of ['appearance: none', 'color-scheme: dark', 'color: var(--surface-tx1)',
    'background-color: var(--surface-2)', 'background-image:', 'background-position:'])
    expect(select).toContain(rule);
  expect(css).toContain('.twr-app select:focus-visible { outline: 2px solid var(--signal-blue)');
  expect(readFileSync('src/app/offers/offers.css', 'utf8')).not.toMatch(/select/);
});
it('uses five Go-width segments per bar, one legend, and all textual counts', () => {
  const html = screen(parseOffers(wire));
  expect(html.match(/class="offers-legend"/g)).toHaveLength(1);
  expect(html.match(/class="offer-counts"/g)).toHaveLength(8);
  expect(html.match(/class="offer-count"/g)).toHaveLength(40);
  for (const stretch of wire.value.stretches) for (const outcome of stretch.outcomes) {
    expect(html).toContain(`data-outcome="${outcome.id}" style="width:${outcome.width}%"`);
    expect(html).toContain(outcome.text);
  }
});
it('localizes cutoff and other visible timestamps using the existing app helper', () => {
  const r = parseOffers(wire);
  r.buildStamp = `dev ${wire.cutoff}`;
  r.howCounted = `History through ${wire.cutoff}`;
  const html = screen(r);
  expect(html).toContain(`through ${localDate(wire.cutoff)}`);
  expect(html).toContain(`dev ${localDate(wire.cutoff)}`);
  expect(html).not.toContain(wire.cutoff);
});
it('offers Try again only for failed readings', () => {
  const r = parseOffers(wire);
  for (const status of ['ok', 'thin', 'failed', 'not_wired'] as const) {
    const html = screen({ ...r, status, absenceReason: status === 'ok' ? null : 'test reason' });
    expect(html.includes('Try again')).toBe(status === 'failed');
    expect(html).not.toContain('Read again');
  }
});
