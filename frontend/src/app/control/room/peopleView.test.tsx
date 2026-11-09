import { describe, expect, it } from 'vitest';
import { button, click, present } from '../../commissioner/commissionerDom';
import { region, room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · People', () => {
  it('People lists the seats; GRANT ROLE opens the role form', async () => {
    const { host } = await room();
    await tab(host, 'PEOPLE');
    expect(present(host, 'Franchise 1')).toBe(true);
    await click(button(host, 'GRANT ROLE'));
    expect(region(host, 'Give or take a role')).toBeTruthy();
  });
});
