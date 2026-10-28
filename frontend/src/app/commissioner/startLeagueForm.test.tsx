import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { act } from 'react';
import { all, button, control, disabled, flush, mount, present, render, settle, type, unmount }
  from './commissionerDom';
import { open, receipt, stub } from './commissionerStub';
import { leagueState } from './proxy';
import { StartLeagueForm } from './StartLeagueForm';
import controlsWire from './controls.wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const start = 'Start this league';
const thisYear = new Date().getFullYear();
describe('Start an app league', () => {
  it('lists exactly the presets in Go’s catalog, in order, with the chosen description', async () => {
    const { app } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await open();
    const options = all(control(host, 'Starting rules'), 'option');
    expect(options.map(o => [o.props().value, o.textContent]))
      .toEqual(controlsWire.presets.map(p => [p.id, p.label]));
    expect(present(host, controlsWire.presets[0].description)).toBe(true);
    await type(control(host, 'Starting rules'), controlsWire.presets[1].id);
    expect(present(host, controlsWire.presets[1].description)).toBe(true);
  });
  it('starts nothing while the catalog cannot be read', async () => {
    const { app, calls } = stub({ TargetCommissionerControls: async () => { throw new Error('offline'); } });
    const host = await mount(render(StartLeagueForm), app);
    // A new seat drops the controls read before it, so this read is the one that fails.
    await act(() => leagueState.changeSeat());
    await type(control(host, 'League name'), 'Our league');
    await type(control(host, 'Reason'), 'Start');
    await settle();
    expect(all(control(host, 'Starting rules'), 'option')).toHaveLength(0);
    expect(present(host, 'The starting presets could not be read.')).toBe(true);
    expect(disabled(button(host, start))).toBe(true);
    expect(calls.sent).toHaveLength(0);
  });
  it('cannot be confirmed without a typed reason, then sends one create with preset and key', async () => {
    const { app, calls } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await open();
    await type(control(host, 'League name'), 'Our league');
    await settle();
    expect(disabled(button(host, start))).toBe(true);
    expect(present(host, 'Needs a typed reason')).toBe(true);
    await type(control(host, 'Reason'), '   ');
    expect(present(host, 'Needs a typed reason')).toBe(true);
    expect(disabled(button(host, start))).toBe(true);
    await type(control(host, 'Reason'), 'Owners voted to move the league here');
    await settle();
    expect(present(host, 'Needs a typed reason')).toBe(false);
    expect(disabled(button(host, start))).toBe(false);
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent).toHaveLength(1);
    const [sent] = calls.sent;
    expect(sent).toMatchObject({ command: 'commish.league.create',
      reason: 'Owners voted to move the league here', expected: 0, effective: 'now', visibility: 'league',
      args: { name: 'Our league', preset: 'default_32', franchiseCount: 32, season: thisYear } });
    expect(sent.league).toMatch(/^native\.[0-9a-f]{16}$/);
    expect(sent.request).toMatch(/^[0-9a-f]{32}$/);
  });
  it('offers the 32-Team Default at 32 franchises only', async () => {
    const { app, calls } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await open();
    await type(control(host, 'League name'), 'Our league');
    await type(control(host, 'Reason'), 'Start');
    await type(control(host, 'Number of franchises'), '12');
    await settle();
    expect(disabled(button(host, start))).toBe(true);
    expect(present(host, 'The 32-Team Default is for exactly 32 franchises.')).toBe(true);
    await type(control(host, 'Number of franchises'), '32');
    await settle();
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0].args).toEqual({ name: 'Our league', preset: 'default_32', franchiseCount: 32,
      season: thisYear });
  });
  it('offers an empty league of any size and sends that preset', async () => {
    const { app, calls } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await open();
    await type(control(host, 'League name'), 'Small league');
    await type(control(host, 'Starting rules'), 'empty');
    await type(control(host, 'Number of franchises'), '12');
    await type(control(host, 'Reason'), 'A new league');
    await settle();
    expect(present(host, 'Start empty: every rule Needs a ruling')).toBe(true);
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0].args).toEqual({ name: 'Small league', preset: 'empty', franchiseCount: 12,
      season: thisYear });
  });
  it('sends the NFL season the commissioner types, and needs one', async () => {
    const { app, calls } = stub();
    const host = await mount(render(StartLeagueForm), app);
    await open();
    await type(control(host, 'League name'), 'Our league');
    await type(control(host, 'Reason'), 'Start');
    await type(control(host, 'First NFL season'), '');
    await settle();
    expect(disabled(button(host, start))).toBe(true);
    expect(present(host, 'Enter the NFL season the league starts in.')).toBe(true);
    await type(control(host, 'First NFL season'), '2027');
    await settle();
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0].args).toMatchObject({ season: 2027 });
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
    await settle();
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(leagueState.peek().selected).toBe('native.created');
    expect([calls.ledger, calls.rules, calls.people]).toEqual(Array(3).fill(['native.created']));
    expect(present(host, 'Recorded in the league ledger.')).toBe(true);
    expect(control(host, 'League name').props().value).toBe('');
  });
  it('shows Go’s refusal as written and changes nothing else', async () => {
    const words = 'Choose between 2 and 32 franchises';
    const { app, calls } = stub({ TargetCommissionerExecute: async () => { throw new Error(words); } });
    const host = await mount(render(StartLeagueForm), app);
    await open();
    calls.ledger.length = calls.rules.length = calls.people.length = 0;
    await type(control(host, 'League name'), 'Our league');
    await type(control(host, 'Starting rules'), 'empty');
    await type(control(host, 'Number of franchises'), '12');
    await type(control(host, 'Reason'), 'Start');
    await settle();
    await act(async () => { (button(host, start).props().onClick as () => void)(); });
    await settle();
    expect(host.textContent).toContain(words);
    expect([calls.ledger, calls.rules, calls.people]).toEqual([[], [], []]);
    expect(leagueState.peek().selected).toBe('native.rules');
    expect(control(host, 'League name').props().value).toBe('Our league');
    await flush();
  });
});
