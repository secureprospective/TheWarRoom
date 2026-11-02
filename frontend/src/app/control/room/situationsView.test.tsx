import { describe, expect, it } from 'vitest';
import { all, click, settle } from '../../commissioner/commissionerDom';
import { region, room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · Situations', () => {
  it('a situation lists its steps and OPEN runs the step’s form', async () => {
    const { host } = await room();
    await tab(host, 'SITUATIONS');
    await click(all(host, 'button').find(b => b.textContent.includes('The feed has a player at the wrong position'))!);
    await click(all(host, 'button').find(b => b.getAttribute('data-step') === 'commish.player.position')!);
    await settle();
    expect(region(host, 'Correct a player position')).toBeTruthy();
  });
});
