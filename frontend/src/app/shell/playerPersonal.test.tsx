import { afterEach, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, button, click, flush, mount, render, type, unmount } from
  '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';
import { heldSnapshot } from '../data/lineupTestData';
import { HQInspector } from './HQInspector';
import { setHostedPlayerWatch } from './playerPersonal';
import { WarRoomInspector } from './WarRoomInspector';
import type { WarRoomContext } from './warroomReads';

afterEach(async () => { await unmount(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });

it('saves hosted private notes and watch stars from HQ and the War Room inspector', async () => {
  const league = 'native.test';
  const snapshot = heldSnapshot();
  const player = snapshot.players.value[0].id;
  let notes: { kind: string; entity: string; id: string; text: string; at: string; order: number }[] = [];
  let watched = false;
  const port = vi.fn<HostedPort>(async request => {
    const args = request.arguments as { player?: string; text?: string; at?: string; watched?: boolean };
    if (request.command === 'player.note.read') return { label: '', writable: true, football: 'pin', data: {
      league, player, notes, count: notes.length, watched,
    } };
    if (request.command === 'player.note.set') {
      notes = args.at
        ? notes.filter(note => note.at !== args.at)
        : [{
          kind: 'note', entity: 'player', id: player, text: args.text ?? '',
          at: '2026-11-17T15:00:00Z', order: 1,
        }];
      return { label: '', writable: true, football: 'pin', data: { league, player } };
    }
    if (request.command === 'player.watch.set') {
      watched = args.watched ?? false;
      return { label: '', writable: true, football: 'pin', data: { league, player } };
    }
    throw new Error(`Unexpected hosted command ${request.command}`);
  });
  vi.stubGlobal('fetch', vi.fn(async () => { throw new Error('Unconfigured hosted read'); }));

  let host = await mount(render(HQInspector, {
    native: true, league, port, snapshot, mine: 'franchise.01', id: player,
  }));
  await flush();
  expect(button(host, 'WATCH').attributes.has('disabled')).toBe(false);
  await type(all(host, 'textarea')[0], 'HQ private note');
  await act(async () => {
    const submit = all(host, 'form')[0].props().onSubmit as (event: { preventDefault(): void }) => void;
    submit({ preventDefault() {} });
  });
  await flush();
  expect(host.textContent).toContain('HQ private note');
  await click(button(host, 'WATCH'));
  await flush();
  expect(host.textContent).toContain('WATCHED');
  expect(port.mock.calls.some(([request]) => request.command === 'player.note.set' &&
    request.mode === 'execute')).toBe(true);
  expect(port.mock.calls.some(([request]) => request.command === 'player.watch.set' &&
    request.mode === 'execute')).toBe(true);
  await unmount();

  notes = [];
  watched = false;
  const context: WarRoomContext = { snapshot, mine: 'franchise.01', week: 1,
    native: { league, mine: 'franchise.01', owner: true, port, leave: () => {} } };
  host = await mount(render(WarRoomInspector, { context, asset: { token: player, name: 'Hosted Player' },
    watch: current => setHostedPlayerWatch(league, player, !current, port) }));
  await flush();
  const notesDrawer = all(host, 'button').find(item => item.textContent.startsWith('NOTES'));
  if (!notesDrawer) throw new Error('Notes drawer unavailable');
  await click(notesDrawer);
  await type(all(host, 'textarea')[0], 'War Room private note');
  await act(async () => {
    const submit = all(host, 'form')[0].props().onSubmit as (event: { preventDefault(): void }) => void;
    submit({ preventDefault() {} });
  });
  await flush();
  expect(host.textContent).toContain('War Room private note');
  expect(button(host, 'WATCH').attributes.has('disabled')).toBe(false);
  await click(button(host, 'WATCH'));
  await flush();
  expect(host.textContent).toContain('WATCHED');
  const noteRequests = port.mock.calls.filter(([request]) => request.command === 'player.note.set');
  const watchRequests = port.mock.calls.filter(([request]) => request.command === 'player.watch.set');
  expect(noteRequests.length).toBeGreaterThan(1);
  expect(watchRequests.length).toBeGreaterThan(1);
});
