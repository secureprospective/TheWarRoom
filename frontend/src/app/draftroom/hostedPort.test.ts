import { expect, it, vi } from 'vitest';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import { createDraftRoom } from './state';
import { hostedDraftPort } from './hostedPort';

it('uses the hosted native transport for draft reads and owner picks', async () => {
  const transport = vi.fn(async request => ({ data: { command: request.command } }));
  const port = hostedDraftPort(transport);
  const scope = { league: 'native.test', season: 2026, franchise: 'franchise.01',
    at: '0001-01-01T00:00:00Z' };
  const choice = { ...scope, player: '1001', request: 'request-1', expected: 4 };

  expect(await port.read(scope)).toEqual({ command: 'draft.read' });
  expect(await port.pick(choice)).toEqual({ command: 'draft.pick' });
  expect(transport.mock.calls.map(([request]) => [request.command, request.mode])).toEqual([
    ['draft.read', 'read'], ['draft.pick', 'execute'],
  ]);
  expect(transport.mock.calls[1][0].arguments).toMatchObject({ franchise: 'franchise.01', player: '1001' });
});

it('opens a startup draft and keeps the recorded pick visible after refreshing', async () => {
  const league = 'native.startup', franchise = 'franchise.01', player = '1001';
  const at = '0001-01-01T00:00:00Z', openedAt = '2026-05-01T00:00:00Z';
  const first = { round: 1, original: franchise, owner: franchise, slot: 1,
    skipped: false, forfeited: false };
  let recorded = false;
  const transport = vi.fn(async (request: NativeRequest) => {
    const sent = nativeEnvelope(request);
    if (sent.command === 'draft.pick') {
      recorded = 'startup' in sent.arguments && sent.arguments.startup === true;
      return { data: { league, entry: 'pick.1', revision: 5, recordedAt: openedAt, message: 'Pick made.' } };
    }
    if (sent.command !== 'draft.read') throw new Error('Unexpected draft command');
    const scope = sent.arguments;
    const startup = scope.startup === true || (scope.league === league && scope.season === 0);
    return { data: { league: scope.league, season: 2026, startup, revision: recorded ? 5 : 4,
      words: startup ? 'Startup in progress.' : 'No rookie picks were recorded.',
      nextOwnedPick: null, closed: null, rules: [], owners: [], order: [], players: {},
      available: [], checkpoints: [], draft: { start: openedAt, started: startup, exhausted: false,
        onClock: null, picks: startup ? [{ ...first, ...(recorded ? { player } : {}) }] : [] } } };
  });
  const room = createDraftRoom(hostedDraftPort(transport));
  await room.read(league, franchise);
  expect(transport.mock.calls[0][0].arguments).toEqual({ league, franchise, season: 0, at });
  expect(room.peek().draft).toMatchObject({ startup: true, revision: 4, picks: [{ player: '' }] });

  await room.pick(franchise, player);
  expect(room.peek().pick.status).toBe('done');
  expect(transport.mock.calls[1][0]).toMatchObject({ command: 'draft.pick', mode: 'execute',
    arguments: { league, franchise, season: 2026, startup: true, expected: 4, player } });
  expect(room.peek().draft).toMatchObject({ startup: true, revision: 5, picks: [{ player }] });
  expect(transport.mock.calls[2][0]).toEqual({ command: 'draft.read', mode: 'read',
    arguments: { league, franchise, season: 2026, at, startup: true } });

  await room.read('native.rookie', franchise);
  expect(transport.mock.calls[3][0].arguments).toEqual({ league: 'native.rookie', franchise, season: 0, at });
  expect(room.peek().draft?.startup).toBe(false);
});
