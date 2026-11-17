import { expect, it, vi } from 'vitest';
import { heldSnapshot } from '../data/lineupTestData';
import type { HostedPort } from '../commissioner/hostedRoom';
import { executeWarRoomOffer, type WarRoomCatalog } from './warroomTradeReads';

it('executes a hosted offer with its request key and verifies it in trades.read', async () => {
  const calls: string[] = [];
  const port: HostedPort = vi.fn(async request => {
    calls.push(`${request.command}:${request.mode}`);
    if (request.command === 'trade.offer' && request.mode === 'execute') {
      expect(request.arguments.expected).toBe(12);
      expect(request.arguments.request).toBe('retry-key');
      expect(request.arguments.visibility).toBe('league');
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
