import type { HostedPort } from '../commissioner/hostedRoom';

const clubs = [['KC', '12'], ['BUF', '2'], ['PHI', '21'], ['DAL', '6'], ['SF', '25'], ['SEA', '26'],
  ['DET', '8'], ['GB', '9'], ['BAL', '33'], ['CIN', '4'], ['MIA', '15'], ['NYJ', '20']];
const state = { possession: '12', down: 2, distance: 7, yardLine: 57, toEndzone: 57 };

const demoPlays = (home: string, away: string, homeId: string, awayId: string,
  st: typeof state, kickoff: string, homeScore: number) => [
  `${home} pass short right to T. Kelce for 8 yards`,
  `${away} rush left guard for 4 yards`,
  `${home} pass deep middle to R. Rice for 22 yards`,
  `${home} rush up the middle for 6 yards`,
  `${away} pass short left to D. Kincaid for 11 yards`,
  `${home} pass short middle to N. Gray for 5 yards`,
].map((text, idx) => ({
  id: `p${homeId}-${idx + 1}`, sequence: idx + 1, text, clock: `8:${42 - idx * 7}`, period: 2,
  at: kickoff, homeScore, awayScore: 3,
  start: { ...st, possession: idx % 2 === 1 ? awayId : homeId }, end: st, participants: [],
}));

/** wallGame builds game i of the fixture board (pairs of clubs); `status` sets live, pre or post. */
export function wallGame(i: number, status: 'in' | 'pre' | 'post' = 'in',
  kickoff = '2026-11-22T19:00:00Z') {
  const [home, away] = [clubs[i * 2], clubs[i * 2 + 1]];
  const poss = i % 2 ? away[1] : home[1];
  const st = { ...state, possession: poss, yardLine: 25 + i * 9, toEndzone: 75 - i * 9, down: 1 + (i % 3) };
  const plays = demoPlays(home[0], away[0], home[1], away[1], st, kickoff, 7 + i);
  return { season: 2026, week: 11, seasonType: 2, id: `40${i}`, kickoff,
    home: { id: home[1], abbreviation: home[0], score: 7 + i, record: '8-2' },
    away: { id: away[1], abbreviation: away[0], score: 3, record: '6-4' },
    status, clock: '8:42', period: 2, state: st, last: plays[plays.length - 1], plays, box: [] };
}
const reply = (data: unknown) => ({ data, football: '', writable: false, label: 'Server' });

/** fakeWallPort answers the two NFL reads from a fixed board (the style of phone/LiveTab.test.tsx). */
export function fakeWallPort(games: ReturnType<typeof wallGame>[], league = 'league') {
  const port: HostedPort = async request => {
    if (request.command === 'nfl.games.read') return reply({ league, board: { stale: false, source: 'ESPN',
      at: '2026-11-22T19:00:00Z', failure: '', games: games.map(g => ({ ...g, plays: [], last: null })) } });
    if (request.command === 'nfl.game.read') {
      const id = (request.arguments as { game: string }).game;
      return reply({ league, game: games.find(g => g.id === id), at: '2026-11-22T19:00:00Z',
        stale: false, words: '' });
    }
    throw new Error(`Unexpected read ${request.command}`);
  };
  return port;
}
