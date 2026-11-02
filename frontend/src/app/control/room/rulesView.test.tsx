import { describe, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, button, click, control, flush, holder, present, type, settle } from '../../commissioner/commissionerDom';
import { receipt } from '../../commissioner/commissionerStub';
import rulesWire from '../../commissioner/rules.wire.testdata.json';
import { room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · Rules and settings congruence', () => {
  it('a rule change made here runs commish.rules.set and the room shows it pending at once', async () => {
    let landed = false;
    const pendingRules = { ...rulesWire, revision: 4, sections: rulesWire.sections.map(s => ({ ...s, rows: s.rows.map(r =>
      r.field !== 'roster.limits' ? r : { ...r, pending: [{ entry: 'act.new', value: '25 to 60', when: 'Waits for the next rollover' }] }) })) };
    const { host, calls } = await room({ TargetCommissionerExecute: async body => {
      calls.sent.push(JSON.parse(body)); landed = true; return receipt();
    } }, { rules: () => (landed ? pendingRules : rulesWire) });
    await tab(host, 'RULES');
    expect(present(host, 'NEXT ROLLOVER')).toBe(false);
    await click(all(host, 'tr').find(t => t.getAttribute('data-rule') === 'roster.limits')!);
    await click(button(host, 'CHANGE ›'));
    await type(control(host, 'Minimum'), '25');
    await type(control(host, 'Maximum'), '60');
    await settle();
    vi.useFakeTimers();
    const b = holder(host, 'Record this rule change');
    await act(async () => { (b.props().onKeyDown as (e: object) => void)({ key: 'Enter', repeat: false, preventDefault() {} }); });
    await act(async () => { vi.advanceTimersByTime(600); });
    await act(async () => { (b.props().onKeyUp as (e: object) => void)({ key: 'Enter' }); });
    await flush();
    vi.useRealTimers();
    await settle();
    expect(calls.sent.map(s => s.command)).toEqual(['commish.rules.set']);
    expect(calls.sent[0]!.args).toMatchObject({ field: 'roster.limits' });
    expect(present(host, 'NEXT ROLLOVER')).toBe(true);
  });
  it('the Season desk shows the rule values the league holds, not its own', async () => {
    const changed = { ...rulesWire, sections: rulesWire.sections.map(s => ({ ...s, rows: s.rows.map(r =>
      r.field !== 'waivers.method' ? r : { ...r, value: 'Reverse standings', state: 'Set' }) })) };
    const { host } = await room({}, { rules: changed });
    await tab(host, 'SEASON DESK');
    expect(present(host, 'PriorityReverse standings')).toBe(true);
  });
});
