import { ReleaseCard } from './ReleasePlans';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { clockDom } from '../clock/testDom';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { confirmRelease, parseReleases, releaseAction, type ReleasePlan } from './releases';
import { runContractsAction } from './state';
import { WhatIfPanel } from './WhatIfPanel';
import * as providers from '../data/provider';

const at = '2026-10-11T01:00:00Z';
function plan(): ReleasePlan {
  return {
    id: 'release-1', player: 'Example', label: 'Cut Example', charge: '$7,000,000',
    confirmation: 'This releases Example; $7,000,000 dead cap this season. It cannot be undone.',
    post: 'Go-authored request. $7,000,000', channel: 'MFL drop + ProBoards post',
    canHandOff: true, canRebase: false, cut: true,
    rebaseLabel: "Reprice from today's MFL rosters", withdrawLabel: "Remove plan",
    receipt: { state: 'ready', spec: { intent: 'roster.drop' }, audit: [
      { at, from: 'draft', to: 'ready', note: 'Roster checked; $7,000,000 planned' },
    ] },
  };
}
function wire(p = plan()) {
  return {
    status: 'ok', absenceReason: null, value: [{ ...p, receipt: {
      correlationId: p.id, spec: { intent: p.receipt.spec.intent, franchiseId: '0001' },
      state: p.receipt.state, audit: p.receipt.audit,
    } }], provenance: { source: 'plans', kind: 'live', freshness: {
      state: 'live', fetchedAt: at, note: 'Local plans, no mirror writes',
    } }, count: 1, threshold: 0, baseline: at, cutoff: '', method: 'Go', inputSHA: '',
    buildStamp: 'dev', howCounted: 'Engine cents from pinned what-if',
  };
}
const globals = ['window', 'navigator', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map(key => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => { for (const root of roots.splice(0)) root.unmount(); });
  vi.restoreAllMocks(); confirmRelease('');
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
describe('Release plans', () => {
  it('Make this a plan is an Act on the move, not an inferred amount or roster mutation', () => {
    const reading = { ...wire(), value: {
      stale: false, banner: '', empty: '', notice: '', choices: [], figures: [],
      moves: [{ id: 'move-1', player: '0042', tag: 'Cut in what-if', status: 'Modelled',
        label: 'Cut Example', charge: { cents: 700000000, text: '$7,000,000' }, explanation: '35%' }],
    } };
    const render = () => renderToStaticMarkup(createElement(WhatIfPanel, {
      mine: '0001', reading: reading as never,
    }));
    expect(render()).toContain('Make this a plan');
    expect(render()).toContain('Make this a plan: Cut Example');
    expect(render()).toMatch(/class="act act-command"[^>]*aria-label="Make this a plan: Cut Example"/);
    expect(render()).toContain('class="whatif-plan-line"');
    Object.assign(reading.value.moves[0], { plan: 'Plan: Ready' });
    expect(render()).toContain('Plan: Ready');
    expect(render()).toContain('class="whatif-plan-line"');
    expect(render()).not.toContain('Make this a plan');
  });
  it('requires one named, priced confirmation; the shared rail and history render Go notes', async () => {
    const p = plan();
    const dom = clockDom([]);
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    vi.stubGlobal('document', dom.document);
    vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
    const root = createRoot(dom.element); roots.push(root);
    const render = async () => {
      await act(async () => { root.render(createElement(ReleaseCard, { plan: p, mine: '0001' })); });
      return dom.root.textContent;
    };
    expect(await render()).toContain('Open MFL drop page');
    const buttons = () => dom.root.querySelectorAll('[type]');
    expect(buttons()[0].getAttribute('class')).toBe('act act-command');
    expect(buttons()[0].getAttribute('type')).toBe('button');
    expect(await render()).not.toContain('Confirm and open MFL');
    await act(async () => { confirmRelease(`${p.id}:${p.charge}:${at}`); });
    expect(await render()).toContain(p.confirmation);
    expect(await render()).toContain('Confirm and open MFL');
    expect(buttons()[0].getAttribute('class')).toBe('act act-command');
    expect(buttons()[1].getAttribute('class')).toBe('act act-command');
    expect(buttons()[1].textContent).toBe('Cancel');
    expect(dom.root.querySelector('[data-primary]')).not.toBeNull();
    expect(dom.root.querySelector('[aria-label]')).not.toBeNull();
    expect(await render()).toContain('History');
    expect(await render()).toContain('Roster checked; $7,000,000 planned');
    p.charge = '$14,000,000';
    expect(await render()).not.toContain('Confirm and open MFL');
    p.receipt.state = 'stale'; p.canHandOff = false; p.canRebase = true;
    expect(await render()).toContain("Reprice from today's MFL rosters");
    expect(await render()).not.toContain('Open MFL drop page');
    p.cut = false; p.canHandOff = true; p.canRebase = false;
    expect(await render()).toContain('Copy request and hand off');
  });
  it('renders Go withdrawal labels and Blocked reasons without Reprice', async () => {
    const p = plan(); p.canHandOff = false; p.receipt.state = 'blocked';
    p.receipt.audit[0].note = "Offseason only: can't be done until the offseason.";
    const render = () => renderToStaticMarkup(createElement(ReleaseCard, { plan: p, mine: '0001' }));
    expect(render()).toContain('Remove plan');
    expect(render()).toContain('act act-command');
    expect(render()).toContain("Offseason only: can&#x27;t be done until the offseason.");
    expect(render()).not.toContain('Reprice');
    for (const state of ['handed_off', 'not_yet_done'] as const) {
      p.receipt.state = state; p.withdrawLabel = "I didn't drop him";
      expect(render()).toContain('I didn&#x27;t drop him');
    }
    vi.spyOn(providers, 'selectProvider').mockReturnValue({ kind: 'live' } as never);
    const call = vi.fn().mockResolvedValue(wire(p));
    vi.stubGlobal('window', { go: { main: { App: { TargetReleasePlans: call } } } });
    await runContractsAction({ field: 'release', value: { mine: '0001', action: 'withdraw', id: p.id } });
    expect(call).toHaveBeenCalledWith({
      mine: '0001', action: 'withdraw', id: p.id, move: '', confirmed: false,
    });
    p.receipt.state = 'withdrawn'; p.withdrawLabel = '';
    expect(render()).not.toContain('Remove plan');
    expect(render()).not.toContain('I didn');
  });
  it('validates the Go wire and sends the move id without recomputing cents', async () => {
    expect(parseReleases(wire()).value?.[0].charge).toBe('$7,000,000');
    const bad = wire(); bad.value[0].canHandOff = 'yes' as never;
    expect(() => parseReleases(bad)).toThrow('expected boolean');
    vi.spyOn(providers, 'selectProvider').mockReturnValue({ kind: 'live' } as never);
    const call = vi.fn().mockResolvedValue(wire());
    vi.stubGlobal('window', { go: { main: { App: { TargetReleasePlans: call } } } });
    await runContractsAction({ field: 'release', value: { mine: '0001', action: 'draft', move: 'move-1' } });
    expect(call).toHaveBeenCalledWith({
      mine: '0001', action: 'draft', move: 'move-1', id: '', confirmed: false,
    });
    await runContractsAction({ field: 'release-confirm', value: 'release-1' });
    await releaseAction({ mine: '0001', action: 'handoff', id: 'release-1', confirmed: true });
    expect(call.mock.calls.at(-1)?.[0].confirmed).toBe(true);
  });
  it('copies the buyout text before hand-off and never opens an invented commissioner URL', async () => {
    const p = plan(); p.cut = false; p.label = 'Buy out Example'; p.receipt.spec.intent = 'roster.buyout';
    vi.spyOn(providers, 'selectProvider').mockReturnValue({ kind: 'live' } as never);
    const call = vi.fn().mockResolvedValue(wire(p));
    vi.stubGlobal('window', { go: { main: { App: { TargetReleasePlans: call } } } });
    const writeText = vi.fn().mockResolvedValue(true);
    Object.assign(window, { runtime: { ClipboardSetText: writeText } });
    await runContractsAction({ field: 'release', value: { mine: '0001', action: 'handoff', id: p.id } });
    expect(writeText).toHaveBeenCalledWith(p.post);
    expect(call.mock.calls.map(c => c[0].action)).toEqual(['read', 'handoff']);
    writeText.mockResolvedValueOnce(false);
    await expect(runContractsAction({ field: 'release', value: {
      mine: '0001', action: 'handoff', id: p.id,
    } })).rejects.toThrow('Clipboard unavailable');
    expect(call.mock.calls.at(-1)?.[0].action).toBe('read');
  });
  it('keeps release bindings and screens lazy, with ordinary button type and one-line headers', () => {
    const source = readFileSync('src/app/contracts/releases.ts', 'utf8');
    expect(source).toContain('import type * as Bindings');
    expect(source).toContain('go.TargetReleasePlans');
    const moves = readFileSync('src/app/shell/MyMoves.tsx', 'utf8');
    expect(moves).toContain("lazy(() => import('../contracts/ReleasePlans'))");
    const css = readFileSync('src/app/contracts/contracts.css', 'utf8');
    expect(css).toContain('th:nth-child(3) { width: 25%; white-space: nowrap; }');
    expect(css).toContain('.contract-whatif-acts .act { font-family: inherit; }');
    expect(css).toContain('.whatif-moves .whatif-plan-line' +
      ' { height: 30px; margin: 4px 0; align-items: center; }');
    expect(css).toContain('.whatif-plan-line > p { margin: 0; line-height: 18px; }');
    expect(css).toContain('.release-confirm { display: contents; }');
    expect(css).toContain('.release-confirm[data-primary="handoff"] .act-command:first-of-type' +
      ' { background: var(--surface-tx1); color: var(--surface-0); }');
  });
});

it('cut and buyout cards reuse bid frames, copy blocks and the same action row', () => {
  for (const cut of [true, false]) {
    const html = renderToStaticMarkup(createElement(ReleaseCard, { plan: { ...plan(), cut }, mine: '0001' }));
    expect(html).toContain('class="bid-plan"');
    expect(html).toContain('class="bid-post"');
    const actions = html.slice(html.indexOf('class="bid-actions"'), html.indexOf('<details'));
    expect(actions).toContain('Remove plan');
    expect(actions).toContain(cut ? 'Open MFL drop page' : 'Copy request and hand off');
  }
});

it('replacement hand-off and make-plan buttons have accessible labels but no native title', () => {
  const html = renderToStaticMarkup(createElement(ReleaseCard, { plan: plan(), mine: '0001' }));
  expect(html).toContain('aria-label="Open MFL drop page for Example"');
  expect(html).not.toContain('title=');
});
