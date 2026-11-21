import type { HostedPort } from '../commissioner/hostedRoom';

const clubs = [
  ['KC', '12'], ['BUF', '2'], ['PHI', '21'], ['DAL', '6'], ['SF', '25'], ['SEA', '26'],
  ['DET', '8'], ['GB', '9'], ['BAL', '33'], ['CIN', '4'], ['MIA', '15'], ['NYJ', '20'],
  ['MIN', '16'], ['CHI', '3'], ['LAR', '14'], ['ARI', '22'], ['HOU', '34'], ['IND', '11'],
  ['PIT', '23'], ['CLE', '5'], ['DEN', '7'], ['LV', '13'], ['TB', '27'], ['ATL', '1'],
];

const baseState = { possession: '12', down: 2, distance: 7, yardLine: 57, toEndzone: 57 };

type RawPlay = {
  id: string; sequence: number; text: string; clock: string; period: number; at: string;
  homeScore: number; awayScore: number;
  start: typeof baseState; end: typeof baseState; participants: string[];
};

const addSec = (iso: string, sec: number) => new Date((Date.parse(iso) || 0) + sec * 1000).toISOString();

type MatchCast = {
  wr: string; te: string; rb: string; oppWr: string; db: string;
  bigYds: number; bigKind: 'td' | 'int' | 'gain' | 'fg';
};

const casts: MatchCast[] = [
  { wr: 'R. Rice', te: 'T. Kelce', rb: 'I. Pacheco', oppWr: 'D. Kincaid', db: 'T. McDuffie',
    bigYds: 22, bigKind: 'td' },
  { wr: 'A. Brown', te: 'D. Goedert', rb: 'S. Barkley', oppWr: 'C. Lamb', db: 'D. Slay',
    bigYds: 26, bigKind: 'int' },
  { wr: 'B. Aiyuk', te: 'G. Kittle', rb: 'C. McCaffrey', oppWr: 'D. Metcalf', db: 'F. Warner',
    bigYds: 29, bigKind: 'gain' },
  { wr: 'A. St. Brown', te: 'S. LaPorta', rb: 'J. Gibbs', oppWr: 'J. Reed', db: 'B. Branch',
    bigYds: 24, bigKind: 'td' },
  { wr: 'Z. Flowers', te: 'M. Andrews', rb: 'D. Henry', oppWr: 'J. Chase', db: 'K. Hamilton',
    bigYds: 51, bigKind: 'fg' },
  { wr: 'T. Hill', te: 'J. Smith', rb: 'D. Achane', oppWr: 'G. Wilson', db: 'J. Ramsey',
    bigYds: 27, bigKind: 'gain' },
  { wr: 'J. Jefferson', te: 'T. Hockenson', rb: 'A. Jones', oppWr: 'D. Moore', db: 'B. Murphy',
    bigYds: 33, bigKind: 'td' },
  { wr: 'P. Nacua', te: 'T. Higbee', rb: 'K. Williams', oppWr: 'M. Harrison', db: 'K. Curl',
    bigYds: 23, bigKind: 'gain' },
];

const demoPlays = (i: number, home: string, away: string, homeId: string, awayId: string,
  st: typeof baseState, kickoff: string, homeScore: number, awayScore: number): RawPlay[] => {
  const c = casts[i % casts.length];
  const baseAt = addSec(kickoff, 1800 + i * 35);
  const sHome = (toEz: number, down = 2, dist = 7) =>
    ({ possession: homeId, down, distance: dist, yardLine: toEz, toEndzone: toEz });
  const sAway = (toEz: number, down = 2, dist = 7) =>
    ({ possession: awayId, down, distance: dist, yardLine: toEz, toEndzone: toEz });
  const bigLine = c.bigKind === 'td'
    ? { text: `${home} pass deep middle to ${c.wr} for ${c.bigYds} yards, TOUCHDOWN.`,
      start: sHome(c.bigYds, 2, 6), end: sAway(75, 1, 10), hs: homeScore, as: awayScore }
    : c.bigKind === 'int'
      ? { text: `${away} pass short right intended for ${c.oppWr} INTERCEPTED by ${c.db} at ${home} 32.`,
        start: sAway(40, 3, 8), end: sHome(68, 1, 10), hs: homeScore, as: awayScore }
      : c.bigKind === 'fg'
        ? { text: `${home} ${c.bigYds} yard field goal is GOOD.`,
          start: sHome(33, 4, 6), end: sAway(75, 1, 10), hs: homeScore, as: awayScore }
        : { text: `${home} pass deep middle to ${c.wr} for ${c.bigYds} yards`,
          start: sHome(54, 1, 10), end: sHome(54 - c.bigYds, 1, 10), hs: homeScore, as: awayScore };
  const lastToEz = Math.max(12, Math.min(82, st.toEndzone));
  const lastStart = { possession: st.possession, down: st.down, distance: st.distance,
    yardLine: lastToEz + 5, toEndzone: lastToEz + 5 };
  const lastEnd = { possession: st.possession, down: Math.min(4, st.down + 1),
    distance: Math.max(1, st.distance - 5), yardLine: lastToEz, toEndzone: lastToEz };
  const lastOff = st.possession === homeId ? home : away;
  const lastRec = i === 0 ? 'N. Gray' : st.possession === homeId ? c.te : c.oppWr;
  const prevHome = Math.max(0, homeScore - (c.bigKind === 'td' ? 7 : c.bigKind === 'fg' ? 3 : 0));
  const openerYds = i % 3 === 0 ? 21 + (i % 4) : 8;
  const openerText = openerYds >= 20
    ? `${home} pass deep right to ${c.te} for ${openerYds} yards`
    : `${home} pass short right to ${c.te} for 8 yards`;
  const lines = [
    { text: openerText,
      start: sHome(58, 1, 10), end: sHome(58 - openerYds, 1, 10), hs: prevHome, as: awayScore },
    { text: `${away} rush left guard for 4 yards`,
      start: sAway(62, 1, 10), end: sAway(58, 2, 6), hs: prevHome, as: awayScore },
    bigLine,
    { text: `${home} rush up the middle ${c.rb} for 6 yards`,
      start: sHome(48, 1, 10), end: sHome(42, 2, 4), hs: homeScore, as: awayScore },
    { text: `${away} pass short left to ${c.oppWr} for 11 yards`,
      start: sAway(55, 2, 8), end: sAway(44, 1, 10), hs: homeScore, as: awayScore },
    { text: `${lastOff} pass short middle to ${lastRec} for 5 yards`,
      start: lastStart, end: lastEnd, hs: homeScore, as: awayScore },
  ];
  return lines.map((line, idx) => ({
    id: `p${homeId}-${idx + 1}`, sequence: idx + 1, text: line.text,
    clock: `8:${String(42 - idx * 7).padStart(2, '0')}`, period: 2,
    at: addSec(baseAt, idx * 45), homeScore: line.hs, awayScore: line.as,
    start: line.start, end: line.end, participants: [],
  }));
};

/** wallGame builds game i of the fixture board (pairs of clubs); `status` sets live, pre or post. */
export function wallGame(i: number, status: 'in' | 'pre' | 'post' = 'in',
  kickoff = '2026-11-22T19:00:00Z') {
  const pair = i % (clubs.length / 2);
  const [home, away] = [clubs[pair * 2], clubs[pair * 2 + 1]];
  const poss = i % 2 ? away[1] : home[1];
  const st = { ...baseState, possession: poss, yardLine: 25 + (i % 6) * 9,
    toEndzone: 75 - (i % 6) * 9, down: 1 + (i % 3) };
  const homeScore = status === 'pre' ? 0 : 7 + (i % 6) * 3;
  const awayScore = status === 'pre' ? 0 : 3 + (i % 4) * 3;
  const plays = status === 'pre' ? []
    : demoPlays(i, home[0], away[0], home[1], away[1], st, kickoff, homeScore, awayScore);
  const last = plays.length ? plays[plays.length - 1] : null;
  return {
    season: 2026, week: 11, seasonType: 2, id: `40${i}`, kickoff,
    home: { id: home[1], abbreviation: home[0], score: homeScore, record: '8-2' },
    away: { id: away[1], abbreviation: away[0], score: awayScore, record: '6-4' },
    status, clock: status === 'post' ? '0:00' : status === 'pre' ? '' : '8:42',
    period: status === 'pre' ? 0 : status === 'post' ? 4 : 2,
    state: status === 'in' ? (last?.end ?? st) : null,
    last,
    plays, box: [],
  };
}

export type FixtureGame = ReturnType<typeof wallGame>;

/** demoBoard builds a 12-game NFL slate for ?demo: 8 live, 2 upcoming, 2 final. */
export function demoBoard(): FixtureGame[] {
  const liveKick = '2026-11-22T18:00:00Z';
  const lateKick = '2026-11-22T21:25:00Z';
  const earlyKick = '2026-11-22T14:30:00Z';
  return [
    ...[0, 1, 2, 3, 4, 5, 6, 7].map(i => wallGame(i, 'in', liveKick)),
    wallGame(8, 'pre', lateKick),
    wallGame(9, 'pre', lateKick),
    wallGame(10, 'post', earlyKick),
    wallGame(11, 'post', earlyKick),
  ];
}

const reply = (data: unknown) => ({ data, football: '', writable: false, label: 'Server' });

/** fakeWallPort answers the two NFL reads from a fixed board (the style of phone/LiveTab.test.tsx). */
export function fakeWallPort(games: FixtureGame[], league = 'league') {
  const port: HostedPort = async request => {
    if (request.command === 'nfl.games.read') return reply({
      league, board: { stale: false, source: 'ESPN', at: '2026-11-22T19:00:00Z', failure: '',
        games: games.map(g => ({ ...g, plays: [], last: null })) },
    });
    if (request.command === 'nfl.game.read') {
      const id = (request.arguments as { game: string }).game;
      return reply({ league, game: games.find(g => g.id === id), at: '2026-11-22T19:00:00Z',
        stale: false, words: '' });
    }
    throw new Error(`Unexpected read ${request.command}`);
  };
  return port;
}

type DemoEvent = {
  gameIndex: number; side: 'home' | 'away'; points: number;
  text: string; toEndzone: number; clock: string; turnover?: boolean;
};

const demoScript: DemoEvent[] = [
  { gameIndex: 0, side: 'home', points: 7, toEndzone: 8, clock: '7:54',
    text: 'KC pass short right to T. Kelce for 8 yards, TOUCHDOWN.' },
  { gameIndex: 1, side: 'away', points: 0, toEndzone: 22, clock: '7:31', turnover: true,
    text: 'DAL pass deep left intended for C. Lamb INTERCEPTED by D. Slay at PHI 18.' },
  { gameIndex: 2, side: 'home', points: 7, toEndzone: 24, clock: '7:10',
    text: 'SF rush right end C. McCaffrey for 24 yards, TOUCHDOWN.' },
  { gameIndex: 4, side: 'home', points: 3, toEndzone: 34, clock: '6:48',
    text: 'BAL 52 yard field goal is GOOD.' },
  { gameIndex: 3, side: 'home', points: 7, toEndzone: 31, clock: '6:19',
    text: 'DET pass deep middle to A. St. Brown for 31 yards, TOUCHDOWN.' },
  { gameIndex: 5, side: 'away', points: 0, toEndzone: 38, clock: '5:52', turnover: true,
    text: 'NYJ pass short middle intended for G. Wilson INTERCEPTED by J. Ramsey at MIA 40.' },
];

export const demoStepMs = 20_000;

/** createDemoFeed serves a live 12-game fixture board that emits a scripted scoring or turnover play
 *  every `stepMs` (default 20 s) to exercise the spotlight and Big Plays rail in dev `?demo`. */
export function createDemoFeed(stepMs = demoStepMs, league = 'league') {
  const games = demoBoard();
  const listeners = new Set<() => void>();
  let step = 0;
  let timer: ReturnType<typeof setInterval> | undefined;

  const advance = () => {
    const ev = demoScript[step % demoScript.length];
    step += 1;
    const g = games[ev.gameIndex];
    if (!g) return;
    const nextSeq = (g.plays[g.plays.length - 1]?.sequence ?? 0) + 1;
    const offId = ev.side === 'home' ? g.home.id : g.away.id;
    const defId = ev.side === 'home' ? g.away.id : g.home.id;
    const homeScore = g.home.score + (ev.side === 'home' ? ev.points : 0);
    const awayScore = g.away.score + (ev.side === 'away' ? ev.points : 0);
    const start = { possession: offId, down: 2, distance: 6,
      yardLine: ev.toEndzone, toEndzone: ev.toEndzone };
    const end = { possession: ev.turnover || ev.points > 0 ? defId : offId, down: 1, distance: 10,
      yardLine: ev.turnover ? 32 : 75, toEndzone: ev.turnover ? 68 : 75 };
    const latestAt = games.reduce((mx, gm) =>
      gm.plays.reduce((m, p) => Math.max(m, Date.parse(p.at) || 0), mx), Date.now());
    const play: RawPlay = {
      id: `demo-${g.id}-${nextSeq}`, sequence: nextSeq, text: ev.text,
      clock: ev.clock, period: g.period, at: new Date(latestAt + 1000).toISOString(),
      homeScore, awayScore, start, end, participants: [],
    };
    g.home = { ...g.home, score: homeScore };
    g.away = { ...g.away, score: awayScore };
    g.clock = ev.clock;
    g.state = end;
    g.plays = [...g.plays, play];
    g.last = play;
    for (const fn of listeners) fn();
  };

  const subscribe = (load: () => void) => {
    listeners.add(load);
    if (!timer && stepMs > 0) timer = setInterval(advance, stepMs);
    return () => {
      listeners.delete(load);
      if (listeners.size === 0 && timer) { clearInterval(timer); timer = undefined; }
    };
  };

  const port: HostedPort = async request => {
    if (request.command === 'nfl.games.read') return reply({
      league, board: { stale: false, source: 'ESPN', at: new Date().toISOString(), failure: '',
        games: games.map(g => ({ ...g, plays: [], last: null })) },
    });
    if (request.command === 'nfl.game.read') {
      const id = (request.arguments as { game: string }).game;
      const found = games.find(g => g.id === id);
      return reply({ league, game: found ? { ...found, plays: [...found.plays] } : null,
        at: new Date().toISOString(), stale: false, words: '' });
    }
    throw new Error(`Unexpected read ${request.command}`);
  };

  return { port, subscribe, advance, games };
}
