import { afterEach, expect, it, vi } from 'vitest';
import { addHQTrade } from './hqReads';
import { heldSnapshot } from '../data/lineupTestData';
import { commands } from '../commands/registry';
afterEach(() => { vi.restoreAllMocks(); });
it.each([undefined, 'native.league'])('hands a held player to War Room without the old builder (%s)',
  async league => {
  const snapshot = heldSnapshot();
  const mine = '0025', player = snapshot.rosters.value[0].players[0].id;
  const navigate = vi.spyOn(commands, 'dispatch').mockImplementation(() => undefined);
  await addHQTrade(mine, player, snapshot, league);
  expect(navigate).toHaveBeenCalledWith('nav.open', { node: 'war', workspace: 'trade-desk',
    trade: { mine, player, league } });
});
it('does not navigate when the player is not held by the selected franchise', async () => {
  const navigate = vi.spyOn(commands, 'dispatch');
  await expect(addHQTrade('other', '0042', heldSnapshot())).rejects.toThrow('held trade roster');
  expect(navigate).not.toHaveBeenCalled();
});
