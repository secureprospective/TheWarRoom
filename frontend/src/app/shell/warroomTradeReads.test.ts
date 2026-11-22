import { expect, it, vi } from 'vitest';
import { heldSnapshot } from '../data/lineupTestData';
import type { HostedPort } from '../commissioner/hostedRoom';
import { executeWarRoomOffer, readWarRoomPartner, type WarRoomCatalog } from './warroomTradeReads';

it('executes a hosted offer with its request key and verifies it in trades.read', async () => {
  const calls: string[] = [];
  const port: HostedPort = vi.fn(async request => {
    calls.push(`${request.command}:${request.mode}`);
    if (request.command === 'trade.offer' && request.mode === 'execute') {
      expect(request.arguments.expected).toBe(12);
      expect(request.arguments.request).toBe('retry-key');
      expect(request.arguments.visibility).toBe('private');
      return { data: { entry: 'offer-13', league: 'native.league', revision: 13,
        recordedAt: '2026-11-17T12:00:00Z', message: 'Recorded' }, label: 'server', writable: true,
        football: 'held' };
    }
    if (request.command === 'trades.read') return { data: { league: 'native.league', viewer: 'mine',
      revision: 13, rules: {}, notes: [], offers: [{ id: 'offer-13', proposer: 'mine',
        parties: ['mine', 'them'],
        assets: [], assetDetails: [], accepted: ['mine'], policy: '', offeredAt: 'now', expiresAt: '',
        status: '', words: '', expires: '', votes: [], recusals: [], appeals: [], overrides: [],
        vetoNumber: 0, appealedVetoNumber: 0 }] }, label: 'server', writable: true, football: 'held' };
    throw new Error(`Unexpected call ${request.command}`);
  });
  const catalog: WarRoomCatalog = { revision: 12, teams: [
    { id: 'mine', name: 'Mine', assets: [{ token: 'p1', name: 'Player' }] },
    { id: 'them', name: 'Them', assets: [] },
  ] };
  const receipt = await executeWarRoomOffer({ snapshot: heldSnapshot(), mine: 'mine', week: 1,
    native: { league: 'native.league', mine: 'mine', leave: () => {}, port } }, catalog,
  { partner: 'them', give: ['p1'], get: [] }, 'retry-key');
  expect(receipt.entry).toBe('offer-13');
  expect(calls).toEqual(['trade.offer:execute', 'trades.read:read']);
});

it('reads the checked partner standings row and scopes cap lookup to that partner', async () => {
  const port: HostedPort = vi.fn(async request => {
    if (request.command === 'standings.read') return { data: { league: 'native.league', revision: 4,
      words: 'Current', criteria: [], noData: [], computed: [
        { rank: 1, franchise: 'mine', name: 'Mine', record: '9-1', pointsWords: '900', why: [] },
        { rank: 2, franchise: 'them', name: 'Them', record: '8-2', pointsWords: '800', why: [] },
      ] }, label: 'server', writable: true, football: 'held' };
    if (request.command === 'native.cap.check') return { data: { league: 'native.league', rulesSeason: 2026,
      franchises: ['mine', 'them'].map((franchise, i) => ({ franchise, irCents: null, runway: [{
        season: 2026, committedCents: 1000, deadCents: 0, capCents: 5000, roomCents: 4000 + i,
      }] })) }, label: 'server', writable: true, football: 'held' };
    throw new Error(`Unexpected call ${request.command}`);
  });
  const context = { snapshot: heldSnapshot(), mine: 'mine', week: 1,
    native: { league: 'native.league', mine: 'mine', leave: () => {}, port } };

  expect(await readWarRoomPartner(context, 'them')).toEqual({ record: '8-2', room: 4001 });
  expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'standings.read', mode: 'read',
    arguments: { league: 'native.league', season: 2026 } }));
  expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'native.cap.check', mode: 'read',
    arguments: { league: 'native.league', viewerZone: expect.any(String) } }));
});
