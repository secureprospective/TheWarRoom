import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { all, control, mount, present, render, settle, unmount } from './commissionerDom';
import { ownerPeople, stub } from './commissionerStub';
import LeagueDesk from './LeagueDesk';
import { leagueState } from './proxy';
import peopleWire from './people.wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const headings = (host: Parameters<typeof all>[0]) => all(host, 'h5').map(h => h.textContent);
const coCommissioner = { ...peopleWire.commissioner, yourRole: 'Co-commissioner', mayGrantRoles: false };

describe('The commissioner desk', () => {
  it('gives the commissioner all three forms and the one league picker', async () => {
    const host = await mount(render(LeagueDesk), stub().app);
    await settle();
    expect(headings(host)).toEqual(['Start an app league', 'Change a rule', 'Give or take a role']);
    expect(all(host, 'option').some(o => o.textContent === 'Rules league')).toBe(true);
    expect(control(host, 'App league').props().value).toBe('native.rules');
  });
  it('leaves out the role form for a co-commissioner, as the people reading says', async () => {
    const host = await mount(render(LeagueDesk), stub({}, { people: coCommissioner }).app);
    await settle();
    expect(headings(host)).toEqual(['Start an app league', 'Change a rule']);
  });
  it('shows an owner only the way to start their own league', async () => {
    const host = await mount(render(LeagueDesk), stub({}, { people: peopleWire.owner }).app);
    await settle();
    expect(headings(host)).toEqual(['Start an app league']);
    expect(ownerPeople.mayChangeRules).toBe(false);
    expect(present(host, 'Change a rule')).toBe(false);
  });
});
