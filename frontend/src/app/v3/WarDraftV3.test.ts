import { describe, expect, it } from 'vitest';
import { madePicks } from './WarDraftV3';
import type { Draft } from '../draftroom/draft';

const pick = (round: number, slot: number, player: string) => ({ round, slot, original: `f${slot}`,
  owner: `f${slot}`, extra: 0, player, skipped: false, forfeited: false, madeAt: '', contract: null,
  how: 'owner' as const, changed: false });

describe('madePicks', () => {
  it('lists picks made newest first with the team and player', () => {
    const draft = { owners: [{ id: 'f1', name: 'One' }, { id: 'f2', name: 'Two' }],
      picks: [pick(1, 1, 'p1'), pick(1, 2, 'p2'), pick(2, 1, '')],
      players: { p1: { id: 'p1', name: 'Ace', position: 'RB', team: 'ARI' },
        p2: { id: 'p2', name: 'Bo', position: 'WR', team: 'BUF' } } } as unknown as Draft;
    expect(madePicks(draft).map(p => `${p.no} ${p.team} ${p.player}`))
      .toEqual(['1.02 Two Bo', '1.01 One Ace']);
  });
});
