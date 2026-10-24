import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { act } from 'react';
import { all, button, control, disabled, mount, present, render, settle, type, unmount }
  from './commissionerDom';
import { commissionerPeople, open, stub } from './commissionerStub';
import { leagueState } from './proxy';
import { RoleForm } from './RoleForm';
import peopleWire from './people.wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const give = 'Give this role';
const click = (host: Parameters<typeof button>[0], label: string) =>
  act(async () => { (button(host, label).props().onClick as () => void)(); });

describe('Give or take a role', () => {
  it('lists the seated franchises only, and says role changes are private before confirming', async () => {
    const host = await mount(render(RoleForm, { people: commissionerPeople }), stub().app);
    const options = all(host, 'option').map(o => o.textContent);
    expect(options.filter(o => o.startsWith('Franchise'))).toEqual(
      ['Franchise 1 (owner-one)', 'Franchise 2 (owner-two)', 'Franchise 3 (owner-three)']);
    expect(present(host, 'Role changes are private: owners do not see them in the ledger.')).toBe(true);
    expect(all(host, 'option').map(o => o.textContent)).toContain('Trade review seat');
    expect(disabled(button(host, give))).toBe(true);
  });
  it('needs a franchise and a typed reason, then sends a private grant for its owner', async () => {
    const { app, calls } = stub();
    const host = await mount(render(RoleForm, { people: commissionerPeople }), app);
    await type(control(host, 'Franchise'), 'franchise.02');
    expect(disabled(button(host, give))).toBe(true);
    await type(control(host, 'Reason'), 'Owner two will help run the league');
    await settle();
    expect(disabled(button(host, give))).toBe(false);
    await click(host, give);
    await settle();
    expect(calls.sent).toHaveLength(1);
    expect(calls.sent[0]).toMatchObject({ command: 'commish.role.grant', league: 'native.people',
      expected: 7, effective: 'now', visibility: 'private',
      reason: 'Owner two will help run the league',
      args: { people: { principal: 'owner-two', role: 'co-commissioner' } } });
  });
  it('takes a role away through the revoke command', async () => {
    const { app, calls } = stub();
    const host = await mount(render(RoleForm, { people: commissionerPeople }), app);
    await type(control(host, 'Franchise'), 'franchise.01');
    await type(control(host, 'What to do'), 'revoke');
    await type(control(host, 'Role'), 'DOT');
    await type(control(host, 'Reason'), 'Stepping down from trade review');
    await settle();
    await click(host, 'Take this role away');
    await settle();
    expect(calls.sent[0]).toMatchObject({ command: 'commish.role.revoke',
      args: { people: { principal: 'owner-one', role: 'DOT' } } });
  });
  it('shows Go’s refusal of a duplicate co-commissioner as written, and changes nothing', async () => {
    const refusal = peopleWire.duplicateCoCommissionerRefusal;
    const { app, calls } = stub({ TargetCommissionerExecute: async () => { throw new Error(refusal); } });
    const host = await mount(render(RoleForm, { people: commissionerPeople }), app);
    await open('native.people');
    calls.ledger.length = calls.rules.length = calls.people.length = 0;
    await type(control(host, 'Franchise'), 'franchise.02');
    await type(control(host, 'Reason'), 'A second deputy');
    await settle();
    await click(host, give);
    await settle();
    expect(refusal).toContain('already holds that role');
    expect(host.textContent).toContain(refusal);
    expect([calls.ledger, calls.rules, calls.people]).toEqual([[], [], []]);
  });
});

it('explains vacant seats instead of presenting an empty franchise choice', async () => {
  const people = { ...commissionerPeople,
    seats: commissionerPeople.seats.map(s => ({ ...s, owner: 'Vacant' })) };
  const host = await mount(render(RoleForm, { people }), stub().app);
  expect(host.textContent)
    .toContain('Nobody else has joined this league yet. Roles go to owners who hold a franchise.');
  expect(host.textContent).not.toContain('Choose a franchise…');
  expect(disabled(button(host, give))).toBe(true);
  expect(control(host, 'What to do')).toBeDefined();
});
