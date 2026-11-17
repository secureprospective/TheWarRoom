import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { button, click, flush, mount, unmount } from '../commissioner/commissionerDom';
import type { WarRoomContext } from './warroomReads';

const reading = vi.hoisted(() => ({
  players: [
    { id: '0001', name: 'Need and affordable', position: 'WR', minimum: '$85M', minimumCents: 8500000000,
      age: '25', points: '12', value: '10' },
    { id: '0002', name: 'Affordable only', position: 'RB', minimum: '$50M', minimumCents: 5000000000,
      age: '24', points: '8', value: '7' },
    { id: '0003', name: 'Need but costly', position: 'WR', minimum: '$90M', minimumCents: 9000000000,
      age: '26', points: '4', value: '3' },
    { id: '0004', name: 'Kicker need', position: 'K', minimum: '$1M', minimumCents: 100000000,
      age: '23', points: '2', value: '2' },
  ], capRoom: 8500000000 as number | undefined, targets: [], errors: [],
}));
vi.mock('./warroomReads', () => ({ useWarRoomRead: () => ({ reading }) }));
import { WarRoomMarket } from './WarRoomMarket';

beforeEach(() => { reading.capRoom = 8500000000; });
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

const context = { mine: 'franchise.mine', snapshot: {} as WarRoomContext['snapshot'], week: 1,
  native: { league: 'native.test', mine: 'franchise.mine', owner: true, leave: () => {} } } as WarRoomContext;

it('filters hosted players by the header needs and server cap room', async () => {
  const host = await mount(<WarRoomMarket context={context} needs={[
    { position: 'WR', state: 'hole', have: 0, starts: 2 },
    { position: 'PK', state: 'thin', have: 1, starts: 2 },
    { position: 'RB', state: 'ok', have: 2, starts: 2 },
  ]} select={() => {}} />);
  await flush();
  expect(host.textContent).toContain('Need and affordable');
  expect(host.textContent).toContain('Affordable only');
  expect(host.textContent).toContain('Need but costly');
  expect(host.textContent).toContain('Kicker need');

  await click(button(host, 'My needs'));
  expect(host.textContent).toContain('Need and affordable');
  expect(host.textContent).toContain('Need but costly');
  expect(host.textContent).toContain('Kicker need');
  expect(host.textContent).not.toContain('Affordable only');

  await click(button(host, 'Affordable'));
  expect(host.textContent).toContain('Need and affordable');
  expect(host.textContent).toContain('Affordable only');
  expect(host.textContent).toContain('Kicker need');
  expect(host.textContent).not.toContain('Need but costly');
});

it('leaves My needs and Affordable disabled when their held inputs are unavailable', async () => {
  reading.capRoom = undefined;
  const host = await mount(<WarRoomMarket context={context} select={() => {}} />);
  await flush();
  expect(button(host, 'My needs').disabled).toBe(true);
  expect(button(host, 'Affordable').disabled).toBe(true);
});
