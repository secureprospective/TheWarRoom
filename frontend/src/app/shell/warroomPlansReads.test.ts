import { describe, expect, it } from 'vitest';
import { lookingBack } from './warroomPlansReads';

const offer = (id: string, status: string, parties: string[], offeredAt: string) => ({ id, status, parties,
  offeredAt, teams: [['f1', 'Syndicate'], ['f2', 'Iron Curtain'], ['f3', 'Tundra Line']] as [string,
    string][],
  words: '', expires: '', expiresAt: '', proposer: '', accepted: [], details: [], reasons: [] });

describe('lookingBack', () => {
  const all = [
    offer('a', 'approved', ['f1', 'f2'], '2026-08-14T12:00:00Z'),
    offer('b', 'declined', ['f1', 'f3'], '2026-08-02T12:00:00Z'),
    offer('c', 'expired', ['f2', 'f1'], '2026-07-20T12:00:00Z'),
    offer('d', 'offered', ['f1', 'f3'], '2026-08-20T12:00:00Z'),
    offer('e', 'approved', ['f2', 'f3'], '2026-08-21T12:00:00Z'),
  ];
  it.each([
    ['ended offers only, newest first, accepted wording', (r: ReturnType<typeof lookingBack>) =>
      expect(r.offers.map(o => [o.id, o.partner, o.status])).toEqual([
        ['a', 'Iron Curtain', 'accepted'], ['b', 'Tundra Line', 'declined'], ['c', 'Iron Curtain',
          'expired']])],
    ['counterparties busiest first with deal counts', (r: ReturnType<typeof lookingBack>) =>
      expect(r.partners).toEqual([{ name: 'Iron Curtain', talks: 2, deals: 1 },
        { name: 'Tundra Line', talks: 1, deals: 0 }])],
  ])('%s', (_, check) => check(lookingBack('f1', all)));
  it('is empty for a franchise with no finished talks', () =>
    expect(lookingBack('f9', all)).toEqual({ offers: [], partners: [] }));
});
