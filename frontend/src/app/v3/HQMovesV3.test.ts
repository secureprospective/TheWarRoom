import { describe, expect, it } from 'vitest';
import { movesLanes } from './HQMovesV3';
import type { HQMove } from '../shell/HQMoves';

const m = (id: string, state: HQMove['state']): HQMove => ({
  id, state, intent: 'x', note: '', at: '', players: [] });

describe('movesLanes', () => {
  it('puts planned and failed moves under Needs you, then waiting, then landed', () => {
    const [need, wait, done] = movesLanes([m('a', 'ready'), m('b', 'needs_you'), m('c', 'handed_off'),
      m('d', 'landed')]);
    expect(need.map(x => x.id)).toEqual(['a', 'b']);
    expect(wait.map(x => x.id)).toEqual(['c']);
    expect(done.map(x => x.id)).toEqual(['d']);
  });
});
