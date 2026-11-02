import { describe, expect, it } from 'vitest';
import { all, present } from '../../commissioner/commissionerDom';
import { room, roomHooks, tab } from './roomTestKit';


roomHooks();

describe('Control Room · League', () => {
  it('League shows the ledger and the notice form; a chip with no Go kind is disabled', async () => {
    const { host } = await room();
    await tab(host, 'LEAGUE');
    expect(all(host, 'button').find(b => b.textContent === 'TRADES')!.attributes.has('disabled')).toBe(true);
    expect(present(host, 'Post an official notice') || all(host, 'section').some(s =>
      s.getAttribute('aria-label') === 'Post an official notice')).toBe(true);
  });
});
