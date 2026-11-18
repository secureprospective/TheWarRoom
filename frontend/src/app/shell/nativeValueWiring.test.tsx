import { afterEach, expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { heldSnapshot } from '../data/lineupTestData';
import { parseNativeValues } from '../data/nativeValues';
import { mount, render, flush, unmount } from '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';
import { readNativeHQ, readNativeCut } from '../commissioner/hqReading';
import { WarRoomHeader } from './WarRoomHeader';
import { HQInspector } from './HQInspector';
import { HQContracts, HQReserves } from './HQPanels';
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
  c.native!.reading = { revision: 7, snapshot: c.snapshot, searchPlayers: [],
    settings: { positions: [], starters: 21 },
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
it('renders native values, annual contracts, selected quote, calendar decisions and taxi value', () => {
  const snapshot = context().snapshot;
  const rosterPlayer = snapshot.rosters.value.find(r => r.franchiseId === 'mine')!.players[0];
  const base = wire();
  const values = parseNativeValues({ ...base, franchise: 'mine', best: { ...base.best, franchise: 'mine' },
    players: [{ ...player(), player: rosterPlayer.id, franchise: 'mine' }] }, 'native.test', 'mine', 2026);
  const contracts = renderToStaticMarkup(<HQContracts native snapshot={snapshot} mine="mine"
    values={values} annual={new Map([[rosterPlayer.id, [{ season: 2026, cents: 850000000 }]]])}
    cutCosts={new Map([[rosterPlayer.id, 120000000]])} runway={[{ season: 2026, dead: 250000000,
      committed: 850000000, max: 900000000, room: -200000000 }]}
    decisions={[{ label: 'lineup', at: '2026-11-20T18:00:00Z', live: true }]}
    select={() => {}} />);
  for (const fact of ['$8.50M', '$1.20M', '12.3', '0.0', 'lineup', 'DEAD CAP TOTAL · $2.50M'])
    expect(contracts).toContain(fact);
  const taxiSnapshot = heldSnapshot();
  const taxiPlayer = taxiSnapshot.rosters.value[0].players.find(p => p.rosterStatus === 'TAXI_SQUAD')!;
  const taxiValues = parseNativeValues({ ...base, franchise: '0025',
    best: { ...base.best, franchise: '0025' },
    players: [{ ...player(), player: taxiPlayer.id, franchise: '0025' }] }, 'native.test', '0025', 2026);
  const taxi = renderToStaticMarkup(<HQReserves native kind="taxi" snapshot={taxiSnapshot} mine="0025"
    settings={{ positions: [], taxi: 12 }} values={taxiValues} close={() => {}} />);
  expect(taxi).toContain('12.3');
  expect(taxi).not.toContain('NFL YR'); expect(taxi).not.toContain('TAXI LEFT');
});
it('renders hosted HQ readings from one fake workflow without desktop data', async () => {
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  const assets = { league: 'native.test', revision: 4, seeded: true, identities: [
    { player: '0042', franchise: 'mine', side: 'mine', name: 'Hosted receiver', position: 'WR' },
    { player: 'taxi1', franchise: 'mine', side: 'mine', name: 'Taxi receiver', position: 'WR' },
  ], pool: [], snapshot: { season: 2026, franchises: [{ id: 'mine', name: 'My team' }],
    players: [{ id: '0042', franchise: 'mine', name: 'Hosted receiver', status: 'ROSTER',
      salaryCents: 500000000, expirationYear: 2027, contractStatus: 'UFA' },
      { id: 'taxi1', franchise: 'mine', name: 'Taxi receiver', status: 'TAXI_SQUAD',
        salaryCents: 100000000, expirationYear: 2027, contractStatus: 'RFA' }], picks: [], pool: [] } };
  const values = { ...wire('mine'), league: 'native.test', season: 2026, franchise: 'mine',
    best: { ...wire('mine').best, franchise: 'mine' }, players: [
      { ...player(), player: '0042', franchise: 'mine', nowPPG: 8.7, dynastyPPG: 18.4 },
      { ...player(), player: 'taxi1', franchise: 'mine', nowPPG: 4.2, dynastyPPG: 7.6 },
    ] };
  const cap = { league: 'native.test', revision: 4, corrections: [], name: 'Test', rosterSeason: 2026,
    rulesSeason: 2026, seeded: true, source: '', counting: {}, rules: {}, franchises: [{ franchise: 'mine',
      exception: '', capCents: 10000000000, salaryCents: 0, activeCents: 0, irCents: 0, taxiCents: 0,
      adjustmentCents: 0, committedCents: 500000000, roomCents: 9000000000, quantumCents: 1, mode: '',
      deadline: '', words: '', enforcement: '', name: 'Mine', abbreviation: 'M', charges: [],
      runway: [{ season: 2026, committedCents: 500000000, deadCents: 300000000,
        capCents: 10000000000, roomCents: 9000000000 }],
      contracts: [{ player: '0042', years: [{ season: 2026, cents: 500000000 }] },
        { player: 'taxi1', years: [{ season: 2026, cents: 100000000 }] }] }] };
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'results.read') {
      const week = Number(request.arguments.week);
      return { label: 'Fake hosted workflow', writable: false, football: '', data: {
        league: 'native.test', season: 2026, week, revision: 4, published: true, words: `Week ${week}`,
        games: [{ week, home: 'mine', away: 'other', homeName: 'Mine', awayName: 'Other',
          status: 'Final', winner: week === 1 ? 'mine' : 'other', words: '', why: [] }],
        missed: [], scores: [],
      } };
    }
    const data: Record<string, unknown> = {
      'native.assets.read': assets, 'value.read': values, 'native.cap.check': cap,
      'native.reserve.check': { league: 'native.test', name: 'Test', revision: 4, rosterSeason: 2026,
        rulesSeason: 2026, seeded: true, source: 'Held player facts', rules: [], corrections: [],
        franchises: [{ franchise: 'mine', name: 'Mine', abbreviation: 'M',
          ir: { used: 0, slots: 4, judged: true, words: 'IR 0 of 4' },
          taxi: { used: 1, slots: 4, judged: true, words: 'Taxi 1 of 4' }, players: [{ player: 'taxi1',
            name: 'Taxi receiver', squad: 'Taxi', standing: 'stays', words: 'May stay',
            experienceYears: 2, taxiYearsLeft: 1 }] }] },
      'commish.rules.read': {}, 'moves.read': {},
      'standings.read': { league: 'native.test', name: 'Test', season: 2026, revision: 4, status: 'Ready',
        words: 'Two weeks have results', through: 2, decided: 2, pending: 0, criteria: [], noData: [],
        computed: [] },
      'contract.decisions.read': { league: 'native.test', franchise: 'mine', revision: 4, season: 2026,
        words: 'No priceable owner contract decisions are due.', players: [] },
      'week.read': { schedule: { league: 'native.test', franchise: 'mine', season: 2026, revision: 4,
        message: '', games: [{ week: 1, venue: 'Home', opponent: 'other' },
          { week: 2, venue: 'Away', opponent: 'other' }] }, current: null, nextWeek: null, message: '' },
      'calendar.read': { league: 'native.test', revision: 4, clock: { windows: [{ id: 'lineup',
        opens: ['lineup'], start: '2026-11-20T18:00:00Z', close: null, live: true,
        secondsToClose: null }], exceptions: [], tradeDeadline: null, secondsToTradeDeadline: null },
        decision: { open: true, reason: '', window: 'lineup', next: null } },
      'contract.quote': { league: 'native.test', revision: 4, season: 2026, player: '0042', name: '',
        franchise: 'mine', action: 'cut', actionName: '', allowed: true, reason: '', needs: [], years: 0,
        salaryCents: 0, salaries: [], deadMoney: [{ season: 2026, cents: 120000000 }], reliefCents: 0,
        cap: { beforeCents: 9000000000, afterCents: 9120000000, changeCents: 120000000,
          roomAfterCents: 9120000000, words: '' }, basis: '', notes: [], source: '' },
    };
    if (!data[request.command]) throw new Error('Unexpected workflow command ' + request.command);
    return { label: 'Fake hosted workflow', writable: false, football: '', data: data[request.command] };
  });
  const reading = await readNativeHQ('native.test', 'mine', port, undefined, true, true);
  expect(reading.playerForm?.['0042']).toEqual([]);
  expect(port.mock.calls.flatMap(([r]) => r.command === 'results.read' ? [r.arguments.week] : []))
    .toEqual([1, 2]);
  const cut = await readNativeCut('native.test', 'mine', '0042', port);
  const inspector = renderToStaticMarkup(<HQInspector native league="native.test" snapshot={reading.snapshot}
    mine="mine" id="0042" leagueValue={reading.values?.players.find(p => p.player === '0042')}
    annual={reading.contracts?.get('0042')} cut={cut} />);
  expect(inspector).toContain('8.7'); expect(inspector).toContain('18.4');
  expect(inspector).toContain('$5.00M'); expect(inspector).toContain('$1.20M');
  const contracts = renderToStaticMarkup(<HQContracts native snapshot={reading.snapshot} mine="mine"
    runway={reading.runway} annual={reading.contracts} values={reading.values}
    decisions={reading.decisions} contractDecisions={reading.contractDecisions}
    cutCosts={new Map([['0042', cut.deadNow!]])} select={() => {}} />);
  for (const visible of ['$5.00M', '$90.00M', '18.4', '0.3', 'No priceable owner contract decisions',
    'Lineup', 'DEAD CAP TOTAL · $3.00M'])
    expect(contracts).toContain(visible);
  const taxi = renderToStaticMarkup(<HQReserves native kind="taxi" snapshot={reading.snapshot}
    mine="mine" settings={{ positions: [], taxi: 4 }} values={reading.values}
    reserveExperience={reading.reserveExperience} close={() => {}} />);
  expect(taxi).toContain('7.6');
  expect(taxi).toContain('NFL YR'); expect(taxi).toContain('TAXI LEFT');
  expect(taxi).toContain('<td>2</td>'); expect(taxi).toContain('<td>1</td>');
  expect(reading.contractDecisions).toMatchObject({ franchise: 'mine', revision: 4, season: 2026,
    words: 'No priceable owner contract decisions are due.' });
  const viewerStart = port.mock.calls.length;
  const viewer = await readNativeHQ('native.test', 'mine', port, undefined, false, true);
  expect(viewer.contractDecisions).toBeUndefined();
  expect(port.mock.calls.slice(viewerStart).some(([request]) =>
    request.command === 'contract.decisions.read')).toBe(false);
  for (const command of ['value.read', 'native.cap.check', 'calendar.read', 'contract.quote',
    'contract.decisions.read'])
    expect(port).toHaveBeenCalledWith(expect.objectContaining({ command }));
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
