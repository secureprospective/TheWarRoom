import { afterEach, expect, it, vi } from 'vitest';
import { act } from 'react';
import { mount, rerender, flush, unmount, all, click, button } from '../commissioner/commissionerDom';
import { heldSnapshot } from '../data/lineupTestData';
import { HQAct } from './HQAct';
import { FixtureProvider } from '../data/provider';
import { commands } from '../commands/registry';
import WarRoom from './WarRoom';
import { placementRoute } from '../registry';
vi.mock('./warroomReads', () => ({ useWarRoomRead: () => ({}),
  nativeWarRoomHeader: () => undefined }));
vi.mock('./WarRoomTrade', () => ({ WarRoomTrade: ({ deal }: { deal: { give: string[] } }) =>
  <section aria-label="Trade desk">{deal.give.join(',')}</section> }));
vi.mock('./WarRoomDraft', () => ({ WarRoomDraft: () => <section>Native draft destination</section> }));
vi.mock('./WarRoomDrawers', () => ({ WarRoomDrawers: ({ openOffer }: {
  openOffer: (id: string) => void }) => <HQAct onAction={() => openOffer('901')}>Open offer</HQAct> }));
vi.mock('./EntityInspector', () => ({ default: ({ entity }: { entity: { id: string } }) =>
  <p>Offer inspector {entity.id}</p> }));
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });
it('opens the requested draft tab and switches back to the desk', async () => {
  act(() => { commands.dispatch('nav.open', { node: 'war', workspace: 'draft-room' }); });
  const host = await mount(<WarRoom snapshot={heldSnapshot()} native={{ league: 'local',
    mine: '0025', embedded: true, leave: () => {} }} />); await flush();
  expect(host.textContent).toContain('Native draft destination');
  await click(button(host, 'TRADE DESK')); await flush();
  expect(all(host, 'section').find(e => e.getAttribute('aria-label') === 'Trade desk')).toBeDefined();
  const read = commands.read;
  expect(read().workspace.war).toBe('trade-desk');
});
it('loads the handed-off player into the existing deal once, scoped to league and franchise', async () => {
  act(() => { commands.dispatch('nav.open', { node: 'war', workspace: 'trade-desk',
    trade: { mine: '0025', player: '0042', league: 'local' } }); });
  const host = await mount(<WarRoom snapshot={heldSnapshot()} native={{ league: 'local',
    mine: '0025', embedded: true, leave: () => {} }} />); await flush();
  const desk = () => all(host, 'section').find(e => e.getAttribute('aria-label') === 'Trade desk');
  expect(desk()?.textContent).toBe('0042');
  await rerender(<WarRoom snapshot={heldSnapshot()} native={{ league: 'another',
    mine: '0025', embedded: true, leave: () => {} }} />); await flush();
  expect(desk()?.textContent).toBe('');
});
it('repoints command-bar endpoint placements without asserting report parity', () => {
  expect(placementRoute('War Room › Trade desk')).toEqual({ node: 'war', workspace: 'trade-desk' });
  expect(placementRoute('War Room › Draft room')).toEqual({ node: 'war', workspace: 'draft-room' });
});

it('opens a negotiation in the War Room inspector and closes back to the player inspector', async () => {
  const snapshot = await new FixtureProvider().snapshot();
  act(() => {
    commands.loadSnapshot(snapshot);
    commands.dispatch('franchise.set', { franchiseId: snapshot.franchises.value[0].id });
    commands.dispatch('nav.open', { node: 'war', workspace: 'trade-desk' });
  });
  const host = await mount(<WarRoom snapshot={snapshot} />);
  await click(button(host, 'NEGOTIATIONS'));
  await click(button(host, 'Open offer'));
  expect(host.textContent).toContain('Offer inspector 901');
  await click(button(host, 'Close offer'));
  expect(host.textContent).not.toContain('Offer inspector 901');
});
