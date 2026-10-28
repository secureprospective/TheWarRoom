import { act } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { all, button, click, control, disabled, holder, mount, render, settle, type, unmount,
  type Element } from './commissionerDom';
import { stub } from './commissionerStub';
import LeagueLanding from './LeagueLanding';
import { leagueState } from './proxy';
import { RoleForm } from './RoleForm';
import { commissionerPeople } from './commissionerStub';
import { surfaceActions } from './surfaceActions';
import { active, dialogOf, focused, key, modelFocusApis } from './trayHarness';
import controlsWire from './controls.wire.testdata.json';
import formsWire from './preview-forms.wire.testdata.json';
import jobsWire from './jobs.wire.testdata.json';
import ledgerWire from './wire.testdata.json';
import peopleWire from './people.wire.testdata.json';

let restoreFocus: () => void;
beforeEach(() => {
  leagueState.forget();
  restoreFocus = modelFocusApis();
});
afterEach(async () => {
  await unmount();
  vi.useRealTimers();
  restoreFocus();
});

type Preview = typeof formsWire.start;
const answering = (source: Preview | Record<string, unknown>) => async (body: string) => {
  const sent = JSON.parse(body);
  const wire = source as Preview;
  return { ...wire, target: { ...wire.target, league: sent.league, version: sent.expected } };
};
const undoable = { ...ledgerWire, entries: ledgerWire.entries.map(e => ({ ...e, undo: 'league' })) };
const landing = async (over: Parameters<typeof stub>[0] = {}) => {
  await Promise.all([import('./LeagueRules'), import('./LeaguePeople'), import('./LeagueLedger'),
    import('./ActionTray')]);
  const { app, calls } = stub({ TargetCommissionerControls: async () => controlsWire,
    TargetCommissionerJobs: async () => jobsWire.fresh,
    TargetCommissionerLedger: async () => undoable,
    TargetCommissionerPreview: async body => {
      const command = JSON.parse(body).command as string;
      const source = command.includes('create') ? formsWire.start
        : command.includes('revert') ? formsWire.undo : formsWire.role;
      return answering(source)(body);
    }, ...over });
  const host = await mount(render(LeagueLanding), app);
  active(host);
  await settle();
  return { host, calls };
};

// The three changes that join the rule change in the tray: where each starts and what fills it in.
const forms = [
  { name: 'start a league', group: 'constitution', launcher: 'Start another app league',
    title: 'Start an app league', close: 'Close league start', confirm: 'Start this league',
    preview: formsWire.start,
    fill: async (panel: Element) => { await type(control(panel, 'League name'), 'Our league'); } },
  { name: 'give or take a role', group: 'people', launcher: 'Give or take a role',
    title: 'Give or take a role', close: 'Close role change', confirm: 'Give this role',
    preview: formsWire.role,
    fill: async (panel: Element) => { await type(control(panel, 'Franchise'), 'franchise.02'); } },
  { name: 'undo', group: 'ledger', launcher: `Undo this change: ${ledgerWire.entries[0].what}`,
    title: 'Undo this change', close: 'Close undo', confirm: 'Undo this change',
    preview: formsWire.undo, fill: async () => {} },
];
const showGroup = async (group: string) => {
  await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: group }));
  await settle();
};
const enabled = (panel: Element) => all(panel, 'button').concat(all(panel, 'input'), all(panel, 'select'),
  all(panel, 'textarea')).filter(e => !disabled(e));

describe('The launchers replace the old inline desk', () => {
  it('offers a first league in Constitution, then opens it rather than leaving a form on the page',
    async () => {
      const { host } = await landing({ TargetCommissionerLeagues: async () => [] });
      expect(host.textContent).toContain('No app league has been started yet');
      const opener = button(host, 'Start an app league');
      expect(all(host, 'textarea')).toHaveLength(0);
      opener.focus();
      await click(opener);
      await settle();
      expect(control(dialogOf(host)!, 'League name')).toBeDefined();
    });
  it('keeps one league picker with the selected league', async () => {
    const { host } = await landing();
    expect(all(host, 'label').filter(l => l.textContent.startsWith('App league'))).toHaveLength(1);
    expect(control(host, 'App league').props().value).toBe('native.rules');
  });
  it.each([
    { role: 'owner', people: peopleWire.owner },
    { role: 'co-commissioner', people: { ...peopleWire.commissioner, mayGrantRoles: false } },
  ])('does not offer role changes to the $role', async ({ people }) => {
    const { host } = await landing({ TargetCommissionerPeople: async () => people });
    await showGroup('people');
    expect(all(host, 'button').some(b => b.getAttribute('aria-label') === 'Give or take a role'))
      .toBe(false);
    expect(all(host, 'textarea')).toHaveLength(0);
    await showGroup('constitution');
    expect(button(host, 'Start another app league')).toBeDefined();
    if (!people.mayChangeRules) {
      expect(all(host, 'button').some(b => b.getAttribute('aria-label')?.startsWith('Change ')))
        .toBe(false);
    }
  });
});

describe.each(forms)('The $name form', form => {
  it('opens in the side tray in the shared order, traps focus and returns it to the control', async () => {
    const { host } = await landing();
    await showGroup(form.group);
    expect(all(host, 'section').some(e => e.getAttribute('class') === 'league-form')).toBe(false);
    expect(all(host, 'textarea')).toHaveLength(0);
    const opener = button(host, form.launcher);
    opener.focus();
    await click(opener);
    await settle();
    const panel = dialogOf(host)!;
    expect(panel).toBeDefined();
    expect(all(panel, 'h3').map(h => h.textContent)).toEqual([form.title]);
    expect(focused(host)).toBe(button(panel, form.close));
    await form.fill(panel);
    await settle();
    const text = panel.textContent;
    const ordered = [form.preview.question, 'Values and sources', 'Before → after',
      'Affects', form.preview.affects.words, 'Before you confirm', form.confirm];
    const at = ordered.map(word => text.lastIndexOf(word));
    expect(at.every(n => n >= 0)).toBe(true);
    expect(at).toEqual([...at].sort((a, b) => a - b));
    expect(text).not.toMatch(/CM-\d|G[123]\b/);
    const items = enabled(panel);
    const close = button(panel, form.close);
    items[items.length - 1].focus();
    await key(panel, 'Tab');
    expect(focused(host)).toBe(close);
    await key(panel, 'Tab', true);
    expect(focused(host)).toBe(items[items.length - 1]);
    await key(panel, 'Escape');
    expect(dialogOf(host)).toBeUndefined();
    expect(focused(host)).toBe(opener);
    expect(all(host, 'div').some(d => d.attributes.has('inert'))).toBe(false);
  });
  it('returns focus to the control when the tray’s own Close is used', async () => {
    const { host } = await landing();
    await showGroup(form.group);
    expect(all(host, 'section').some(e => e.getAttribute('class') === 'league-form')).toBe(false);
    expect(all(host, 'textarea')).toHaveLength(0);
    const opener = button(host, form.launcher);
    opener.focus();
    await click(opener);
    await settle();
    await click(button(dialogOf(host)!, form.close));
    expect(dialogOf(host)).toBeUndefined();
    expect(focused(host)).toBe(opener);
  });
  it('shows Go’s blockers word for word and keeps the confirm off', async () => {
    const words = 'Go says: not this one.';
    const { host, calls } = await landing({ TargetCommissionerPreview:
      answering({ ...form.preview, blockers: [words] }) });
    await showGroup(form.group);
    button(host, form.launcher).focus();
    await click(button(host, form.launcher));
    await settle();
    const panel = dialogOf(host)!;
    await form.fill(panel);
    await type(control(panel, 'Reason'), 'Because we agreed');
    await settle();
    expect(panel.textContent).toContain(words);
    expect(disabled(button(panel, form.confirm))).toBe(true);
    expect(calls.sent).toHaveLength(0);
  });
  it('keeps Go’s refusal words when it cannot describe the target or offer a safeguard', async () => {
    const words = 'Choose between 2 and 1000 franchises';
    const refused = { question: '', target: { league: '', name: '', version: 0 }, fields: [],
      affects: { count: null, readings: [], words: 'Not known yet' }, blockers: [words], safeguard: '' };
    const { host, calls } = await landing({ TargetCommissionerPreview: async () => refused });
    await showGroup(form.group);
    button(host, form.launcher).focus();
    await click(button(host, form.launcher));
    await settle();
    const panel = dialogOf(host)!;
    await form.fill(panel);
    await settle();
    expect(panel.textContent).toContain(words);
    expect(panel.textContent).not.toContain('does not match');
    expect(disabled(button(panel, form.confirm))).toBe(true);
    expect(calls.sent).toHaveLength(0);
  });
  it('takes what it affects from Go, and says plainly when Go does not know', async () => {
    const unknown = { ...form.preview, affects: { readings: [], count: null, words: 'Made up elsewhere' } };
    const { host } = await landing({ TargetCommissionerPreview: answering(unknown) });
    await showGroup(form.group);
    button(host, form.launcher).focus();
    await click(button(host, form.launcher));
    await settle();
    const panel = dialogOf(host)!;
    await form.fill(panel);
    await settle();
    expect(panel.textContent).toContain('Not known yet');
    expect(panel.textContent).not.toContain('Made up elsewhere');
  });
});

describe('The confirm Go asks for', () => {
  it.each(['hold', 'reason', 'confirm'] as const)('is the %s Go reports for a role change', async name => {
    const sent: unknown[] = [];
    const { app } = stub({ TargetCommissionerExecute: async body => { sent.push(body); return {}; },
      TargetCommissionerPreview: answering({ ...formsWire.role, safeguard: name }) });
    const host = await mount(render(RoleForm, { people: commissionerPeople }), app);
    await type(control(host, 'Franchise'), 'franchise.02');
    await type(control(host, 'Reason'), 'Agreed');
    await settle();
    const confirm = button(host, 'Give this role');
    expect(disabled(confirm)).toBe(false);
    if (name === 'hold') {
      expect(confirm.props().onClick).toBeUndefined();
      expect(holder(host, 'Give this role').props().onKeyDown).toBeTypeOf('function');
    } else {
      expect(confirm.props().onClick).toBeTypeOf('function');
    }
    expect(sent).toHaveLength(0);
  });
});

