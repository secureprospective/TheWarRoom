import { describe, expect, it, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { ContractActCard, ContractActMenu, ContractMovePreview, ContractMoveFeedback,
  contractActOptions } from './ContractActs';
import { menuPlacement, placeRowMenu } from './RowMenu';
import { keepContractMove, keepLabel, openContractMove, revealContractPlan, useContractMove,
  type ContractActPlan, type ContractActRequest } from './contractActs';
import { commands } from '../commands/registry';
import type { Reading } from './contract';
import { ReleaseCard } from './ReleasePlans';
import type { ReleasePlan } from './releases';

const at = '2026-10-11T01:00:00Z';
// Like a DOMRect: every field is a prototype getter, so a spread copies nothing.
function rect(left: number, top: number, width: number, height: number) {
  return Object.create({ get left() { return left; }, get right() { return left + width; },
    get top() { return top; }, get bottom() { return top + height; } }) as DOMRect;
}
function plan(): ContractActPlan {
  return { id: 'contract-1', intent: 'contract.extend',
    teamFigures: [{ label: 'Used', before: '$49M', after: '$48M', change: '−$1M' }],
    player: 'Example', subject: 'Example · QB · $10,000,000 a year through 2028',
    label: 'Extension · Example', price: '$15,000,000',
    post: 'Extension Example\n2029: $15,000,000\nPlease process this request on ProBoards.',
      how: '§10: Go exact cents then snap.',
    reasons: ['Not checked: prior extension'], figures: [{ season: 2029, before: '$0',
      after: '$15,000,000', change: '$15,000,000' }],
    receipt: { state: 'blocked', spec: { intent: 'contract.extend' }, audit: [{ at, from: 'draft',
      to: 'blocked', note: 'Rule unclear: window' }] },
    canHandOff: false, canPost: false, canWithdraw: true, canRebase: false };
}
describe('Contract Acts', () => {
  it('lists the five Acts behind a compact lazy menu, not added table rows', () => {
    expect(contractActOptions.map(v => v[1])).toEqual(['Tender him (RFA)', 'Re-sign him',
      'Franchise tag', 'Extend him', 'Restructure']);
    expect(contractActOptions.map(v => v[0])).toEqual(['rfa.tender', 'fa.resign', 'contract.tag',
      'contract.extend', 'contract.restructure']);
    const html = renderToStaticMarkup(createElement(ContractActMenu, { mine: '0001', player: '0042' }));
    expect(html).toContain('Contract Acts…</button>');
    expect(html).not.toContain('<table');
    expect(html).not.toContain('<select');
    const source = readFileSync(new URL('./WhatIfPanel.tsx', import.meta.url), 'utf8');
    expect(source).toContain('<ContractActMenu mine={mine} player={player} />');
  });
  it('places below, flips above and clamps to the work column, never the Inspector', () => {
    const work = { left: 240, right: 890, top: 100, bottom: 780 };
    expect(menuPlacement({ left: 780, right: 880, top: 300, bottom: 320 }, work, 200))
      .toEqual({ left: 500, top: 320, width: 380, maxHeight: 460 });
    expect(menuPlacement({ left: 780, right: 880, top: 700, bottom: 720 }, work, 200))
      .toEqual({ left: 500, top: 500, width: 380, maxHeight: 600 });
    expect(menuPlacement({ left: 250, right: 280, top: 400, bottom: 420 }, work, 900))
      .toEqual({ left: 240, top: 420, width: 380, maxHeight: 360 });
    expect(menuPlacement({ left: 820, right: 910, top: 650, bottom: 670 }, work, 900))
      .toEqual({ left: 510, top: 100, width: 380, maxHeight: 550 });
    expect(menuPlacement({ left: 250, right: 280, top: 200, bottom: 220 },
      { ...work, right: 500 }, 200).width).toBe(260);
  });
  it('reads a real rect field by field, so the menu gets a finite left', () => {
    const work = { left: 240, right: 890, top: 100, bottom: 780 };
    const placed = menuPlacement(rect(780, 300, 100, 20), work, 200);
    expect(placed).toEqual({ left: 500, top: 320, width: 380, maxHeight: 460 });
    expect(Object.values(placed).every(Number.isFinite)).toBe(true);
  });
  it('applies a numeric left to the open menu from DOM rects (WebKit live-check geometry)', () => {
    vi.stubGlobal('innerWidth', 1280); vi.stubGlobal('innerHeight', 800);
    const popup = { style: {} as Record<string, string>, scrollHeight: 139 };
    const control = { getBoundingClientRect: () => rect(1007, 465, 162, 15) };
    const work = { getBoundingClientRect: () => rect(193, 46, 998, 390) };
    const host = { getAttribute: () => 'closed', querySelector: () => null };
    placeRowMenu(popup as unknown as HTMLElement, control as unknown as Element,
      work as unknown as Element, host as unknown as Element);
    expect(popup.style.left).toMatch(/^\d+px$/);
    expect(popup.style).toEqual({ left: '789px', top: '289px', width: '380px', maxHeight: '374px' });
    for (const key of ['innerWidth', 'innerHeight']) Reflect.deleteProperty(globalThis, key);
  });
  it('shows Kept once saved, and offers Keep again only for different terms', async () => {
    const request: ContractActRequest = { mine: '0001', player: '0042', action: 'preview',
      intent: 'contract.extend', level: 1, years: 1, move: '1', source: 0, destination: 0 };
    useContractMove.setState({ kept: undefined, keptTerms: '', saving: false, error: '' });
    expect(keepLabel(request)).toBe('Keep plan');
    const result = (plans: ContractActPlan[]) => ({ value: plans } as Reading<ContractActPlan[]>);
    await keepContractMove(request, vi.fn().mockResolvedValueOnce(result([]))
      .mockResolvedValueOnce(result([plan()])));
    expect(keepLabel(request)).toBe('Kept');
    expect(keepLabel({ ...request, years: 2 })).toBe('Keep plan');
    expect(keepLabel({ ...request, years: 2, action: 'draft' })).toBe('Keep plan');
    expect(keepLabel({ ...request, action: 'draft' })).toBe('Kept');
    expect(keepLabel(request, { ...useContractMove.getState(), saving: true })).toBe('Keeping…');
    const form = readFileSync(new URL('./ContractActs.tsx', import.meta.url), 'utf8');
    expect(form).not.toContain('kept: undefined');
    expect(form).toContain("disabled={keep !== 'Keep plan'}");
    useContractMove.setState({ kept: undefined, keptTerms: '' });
  });
  it('sends See it in My moves to the kept card, not the top of My moves', () => {
    const route = vi.spyOn(commands, 'dispatch').mockImplementation(() => {});
    revealContractPlan('contract-new');
    expect(useContractMove.getState().reveal).toBe('contract-new');
    expect(route).toHaveBeenCalledWith('nav.open', { node: 'hq', workspace: 'my-moves' });
    useContractMove.setState({ reveal: '' });
    route.mockRestore();
    const form = readFileSync(new URL('./ContractActs.tsx', import.meta.url), 'utf8');
    expect(form).toContain("args={{ field: 'contract-reveal', value: plan.id }}>See it in My moves");
    const moves = readFileSync(new URL('../shell/MyMoves.tsx', import.meta.url), 'utf8');
    expect(moves).toContain('<ContractActs mine={franchiseId} revealing />');
  });
  it('frames the plan card like a bid plan, sections spaced 8–12 px', () => {
    const p = plan(); p.receipt!.state = 'ready'; p.canHandOff = true;
    const html = renderToStaticMarkup(createElement(ContractActCard, { plan: p, mine: '0001' }));
    expect(html).toContain('class="bid-plan contract-act-card"');
    expect(html).toContain('<pre class="bid-post" aria-label="Contract ProBoards post">');
    expect(html).toMatch(/<div class="bid-actions"><button[^>]*>Copy request and hand off/);
    expect(html).toContain('<p class="bid-note">Post this to the commissioner');
    expect(html).not.toContain('Filter or sort contracts');
    const css = readFileSync(new URL('./contracts.css', import.meta.url), 'utf8');
    const gap = /article\.bid-plan\.contract-act-card \{ display: grid; gap: (\d+)px; \}/.exec(css);
    expect(Number(gap?.[1])).toBeGreaterThanOrEqual(8); expect(Number(gap?.[1])).toBeLessThanOrEqual(12);
    expect(css).toContain('article.bid-plan.contract-act-card > * { margin: 0; }');
    expect(css).not.toMatch(/\.contract-act-card[^{]*\{[^}]*(border|background)/);
  });
  it('shows whose contract the panel changes, from Go', () => {
    const form = readFileSync(new URL('./ContractActs.tsx', import.meta.url), 'utf8');
    expect(form).toContain('<p className="contract-move-subject">{options[0]?.subject ?? plan?.subject}</p>');
    expect(form.indexOf('contract-move-subject')).toBeLessThan(form.indexOf('<fieldset'));
  });
  it('shows team cap before keeping, outside the counting disclosure', () => {
    const html = renderToStaticMarkup(createElement(ContractMovePreview, { plan: plan() }));
    expect(html.indexOf('Team cap before and after this Act'))
      .toBeLessThan(html.indexOf('<summary>How this is counted'));
    expect(html).toContain('Price:'); expect(html).toContain('Not checked: prior extension');
    const source = readFileSync(new URL('./Contracts.tsx', import.meta.url), 'utf8');
    expect(source.indexOf('<ContractMovePanel')).toBeGreaterThan(source.indexOf('<WhatIfPanel'));
    expect(source.indexOf('<ContractMovePanel'))
      .toBeLessThan(source.indexOf('title="My franchise cap"'));
    const form = readFileSync(new URL('./ContractActs.tsx', import.meta.url), 'utf8');
    const actions = readFileSync(new URL('./contractActs.ts', import.meta.url), 'utf8');
    expect(form).toContain('className="contract-primary"');
    expect(actions).toContain("reading.value?.find(p => !before.value?.some(old => old.id === p.id))");
    const css = readFileSync(new URL('./contracts.css', import.meta.url), 'utf8');
    expect(css).toContain('.release-plans > h4 { margin-top: 12px; margin-bottom: 6px; }');
  });
  it('opens the same panel from a command, then reports the newly kept plan, not an older Ready one',
    async () => {
      const route = vi.spyOn(commands, 'dispatch').mockImplementation(() => {});
      openContractMove('0001', '0042');
      expect(route).toHaveBeenCalledWith('nav.open', { node: 'hq', workspace: 'contracts-and-cap' });
      expect(useContractMove.getState().player).toBe('0042');
      const old = plan(); old.receipt!.state = 'ready';
      const added = { ...plan(), id: 'contract-new' };
      const result = (plans: ContractActPlan[]) => ({ value: plans } as Reading<ContractActPlan[]>);
      const action = vi.fn().mockResolvedValueOnce(result([old])).mockResolvedValueOnce(result([old, added]));
      await keepContractMove({ mine: '0001', player: '0042', action: 'preview' }, action);
      expect(action.mock.calls.map(([request]) => request.action)).toEqual(['read', 'draft']);
      expect(useContractMove.getState().kept?.id).toBe('contract-new');
      expect(useContractMove.getState().saving).toBe(false);
      const failed = vi.fn().mockRejectedValue(new Error('mirror unavailable'));
      await keepContractMove({ mine: '0001', player: '0042', action: 'preview' }, failed);
      expect(useContractMove.getState().error).toContain('mirror unavailable');
      expect(useContractMove.getState().saving).toBe(false);
      useContractMove.setState({ mine: '', player: '', label: '', kept: undefined, error: '' });
      route.mockRestore();
    });
  it('shows kept Ready / Blocked feedback and a real My moves link', () => {
    const p = plan();
    let html = renderToStaticMarkup(createElement(ContractMoveFeedback, { plan: p }));
    expect(html).toContain('Plan: Blocked'); expect(html).toContain('Not checked: prior extension');
    expect(html).toContain('See it in My moves');
    p.receipt!.state = 'ready';
    html = renderToStaticMarkup(createElement(ContractMoveFeedback, { plan: p }));
    expect(html).toContain('Plan: Ready'); expect(html).not.toContain('Not checked:');
  });
  it('keeps unchecked notes below the strip, not inside its Ready cell', () => {
    const p = plan(); p.receipt!.state = 'ready';
    p.reasons.push('Not checked: team allowance');
    p.receipt!.audit = [{ at, from: 'draft', to: 'ready', note: p.reasons.join('; ') }];
    const html = renderToStaticMarkup(createElement(ContractActCard, { plan: p, mine: '0001' }));
    const strip = html.slice(html.indexOf('aria-label="Move stages"'), html.indexOf('</ol>'));
    expect(strip).toContain('Ready: 2 things not checked');
    expect(strip).not.toContain('Not checked:');
    expect(html.slice(html.indexOf('</ol>'))).toContain('Not checked: team allowance');
  });
  it('renders Go reasons, post and figures without computing money or hiding uncertainty', () => {
    const html = renderToStaticMarkup(createElement(ContractActCard, { plan: plan(), mine: '0001' }));
    expect(html).toContain('Not checked: prior extension');
    expect(html).toContain('Move stages');
    expect(html).toContain('2029: $15,000,000');
    expect(html).toContain('My team&#x27;s cap this season');
    expect(html).toContain('How this is counted');
    expect(html).toContain('Team cap before and after this Act');
    expect(html).toContain('Change');
    expect(html).not.toContain('Copy request and hand off');
    expect(html).toContain('Remove plan');
  });
  it('uses the shared strip and separate copy / posted / withdraw actions', () => {
    const p = plan(); p.receipt!.state = 'handed_off'; p.canPost = true;
    const html = renderToStaticMarkup(createElement(ContractActCard, { plan: p, mine: '0001' }));
    expect(html).toContain('I posted it'); expect(html).toContain('I didn&#x27;t post it');
    p.receipt!.state = 'bid_pending'; p.canPost = false;
    p.receipt!.audit.push({ at, from: 'handed_off', to: 'bid_pending', note: 'You posted it' });
    const pending = renderToStaticMarkup(createElement(ContractActCard, { plan: p, mine: '0001' }));
    expect(pending).toContain('Pending'); expect(pending).not.toContain('Bid pending');
  });
  it.each(['withdrawn', 'landed', 'failed'] as const)('folds %s contract post under History', state => {
    const p = plan(); p.receipt!.state = state;
    const html = renderToStaticMarkup(createElement(ContractActCard, { plan: p, mine: '0001' }));
    expect(html.indexOf('Contract ProBoards post'))
      .toBeGreaterThan(html.indexOf('<summary>History</summary>'));
    expect(html).not.toContain('Indicative price'); expect(html).not.toContain('Remove plan');
    expect(html).not.toContain('future baseline cells are inferred');
  });
  it.each(['withdrawn', 'landed', 'failed'] as const)('folds %s release post under History', state => {
    const p: ReleasePlan = { id: 'release-1', player: 'Example', label: 'Buy out Example',
      charge: '$5,000,000', confirmation: '',
      post: 'Full Go post', channel: 'Post this to the commissioner on ProBoards.', receipt: { state,
      spec: { intent: 'roster.buyout' }, audit: [] },
      canHandOff: false, canRebase: false, cut: false, rebaseLabel: '', withdrawLabel: '' };
    const html = renderToStaticMarkup(createElement(ReleaseCard, { plan: p, mine: '0001' }));
    expect(html.indexOf('Release request text')).toBeGreaterThan(html.indexOf('<summary>History</summary>'));
    expect(html).not.toContain('Planned dead cap this season:');
  });
});
