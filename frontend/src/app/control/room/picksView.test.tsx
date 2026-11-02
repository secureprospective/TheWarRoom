import { describe, expect, it } from 'vitest';
import { all, click, present, settle } from '../../commissioner/commissionerDom';
import { region, room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · Picks', () => {
  it('Picks lists each pick by team name and marks a traded one; an unbuilt act says so', async () => {
    const { host } = await room();
    await tab(host, 'PICKS');
    expect(present(host, '1 non-standard picks in 2027')).toBe(true);
    expect(present(host, 'Held Bison')).toBe(true);
    await click(all(host, 'button').find(b => b.getAttribute('data-pickact') === '0')!);
    expect(present(host, 'Not built yet: nothing would change. (pick.grant)')).toBe(true);
    await click(all(host, 'button').find(b => b.getAttribute('data-pickact') === '1')!);
    await settle();
    expect(region(host, 'Move a pick') ?? all(host, 'div').find(d => d.getAttribute('role') === 'region')).toBeTruthy();
  });
});
