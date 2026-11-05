import { afterEach, expect, it, vi } from 'vitest';
import { flush, mount, render, unmount } from '../commissioner/commissionerDom';
import arc from '../data/age-arc.wire.testdata.json';
import { heldSnapshot } from '../data/lineupTestData';
import { WarRoomInspector } from './WarRoomInspector';
import { HQInspector } from './HQInspector';
import type { HostedPort } from '../commissioner/hostedRoom';

afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

it('reads the same held player arc for an app War Room without mirror ownership or notes', async () => {
  const read = vi.fn(async () => arc);
  const port = vi.fn<HostedPort>(async () => { throw new Error('No quote'); });
  const host = await mount(render(WarRoomInspector, { context: { snapshot: heldSnapshot(),
    mine: 'franchise.01', week: 0, native: { league: 'native.any', mine: 'franchise.01',
      port, leave: () => {} } }, asset: { token: '0001', name: 'Held player', position: 'WR' } }), {}, {
    market: { API: { TargetAgeArc: read } },
  });
  await flush();
  expect(read).toHaveBeenCalledWith('0001');
  expect(host.textContent).toContain(`age ${Math.floor(arc.points[0].age)} → ${Math.floor(arc.points.at(-1)!.age)}`);
  expect(host.textContent).toContain('Held player');
  expect(port.mock.calls.every(([r]) => ['contract.quote', 'value.read'].includes(r.command))).toBe(true);
});

it('shows held age arc and app projection in the HQ inspector', async () => {
  const snapshot = heldSnapshot(), id = snapshot.players.value[0].id;
  const read = vi.fn(async () => arc);
  const host = await mount(render(HQInspector, { native: true, snapshot, mine: 'franchise.01', id,
    projection: 12.75 }), {}, { market: { API: { TargetAgeArc: read } } });
  await flush();
  expect(read).toHaveBeenCalledWith(id);
  expect(host.textContent).toContain('12.8');
  expect(host.textContent).toContain(`${arc.peakStart}–${arc.peakEnd}`);
});

it('keeps the app projection when a held arc is absent', async () => {
  const snapshot = heldSnapshot(), id = snapshot.players.value[0].id;
  const host = await mount(render(HQInspector, { native: true, snapshot, mine: 'franchise.01', id,
    projection: 0 }), {}, { market: { API: { TargetAgeArc: async () => {
      throw new Error('No model score');
    } } } });
  await flush();
  expect(host.textContent).toContain('0.0');
  expect(host.textContent).toContain('Not available yet');
  expect(host.textContent).not.toContain('No model score');
});
