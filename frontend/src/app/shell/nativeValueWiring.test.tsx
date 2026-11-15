import { afterEach, expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { heldSnapshot } from '../data/lineupTestData';
import { parseNativeValues } from '../data/nativeValues';
import { mount, render, flush, unmount } from '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';
import { WarRoomHeader } from './WarRoomHeader';
import { HQInspector } from './HQInspector';
import { WarRoomInspector } from './WarRoomInspector';
import { readInspectorFit } from './inspectorFit';
import { readWarRoomHeader, type WarRoomContext } from './warroomReads';

const player = () => ({ player: '0042', franchise: 'other', position: 'WR', nowPPG: 0,
  dynastyPPG: 12.3, age: 25, ageKnown: true, available: true, words: 'Held model inputs' });
const wire = (franchise = '') => ({ league: 'native.test', season: 2026, franchise, players: [player()],
  best: { franchise, starterN: 21, value: 321.9, age: 25, ageKnown: true, rank: 2, of: 32,
    available: true, words: '21 of 21 players valued' },
  age: [{ from: 24, to: 25, open: false, value: 12.3, players: 1 }], notes: [] });
const side = (franchise: string) => ({ franchise, before: [], after: [], perWeek: 2.4, thisWeek: 1.2,
  changes: [{ kind: 'benched', player: '0043', position: 'WR' }], unprojected: [] });
const delta = (league: string) => ({ league, season: 2026, week: 9, you: side('mine'), them: side('other'),
  notes: ['Held projections'] });
function context(port?: HostedPort): WarRoomContext {
  const snapshot = heldSnapshot(); snapshot.league.value.season = 2026;
  snapshot.franchises.value = [{ id: 'mine' }, { id: 'other' }];
  snapshot.players.value = [{ id: '0042', name: 'Incoming receiver' }, { id: '0043',
    name: 'Bench receiver' }];
  const contract = { salary: 100, contractStatus: 'UFA' as const, rosterStatus: 'ROSTER' as const };
  snapshot.rosters.value = [{ franchiseId: 'mine', players: [{ ...contract, id: '0043' }] },
    { franchiseId: 'other', players: [{ ...contract, id: '0042' }] }];
  return { snapshot, mine: 'mine', week: 9, native: port ? { league: 'native.test', mine: 'mine', port,
    leave: () => {} } : undefined };
}
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });
it('reuses the mounted native header and cap runway without a second read', async () => {
  const port = vi.fn<HostedPort>(async () => { throw new Error('Must not read twice'); });
  const c = context(port);
  c.native!.reading = { revision: 7, snapshot: c.snapshot, settings: { positions: [], starters: 21 },
    values: parseNativeValues(wire(), 'native.test', '', 2026), errors: [],
    runway: [{ season: 2026, max: 12500000000, room: 108000000 },
      { season: 2027, max: 12500000000, room: 4558000000 },
      { season: 2028, max: 12500000000, room: 6994000000 }] };
  const header = await readWarRoomHeader(c);
  expect(port).not.toHaveBeenCalled();
  expect(header.best).toBe('321.9');
  expect(header.runway?.map(y => y.room)).toEqual([108000000, 4558000000, 6994000000]);
  const html = renderToStaticMarkup(<WarRoomHeader reading={header} />);
  for (const amount of ['$1.08M', '$45.58M', '$69.94M']) expect(html).toContain(amount);
});
it('checks league/owner/season, finite values, duplicates and known zero independently of absence', () => {
  expect(parseNativeValues(wire(), 'native.test', '', 2026).players[0].nowPPG).toBe(0);
  for (const broken of [{ ...wire(), league: 'wrong' }, { ...wire(), season: 2025 },
    { ...wire(), players: [player(), player()] }, { ...wire(), players: [{ ...player(), nowPPG: NaN }] },
    { ...wire(), players: [{ ...player(), available: 'true' }] }])
    expect(() => parseNativeValues(broken, 'native.test', '', 2026)).toThrow();
  expect(() => parseNativeValues(wire('mine'), 'native.test', 'mine', 2026)).toThrow('wrong owner');
  const missing = { ...wire(), players: [{ ...player(), available: false }] };
  expect(parseNativeValues(missing, 'native.test', '', 2026).players[0].available).toBe(false);
});
it('renders league Best N, dynasty age amounts and the native card, including known zero', () => {
  const values = parseNativeValues(wire(), 'native.test', '', 2026);
  const header = renderToStaticMarkup(<WarRoomHeader reading={{ starterN: values.best.starterN,
    best: values.best.value.toFixed(1), rank: '2 of 32', age: values.age, errors: [] }} />);
  for (const text of ['Best 21', '321.9', '2 of 32', 'Age 24', '12.3']) expect(header).toContain(text);
  const html = renderToStaticMarkup(<HQInspector native snapshot={context().snapshot} mine="mine"
    id="0042" leagueValue={player()} />);
  expect(html).toContain('>0.0</b>'); expect(html).toContain('>12.3</b>');
  const absent = renderToStaticMarkup(<HQInspector native snapshot={context().snapshot} mine="mine"
    id="0042" leagueValue={{ ...player(), available: false }} />);
  expect(absent).not.toContain('>0.0</b>'); expect(absent).not.toContain('>12.3</b>');
});
it('shares the app/mirror lineup read, fences scope and renders the inspector fit', async () => {
  const port = vi.fn<HostedPort>(async r => {
    if (r.command !== 'value.read' && r.command !== 'lineupdelta.read') throw new Error('Not held');
    return { label: 'Local', writable: false, football: '', data: r.command === 'value.read' ? wire() :
      delta('native.test') };
  });
  const app = await readInspectorFit(context(port), '0042');
  expect(app.side.perWeek).toBe(2.4);
  expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'lineupdelta.read',
    arguments: expect.objectContaining({ get: ['0042'], give: [], partner: 'other' }) }));
  const mirror = vi.fn(async () => delta(''));
  const host = await mount(render(WarRoomInspector, { context: context(port), asset: {
    token: '0042', name: 'Incoming receiver' } }), {}, { commissioner: { API: {
      TargetNativeLineupDelta: mirror } } });
  await flush(); expect(host.textContent).toContain('Bench receiver');
  expect(host.textContent).toContain('Adds / week2.4'); expect(host.textContent).toContain('12.3');
  expect((await readInspectorFit(context(), '0042')).side.perWeek).toBe(2.4);
  expect(mirror).toHaveBeenCalledWith(expect.objectContaining({ league: '', get: ['0042'] }));
  port.mockResolvedValue({ label: 'Local', writable: false, football: '', data: delta('wrong') });
  await expect(readInspectorFit(context(port), '0042')).rejects.toThrow('another player context');
});

it('checks preparation shape and keeps unavailable values distinct from preparation', () => {
  const pending = { ...wire(), preparing: true, players: [], age: [],
    best: { ...wire().best, available: false, words: 'Values are being prepared' } };
  expect(parseNativeValues(pending, 'native.test', '', 2026).preparing).toBe(true);
  expect(() => parseNativeValues({ ...wire(), preparing: true }, 'native.test', '', 2026)).toThrow();
  expect(() => parseNativeValues({ ...pending, preparing: 'true' }, 'native.test', '', 2026)).toThrow();
  const html = renderToStaticMarkup(<WarRoomHeader reading={{ valuesPreparing: true,
    runway: [{ season: 2026, max: 12500000000, room: 108000000 }], errors: [] }} />);
  expect(html).toContain('Values are being prepared'); expect(html).toContain('$1.08M');
  expect(html).not.toContain('role="alert"');
  const inspector = renderToStaticMarkup(<HQInspector native snapshot={context().snapshot}
    mine="mine" id="0042" valuesPreparing />);
  expect(inspector).toContain('Values are being prepared');
  expect(inspector).not.toContain('>0.0</b>');
});
