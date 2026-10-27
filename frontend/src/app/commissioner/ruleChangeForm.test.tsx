import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, button, control, disabled, flush, holder, mount, present, render, rerender, settle, type,
  unmount, type Element } from './commissionerDom';
import { open, rules, stub, receipt } from './commissionerStub';
import { leagueState, staleWords } from './proxy';
import { RuleChangeForm } from './RuleChangeForm';
import LeagueRules from './LeagueRules';
import rulesWire from './rules.wire.testdata.json';
import previewWire from './preview.wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(async () => { await unmount(); vi.useRealTimers(); });

const confirm = 'Record this rule change';
const hold = async (host: Element, label = confirm) => {
  await settle();
  vi.useFakeTimers();
  const b = holder(host, label);
  await act(async () => { (b.props().onKeyDown as (e: object) => void)({ key: 'Enter', repeat: false,
    preventDefault() {} }); });
  await act(async () => { vi.advanceTimersByTime(600); });
  await act(async () => { (b.props().onKeyUp as (e: object) => void)({ key: 'Enter' }); });
  await flush();
  vi.useRealTimers();
  await settle();
};
const form = (field = 'roster.limits', held = rules) =>
  render(RuleChangeForm, { rules: held ?? rules, initialField: field });
const waiting = (value: string) => [{ entry: 'act.new', value, when: 'Waits for the next rollover' }];
const afterNext = (value: string, revision = 4) => ({ ...rulesWire, revision,
  sections: rulesWire.sections.map(s => ({ ...s, rows: s.rows.map(r =>
    r.field !== 'roster.limits' ? r : { ...r, pending: waiting(value) }) })) });

describe('Change a rule', () => {
  it('lists every rule by its plain label under its section, never a path', async () => {
    const host = await mount(form(''), stub().app);
    const options = all(host, 'option').map(o => o.textContent);
    expect(options).toContain('Active roster size');
    expect(all(host, 'optgroup').map(g => g.getAttribute('label'))).toEqual(rules.sections.map(s => s.label));
    expect(options.length).toBe(rules.sections.flatMap(s => s.rows).length + 1);
    expect(options.some(o => o.includes('roster.limits') || o.includes('.'))).toBe(false);
  });
  it('offers a range with Go’s bounds, today’s value in words, and its source', async () => {
    const host = await mount(form(), stub().app);
    for (const end of ['Minimum', 'Maximum']) {
      expect(control(host, end).props()).toMatchObject({ type: 'number', min: 0, step: 1 });
    }
    expect(control(host, 'Minimum').props().value).toBe('20');
    expect(present(host, 'In force now20 to 70')).toBe(true);
    expect(host.textContent).not.toContain('(Set)');
    const local = new Date('2026-10-14T04:17:03Z').toLocaleString();
    expect(present(host, `Changed in the league ledger on ${local}`)).toBe(true);
  });
  it('shows the safeguard Go reports before anything is confirmed', async () => {
    const host = await mount(form(), stub().app);
    expect(present(host, 'Hold the confirm button to record it. A reason is optional.')).toBe(true);
    expect(present(host, 'The next rollover has no date yet; the commissioner sets it.')).toBe(true);
    await type(control(host, 'When it takes effect'), 'now');
    expect(present(host, 'Needs a typed reason before it can be recorded.')).toBe(true);
    expect(present(host, 'Reason (required)')).toBe(true);
  });
  it('sends the next-rollover change once held; the screen shows it waiting by the old value', async () => {
    let landed = false;
    const { app, calls } = stub({ TargetCommissionerExecute: async body => {
      calls.sent.push(JSON.parse(body));
      landed = true;
      return receipt();
    } }, { rules: () => (landed ? afterNext('25 to 60') : rulesWire) });
    const host = await mount(render('div', null, form(), render(LeagueRules)), app);
    await open();
    await settle();
    expect(present(host, '25 to 60')).toBe(false);
    await type(control(host, 'Minimum'), '25');
    await type(control(host, 'Maximum'), '60');
    await hold(host);
    expect(calls.sent).toHaveLength(1);
    expect(calls.sent[0]).toMatchObject({ command: 'commish.rules.set', league: 'native.rules', expected: 3,
      effective: 'next_rollover', visibility: 'league', reason: '',
      args: { field: 'roster.limits', value: { Min: 25, Max: 60 } } });
    expect(present(host, '25 to 60: Waits for the next rollover')).toBe(true);
    expect(present(host, '20 to 70')).toBe(true);
    expect(host.textContent).not.toContain('Confirmed. Sending the change.');
  });
  it('cannot confirm Now until a reason is typed, then sends effective now', async () => {
    const { app, calls } = stub();
    const host = await mount(form(), app);
    await type(control(host, 'When it takes effect'), 'now');
    expect(disabled(button(host, confirm))).toBe(true);
    await type(control(host, 'Reason'), ' ');
    expect(disabled(button(host, confirm))).toBe(true);
    await type(control(host, 'Reason'), 'Owners voted at the draft');
    expect(disabled(button(host, confirm))).toBe(false);
    await act(async () => { (button(host, confirm).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0]).toMatchObject({ effective: 'now', reason: 'Owners voted at the draft' });
  });
  it('will not send a value Go’s bounds or the field’s kind refuse, and says why', async () => {
    const host = await mount(form(), stub().app);
    await type(control(host, 'Minimum'), '-1');
    expect(disabled(button(host, confirm))).toBe(true);
    expect(present(host, 'Active roster size › Minimum: must be at least 0')).toBe(true);
    await type(control(host, 'Minimum'), 'many');
    expect(present(host, 'enter a whole number')).toBe(true);
  });
  it('shows an owner’s refusal as Go wrote it and changes nothing else', async () => {
    const words = 'Your league role does not permit this action';
    const { app, calls } = stub({ TargetCommissionerExecute: async () => { throw new Error(words); } });
    const host = await mount(form(), app);
    await open();
    calls.ledger.length = calls.rules.length = calls.people.length = 0;
    await hold(host);
    expect(host.textContent).toContain(words);
    expect([calls.ledger, calls.rules, calls.people]).toEqual([[], [], []]);
    expect(control(host, 'Minimum').props().value).toBe('20');
  });
  it('says the league changed on a stale version, reloads, and takes a fresh key next try', async () => {
    let attempt = 0;
    const { app, calls } = stub({ TargetCommissionerExecute: async body => {
      calls.sent.push(JSON.parse(body));
      if (attempt++ === 0) throw new Error('League version is 9; reread before changing it (expected 3)');
      return receipt();
    } }, { rules: () => (attempt > 0 ? afterNext('20 to 70', 9) : rulesWire) });
    const host = await mount(form(), app);
    await open();
    calls.rules.length = 0;
    await hold(host);
    expect(host.textContent).toContain(staleWords);
    expect(calls.rules).toEqual(['native.rules']);
    expect(leagueState.peek().rules?.revision).toBe(9);
    await rerender(form('roster.limits', leagueState.peek().rules));
    await hold(host);
    expect(calls.sent.map(s => s.expected)).toEqual([3, 9]);
    expect(calls.sent[1].request).not.toBe(calls.sent[0].request);
  });
  it('reuses the same key when the very same submit is tried again', async () => {
    let attempt = 0;
    const { app, calls } = stub({ TargetCommissionerExecute: async body => {
      calls.sent.push(JSON.parse(body));
      if (attempt++ === 0) throw new Error('the league could not be reached');
      return receipt();
    } });
    const host = await mount(form(), app);
    await open();
    await hold(host);
    await hold(host);
    expect(calls.sent).toHaveLength(2);
    expect(calls.sent[1].request).toBe(calls.sent[0].request);
    expect(calls.sent[1]).toEqual(calls.sent[0]);
  });
  it('builds a list through the add and remove verbs and sends it', async () => {
    const { app, calls } = stub();
    const host = await mount(form('ir.eligible'), app);
    const press = async (label: string) => {
      await act(async () => { (button(host, label).props().onClick as () => void)(); });
      await settle();
    };
    const list = 'New value for IR eligible designations';
    await press(`Add to ${list}`);
    await press(`Add to ${list}`);
    expect(present(host, `${list} #2`)).toBe(true);
    await type(control(host, `${list} #1`), 'IR');
    await type(control(host, `${list} #2`), 'PUP');
    await press(`Remove ${list} #1`);
    expect(present(host, `${list} #2`)).toBe(false);
    await hold(host);
    expect(calls.sent[0].args).toEqual({ field: 'ir.eligible', value: ['PUP'] });
  });
  it('sets the conference of each division through the map editor', async () => {
    const { app, calls } = stub();
    const host = await mount(form('league.divisionConferences'), app);
    const map = 'New value for Conference of each division';
    expect(all(host, 'option').map(o => o.textContent)).toContain('Conference of each division');
    expect(present(host, 'In force nowNeeds a ruling')).toBe(true);
    await act(async () => { (button(host, `Add to ${map}`).props().onClick as () => void)(); });
    await settle();
    const division = all(host, 'input').find(i => i.getAttribute('aria-label') === `${map}: entry 1 name`);
    await type(division!, 'AFC East');
    await type(control(host, `${map} › AFC East`), 'AFC');
    await hold(host);
    expect(calls.sent[0].args).toEqual({ field: 'league.divisionConferences', value: { 'AFC East': 'AFC' } });
  });
  it('sends a Teams change like any rule and shows the teams it adds before it is recorded', async () => {
    const teams = { ...previewWire.reason, fields: [{ label: 'Teams', before: '4',
      beforeSource: 'Counted from the league\'s franchises', after: '6 (adds Franchise 5, Franchise 6)',
      afterSource: 'Your proposed change', when: 'in force now' }] };
    const { app, calls } = stub({ TargetCommissionerPreview: async body => {
      const sent = JSON.parse(body) as { league: string; expected: number };
      return { ...teams, target: { ...teams.target, league: sent.league, version: sent.expected } };
    } });
    const host = await mount(form('league.teams'), app);
    expect(present(host, 'In force now4')).toBe(true);
    await type(control(host, 'New value for Teams'), '6');
    await type(control(host, 'When it takes effect'), 'now');
    await type(control(host, 'Reason'), 'Two expansion teams');
    await settle();
    expect(present(host, 'Teams: 4 → 6 (adds Franchise 5, Franchise 6) (in force now)')).toBe(true);
    await act(async () => { (button(host, confirm).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0]).toMatchObject({ command: 'commish.rules.set', effective: 'now', reason: 'Two expansion teams',
      args: { field: 'league.teams', value: 6 } });
  });
  it('will not send a list with a blank entry, and says which', async () => {
    const host = await mount(form('ir.eligible'), stub().app);
    const list = 'New value for IR eligible designations';
    await act(async () => { (button(host, `Add to ${list}`).props().onClick as () => void)(); });
    await settle();
    expect(disabled(button(host, confirm))).toBe(true);
    expect(present(host, 'IR eligible designations #1: fill this in or remove it')).toBe(true);
  });
});
