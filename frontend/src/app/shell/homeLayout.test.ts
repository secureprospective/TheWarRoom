import { expect, it } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import { bump, createHomeLayoutStore, defaultLayout, DEFAULTS, pack, parseLayout, place, swap,
  type Layout } from './homeLayout';

it('packs both phase defaults into the 3 × 2 grid', () => {
  expect(pack(DEFAULTS.in)).toHaveLength(5);
  expect(pack(DEFAULTS.off)).toHaveLength(5);
});
it('a full grid asks for a tile to bump, and the bumped tile lands in overflow', () => {
  expect(place(defaultLayout(), 'in', 'cap', '1x1')).toBeUndefined();
  const next = bump(defaultLayout(), 'in', { id: 'cap', span: '1x1' }, 'race')!;
  expect(next.over).toEqual(['race']);
  expect(next.tiles.map(t => t[0])).toContain('cap');
});
it('reorders phone boards using the same validated layout rules', () => {
  const reordered = swap(defaultLayout(), 'in', 'mu', 'news');
  expect(reordered?.tiles.slice(0, 2).map(([id]) => id)).toEqual(['news', 'mu']);
  expect(pack(reordered?.tiles ?? [])).not.toBeNull();
});

it('a saved layout that no longer packs falls back to the default', () => {
  expect(parseLayout(JSON.stringify({ mode: 'custom', tiles: [['news', '1x2'], ['wire', '1x2'],
    ['mu', '1x2'], ['race', '1x1']], over: [], pins: {} })).mode).toBe('default');
  expect(parseLayout('not json').mode).toBe('default');
});

it('a new device reads the account layout saved by the first device', async () => {
  let layout = defaultLayout();
  let revision = 0;
  const port: HostedPort = async request => {
    if (request.command === 'home.layout.read')
      return { label: 'Account home layout', writable: true, football: '',
        data: { layout, revision } };
    if (request.command === 'home.layout.set') {
      const args = request.arguments as { layout: Layout; expectedRevision: number };
      if (args.expectedRevision !== revision)
        return { label: 'Account home layout', writable: true, football: '',
          data: { layout, revision, conflict: true } };
      layout = args.layout;
      revision++;
      return { label: 'Account home layout', writable: true, football: '', data: { layout, revision } };
    }
    throw new Error('unexpected command');
  };
  const deviceOne = createHomeLayoutStore(port);
  await deviceOne.read();
  const saved: Layout = { mode: 'custom', tiles: [['cap', '1x1']], over: [], pins: { cap: true } };
  expect(await deviceOne.write(saved)).toMatchObject({ layout: saved, revision: 1, offline: false });

  const deviceTwo = createHomeLayoutStore(port);
  expect(await deviceTwo.read()).toMatchObject({ layout: saved, revision: 1, offline: false });
});

it('keeps an in-memory default while offline and adopts the server copy on conflict', async () => {
  const offline = createHomeLayoutStore(async () => { throw new Error('offline'); });
  expect(await offline.read()).toMatchObject({ layout: defaultLayout(), offline: true });
  const offlineEdit: Layout = { mode: 'custom', tiles: [['wire', '1x1']], over: [], pins: {} };
  expect(await offline.write(offlineEdit)).toMatchObject({ layout: offlineEdit, offline: true });

  const saved: Layout = { mode: 'custom', tiles: [['cap', '1x1']], over: [], pins: {} };
  let remote: Layout = defaultLayout();
  let revision = 2;
  const port: HostedPort = async request => {
    if (request.command === 'home.layout.read')
      return { label: 'Account home layout', writable: true, football: '',
        data: { layout: remote, revision } };
    if (request.command === 'home.layout.set')
      return { label: 'Account home layout', writable: true, football: '',
        data: { layout: remote, revision, conflict: true } };
    throw new Error('unexpected command');
  };
  const device = createHomeLayoutStore(port);
  await device.read();
  const result = await device.write(saved);
  expect(result).toMatchObject({ layout: remote, revision: 2, conflict: true, offline: false });
});
