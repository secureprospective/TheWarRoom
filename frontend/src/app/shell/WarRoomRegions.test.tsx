import { afterEach, describe, expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { act } from 'react';
import { all, button, click, flush, mount, render, rerender, type, unmount }
  from '../commissioner/commissionerDom';
import { tradeReading } from '../data/tradeTestData';
import { heldSnapshot } from '../data/lineupTestData';
import type { Report } from '../reports/contract';
import { WarRoomEmpty, WarRoomHeader } from './WarRoomHeader';
import { WarRoomAssetBlock, WarRoomDeltaSidePanel, WarRoomTrade } from './WarRoomTrade';
import { WarRoomOwnBids, WarRoomMarket } from './WarRoomMarket';
import { WarRoomInspector } from './WarRoomInspector';
import { WarRoomPlans } from './WarRoomPlans';
import { WarRoomDrawers } from './WarRoomDrawers';
import { WarRoomDraft } from './WarRoomDraft';
import { parseWarRoomDelta, readWarRoomCatalog, readWarRoomPreview, type WarRoomCatalog }
  from './warroomTradeReads';
import { readWarRoomHeader, type WarRoomContext } from './warroomReads';
import { readWarRoomMarket, parseWarRoomOwnBids, changeWarRoomTarget, parseWarRoomProjections }
  from './warroomMarketReads';
import { readWarRoomInspector } from './warroomInspectorReads';
import { readWarRoomCalendar } from './warroomCalendar';
import { readWarRoomOffers } from './WarRoomDrawers';
import { useWarRoomRead } from './warroomReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import assetsWire from '../../../../testdata/wire/native/assets.json';
import bidsWire from '../../../../testdata/wire/native/bids.json';
import draftWire from '../../../../testdata/wire/native/draft.json';
import tradesWire from '../../../../testdata/wire/native/trades.json';
import { researchAPI } from '../research/api';
import * as clockProvider from '../data/provider';
import { controlAPI } from '../control/api';
import { raceAPI } from '../race/api';
import { contractsProvider } from '../contracts/provider';
import { poolProvider } from '../pool/provider';
import { personalWire } from '../offers/api';
import { unavailable } from './hqModel';
const context = (): WarRoomContext => ({ snapshot: heldSnapshot(), mine: '0025', week: 9 });
const emptyReport = (): Report => ({ id: 'M-002', title: '', note: '', status: 'ok', source: '', asOf: '',
  how: '', columns: ['Setting', 'Value'], summary: [], link: null, rows: [] });
const reply = (data: unknown) => ({ label: 'Native', writable: false, football: '', data });
const nativeContext = (port: HostedPort): WarRoomContext => ({ ...context(), mine: 'franchise.01',
  native: { league: 'native.seed', mine: 'franchise.01', port, leave: () => {} } });
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });
describe('War Room header and settings congruence', () => {
  it('renders record, Best-N model output, rank and runway from real read adapters', async () => {
    vi.spyOn(controlAPI, 'settings').mockResolvedValue({ ...emptyReport(), rows: [
      { cells: ['Starters each week', '17'], range: null }] });
    vi.spyOn(raceAPI, 'read').mockResolvedValue({ standings: { ...emptyReport(),
      columns: ['Record'], rows: [{ cells: ['5–3'], mine: true, range: null, group: 'North' }] },
      board: { title: '', loading: '', refresh: '', howLabel: '', mflLabel: '', mflUrl: '', reports: [{
        ...emptyReport(), id: 'M-008', columns: ['Rank', 'Roster value'], rows: [{ cells: ['4', '321.9'],
          mine: true, range: null }] }] }, starterN: 17, team: '', teams: [], week: 9, weeks: [] });
    const reading = await readWarRoomHeader(context());
    const html = renderToStaticMarkup(<WarRoomHeader name="Actual franchise" reading={reading} />);
    expect(html).toContain('class="franchise-context"');
    expect(html).not.toContain('class="table fixed"');
    expect(html).toContain('Actual franchise'); expect(html).toContain('5–3');
    expect(html).toContain('Best 17 (any position)'); expect(html).toContain('321.9');
    expect(raceAPI.read).toHaveBeenCalledWith(9, '0025', '0025', 'topn');
    expect(html).not.toContain('1,412');
  });
  it('changes Best-N when the league starter setting changes, with no preset branch', async () => {
    let starters = 8;
    vi.spyOn(controlAPI, 'settings').mockImplementation(async () => ({ ...emptyReport(), rows: [
      { cells: ['Starters each week', String(starters)], range: null }] }));
    vi.spyOn(raceAPI, 'read').mockRejectedValue(new Error('Model not held'));
    const before = renderToStaticMarkup(<WarRoomHeader reading={await readWarRoomHeader(context())} />);
    starters = 13;
    const after = renderToStaticMarkup(<WarRoomHeader reading={await readWarRoomHeader(context())} />);
    expect(before).toContain('Best 8 (any position)'); expect(after).toContain('Best 13 (any position)');
  });
  it('does not expose internal read failures in the empty-state tooltip', () => {
    const html = renderToStaticMarkup(<WarRoomEmpty reason="native.secret.read failed" />);
    expect(html).not.toContain('native.secret.read');
    expect(html).toContain('title="Not available yet"');
  });
  it('labels compact missing numeric values without wrapping the full phrase', () => {
    const html = renderToStaticMarkup(<WarRoomHeader />);
    expect(html).toContain('aria-label="Not available yet">—</span>');
  });
  it('uses the public direction words and marks the unset choice', () => {
    const html = renderToStaticMarkup(<WarRoomHeader />);
    for (const word of ['Win now', 'Retool', 'Rebuild', 'Not set']) expect(html).toContain(word);
    expect(html).not.toContain('CONTEND');
  });
  it('names the open action window and says plainly when no window is open', () => {
    expect(renderToStaticMarkup(<WarRoomHeader windowLabel="Trade" />)).toContain('window Trade');
    expect(renderToStaticMarkup(<WarRoomHeader windowLabel="none open" />))
      .toContain('window none open');
  });
  it('says next-season needs are not built rather than calling them unavailable', () => {
    const html = renderToStaticMarkup(<WarRoomHeader />);
    expect(html).toContain('NEEDS · NEXT SEASON');
    expect(html.split('NEEDS · NEXT SEASON')[1]).toContain('Not built yet');
  });
  it('renders honest empty regions rather than mock figures when all reads fail', () => {
    const html = renderToStaticMarkup(<WarRoomHeader />);
    expect(html).toContain(unavailable); expect(html).not.toContain('2026–27');
    expect(html).not.toContain('$6.4M'); expect(html).not.toContain('BEST 21');
  });
});
describe('private trade workspace and lineup changes', () => {
  it('keeps the full shelf name and compact contract in separate row content', () => {
    const html = renderToStaticMarkup(<WarRoomAssetBlock asset={{ token: '0042',
      name: 'McCaffrey, Christian', position: 'RB', contract: '$3.00M · 2y' }}
      from="0025" select={() => {}} toggle={() => {}} />);
    expect(html).toContain('McCaffrey, Christian');
    expect(html).toContain('$3.00M · 2y');
  });

  it('uses the HQ-resolved name for a native roster asset', async () => {
    const c = nativeContext(async () => reply({ ...assetsWire, identities: undefined }));
    const id = assetsWire.snapshot.players[0].id;
    c.snapshot.players.value = [{ id, name: 'Resolved player', position: 'QB' }];
    const catalog = await readWarRoomCatalog(c);
    const asset = catalog.teams.flatMap(t => t.assets).find(a => a.token === id);
    expect(asset?.name).toBe('Resolved player');
    expect(asset?.position).toBe('QB');
  });
  it('uses public identities for both shelves and the inspector, scoped to the franchise copy', async () => {
    const wire = structuredClone(assetsWire);
    const first = wire.snapshot.players[0];
    const partner = wire.snapshot.franchises.find(f => f.id !== first.franchise)!;
    wire.snapshot.players.push({ ...first, franchise: partner.id });
    const identities = [
      { player: first.id, franchise: first.franchise, side: 'AFC', name: 'Own identity', position: 'QB' },
      { player: first.id, franchise: partner.id, side: 'NFC', name: 'Partner identity', position: 'RB' },
    ];
    const c = nativeContext(async () => reply({ ...wire, identities }));
    c.snapshot.players.value = [{ id: first.id, name: 'Old HQ identity', position: 'TE' }];
    const catalog = await readWarRoomCatalog(c);
    const own = catalog.teams.find(t => t.id === first.franchise)!.assets.find(a => a.token === first.id)!;
    const incoming = catalog.teams.find(t => t.id === partner.id)!.assets.find(a => a.token === first.id)!;
    expect(own).toMatchObject({ name: 'Own identity', position: 'QB' });
    expect(incoming).toMatchObject({ name: 'Partner identity', position: 'RB' });
    const html = renderToStaticMarkup(<WarRoomInspector context={c} asset={incoming} />);
    expect(html).toContain('Partner identity');
    expect(html).toContain('class="pg">RB</span>');
  });
  it('does not invent absent public identity fields or use stale HQ values in their place', async () => {
    const first = assetsWire.snapshot.players[0];
    const c = nativeContext(async () => reply({ ...assetsWire, identities: [
      { player: first.id, franchise: first.franchise, name: '', position: '' },
    ] }));
    c.snapshot.players.value = [{ id: first.id, name: 'Stale', position: 'QB' }];
    const catalog = await readWarRoomCatalog(c);
    expect(catalog.teams.flatMap(t => t.assets).find(a => a.token === first.id))
      .toMatchObject({ name: '', position: undefined });
  });
  const catalog: WarRoomCatalog = { revision: 7, teams: [
    { id: '0025', name: 'Mine', assets: [{ token: '0042', name: 'Held player', position: 'RB',
      contract: '$3M thru 2030' }] },
    { id: '0002', name: 'Partner', assets: [{ token: '0043', name: 'Incoming', position: 'DE' }] }] };
  it('renders both shelves and changes the proposal only through selecting a held asset', async () => {
    const change = vi.fn();
    const host = await mount(render(WarRoomTrade, { context: context(), catalog,
      deal: { partner: '0002', give: [], get: [] }, change, select: vi.fn() }));
    await flush(); expect(host.textContent).toContain('Held player');
    expect(host.textContent).toContain('$3M thru 2030'); expect(host.textContent).toContain('Incoming');
    await click(button(host, 'Add Held player'));
    expect(change).toHaveBeenCalledWith({ partner: '0002', give: ['0042'], get: [] });
    const partner = all(host, 'select')[0]; await type(partner, '');
    expect(change).toHaveBeenLastCalledWith({ partner: '', give: [], get: [] });
  });
  it('shows a real preview and forces native trade drafts private without executing', async () => {
    const port = vi.fn<HostedPort>(async () => reply({ question: 'Trade preview', target: {
      league: 'native.seed', name: 'Native', version: 7 }, fields: [{ label: 'Cap room', before: '$20M',
      beforeSource: 'Ledger', after: '$17M', afterSource: 'Ledger', when: 'now' }], affects: {
      readings: [], count: 0, words: '' }, blockers: [], safeguard: 'confirm' }));
    const c = nativeContext(port);
    const nativeCatalog = { ...catalog, teams: catalog.teams.map((t, i) => ({ ...t,
      id: i ? 'franchise.02' : 'franchise.01' })) };
    const preview = await readWarRoomPreview(c, nativeCatalog, { partner: 'franchise.02',
      give: ['0042'], get: [] });
    expect(preview.teams[0].figures[0].after).toBe('$17M');
    expect(port.mock.calls[0][0]).toMatchObject({ command: 'trade.offer', mode: 'preview', arguments: {
      visibility: 'private' } });
    expect(port).toHaveBeenCalledTimes(1);
  });
  it('names every unprojected player and never presents a partial numeric delta or zero', () => {
    const side = { franchise: '0025', perWeek: 0, thisWeek: 0, before: [], after: [],
      changes: [{ kind: 'benched' as const, player: '0042', position: 'RB' }], unprojected: ['0043'] };
    const html = renderToStaticMarkup(<WarRoomDeltaSidePanel name="YOU" side={side}
      names={new Map([['0042', 'Kept veteran'], ['0043', 'Missing projection']])} />);
    expect(html).toContain('Your best lineup would bench Kept veteran');
    expect(html).toContain(`Missing projection · projection ${unavailable}`);
    expect(html).not.toContain('0.0'); expect(html).not.toContain('goes to your bench');
  });
  it('renders both before and after starter names, and signed deltas when fully projected', () => {
    const html = renderToStaticMarkup(<WarRoomDeltaSidePanel name="THEM" names={new Map([
      ['0042', 'Old starter'], ['0043', 'New starter']])} side={{ franchise: '0002', perWeek: 3.7,
      thisWeek: 2.1, changes: [], unprojected: [], before: [{ player: '0042', position: 'RB', ppg: 9.2 }],
      after: [{ player: '0043', position: 'RB', ppg: 12.9 }] }} />);
    expect(html).toContain('+3.7/wk'); expect(html).toContain('2.1');
    expect(html).toContain('Old starter'); expect(html).toContain('New starter');
  });
  it('rejects malformed delta kinds and nonfinite numbers at the read boundary', () => {
    const side = { franchise: '0025', before: [], after: [], perWeek: 1, thisWeek: 2, changes: [],
      unprojected: [] };
    expect(parseWarRoomDelta({ league: 'native.seed', season: 2026, week: 9, you: side, them: side,
      notes: [] }).you.perWeek).toBe(1);
    expect(() => parseWarRoomDelta({ league: '', season: 2026, week: 9, notes: [], you: { ...side,
      perWeek: NaN }, them: side })).toThrow('finite');
  });
});
describe('market, stars and sealed-bid privacy', () => {
  it('discards rival leaders and rival bid counts before rendering', () => {
    const wire = structuredClone(bidsWire);
    wire.players[0].leader = { ...wire.players[0].leader, franchise: 'franchise.02',
      team: 'Secret rival', contract: '$987654321 secret', points: '999' };
    wire.players[0].youLead = false;
    const own = parseWarRoomOwnBids(wire, 'native.seed', 'franchise.01');
    const html = renderToStaticMarkup(<WarRoomOwnBids reading={{ bids: own, errors: [] }} />);
    expect(html).toContain('2 years at $5,000,000 a year'); expect(html).toContain('OUTBID');
    expect(html).not.toContain('Secret rival'); expect(html).not.toContain('987654321');
    expect(html).not.toContain('999');
  });
  it('rejects an own-bid payload for another owner or a wrong own-row identity', () => {
    expect(() => parseWarRoomOwnBids(bidsWire, 'native.seed', 'franchise.02')).toThrow('another franchise');
    const wrong = structuredClone(bidsWire); wrong.players[0].yourBid.franchise = 'franchise.02';
    expect(() => parseWarRoomOwnBids(wrong, 'native.seed', 'franchise.01')).toThrow('scope');
  });
  it('never reads local private targets or plans for native leagues', async () => {
    const privateRead = vi.spyOn(personalWire, 'read'); const plans = vi.spyOn(poolProvider, 'plans');
    const port = vi.fn<HostedPort>(async r => r.command === 'bids.read' ? reply(bidsWire) :
      Promise.reject(new Error('No pool')));
    const reading = await readWarRoomMarket(nativeContext(port));
    expect(reading.bids?.[0].contract).toContain('$5,000,000'); expect(reading.targets).toBeUndefined();
    expect(privateRead).not.toHaveBeenCalled(); expect(plans).not.toHaveBeenCalled();
    await expect(changeWarRoomTarget(nativeContext(port), '0042', false)).rejects.toThrow('Hosted');
  });
  it('stars a real pool player, re-reads the lane, then swaps within that lane', async () => {
    const entries: { kind: string; entity: string; id: string; text: string; at: string; order: number }[]
      = [{ kind: 'watch', entity: 'player', id: '0043', text: '', at: '2026-10-31T10:00:00Z', order: 1 }];
    const personal = { entries, bait: emptyReport(), players: [] };
    const read = vi.fn(async () => personal);
    const change = vi.fn(async (raw: { action: string; id: string; with?: string }) => {
      if (raw.action === 'add') entries.push({ ...entries[0], id: raw.id, order: 2 });
      if (raw.action === 'swap') [entries[0].order, entries[1].order] = [entries[1].order, entries[0].order];
    });
    vi.spyOn(poolProvider, 'pool').mockResolvedValue({ status: 'ok', value: { rows: [
      { id: '0042', name: 'Pool runner', position: 'RB', age: '25', minimum: { text: '$1M', cents:
        100_000_000 }, engineText: '24.8' }, { id: '0043', name: 'Other runner', position: 'RB',
        age: '26', minimum: { text: '$2M', cents: 200_000_000 }, engineText: '22.2' }], positions: [],
      summary: '', sorts: [] } } as unknown as Awaited<ReturnType<typeof poolProvider.pool>>);
    vi.spyOn(poolProvider, 'plans').mockRejectedValue(new Error('No plans'));
    const host = await mount(render(WarRoomMarket, { context: context(), select: vi.fn() }), {}, {
      offers: { API: { TargetPersonal: read, TargetPersonalChange: change } } });
    await flush(); expect(host.textContent).toContain('24.8'); expect(host.textContent).toContain('$1M');
    await click(button(host, 'Watch Pool runner')); await flush();
    expect(change).toHaveBeenCalledWith(expect.objectContaining({ mine: '0025', kind: 'watch',
      entity: 'player', id: '0042', action: 'add' }));
    await click(button(host, 'Move Pool runner up')); await flush();
    expect(change).toHaveBeenLastCalledWith(expect.objectContaining({ action: 'swap', id: '0042',
      with: '0043', mine: '0025' }));
    expect(read).toHaveBeenCalledTimes(3);
  });
  it('keeps missing projection values unavailable instead of treating zero as held', () => {
    const projections = parseWarRoomProjections({ league: 'native.seed', season: 2026, week: 9,
      notes: [], players: [{ player: '0042', floor: 0, median: 0, ceiling: 0, games: 0,
        available: false, words: 'No held weeks' }] }, 'native.seed', 2026, 9);
    expect(projections[0].available).toBe(false);
    expect(() => parseWarRoomProjections({ league: 'other', season: 2026, week: 9 }, 'native.seed',
      2026, 9)).toThrow('another league');
  });
});
describe('calendar draft drawer and plans', () => {
  it('uses the reference calendar reading for deadlines and failure, just like the hosted read', async () => {
    const provider = clockProvider.selectProvider();
    const reading = await provider.clock();
    vi.spyOn(provider, 'clock').mockResolvedValue(reading);
    vi.spyOn(clockProvider, 'selectProvider').mockReturnValue(provider);
    expect(await readWarRoomCalendar(context())).toEqual(reading.value.deadlines);
    vi.mocked(provider.clock).mockResolvedValue({ ...reading, provenance: { ...reading.provenance,
      freshness: { ...reading.provenance.freshness, state: 'fail', note: 'calendar offline' } } });
    await expect(readWarRoomCalendar(context())).rejects.toThrow('calendar offline');
  });

  it('reads commissioner calendar windows and the trade deadline, not mock dates', async () => {
    const port = vi.fn<HostedPort>(async () => reply({ league: 'native.seed', revision: 7, decision: {},
      clock: { windows: [{ id: 'rookies', opens: ['draft'], start: '2027-04-01T12:00:00Z',
        close: null, live: false, secondsToClose: null }], exceptions: [], tradeDeadline:
        '2026-11-13T12:00:00Z', secondsToTradeDeadline: 100 } }));
    const events = await readWarRoomCalendar(nativeContext(port));
    expect(events[0].label).toBe('Draft'); expect(events[1].at).toBe('2026-11-13T12:00:00Z');
    expect(port.mock.calls[0][0]).toMatchObject({ command: 'calendar.read', mode: 'read' });
  });
  it('names the actions an app league window opens, so the header can say which one is live', async () => {
    const port = vi.fn<HostedPort>(async () => reply({ league: 'native.seed', revision: 7, decision: {},
      clock: { windows: [{ id: 'lineups', opens: ['lineup'], start: '2026-03-01T05:00:00Z',
        close: null, live: true, secondsToClose: null }, { id: 'waivers', opens: ['cut', 'waiver_claim'],
        start: '2026-03-01T05:00:00Z', close: null, live: false, secondsToClose: null }],
        exceptions: [], tradeDeadline: null, secondsToTradeDeadline: null } }));
    const events = await readWarRoomCalendar(nativeContext(port));
    expect(events.find(e => e.live)!.label).toBe('Lineup');
    expect(events[1].label).toBe('Cut · Waiver claim');
  });
  it('renders the native draft read and honors round/mine filters', async () => {
    const port = vi.fn<HostedPort>(async () => reply(draftWire));
    const host = await mount(render(WarRoomDraft, { context: nativeContext(port), select: vi.fn() }));
    await flush(); expect(host.textContent).toContain(draftWire.words);
    expect(host.textContent).toContain('DRAFT ORDER'); expect(host.textContent).toContain('MY QUEUE');
    expect(host.textContent).toContain(unavailable);
    await click(button(host, 'MINE'));
    expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'draft.read', mode: 'read' }));
    expect(button(host, '▸ RUN MOCK DRAFT').disabled).toBe(true);
  });
  it('renders plans missing regions without made-up saved counts or cap outcomes', () => {
    const html = renderToStaticMarkup(<WarRoomPlans />);
    for (const label of ['PLANS', 'MY NEEDS', 'CONFLICTS', 'CAP IF THIS WORKS', 'POCKET', 'MARKET'])
      expect(html).toContain(label);
    expect(html).toContain(unavailable); expect(html).not.toContain('WIN NOW 2026');
  });
  it('renders unavailable pocket and closes it without reading private saved versions', async () => {
    const close = vi.fn(); const port = vi.fn<HostedPort>();
    const host = await mount(render(WarRoomDrawers, { context: nativeContext(port), drawer: 'pocket',
      close, openOffer: vi.fn() })); await flush();
    expect(host.textContent).toContain(unavailable); expect(port).not.toHaveBeenCalled();
    await click(button(host, 'ESC')); expect(close).toHaveBeenCalledTimes(1);
  });
  it('opens a mirror negotiation in the preserved offer inspector without a handoff', async () => {
    const openOffer = vi.fn();
    const host = await mount(render(WarRoomDrawers, { context: context(), drawer: 'offers',
      close: vi.fn(), openOffer }), {}, { trades: { API: { TargetTrades: async () => tradeReading() } } });
    await flush();
    await click(button(host, 'Rivals'));
    expect(openOffer).toHaveBeenCalledWith('901');
  });
  it('accepts only the asking native franchise’s offer reading', async () => {
    const wrong = { ...tradesWire, league: 'native.seed', viewer: 'franchise.02' };
    const port = vi.fn<HostedPort>(async () => reply(wrong));
    await expect(readWarRoomOffers(nativeContext(port))).rejects.toThrow('another viewer');
  });
});
describe('inspector and request lifetimes', () => {
  it('renders a selected contract and real age arc, then persists a private note', async () => {
    const id = '0042';
    let notes: { kind: string; entity: string; id: string; text: string; at: string; order: number }[] = [];
    const noteRead = vi.fn(async () => ({ player: id, count: notes.length, notes }));
    const change = vi.fn(async (raw: { text: string }) => { notes = [{ kind: 'note', entity: 'player', id,
      text: raw.text, at: '2026-10-31T12:00:00Z', order: 1 }]; });
    const host = await mount(render(WarRoomInspector, { context: context(), asset: { token: id,
      name: 'Selected runner', position: 'RB' } }), {}, { offers: { API: { TargetPersonalNotes: noteRead,
      TargetPersonalChange: change } }, market: { API: { TargetAgeArc: async () => ({ age: 25,
      peakStart: 2026, peakEnd: 2027, points: [{ season: 2026, age: 25, level: 91, survival: .9 }] }) } },
      contracts: { API: { TargetPlayerContract: async () => { throw new Error('No contract held'); } } } });
    await flush(); expect(host.textContent).toContain('age 25 → 25');
    const notesButton = all(host, 'button').find(b => b.textContent.startsWith('NOTES'))!;
    await click(notesButton); await type(all(host, 'textarea')[0], 'Private observation');
    await act(async () => { (all(host, 'form')[0].props().onSubmit as (e: object) => void)({
      preventDefault() {} }); }); await flush();
    expect(change).toHaveBeenCalledWith(expect.objectContaining({ mine: '0025', entity: 'player',
      kind: 'note', text: 'Private observation' }));
    expect(host.textContent).toContain('Private observation'); expect(noteRead).toHaveBeenCalledTimes(2);
  });
  it('does not call mirror notes or research for an app inspector', async () => {
    const notes = vi.spyOn(personalWire, 'read');
    const port = vi.fn<HostedPort>(async r => r.command === 'projections.read' ? reply({
      league: 'native.seed', season: 2026, week: 9, players: [{ player: '0042', floor: 2, median: 7.1,
        ceiling: 11, games: 6, available: true, words: 'Held projections' }], notes: [] }) :
      Promise.reject(new Error('No cut quote')));
    const c = nativeContext(port); c.snapshot.league.value.season = 2026;
    const facts = await readWarRoomInspector(c, '0042');
    expect(facts.projection?.median).toBe(7.1); expect(facts.notes).toBeUndefined();
    expect(notes).not.toHaveBeenCalled();
    expect(port.mock.calls.every(([r]) => r.mode === 'read')).toBe(true);
  });
  it('fences a slow first selection so it cannot reveal another player’s data', async () => {
    let finish!: (value: string) => void;
    const first = new Promise<string>(yes => { finish = yes; });
    function Probe({ id }: { id: string }) {
      const r = useWarRoomRead(id, () => id === 'first' ? first : Promise.resolve('Second reading'));
      return <p>{r.reading ?? unavailable}</p>;
    }
    const host = await mount(render(Probe, { id: 'first' }));
    await rerender(render(Probe, { id: 'second' })); await flush();
    expect(host.textContent).toBe('Second reading');
    await act(async () => { finish('Private first reading'); });
    expect(host.textContent).toBe('Second reading');
  });
});

it('never calls a blind bid outbid when the read deliberately hides leadership', () => {
  const wire = structuredClone(bidsWire);
  wire.players[0].mode = 'blind'; wire.players[0].youLead = false;
  const own = parseWarRoomOwnBids(wire, 'native.seed', 'franchise.01');
  const html = renderToStaticMarkup(<WarRoomOwnBids reading={{ bids: own, errors: [] }} />);
  expect(html).toContain('PLACED · PRIVATE'); expect(html).not.toContain('OUTBID');
  expect(html).not.toContain('HIGH BID');
});

it('does not retain a rejected research identity in the selected inspector reading', async () => {
  vi.spyOn(researchAPI, 'player').mockResolvedValue({ id: '0043', name: 'Other player', position: 'RB',
    team: '', franchiseId: '0002', owner: 'Another owner', report: emptyReport(), articles: [],
    howLabel: '' });
  const result = await readWarRoomInspector(context(), '0042');
  expect(result.research).toBeUndefined();
  expect(result.errors.join(' · ')).toContain('another player');
});

it('quotes the selected player’s actual roster, not the viewing franchise', async () => {
  const port = vi.fn<HostedPort>(async () => { throw new Error('Read captured'); });
  const c = nativeContext(port);
  const player = { id: '0042', salary: 100, contractStatus: 'UFA' as const, rosterStatus: 'ROSTER' as const };
  c.snapshot.rosters.value = [{ franchiseId: 'franchise.02', players: [player] }];
  await readWarRoomInspector(c, '0042');
  expect(port.mock.calls.find(([r]) => r.command === 'contract.quote')?.[0].arguments)
    .toMatchObject({ franchise: 'franchise.02' });
  port.mockClear();
  c.snapshot.rosters.value.push({ franchiseId: 'franchise.03', players: [player] });
  const ambiguous = await readWarRoomInspector(c, '0042');
  expect(port.mock.calls.some(([r]) => r.command === 'contract.quote')).toBe(false);
  expect(ambiguous.errors.join(' ')).toContain('Choose the rostered copy');
  port.mockClear();
  c.snapshot.rosters.value.push({ franchiseId: c.mine, players: [player] });
  await readWarRoomInspector(c, '0042');
  expect(port.mock.calls.find(([r]) => r.command === 'contract.quote')?.[0].arguments)
    .toMatchObject({ franchise: c.mine });
});
