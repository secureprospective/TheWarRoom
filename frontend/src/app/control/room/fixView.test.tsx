import { describe, expect, it } from 'vitest';
import { all, click, present, settle } from '../../commissioner/commissionerDom';
import { region, room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · Fix', () => {
  it('Fix lists exceptions in force with names, and a fix opens its form', async () => {
    const { host } = await room();
    await tab(host, 'FIX');
    expect(present(host, 'Exception: roster over the maximumHeld Bison')).toBe(true);
    await click(all(host, 'button').find(b => b.getAttribute('data-fix') === '1')!);
    await settle();
    expect(region(host, 'Adjust a player score')).toBeTruthy();
  });
});
