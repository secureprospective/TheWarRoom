import { afterEach, expect, it, vi } from 'vitest';
import { act } from 'react';
import { mount, rerender, flush, unmount, all, click, button } from '../commissioner/commissionerDom';
import { heldSnapshot } from '../data/lineupTestData';
import { commands } from '../commands/registry';
import WarRoom from './WarRoom';
import { placementRoute } from '../registry';
vi.mock('./warroomReads', () => ({ useWarRoomRead: () => ({}),
  nativeWarRoomHeader: () => undefined }));
vi.mock('./WarRoomTrade', () => ({ WarRoomTrade: ({ deal }: { deal: { give: string[] } }) =>
  <section aria-label="Trade desk">{deal.give.join(',')}</section> }));
vi.mock('./WarRoomDraft', () => ({ WarRoomDraft: () => <section>Native draft destination</section> }));
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });
it('opens the requested draft tab and switches back to the desk', async () => {
  act(() => { commands.dispatch('nav.open', { node: 'war', workspace: 'draft-room' }); });
  const host = await mount(<WarRoom snapshot={heldSnapshot()} native={{ league: 'local',
    mine: '0025', embedded: true, leave: () => {} }} />); await flush();
  expect(host.textContent).toContain('Native draft destination');
  await click(button(host, 'TRADE DESK')); await flush();
  expect(all(host, 'section').find(e => e.getAttribute('aria-label') === 'Trade desk')).toBeDefined();
  expect(commands.read().workspace.war).toBe('trade-desk');
});
it('loads the handed-off player into the existing deal once, scoped to league and franchise', async () => {
  act(() => { commands.dispatch('nav.open', { node: 'war', workspace: 'trade-desk',
    trade: { mine: '0025', player: '0042', league: 'local' } }); });
  const host = await mount(<WarRoom snapshot={heldSnapshot()} native={{ league: 'local',
    mine: '0025', embedded: true, leave: () => {} }} />); await flush();
  expect(all(host, 'section').find(e => e.getAttribute('aria-label') === 'Trade desk')?.textContent).toBe('0042');
  await rerender(<WarRoom snapshot={heldSnapshot()} native={{ league: 'another',
    mine: '0025', embedded: true, leave: () => {} }} />); await flush();
  expect(all(host, 'section').find(e => e.getAttribute('aria-label') === 'Trade desk')?.textContent).toBe('');
});
it('repoints command-bar endpoint placements without asserting report parity', () => {
  expect(placementRoute('Trade Floor › Trade desk')).toEqual({ node: 'war', workspace: 'trade-desk' });
  expect(placementRoute('Trade Floor › Draft room')).toEqual({ node: 'war', workspace: 'draft-room' });
});
