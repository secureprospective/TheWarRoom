import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { act } from 'react';
import { button, control, disabled, flush, mount, present, render, settle, type, unmount }
  from './commissionerDom';
import { open, receipt, stub } from './commissionerStub';
import { leagueState } from './proxy';
import { StartLeagueForm } from './StartLeagueForm';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const start = 'Start this league';
describe('Start an app league', () => {
  it('cannot be confirmed without a typed reason, then sends one create with preset and key', async () => {
    const { app, calls } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await type(control(host, 'League name'), 'Our league');
    expect(disabled(button(host, start))).toBe(true);
    expect(present(host, 'Needs a typed reason')).toBe(true);
    await type(control(host, 'Reason'), '   ');
    expect(disabled(button(host, start))).toBe(true);
    await type(control(host, 'Reason'), 'Owners voted to move the league here');
    expect(disabled(button(host, start))).toBe(false);
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent).toHaveLength(1);
    const [sent] = calls.sent;
    expect(sent).toMatchObject({ command: 'commish.league.create',
      reason: 'Owners voted to move the league here', expected: 0, effective: 'now', visibility: 'league',
      args: { name: 'Our league', preset: 'legacy_nfl', franchiseCount: 32 } });
    expect(sent.league).toMatch(/^native\.[0-9a-f]{16}$/);
    expect(sent.request).toMatch(/^[0-9a-f]{32}$/);
  });
  it('offers an empty league of any size and sends that preset', async () => {
    const { app, calls } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await type(control(host, 'League name'), 'Small league');
    await type(control(host, 'Starting rules'), 'empty');
    await type(control(host, 'Number of franchises'), '12');
    await type(control(host, 'Reason'), 'A new league');
    expect(present(host, 'Start empty: every rule Needs a ruling')).toBe(true);
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0].args).toEqual({ name: 'Small league', preset: 'empty', franchiseCount: 12 });
  });
  it('selects the new league on the receipt and reloads the ledger, rules and people for it', async () => {
    const created = receipt({ league: 'native.created', entry: 'act.created' });
    const { app, calls } = stub({
      TargetCommissionerExecute: async () => created,
      TargetCommissionerLeagues: async () => [{ id: 'native.rules', name: 'Rules league' },
        { id: 'native.created', name: 'Our league' }],
    });
    const host = await mount(render(StartLeagueForm), app);
    await open();
    calls.ledger.length = calls.rules.length = calls.people.length = 0;
    await type(control(host, 'League name'), 'Our league');
    await type(control(host, 'Reason'), 'Start');
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(leagueState.peek().selected).toBe('native.created');
    expect([calls.ledger, calls.rules, calls.people]).toEqual(Array(3).fill(['native.created']));
    expect(present(host, 'Recorded in the league ledger.')).toBe(true);
    expect(control(host, 'League name').props().value).toBe('');
  });
  it('shows Go’s refusal as written and changes nothing else', async () => {
    const words = 'The Legacy NFL preset is for 32 franchises';
    const { app, calls } = stub({ TargetCommissionerExecute: async () => { throw new Error(words); } });
    const host = await mount(render(StartLeagueForm), app);
    await open();
    calls.ledger.length = calls.rules.length = calls.people.length = 0;
    await type(control(host, 'League name'), 'Our league');
    await type(control(host, 'Number of franchises'), '12');
    await type(control(host, 'Reason'), 'Start');
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(host.textContent).toContain(words);
    expect([calls.ledger, calls.rules, calls.people]).toEqual([[], [], []]);
    expect(leagueState.peek().selected).toBe('native.rules');
    expect(control(host, 'League name').props().value).toBe('Our league');
    await flush();
  });
});
