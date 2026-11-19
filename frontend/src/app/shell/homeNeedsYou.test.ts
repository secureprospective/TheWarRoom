import { describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { People } from '../commissioner/people';
import type { HomeBrief } from './homeReads';
import { gatherNeedsYou } from './homeNeedsYou';
import type { LeagueChoice } from './leaguePicker';

const leagues: LeagueChoice[] = [
  { id: 'a', name: 'Alpha', teams: 12, app: true },
  { id: 'b', name: 'Beta', teams: 12, app: true },
  { id: 'mirror', name: 'Reference', teams: 12, app: false },
];
const at = (hour: number) => `2026-11-18T${String(hour).padStart(2, '0')}:00:00Z`;
const person = (league: string, mine: string, yourRole = 'Owner'): People =>
  ({ league, yourFranchise: mine, yourRole, mayChangeRules: ['Commissioner', 'Co-commissioner',
    'Deputy'].includes(yourRole) } as People);
const brief = (values: Partial<HomeBrief> = {}) =>
  ({ at: {}, errors: {}, ...values } as HomeBrief);
const waitingTrade = (id: string, mine: string, expiresAt: string) => ({
  id, status: 'offered', parties: [mine, 'other'], accepted: ['other'], expiresAt,
  teams: [[mine, `${mine} team`], ['other', 'Other team']],
});

const port = vi.fn() as unknown as HostedPort;

describe('gatherNeedsYou', () => {
  it('merges two leagues, sorts deadlines, and reads each brief once', async () => {
    const people = vi.fn(async (league: string) => person(league, `${league}-owner`,
      league === 'a' ? 'Commissioner' : 'Owner'));
    const reads = new Map<string, HomeBrief>([
      ['a', brief({
        trades: { league: 'a', franchise: 'a-owner', offers: [
          waitingTrade('a-trade', 'a-owner', at(12)),
          { ...waitingTrade('not-mine', 'a-owner', at(6)), parties: ['other', 'someone'] },
        ] } as never,
        bids: { league: 'a', franchise: 'a-owner', players: [{ player: 'p1', name: 'Player One',
          closes: 'in ten hours', closesAt: at(10), own: { id: 'bid-1', words: 'bid' } }] } as never,
        pulse: { week: 4, fixtures: [{ week: 4, kickoff: at(8) }] } as never,
        results: { scores: [{ franchise: 'a-owner', lineupStatus: 'missed',
          lineupWords: 'No lineup was set' }] } as never,
        calendar: [{ id: 'cut', label: 'Window', closes: at(9) }] as never,
        jobs: { league: 'a', jobs: [
          { id: 'blocked', name: 'Record the season', state: 'blocked', words: 'Waiting', closes: '' },
          { id: 'live', name: 'Settle review', state: 'live', words: 'Due soon', closes: at(11) },
        ] } as never,
      })],
      ['b', brief({ trades: { league: 'b', franchise: 'b-owner', offers: [
        waitingTrade('b-trade', 'b-owner', at(7)),
      ] } as never })],
    ]);
    const readBrief = vi.fn(async (league: string, _mine: string, _port: HostedPort,
      _owner: boolean, _commissioner: boolean) => reads.get(league)!);

    const got = await gatherNeedsYou(leagues, port, {
      readPeople: people, readBrief, now: Date.parse(at(6)),
    });

    expect(people).toHaveBeenCalledTimes(2);
    expect(readBrief).toHaveBeenCalledTimes(2);
    expect(readBrief.mock.calls.map(([league, mine, _port, owner, commissioner]) =>
      [league, mine, owner, commissioner])
      .sort(([a], [b]) => String(a).localeCompare(String(b))))
      .toEqual([['a', 'a-owner', true, true], ['b', 'b-owner', true, false]]);
    expect(got.map(({ leagueName, kind, deadline }) => [leagueName, kind, deadline])).toEqual([
      ['Beta', 'trade', at(7)], ['Alpha', 'lineup', at(8)], ['Alpha', 'deadline', at(9)],
      ['Alpha', 'auction', at(10)], ['Alpha', 'commissioner', at(11)], ['Alpha', 'trade', at(12)],
      ['Alpha', 'commissioner', undefined],
    ]);
    expect(got.every(entry => !entry.words.includes('\n'))).toBe(true);
    expect(got.find(entry => entry.kind === 'lineup')?.screen)
      .toEqual({ node: 'hq', workspace: 'lineup-and-roster' });
  });

  it('keeps one failed league visible without hiding a successful league', async () => {
    const readBrief = vi.fn(async (league: string) => {
      if (league === 'a') throw new Error('server offline');
      return brief({ trades: { league: 'b', franchise: 'b-owner', offers: [
        waitingTrade('b-trade', 'b-owner', at(9)),
      ] } as never });
    });
    const got = await gatherNeedsYou(leagues, port, {
      readPeople: async league => person(league, `${league}-owner`), readBrief,
      now: Date.parse(at(6)),
    });

    expect(got.filter(entry => entry.leagueId === 'a')).toMatchObject([
      { kind: 'unavailable', words: 'server offline' },
    ]);
    expect(got.some(entry => entry.leagueId === 'b' && entry.kind === 'trade')).toBe(true);
    expect(readBrief).toHaveBeenCalledTimes(2);
  });

  it('keeps working trade items when the bids source fails', async () => {
    const words = 'Bids source refused: owner scope unavailable';
    const got = await gatherNeedsYou([leagues[0]], port, {
      readPeople: async () => person('a', 'a-owner'),
      readBrief: async () => brief({
        trades: { league: 'a', franchise: 'a-owner', offers: [
          waitingTrade('a-trade', 'a-owner', at(9)),
        ] } as never,
        errors: { bids: words },
      }),
      now: Date.parse(at(6)),
    });

    expect(got).toMatchObject([
      { kind: 'trade', leagueId: 'a', id: 'a:trade:a-trade' },
      { kind: 'unavailable', leagueId: 'a', words },
    ]);
  });

  it('includes commissioner work only for commissioner roles, not a rules deputy', async () => {
    const readBrief = vi.fn(async (league: string) => brief({
      jobs: { league, jobs: [{ id: 'job', name: 'Review the season', state: 'blocked',
        words: 'Waiting on the commissioner', closes: '' }] } as never,
    }));
    const got = await gatherNeedsYou(leagues, port, {
      readPeople: async league => person(league, `${league}-owner`,
        league === 'a' ? 'Commissioner' : 'Deputy'), readBrief,
      now: Date.parse(at(6)),
    });

    expect(got.filter(entry => entry.kind === 'commissioner').map(entry => entry.leagueId)).toEqual(['a']);
  });

  it('does not turn a normal offseason with no open week into an unavailable item', async () => {
    const got = await gatherNeedsYou([leagues[0]], port, {
      readPeople: async () => person('a', 'a-owner'),
      readBrief: async () => brief({ errors: { results: 'No league week is open',
        calendar: 'The league season is not known' } }),
      now: Date.parse(at(6)),
    });

    expect(got).toEqual([]);
  });
});
