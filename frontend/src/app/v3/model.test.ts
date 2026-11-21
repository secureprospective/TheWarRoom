import { describe, expect, it } from 'vitest';
import { lanesOf, type Brief, type V3League } from './model';

const now = Date.parse('2026-11-20T18:00:00Z');
const L = { id: 'lg', name: 'Test League', app: true, franchise: 'f1', franchiseName: 'Mine', bd: 'TL',
  c: 'red', commish: false, team: true } as unknown as V3League;
const offer = (id: string, status: string, proposer: string, accepted: string[], offeredAt: string) => ({
  id, status, proposer, accepted, offeredAt, expiresAt: '2026-11-22T18:00:00Z', parties: ['f1', 'f2'],
  teams: [['f1', 'Mine'], ['f2', 'Boise']], words: 'awaiting owner · Boise', assets: [] });
const brief = {
  brief: {
    at: {}, errors: {},
    trades: { league: 'lg', franchise: 'f1', revision: 1, notes: [], offers: [
      offer('t1', 'offered', 'f1', ['f1'], '2026-11-19T18:00:00Z'), // mine, waiting on Boise
      offer('t2', 'offered', 'f2', ['f2'], '2026-11-19T18:00:00Z'), // incoming: Needs you, not here
      offer('t3', 'vetoed', 'f1', ['f1', 'f2'], '2026-11-18T18:00:00Z'), // landed, bad
      offer('t4', 'approved', 'f1', ['f1', 'f2'], '2026-10-01T18:00:00Z'), // too old
    ] },
    bids: { league: 'lg', franchise: 'f1', revision: 1, notes: [], limit: '', players: [], awards: [],
      results: [
      { player: 'p1', name: 'Zack Baun', winner: 'f1', contract: '$4.2M',
        decided: '2026-11-19T12:00:00Z', yours: '' },
      { player: 'p2', name: 'Brian Branch', winner: 'f2', contract: '', decided: '2026-11-19T12:00:00Z',
        yours: 'Your bid was not awarded' },
      { player: 'p3', name: 'Nobody', winner: 'f2', contract: '',
        decided: '2026-11-19T12:00:00Z', yours: '' },
    ] },
  },
  waivers: { league: 'lg', franchise: 'f1', revision: 1, words: '', method: '', order: [],
    players: [{ window: 'w1', player: 'p9', name: 'Chase Brown', status: 'On waivers',
      words: 'On waivers · Wed',
      closesAt: '2026-11-21T09:00:00Z' }],
    claims: [{ id: 'c1', player: 'p9', words: 'Claim recorded privately' }] },
} as unknown as Brief;

describe('docket lanes', () => {
  const rows = lanesOf([L], { lg: brief }, now);
  const by = (lane: string) => rows.filter(r => r.lane === lane).map(r => r.t);
  it('puts my open offer and pending claim in Waiting on, never an incoming offer', () => {
    expect(by('wait')).toEqual(['Your offer to Boise', 'Waiver claim: Chase Brown']);
  });
  it('lands finished trades and my awards from the last week, marking losses', () => {
    expect(by('land')).toEqual(['Trade with Boise: vetoed', 'Won Zack Baun for $4.2M',
      'Brian Branch: Your bid was not awarded']);
    expect(rows.filter(r => r.bad).map(r => r.t)).toEqual(['Trade with Boise: vetoed',
      'Brian Branch: Your bid was not awarded']);
  });
  it('leaves a teamless league out', () => {
    expect(lanesOf([{ ...L, franchise: '' }], { lg: brief }, now)).toEqual([]);
  });
});
