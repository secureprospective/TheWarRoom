import { afterEach, expect, it, vi } from 'vitest';
import { poolProvider, parseBoard } from '../pool/provider';
import * as releases from '../contracts/releases';
import * as contracts from '../contracts/contractActState';
import wire from '../pool/wire.testdata.json';
import { readHQMovePlans, mergeHQMoves } from './hqMovePlans';
import type { HQMove } from './HQMoves';

afterEach(() => vi.restoreAllMocks());
it('joins the existing bid, release and contract plan reads without hiding a failed source', async () => {
  const board = parseBoard(wire.board);
  const mine = board.value.plans[0].franchiseId;
  vi.spyOn(poolProvider, 'plans').mockResolvedValue(board);
  vi.spyOn(releases, 'releaseAction').mockRejectedValue(new Error('Release read offline'));
  vi.spyOn(contracts, 'contractActAction').mockRejectedValue(new Error('Contract read offline'));
  const value = await readHQMovePlans(mine);
  expect(value.moves.map(m => m.id)).toContain(board.value.plans[0].id);
  expect(value.moves[0].players).toContain(board.value.plans[0].playerId);
  expect(value.errors).toEqual(['Error: Release read offline', 'Error: Contract read offline']);
});
it('deduplicates move identities and does not count withdrawn plans as needing action', () => {
  const held: HQMove = { id: 'plan', state: 'ready', intent: 'roster.ir', players: [], note: '',
    at: '2026-10-31T10:00:00Z' };
  expect(mergeHQMoves([held], [{ ...held, state: 'landed' }])).toHaveLength(1);
  expect(mergeHQMoves([held], [{ ...held, state: 'landed' }])[0].state).toBe('landed');
  expect(mergeHQMoves([{ ...held, state: 'withdrawn' }], [])).toEqual([]);
});
