import { expect, it } from 'vitest';
import {
  parseHostedBids,
  parseHostedResults,
  parseHostedStandings,
  parseHostedTrades,
} from './hostedMarkets';

// The league server sends its live-score evidence on results and standings, and provisional points on a game
// being played. HQ reads both in its one refresh; a refused field there leaves the lineup read-only.
const evidence = { sources: ['espn'], reduced: false, stale: false, live: true, season: 2026, week: 10,
  asOf: '2026-11-15T18:00:00Z', source: 'ESPN live', provisional: true };

it('keeps the auction close instant from the owner-only bids read', () => {
  const close = '2026-11-18T18:00:00Z';
  const read = parseHostedBids({ league: 'l', franchise: 'f', revision: 4, season: 2026,
    mode: 'open', activeBids: 1, activeLimit: '3', notes: [], awards: [], players: [{ player: 'p',
      name: 'Player', mode: 'open', modeWords: 'Open', youLead: true, lateAt: '', lateWords: '',
      lateNow: false, closesAt: close, closes: 'Closes at 1 PM' }] });
  expect(read.players[0].closesAt).toBe(close);
});

it('reads live results with score evidence and provisional points', () => {
  const read = parseHostedResults({ league: 'l', season: 2026, week: 10, revision: 4, published: true,
    words: 'Week 10', missed: [], evidence, games: [{ week: 10, home: 'a', away: 'b', homeName: 'A',
      awayName: 'B', status: 'live', homeProvisional: 41.5, awayProvisional: 38.2, words: 'A leads',
      why: [] }] });
  expect(read.games).toEqual([{ names: 'A v B', words: 'A leads', why: [] }]);
});

it('shows the public reason for a 24-hour trade review hold', () => {
  const reason = 'Updated injury information needs another review day';
  const read = parseHostedTrades({ league: 'l', viewer: 'f', revision: 5, rules: [], notes: [], offers: [{
    vetoNumber: 0, appealedVetoNumber: 0, assetDetails: [{ fromTeam: 'A', toTeam: 'B', label: 'Pick' }],
    previous: '', id: 'trade-1', proposer: 'a', parties: ['a', 'b'], assets: [{ from: 'a', to: 'b',
      season: 2027, round: 1, original: 'a' }], accepted: ['a', 'b'], policy: {},
    offeredAt: '2026-11-19T00:00:00Z',
    expiresAt: '2026-11-20T00:00:00Z', status: 'review', words: 'Waiting for trade review', votes: [],
    holds: [{ who: 'commissioner', at: '2026-11-19T00:00:00Z', reason }], recusals: [], appeals: [],
    overrides: [], expires: 'Nov 20, 2026 at 12:00 AM UTC',
  }] });
  expect(read.offers[0].reasons).toContain(`Review held for 24 hours: ${reason}`);
});

it('reads standings that carry score evidence', () => {
  const read = parseHostedStandings({ league: 'l', name: 'L', season: 2026, revision: 4, status: 'live',
    words: 'Through week 9', through: 9, decided: 144, pending: 0, criteria: [], noData: [], evidence,
    computed: [{ rank: 1, franchise: 'a', name: 'A', wins: 9, losses: 0, ties: 0, record: '9-0',
      pointsFor: 1000, pointsWords: '1000.0', why: [] }] });
  expect(read.computed).toEqual([{ rank: 1, franchise: 'a', name: 'A', record: '9-0',
    points: '1000.0', why: [] }]);
});

it("reads standings rows that carry a streak, so the lineup stays editable", () => {
  const read = parseHostedStandings({ league: "l", name: "L", season: 2026, revision: 4, status: "live",
    words: "Through week 10", through: 10, decided: 160, pending: 0, criteria: [], noData: [], evidence,
    computed: [{ rank: 1, franchise: "a", name: "A", wins: 9, losses: 1, ties: 0, record: "9-1",
      streak: "W3", pointsFor: 1000, pointsWords: "1000.0", why: [] }] });
  expect(read.computed[0]?.record).toBe("9-1");
});
