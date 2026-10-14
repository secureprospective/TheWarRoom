import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { act } from 'react';
import { KeyInputElement } from '../shell/mflKeyTestDom';
import { all, click, mount, render, settle, type, unmount } from './commissionerDom';
import { stub } from './commissionerStub';
import LeagueLanding from './LeagueLanding';
import { surfaceActions } from './surfaceActions';
import { entriesOf, parseControls } from './controls';
import { parseRules } from './rules';
import { leagueState } from './proxy';
import controlsWire from './controls.wire.testdata.json';
import jobsWire from './jobs.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';

const entries = entriesOf(parseControls(controlsWire));
beforeEach(() => {
  Object.defineProperty(KeyInputElement.prototype, 'closest', { configurable: true, value: () => null });
});
afterEach(async () => {
  await unmount();
  Reflect.deleteProperty(KeyInputElement.prototype, 'closest');
});
const show = async (empty = false) => {
  await Promise.all([import('./LeagueRules'), import('./LeaguePeople'), import('./LeagueLedger')]);
  const { app } = stub({
    TargetCommissionerJobs: async () => jobsWire.fresh,
    TargetCommissionerControls: async () => controlsWire,
    ...(empty ? { TargetCommissionerLeagues: async () => [] } : {}),
  });
  const host = await mount(render(LeagueLanding), app);
  await settle();
  return host;
};

describe('League landing interactions', () => {
  it('opens the searched control group and focuses its question, including an unbuilt control', async () => {
    const host = await show();
    const search = all(host, 'input').find(e => e.getAttribute('id') === 'control-search')!;
    const target = entries.find(e => e.id === 'CM-031')!;
    await type(search, target.question);
    const hit = all(host, 'button').find(b => b.getAttribute('aria-label')?.startsWith(target.name + '.'))!;
    expect(hit.textContent).toContain('Not built yet');
    await click(hit);
    await settle();
    const group = all(host, 'section').find(s => s.getAttribute('aria-label') === 'Season ops')!;
    expect(group.getAttribute('data-open')).toBe('true');
    const row = all(group, 'li').find(e => e.textContent.includes(target.question))!;
    expect(row.textContent).toContain('Not built yet');
    expect((row.ownerDocument as unknown as { focused: unknown }).focused).toBe(row);
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b/);
  });
  it('opens a rule result with only its group rules offered and the matching rule preselected', async () => {
    const host = await show();
    await act(async () => surfaceActions.run({
      screen: 'landing', action: 'open', value: 'CM-024 calendar.timezone',
    }));
    const group = all(host, 'section').find(s => s.getAttribute('aria-label') === 'Calendar')!;
    expect(group.getAttribute('data-open')).toBe('true');
    const form = all(group, 'section').find(s => s.getAttribute('aria-label') === 'Change a rule')!;
    const pick = all(form, 'select')[0];
    expect(pick.props().value).toBe('calendar.timezone');
    expect(all(pick, 'option').map(o => o.props().value).filter(Boolean))
      .not.toContain('roster.limits');
    expect(form.textContent).toContain('In force now');
    await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: 'people' }));
    await settle();
    expect(all(host, 'section').find(s => s.getAttribute('aria-label') === 'People')
      ?.getAttribute('data-open')).toBe('true');
    expect(all(host, 'section').filter(s => s.getAttribute('aria-label') === 'Change a rule'))
      .toHaveLength(0);
    expect(all(host, 'section').some(s => s.getAttribute('aria-label') === 'Give or take a role')).toBe(true);
  });
  it('offers the existing setup form under Constitution and keeps the ledger in its own group', async () => {
    const host = await show();
    await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: 'constitution' }));
    await settle();
    expect(host.textContent).toContain('Start another app league');
    expect(host.textContent).toContain('Start an app league');
    expect(host.textContent).toContain(parseRules(rulesWire).sections[0].label);
    await act(async () => surfaceActions.run({ screen: 'landing', action: 'open', value: 'CM-056' }));
    await settle();
    const ledger = all(host, 'section').find(s => s.getAttribute('aria-label') === 'Ledger')!;
    expect(ledger.getAttribute('data-open')).toBe('true');
    expect(ledger.textContent).toContain('Find changes');
    expect(leagueState.peek().rules).toBeDefined();
  });
});
