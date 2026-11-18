import { afterEach, describe, expect, it, vi } from 'vitest';
import { poolProvider } from '../pool/provider';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { BidFormState, Plan } from '../pool/contract';
import type { NativeRequest } from '../commands/native';
import type { WarRoomContext } from './warroomReads';
import {
  parseWarRoomBidReading,
  readWarRoomMarket,
  warRoomMarketAction,
} from './warroomMarketReads';

const league = 'native.test';
const mine = 'franchise.mine';
const other = 'franchise.other';
const ownerBid = (id = 'bid-1', franchise = mine) => ({ id, franchise, team: 'Mine',
  contract: '$2.00M · 2 years', points: '0', placedAt: '2026-11-17T12:00:00Z', placed: 'Today' });
const bidReading = (revision = 7) => ({ league, franchise: mine, revision, season: 2026,
  mode: 'open', activeBids: 1, activeLimit: '3', notes: [], awards: [], players: [
    { player: '0001', name: 'Player One', mode: 'open', modeWords: 'Open',
      leader: { ...ownerBid('rival-bid', other), contract: '$9.00M · 4 years' }, bids: 4,
      yourBid: ownerBid(), youLead: false, lateAt: '', lateWords: '', lateNow: false,
      closesAt: null, closes: 'Closes tomorrow' },
    { player: '0002', name: 'Player Two', mode: 'open', modeWords: 'Open',
      leader: ownerBid('another-rival', other), bids: 2, youLead: false, lateAt: '', lateWords: '',
      lateNow: false, closesAt: null, closes: 'Closes tomorrow' },
  ] });
const preview = (version = 7) => ({ question: 'Place a private bid',
  target: { league, name: 'League', version }, fields: [{ label: 'Bid', before: 'None',
    beforeSource: 'Ledger', after: '$2.00M · 2 years', afterSource: 'Request', when: 'now' }],
  affects: { readings: [], count: null, words: 'Not known yet' }, blockers: [], safeguard: 'confirm' });
const receipt = { entry: 'entry-8', league, revision: 8,
  recordedAt: '2026-11-17T12:01:00Z', message: 'Bid placed privately' };
const capReading = (roomCents = 8500000000) => ({ league, name: 'League', revision: 8,
  corrections: [], rosterSeason: 2026, rulesSeason: 2026, seeded: true, source: '', counting: {}, rules: {},
  franchises: [{ franchise: mine, exception: '', capCents: 10000000000, salaryCents: 0,
    activeCents: 0, irCents: 0, taxiCents: 0, adjustmentCents: 0, committedCents: 1500000000,
    roomCents, quantumCents: 1, mode: '', deadline: '', words: '', enforcement: '', name: 'Mine',
    abbreviation: 'M', charges: [], runway: [{ season: 2026, committedCents: 1500000000,
      deadCents: 0, capCents: 10000000000, roomCents }], contracts: [] }] });
const context = (port?: HostedPort, owner = true): WarRoomContext => ({ mine,
  snapshot: {} as WarRoomContext['snapshot'], week: 1,
  native: { league, mine, port, owner, leave: () => {} } });
const marketPlayer = { id: '0001', name: 'Player One', position: 'WR' };
const placed = { player: '0001', name: 'Player One', contract: '$2.00M · 2 years',
  state: 'PLACED · PRIVATE', kind: 'placed' as const, bidId: 'bid-1' };
const marketAssets = () => ({ league, revision: 4, seeded: true, entry: 'entry-1', source: 'held',
  pool: [
    { player: '0001', name: 'Held player', position: 'WR', string: 'FA', points: '14.2', age: '—',
      value: 'must not use', minimum: { cents: 8500000000, text: '$85.00M' } },
    { player: '0002', name: 'Facts missing', position: '—', string: '—', points: '—' },
  ], identities: [], corrections: [], snapshot: { season: 2026, franchises: [{ id: mine, heldId: mine,
    name: 'Mine', abbreviation: 'M', division: 'East' }], players: [], picks: [] } });
const marketValues = () => ({ league, season: 2026, franchise: '', preparing: false,
  players: [
    { player: '0001', franchise: '', position: 'WR', nowPPG: 4.2, dynastyPPG: 12.36, age: 25.5,
      ageKnown: true, available: true, words: 'Held value' },
    { player: '0002', franchise: '', position: 'RB', nowPPG: 0, dynastyPPG: 0, age: 0,
      ageKnown: false, available: false, words: 'No held value' },
  ], best: { franchise: '', starterN: 0, value: 0, age: 0, ageKnown: false, rank: 0, of: 0,
    available: false, words: '' }, age: [], notes: [] });

afterEach(() => vi.restoreAllMocks());

describe('War Room hosted market readings', () => {
  it('maps only held pool facts and server values, with missing facts unavailable', async () => {
    const port = vi.fn<HostedPort>(async request => {
      switch (request.command) {
        case 'commish.rules.read': return { label: '', writable: true, football: 'pin', data: {
          league, name: 'Test league', revision: 4, rollover: '', sections: [],
        } };
        case 'native.assets.read': return { label: '', writable: true, football: 'pin',
          data: marketAssets() };
        case 'value.read': return { label: '', writable: true, football: 'pin', data: marketValues() };
        case 'bids.read': return { label: '', writable: true, football: 'pin', data: bidReading() };
        case 'player.watch.read': return { label: '', writable: true, football: 'pin', data: {
          league, players: [], } };
        case 'market.wins.read': return { label: '', writable: true, football: 'pin',
          data: { league, wins: [] } };
        case 'native.cap.check': return { label: '', writable: true, football: 'pin', data: capReading() };
        case 'waivers.read': return { label: '', writable: true, football: 'pin', data: { league,
          franchise: mine, revision: 4, season: 2026, players: [], claims: [], order: [],
          orderSource: '', method: '' } };
        default: throw new Error(`Unexpected ${request.command}`);
      }
    });

    const reading = await readWarRoomMarket(context(port));

    expect(reading.players).toEqual([
      { id: '0001', name: 'Held player', position: 'WR', age: '25.5', string: 'FA', points: '14.2',
        value: '12.4', minimum: '$85.00M', minimumCents: 8500000000 },
      { id: '0002', name: 'Facts missing', position: 'Not available yet', age: 'Not available yet',
        string: 'Not available yet', points: 'Not available yet', value: 'Not available yet',
        minimum: 'Not available yet' },
    ]);
    expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'value.read', mode: 'read',
      arguments: { league, franchise: '', season: 2026 } }));
    expect(reading.capRoom).toBe(8500000000);
    expect(reading.targets).toEqual([]);
    expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'player.watch.read', mode: 'read',
      arguments: { league } }));
    expect(reading.errors).toEqual([]);
  });

  it('keeps held pool rows when value.read is unavailable and does not invent value or age', async () => {
    const port = vi.fn<HostedPort>(async request => {
      if (request.command === 'native.assets.read') return { label: '', writable: true, football: 'pin',
        data: marketAssets() };
      if (request.command === 'commish.rules.read') return { label: '', writable: true, football: 'pin',
        data: { league, name: 'Test league', revision: 4, rollover: '', sections: [] } };
      if (request.command === 'value.read') throw new Error('Values unavailable');
      if (request.command === 'bids.read') return { label: '', writable: true, football: 'pin',
        data: bidReading() };
      if (request.command === 'player.watch.read') return { label: '', writable: true, football: 'pin',
        data: { league, players: [] } };
      if (request.command === 'market.wins.read') return { label: '', writable: true, football: 'pin',
        data: { league, wins: [] } };
      if (request.command === 'native.cap.check') return { label: '', writable: true, football: 'pin',
        data: capReading() };
      if (request.command === 'waivers.read') return { label: '', writable: true, football: 'pin', data: {
        league, franchise: mine, revision: 4, season: 2026, players: [], claims: [], order: [],
        orderSource: '', method: '' } };
      throw new Error(`Unexpected ${request.command}`);
    });

    const reading = await readWarRoomMarket(context(port));

    expect(reading.players?.[0]).toMatchObject({ name: 'Held player', points: '14.2',
      age: 'Not available yet', value: 'Not available yet', minimum: '$85.00M' });
    expect(reading.errors).toContain('Error: Values unavailable');
    expect(port.mock.calls.map(([request]) => request.command)).toContain('value.read');
  });
});

describe('War Room hosted market bid workflow', () => {
  it('keeps own bids sealed and omits rival bids and counts from the parsed view', () => {
    const reading = parseWarRoomBidReading(bidReading(), league, mine);
    expect(reading.revision).toBe(7);
    expect(reading.bids).toEqual([placed]);
    expect(JSON.stringify(reading)).not.toContain('rival-bid');
    expect(JSON.stringify(reading)).not.toContain('another-rival');
    expect(JSON.stringify(reading)).not.toContain('bids":4');
    expect(() => parseWarRoomBidReading(bidReading(), league, other)).toThrow('another franchise');
    const wrongOwner = bidReading();
    wrongOwner.players[0].yourBid = ownerBid('bid-1', other);
    expect(() => parseWarRoomBidReading(wrongOwner, league, mine)).toThrow('scope changed');
  });

  it('previews place and raise privately, then executes with receipt and rereads bids and cap',
    async () => {
      const calls: string[] = [];
      const port = vi.fn<HostedPort>(async request => {
        calls.push(`${request.command}:${request.mode}`);
        if (request.mode === 'preview') {
          if (!('expected' in request.arguments) || typeof request.arguments.expected !== 'number')
            throw new Error('Expected a commissioner preview version.');
          return { label: '', writable: true, football: 'pin', data: preview(request.arguments.expected) };
        }
        if (request.mode === 'execute') return { label: '', writable: true, football: 'pin', data: receipt };
        if (request.command === 'bids.read') return { label: '', writable: true, football: 'pin',
          data: bidReading(8) };
        if (request.command === 'native.cap.check') return { label: '', writable: true, football: 'pin',
          data: capReading() };
        throw new Error(`Unexpected ${request.command}`);
      });
      const c = context(port);
      const actions = [
        { type: 'place' as const, player: marketPlayer, salaryCents: 200000000, years: 2,
          revision: 7, maxYears: 4, capRoom: 9000000000, writable: true, football: 'pin' },
        { type: 'raise' as const, bid: placed, salaryCents: 250000000, years: 3,
          revision: 7, maxYears: 4, capRoom: 9000000000, writable: true, football: 'pin' },
      ];
      const reviews = await Promise.all(actions.map(action => warRoomMarketAction(c, action)));
      expect(reviews.map(result => result.kind)).toEqual(['review', 'review']);
      const requests = port.mock.calls.slice(0, 2).map(([request]) => request) as unknown as
        { command: string; mode: string; arguments: { visibility?: string; args?: { bid?: unknown } } }[];
      expect(requests.map(request => request.command)).toEqual(['bid.place', 'bid.raise']);
      expect(requests.every(request => request.mode === 'preview' &&
        request.arguments.visibility === 'private')).toBe(true);
      expect(requests[0].arguments.args?.bid).toEqual({ player: '0001', franchise: mine,
        salaryCents: 200000000, years: 2 });
      expect(requests[1].arguments.args?.bid).toEqual({ bid: 'bid-1', franchise: mine,
        salaryCents: 250000000, years: 3 });

      const placeReview = reviews[0];
      if (placeReview.kind !== 'review') throw new Error('Expected place preview');
      const placedResult = await warRoomMarketAction(c, { type: 'confirm', request: placeReview.request,
        football: placeReview.football });
      expect(placedResult).toMatchObject({ kind: 'recorded', receipt, capRoom: 8500000000,
        bids: { revision: 8, bids: [placed] } });
      const raiseReview = reviews[1];
      if (raiseReview.kind !== 'review') throw new Error('Expected raise preview');
      await warRoomMarketAction(c, { type: 'confirm', request: raiseReview.request,
        football: raiseReview.football });
      expect(calls).toContain('bid.place:execute');
      expect(calls).toContain('bid.raise:execute');
      expect(calls).toContain('bids.read:read');
      expect(calls).toContain('native.cap.check:read');
    });

  it('refuses stale previews and invalid salary, years or cap before sending a command', async () => {
    const port = vi.fn<HostedPort>(async request => ({ label: '', writable: true, football: 'pin',
      data: preview((request.arguments as { expected: number }).expected + 1) }));
    const c = context(port);
    await expect(warRoomMarketAction(c, { type: 'place', player: marketPlayer,
      salaryCents: 200000000, years: 2, revision: 7, maxYears: 4, capRoom: 9000000000,
      writable: true, football: 'pin' }))
      .rejects.toThrow('different league version');
    expect(port).toHaveBeenCalledTimes(1);
    await expect(warRoomMarketAction(c, { type: 'place', player: marketPlayer,
      salaryCents: 200000000, years: 2, revision: 7, maxYears: 4, capRoom: 9000000000,
      writable: false, football: 'pin' })).rejects.toThrow('read-only');
    expect(port).toHaveBeenCalledTimes(1);
    for (const input of [
      { salaryCents: 0, years: 2, capRoom: 9000000000 },
      { salaryCents: 100, years: 5, capRoom: 9000000000 },
      { salaryCents: 100, years: 1, capRoom: undefined },
      { salaryCents: 9000000001, years: 1, capRoom: 9000000000 },
    ]) {
      await expect(warRoomMarketAction(c, { type: 'place', player: marketPlayer,
        ...input, revision: 7, maxYears: 4, writable: true, football: 'pin' })).rejects.toThrow();
    }
    expect(port).toHaveBeenCalledTimes(1);
  });

  it('reports a committed bid when a follow-up private read fails without retrying the command', async () => {
    const port = vi.fn<HostedPort>(async request => {
      if (request.mode === 'preview') return { label: '', writable: true, football: 'pin', data: preview() };
      if (request.mode === 'execute') return { label: '', writable: true, football: 'pin', data: receipt };
      if (request.command === 'bids.read') throw new Error('Read refused');
      if (request.command === 'native.cap.check') return { label: '', writable: true, football: 'pin',
        data: capReading() };
      throw new Error(`Unexpected ${request.command}`);
    });
    const c = context(port);
    const review = await warRoomMarketAction(c, { type: 'raise', bid: placed, salaryCents: 250000000,
      years: 3, revision: 7, maxYears: 4, capRoom: 9000000000, writable: true, football: 'pin' });
    if (review.kind !== 'review') throw new Error('Expected preview');
    const recorded = await warRoomMarketAction(c, { type: 'confirm', request: review.request,
      football: review.football });
    expect(recorded).toMatchObject({ kind: 'recorded', receipt,
      refreshError: 'Your bid was recorded, but its current status or cap room could not be refreshed. ' +
        'Refresh before trying again.' });
    expect(port.mock.calls.filter(([request]) => request.mode === 'execute')).toHaveLength(1);
  });

  it('rejects a forged withdrawal preview instead of confirming it', async () => {
    const port = vi.fn<HostedPort>();
    const request = { command: 'bid.withdraw', mode: 'preview', arguments: { league,
      command: 'bid.withdraw', visibility: 'private' } } as unknown as NativeRequest;
    await expect(warRoomMarketAction(context(port), { type: 'confirm', request,
      football: 'pin' })).rejects.toThrow('Only a reviewed private bid can be confirmed');
    expect(port).not.toHaveBeenCalled();
  });

  it('keeps local draft and revise on the existing pool provider', async () => {
    const plan = { id: 'plan-1', playerId: '0001' } as Plan;
    const draft = vi.spyOn(poolProvider, 'draft').mockResolvedValue(plan);
    const revise = vi.spyOn(poolProvider, 'revise').mockResolvedValue(plan);
    const form: BidFormState = {
      planId: '', playerId: '0001', playerName: 'Player One', salary: '2', years: 1,
      snipe: false, beat: '', evidence: '', closesOpen: false, closesDate: '', closesTime: '',
    };
    const bid = { ...placed, plan: { ...plan, id: 'plan-1' } as Plan };
    const c = { mine } as WarRoomContext;
    expect(await warRoomMarketAction(c, { type: 'draft', form })).toEqual({ kind: 'plan', plan });
    expect(await warRoomMarketAction(c, { type: 'revise', bid, form })).toEqual({ kind: 'plan', plan });
    expect(draft).toHaveBeenCalledWith(mine, form);
    expect(revise).toHaveBeenCalledWith('plan-1', mine, form);
  });

  it('does not make hosted owner actions for a non-owner view', async () => {
    const port = vi.fn<HostedPort>();
    await expect(warRoomMarketAction(context(port, false), { type: 'raise', bid: placed,
      salaryCents: 250000000, years: 3, revision: 7, maxYears: 4, capRoom: 9000000000,
      writable: true, football: 'pin' })).rejects.toThrow('owner seat');
    expect(port).not.toHaveBeenCalled();
  });
});
