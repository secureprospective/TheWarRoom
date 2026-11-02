import { describe, expect, it } from 'vitest';
import { button, click, present } from '../../commissioner/commissionerDom';
import { region, room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · Server', () => {
  it('Server says what the app does not hold and runs Back up now', async () => {
    const { host } = await room();
    await tab(host, 'SERVER');
    expect(present(host, 'Not available yet')).toBe(true);
    await click(button(host, 'BACK UP NOW'));
    expect(region(host, 'Back up now')).toBeTruthy();
  });
});
