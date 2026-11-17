import { afterEach, describe, expect, it, vi } from 'vitest';
import { poolProvider } from '../pool/provider';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { BidFormState, Plan } from '../pool/contract';
import type { WarRoomContext } from './warroomReads';
import {
  parseWarRoomBidReading,
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

afterEach(() => vi.restoreAllMocks());

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

  it('previews place, raise and withdraw privately, then executes with receipt and rereads bids and cap',
    async () => {
      const calls: string[] = [];
      const port = vi.fn<HostedPort>(async request => {
        calls.push(`${request.command}:${request.mode}`);
        if (request.mode === 'preview') return { label: '', writable: true, football: 'pin',
          data: preview(request.arguments.expected) };
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
        { type: 'withdraw' as const, bid: placed, revision: 7, writable: true, football: 'pin' },
      ];
      const reviews = await Promise.all(actions.map(action => warRoomMarketAction(c, action)));
      expect(reviews.map(result => result.kind)).toEqual(['review', 'review', 'review']);
      const requests = port.mock.calls.slice(0, 3).map(([request]) => request) as unknown as
        { command: string; mode: string; arguments: { visibility?: string; args?: { bid?: unknown } } }[];
      expect(requests.map(request => request.command)).toEqual(['bid.place', 'bid.raise', 'bid.withdraw']);
      expect(requests.every(request => request.mode === 'preview' &&
        request.arguments.visibility === 'private')).toBe(true);
      expect(requests[0].arguments.args?.bid).toEqual({ player: '0001', franchise: mine,
        salaryCents: 200000000, years: 2 });
      expect(requests[1].arguments.args?.bid).toEqual({ bid: 'bid-1', franchise: mine,
        salaryCents: 250000000, years: 3 });
      expect(requests[2].arguments.args?.bid).toEqual({ bid: 'bid-1', franchise: mine });

      const review = reviews[0];
      if (review.kind !== 'review') throw new Error('Expected preview');
      const recorded = await warRoomMarketAction(c, { type: 'confirm', request: review.request,
        football: review.football });
      expect(recorded).toMatchObject({ kind: 'recorded', receipt, capRoom: 8500000000,
        bids: { revision: 8, bids: [placed] } });
      expect(calls).toContain('bid.place:execute');
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
    const review = await warRoomMarketAction(c, { type: 'withdraw', bid: placed, revision: 7,
      writable: true, football: 'pin' });
    if (review.kind !== 'review') throw new Error('Expected preview');
    const recorded = await warRoomMarketAction(c, { type: 'confirm', request: review.request,
      football: review.football });
    expect(recorded).toMatchObject({ kind: 'recorded', receipt,
      refreshError: 'Your bid was recorded, but its current status or cap room could not be refreshed. ' +
        'Refresh before trying again.' });
    expect(port.mock.calls.filter(([request]) => request.mode === 'execute')).toHaveLength(1);
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
    await expect(warRoomMarketAction(context(port, false), { type: 'withdraw', bid: placed,
      revision: 7, writable: true, football: 'pin' })).rejects.toThrow('owner seat');
    expect(port).not.toHaveBeenCalled();
  });
});
