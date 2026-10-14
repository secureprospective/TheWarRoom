import { act } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { KeyInputElement, descendants } from '../shell/mflKeyTestDom';
import { all, button, click, control, disabled, flush, holder, mount, render, settle, type,
  unmount, type Element } from './commissionerDom';
import { open, rules, stub } from './commissionerStub';
import { leagueState } from './proxy';
import LeagueLanding from './LeagueLanding';
import { RuleChangeForm } from './RuleChangeForm';
import { PreviewReview } from './PreviewReview';
import { parsePreview } from './preview';
import previewWire from './preview.wire.testdata.json';
import controlsWire from './controls.wire.testdata.json';
import jobsWire from './jobs.wire.testdata.json';

// Model only the browser focus APIs the tray consumes; the form harness already models rendering.
const descriptors = new Map<string, PropertyDescriptor | undefined>();
beforeEach(() => {
  leagueState.forget();
  for (const key of ['querySelectorAll', 'hasAttribute', 'tabIndex', 'closest', 'contains'])
    descriptors.set(key, Object.getOwnPropertyDescriptor(KeyInputElement.prototype, key));
  Object.defineProperties(KeyInputElement.prototype, {
    querySelectorAll: { configurable: true, value(this: Element, selector: string) {
      return descendants(this).filter(e => selector === '[hidden]' ? e.attributes.has('hidden') :
        ['BUTTON', 'INPUT', 'SELECT', 'TEXTAREA'].includes(e.tagName) || e.attributes.has('tabindex'));
    } },
    hasAttribute: { configurable: true, value(this: Element, key: string) {
      return this.attributes.has(key);
    } },
    tabIndex: { configurable: true, get(this: Element) {
      return Number(this.getAttribute('tabindex') ?? 0);
    } },
    closest: { configurable: true, value() { return null; } },
    contains: { configurable: true, value(this: Element, other: Element) {
      return this === other || descendants(this).includes(other);
    } },
  });
});
afterEach(async () => {
  await unmount();
  vi.useRealTimers();
  for (const [key, descriptor] of descriptors) {
    if (descriptor) Object.defineProperty(KeyInputElement.prototype, key, descriptor);
    else Reflect.deleteProperty(KeyInputElement.prototype, key);
  }
});
const focused = (host: Element) => (host.ownerDocument as unknown as { focused: Element }).focused;
function active(host: Element) {
  Object.defineProperty(host.ownerDocument, 'activeElement', { configurable: true,
    get: () => focused(host) });
}
const key = (panel: Element, name: string, shiftKey = false) => act(async () => {
  (panel.props().onKeyDown as (event: object) => void)({ key: name, shiftKey,
    preventDefault() {}, stopPropagation() {} });
});
const confirm = 'Record this rule change';
const mountForm = (over: Parameters<typeof stub>[0] = {}) => {
  const supplied = stub(over);
  return mount(render(RuleChangeForm, { rules, initialField: 'roster.limits' }), supplied.app)
    .then(host => ({ host, ...supplied }));
};

describe('Commissioner tray', () => {
  it('opens from a rule search, traps both Tab edges, closes with Escape and returns focus', async () => {
    await import('./RuleTray');
    const { app } = stub({ TargetCommissionerControls: async () => controlsWire,
      TargetCommissionerJobs: async () => jobsWire.fresh });
    const host = await mount(render(LeagueLanding), app);
    active(host);
    await settle();
    const search = all(host, 'input').find(e => e.getAttribute('id') === 'control-search')!;
    await type(search, 'timezone');
    const opener = all(host, 'button').find(b => b.textContent.includes('Rule: League timezone'))!;
    opener.focus();
    await click(opener);
    await settle();
    const panel = all(host, 'section').find(s => s.getAttribute('role') === 'dialog')!;
    expect(panel).toBeDefined();
    expect(control(panel, 'Rule').props().value).toBe('calendar.timezone');
    const close = button(panel, 'Close rule change');
    expect(focused(host)).toBe(close);
    await type(control(panel, 'New value for League timezone'), 'America/Chicago');
    expect(all(host, 'div').some(d => d.attributes.has('inert'))).toBe(true);
    const last = button(panel, confirm);
    last.focus();
    await key(panel, 'Tab');
    expect(focused(host)).toBe(close);
    close.focus();
    await key(panel, 'Tab', true);
    expect(focused(host)).toBe(last);
    await key(panel, 'Escape');
    expect(all(host, 'section').some(s => s.getAttribute('role') === 'dialog')).toBe(false);
    expect(focused(host)).toBe(opener);
    expect(all(host, 'div').some(d => d.attributes.has('inert'))).toBe(false);
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b/);
  });
  it('opens a landing Change link for that rule and returns to that exact link', async () => {
    await import('./RuleTray');
    const { app } = stub({ TargetCommissionerControls: async () => controlsWire,
      TargetCommissionerJobs: async () => jobsWire.fresh });
    const host = await mount(render(LeagueLanding), app);
    active(host);
    await settle();
    await click(all(host, 'button').find(b => b.textContent.startsWith('Calendar'))!);
    await settle();
    const opener = button(host, 'Change League timezone');
    opener.focus();
    await click(opener);
    await settle();
    const panel = all(host, 'section').find(s => s.getAttribute('role') === 'dialog')!;
    expect(control(panel, 'Rule').props().value).toBe('calendar.timezone');
    await click(button(panel, 'Close rule change'));
    expect(focused(host)).toBe(opener);
  });
  it('presents Go fields in order and the review stages in order, with unknown counts honest', async () => {
    const p = parsePreview(previewWire.hold);
    p.fields.push({ ...p.fields[0], label: 'Second field' });
    p.affects.count = null;
    const host = await mount(render(PreviewReview, { preview: p, confirm: render('button', null, 'Confirm'),
      children: render('p', null, 'Edit fields') }));
    const text = host.textContent;
    const ordered = [p.question, 'League version: 1', 'Values and sources',
      'Active roster size — in force now',
      'Second field — in force now', 'Edit fields', 'Before → after',
      'Active roster size: 18 to 80 → 20 to 70',
      'Second field: 18 to 80 → 20 to 70', 'Affects', 'Not known yet', 'Before you confirm', 'Confirm'];
    expect(ordered.map(word => text.indexOf(word))).toEqual(
      ordered.map(word => text.indexOf(word)).sort((a, b) => a - b));
    ordered.forEach(word => expect(text).toContain(word));
    expect(text).toContain('Legacy NFL rules');
    expect(text).toContain('Your proposed change');
  });
  it('keeps Go blockers verbatim and disables confirmation', async () => {
    const words = 'Your league role does not permit this action';
    const { host, calls } = await mountForm({ TargetCommissionerPreview: async () => ({ ...previewWire.hold,
      target: { league: rules.league, name: rules.name, version: rules.revision }, blockers: [words] }) });
    expect(host.textContent).toContain(words);
    expect(disabled(button(host, confirm))).toBe(true);
    expect(calls.sent).toHaveLength(0);
  });
  it('cancels an early hold release; Space and Enter each need the full 700 ms', async () => {
    const { host, calls } = await mountForm();
    await open();
    for (const name of [' ', 'Enter']) {
      vi.useFakeTimers();
      const wrapper = holder(host, confirm);
      const event = { key: name, preventDefault() {} };
      await act(async () => { (wrapper.props().onKeyDown as (e: object) => void)(event); });
      await act(async () => { vi.advanceTimersByTime(699); });
      expect(calls.sent).toHaveLength(name === ' ' ? 0 : 1);
      await act(async () => { (wrapper.props().onKeyUp as (e: object) => void)(event); });
      await act(async () => { vi.advanceTimersByTime(1); });
      expect(calls.sent).toHaveLength(name === ' ' ? 0 : 1);
      await act(async () => { (wrapper.props().onKeyDown as (e: object) => void)(event); });
      await act(async () => { vi.advanceTimersByTime(700); });
      await flush();
      vi.useRealTimers();
      await settle();
    }
    expect(calls.sent).toHaveLength(2);
  });
  it('never enables a superseded draft and sends exactly the body whose preview was read', async () => {
    const waiting: { body: string; resolve(value: unknown): void }[] = [];
    const { host, calls } = await mountForm({ TargetCommissionerPreview: body =>
      new Promise(resolve => waiting.push({ body, resolve })) });
    expect(disabled(button(host, confirm))).toBe(true);
    await type(control(host, 'Minimum'), '25');
    const resolve = (index: number) => {
      const sent = JSON.parse(waiting[index].body);
      waiting[index].resolve({ ...previewWire.confirm,
        target: { league: sent.league, name: rules.name, version: sent.expected } });
    };
    await act(async () => { resolve(0); });
    expect(disabled(button(host, confirm))).toBe(true);
    await act(async () => { resolve(1); });
    expect(disabled(button(host, confirm))).toBe(false);
    await click(button(host, confirm));
    expect(calls.sent[0]).toEqual(JSON.parse(waiting[1].body));
  });
  it('refuses a preview for another frozen target or a failed read', async () => {
    const { host } = await mountForm({ TargetCommissionerPreview: async () => previewWire.hold });
    expect(host.textContent).toContain('does not match this league version');
    expect(disabled(button(host, confirm))).toBe(true);
  });
  it('keeps a failed preview read disabled and says why', async () => {
    const { host } = await mountForm({ TargetCommissionerPreview: async () => {
      throw new Error('The league could not be read.');
    } });
    expect(host.textContent).toContain('The league could not be read.');
    expect(disabled(button(host, confirm))).toBe(true);
  });
  it('uses Go’s preview safeguard, even when the older rule reading differs', async () => {
    const { host } = await mountForm({ TargetCommissionerPreview: async body => {
      const sent = JSON.parse(body);
      return { ...previewWire.confirm, target: { league: sent.league, name: rules.name,
        version: sent.expected } };
    } });
    expect(host.textContent).toContain('You will be asked to confirm once. A reason is optional.');
    expect(button(host, confirm).props().onClick).toBeTypeOf('function');
  });
  it('parses all Go safeguard fixtures and rejects unknown or malformed boundary fields', () => {
    for (const [name, wire] of Object.entries(previewWire)) {
      expect(parsePreview(wire)).toEqual(wire);
      expect(parsePreview(wire).safeguard).toBe(name === 'stale' ? 'hold' : name);
    }
    expect(() => parsePreview({ ...previewWire.hold, extra: true })).toThrow('unexpected field');
    expect(() => parsePreview({ ...previewWire.hold, safeguard: 'G2' })).toThrow('expected');
    expect(() => parsePreview({ ...previewWire.hold, affects: { ...previewWire.hold.affects, count: -1 } }))
      .toThrow('safe integer');
  });
  it('reads a refused change as blockers with nothing to confirm, never as a parse error', () => {
    const refused = { ...previewWire.hold, safeguard: '', blockers: ['The league changed; reload it.'] };
    expect(parsePreview(refused).safeguard).toBeNull();
    expect(parsePreview(refused).blockers).toEqual(['The league changed; reload it.']);
    expect(() => parsePreview({ ...previewWire.hold, safeguard: '' })).toThrow('expected');
  });
});
