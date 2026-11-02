import { describe, expect, it } from 'vitest';
import { button, click } from '../../commissioner/commissionerDom';
import { actChange } from '../../commissioner/actBody';
import { actTable } from '../../commissioner/actTable';
import { region, room, roomHooks, tab } from './roomTestKit';

roomHooks();
const held = { league: 'L', revision: 4, principal: 'creator' };
const def = (command: string) => actTable.find(a => a.command === command)!;

describe('Control Room · deputy order and O-Line close forms', () => {
  it('commish.deputy.order sends the ordered co-commissioners under people, privately', () => {
    const change = actChange(def('commish.deputy.order'), { order: ['b', 'a'] }, held, 'swap');
    expect(change).toMatchObject({ command: 'commish.deputy.order', args: { people: { order: ['b', 'a'] } },
      visibility: 'private', expected: 4 });
    expect(actChange(def('commish.deputy.order'), { order: [] }, held, 'x')).toBeNull();
  });
  it('oline.auction.close sends no arguments (Go supplies the outcome)', () => {
    expect(actChange(def('oline.auction.close'), {}, held, 'close')).toMatchObject({
      command: 'oline.auction.close', args: {}, visibility: 'league' });
  });
  it('DEPUTY ORDER and CLOSE O-LINE AUCTION open their forms', async () => {
    const { host } = await room();
    await tab(host, 'PEOPLE');
    await click(button(host, 'DEPUTY ORDER'));
    expect(region(host, 'Set the co-commissioner order')).toBeTruthy();
    await tab(host, 'SEASON');
    await click(button(host, 'CLOSE O-LINE AUCTION'));
    expect(region(host, 'Close the O-Line auction')).toBeTruthy();
  });
});
