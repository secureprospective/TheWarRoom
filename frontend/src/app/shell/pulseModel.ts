import type { PulseGame, PulseResults, PulseScore, PulseSeat, PulseStandings, PulseStarter,
  PulseLineupPlayer, PulseBracket } from '../data/nativePulse';

export type PulseFilter = 'all' | 'mine' | 'live' | 'close';
export const PULSE_FILTERS: readonly { key: PulseFilter; label: string }[] = [
  { key: 'all', label: 'ALL' }, { key: 'mine', label: 'MINE' }, { key: 'live', label: 'LIVE' },
  { key: 'close', label: 'CLOSE' },
];
/** A game is close when fewer than ten points separate two complete-enough scores. */
const CLOSE = 10;

export type PulseRow = {
  key: string; game: PulseGame; mine: boolean; division: string;
  home: number | undefined; away: number | undefined;
  decided: boolean; margin: number | undefined; status: string;
};

export function gameKey(game: PulseGame): string { return `${game.away}-${game.home}`; }

/** statusLabel says where a game stands in the league's own words: a result, or still to come.
 *  Only the week the league is playing now can be live; an earlier week is simply waiting. */
export function statusLabel(game: PulseGame, current: boolean, playing = true): string {
  if (game.status === 'Tie') return 'TIED';
  if (game.status !== 'No result yet') return 'FINAL';
  if (!current) return 'WAITING';
  return playing ? 'LIVE' : 'IN PLAY';
}

export function pulseRows(results: PulseResults, mine: string, seats: PulseSeat[],
  current: boolean, playing = true): PulseRow[] {
  const divisionOf = (franchise: string) =>
    seats.find(s => s.franchise === franchise)?.division ?? '';
  return results.games.map(game => {
    const decided = game.homePoints !== undefined && game.awayPoints !== undefined;
    return { key: gameKey(game), game, mine: game.home === mine || game.away === mine,
      division: divisionOf(game.home), home: game.homePoints ?? game.homeProvisional,
      away: game.awayPoints ?? game.awayProvisional, decided,
      margin: decided ? Math.abs(game.homePoints! - game.awayPoints!) : undefined,
      status: statusLabel(game, current, playing) };
  });
}

export function filterCounts(rows: PulseRow[]): Record<PulseFilter, number> {
  return { all: rows.length, mine: rows.filter(r => r.mine).length,
    live: rows.filter(r => r.status === 'LIVE').length,
    close: rows.filter(r => r.margin !== undefined && r.margin < CLOSE).length };
}

export function applyFilter(rows: PulseRow[], filter: PulseFilter): PulseRow[] {
  if (filter === 'mine') return rows.filter(r => r.mine);
  if (filter === 'live') return rows.filter(r => r.status === 'LIVE');
  if (filter === 'close') return rows.filter(r => r.margin !== undefined && r.margin < CLOSE);
  return rows;
}

/** groupRows puts my game first, then every other game under its division heading, the way the
 *  mock groups by side. A league that has told us no divisions gets one unnamed group. */
export function groupRows(rows: PulseRow[]): { heading: string; rows: PulseRow[] }[] {
  const mine = rows.filter(r => r.mine);
  const rest = rows.filter(r => !r.mine);
  const groups: { heading: string; rows: PulseRow[] }[] = [];
  if (mine.length) groups.push({ heading: 'MY MATCHUP', rows: mine });
  for (const division of [...new Set(rest.map(r => r.division))].sort()) {
    const group = rest.filter(r => r.division === division);
    groups.push({ heading: `${division ? division.toUpperCase() : 'OTHER GAMES'} · ${group.length}`,
      rows: group });
  }
  return groups;
}

export type PulseStarterRow = {
  slot: string; player: string; name: string; team?: string; locked?: boolean;
  points: number | undefined; projection: number | undefined; words: string;
  /** The score is settled, not a running total. */
  final: boolean;
};

/** naturalName reads a directory's "Last, First" the way people say it. */
export function naturalName(name: string): string {
  const [last, first, ...more] = name.split(', ');
  return first && !more.length ? `${first} ${last}` : name;
}

const OFFENSE = ['QB', 'RB', 'WR', 'TE', 'FLX', 'K', 'PK', 'OL'];
const DEFENSE = ['DT', 'DE', 'DL', 'LB', 'CB', 'S', 'DB'];
export type StarterPair = { slot: string; left?: PulseStarterRow; right?: PulseStarterRow };

/** pairStarters sets two lineups across the line like for like: offense then defense, one row per
 *  starter at each position either side fills, so a pair never sets a kicker against a running back. */
export function pairStarters(left: PulseStarterRow[], right: PulseStarterRow[]):
  { heading: 'OFFENSE' | 'DEFENSE'; pairs: StarterPair[] }[] {
  const order = [...OFFENSE, ...DEFENSE];
  const rank = (slot: string) => (order.indexOf(slot) + order.length + 1) % (order.length + 1);
  const slots = [...new Set([...left, ...right].map(r => r.slot))]
    .sort((a, b) => rank(a) - rank(b) || a.localeCompare(b));
  const groups: { heading: 'OFFENSE' | 'DEFENSE'; pairs: StarterPair[] }[] =
    [{ heading: 'OFFENSE', pairs: [] }, { heading: 'DEFENSE', pairs: [] }];
  for (const slot of slots) {
    const l = left.filter(r => r.slot === slot);
    const r = right.filter(x => x.slot === slot);
    const into = groups[OFFENSE.includes(slot) ? 0 : 1];
    for (let i = 0; i < Math.max(l.length, r.length); i++)
      into.pairs.push({ slot, left: l[i], right: r[i] });
  }
  return groups.filter(g => g.pairs.length);
}

/** starterRows lines one franchise's starters up in lineup order. Points come from the week score,
 *  which the whole league may read; the NFL team and the lock come from that franchise's own lineup
 *  read, which is private to its team, so they are absent rather than guessed for anyone else. */
export function starterRows(score: PulseScore | undefined, lineup: PulseLineupPlayer[] | undefined,
  projections: Record<string, number>): PulseStarterRow[] {
  return (score?.starters ?? []).map((s: PulseStarter) => {
    const held = lineup?.find(p => p.player === s.player);
    return { slot: s.position ?? '—', player: s.player, name: naturalName(s.name), team: held?.team,
      final: s.points !== undefined,
      locked: held?.locked, points: s.points ?? s.provisional, projection: projections[s.player],
      words: [s.words, s.source, ...Object.entries(s.fieldSources ?? {}).map(([field, owner]) =>
        `${field}: ${owner}`)].filter(Boolean).join(" · ") };
  });
}

export function scoreOf(results: PulseResults | undefined, franchise: string) {
  return results?.scores.find(s => s.franchise === franchise);
}

export type PulseSeedRow = { seed?: number; bye: boolean; kind: string; franchise: string;
  name: string; division: string; rank: number; record: string; pointsWords: string;
  mine: boolean; inField: boolean };

/** seedRows is the Race standings read as a playoff field: the seeds the playoff read has set,
 *  then every other franchise in standings order below the line. */
export function seedRows(standings: PulseStandings, bracket: PulseBracket | undefined,
  conference: string, mine: string): PulseSeedRow[] {
  const field = bracket?.fields.find(f => f.conference === conference);
  const row = (franchise: string) => standings.rows.find(r => r.franchise === franchise);
  const seeded: PulseSeedRow[] = (field?.seeds ?? []).map(s => {
    const r = row(s.franchise);
    return { seed: s.seed, bye: s.bye, kind: s.kind, franchise: s.franchise, name: s.name,
      division: s.division, rank: r?.rank ?? 0, record: r?.record ?? '', mine: s.franchise === mine,
      pointsWords: r?.pointsWords ?? '', inField: true };
  });
  const inField = new Set(seeded.map(s => s.franchise));
  const bubble = standings.rows.filter(r => !inField.has(r.franchise)).map(r => ({
    bye: false, kind: '', franchise: r.franchise, name: r.name, division: r.division, rank: r.rank,
    record: r.record, pointsWords: r.pointsWords, mine: r.franchise === mine, inField: false }));
  return [...seeded, ...bubble];
}

/** conferences are the sides the playoff read has named. A league with none named has one side. */
export function conferencesOf(bracket: PulseBracket | undefined): string[] {
  return bracket?.fields.map(f => f.conference) ?? [];
}

export function divisionGroups(standings: PulseStandings): { division: string;
  rows: PulseStandings['rows'] }[] {
  return [...new Set(standings.rows.map(r => r.division))].sort().map(division => ({
    division, rows: standings.rows.filter(r => r.division === division) }));
}
