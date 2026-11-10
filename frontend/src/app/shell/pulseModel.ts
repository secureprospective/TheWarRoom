import type { PulseGame, PulseResults, PulseScore, PulseSeat, PulseStandings, PulseStarter,
  PulseLineupPlayer, PulseBracket, PulseRevision } from '../data/nativePulse';

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
  line?: Record<string, number>;
  revision?: PulseRevision;
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
const SLOT_ORDER = [...OFFENSE, ...DEFENSE];
/** bySlot orders positions the way a lineup reads: offense, then defense, unknown slots last. */
export function bySlot(a: string, b: string): number {
  const rank = (slot: string) => (SLOT_ORDER.indexOf(slot) + SLOT_ORDER.length + 1) % (SLOT_ORDER.length + 1);
  return rank(a) - rank(b) || a.localeCompare(b);
}
export type StarterPair = { slot: string; left?: PulseStarterRow; right?: PulseStarterRow };

/** pairStarters sets two lineups across the line like for like: offense then defense, one row per
 *  starter at each position either side fills, so a pair never sets a kicker against a running back. */
export function pairStarters(left: PulseStarterRow[], right: PulseStarterRow[]):
  { heading: 'OFFENSE' | 'DEFENSE'; pairs: StarterPair[] }[] {
  const slots = [...new Set([...left, ...right].map(r => r.slot))].sort(bySlot);
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
      line: s.line, revision: s.revision,
      words: [s.words, s.source, ...Object.entries(s.fieldSources ?? {}).map(([field, owner]) =>
        `${field}: ${owner}`)].filter(Boolean).join(" · ") };
  });
}

export type StatChange = { player: string; name: string; left: boolean; stats: string;
  was: number; now: number; delta: number };

/** statChanges lists the starters whose points the final statistics moved, biggest move first. */
export function statChanges(left: PulseStarterRow[], right: PulseStarterRow[]): StatChange[] {
  const side = (rows: PulseStarterRow[], isLeft: boolean) => rows.flatMap(r => r.revision ? [{
    player: r.player, name: r.name, left: isLeft, was: r.revision.was, now: r.revision.now,
    delta: r.revision.now - r.revision.was,
    // A scoring rule's label ends with the range it prices; the row already says the numbers.
    stats: r.revision.stats.map(s => `${s.label.replace(/ \([^)]*\)$/, '').toLowerCase()} ${s.was} → ${s.now}`)
      .join(' · '),
  }] : []);
  return [...side(left, true), ...side(right, false)]
    .sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta));
}

/** changePoints writes points to the cent a stat change can move, dropping a spare zero. */
export function changePoints(value: number, signed = false): string {
  const words = Math.abs(value).toFixed(2).replace(/0$/, '');
  return `${value < 0 ? '−' : signed ? '+' : ''}${words}`;
}

/** changeVerdict says whether one starter's change decided the matchup. margin is the left side's
 *  final lead; a recorded week's result is the source's and no change of ours can move it. */
export function changeVerdict(change: StatChange, margin: number | undefined, recorded: boolean,
  names: { left: string; right: string; mine: boolean }): string {
  if (recorded) return 'recorded result stands';
  if (margin === undefined) return 'week not final';
  // Points are held to the ten-thousandth; anything finer is float noise, not a lead.
  const lead = (m: number) => Math.sign(Math.round(m * 1e4));
  const now = lead(margin);
  const then = lead(margin - (change.left ? change.delta : -change.delta));
  const call = (side: number) => names.mine ? ['L', 'T', 'W'][side + 1]
    : side ? (side > 0 ? names.left : names.right) : 'level';
  if (then !== now) return `changed · ${call(then)} → ${call(now)}`;
  return `no change · ${call(now)} ${names.mine ? 'stands' : now ? 'still wins' : 'still'}`;
}

export function scoreOf(results: PulseResults | undefined, franchise: string) {
  return results?.scores.find(s => s.franchise === franchise);
}

export type PulseSeedRow = { seed?: number; bye: boolean; kind: string; franchise: string;
  name: string; division: string; rank: number; record: string; pointsWords: string; pointsFor?: number;
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
      pointsWords: r?.pointsWords ?? '', pointsFor: r?.pointsFor, inField: true };
  });
  const inField = new Set(seeded.map(s => s.franchise));
  const bubble = standings.rows.filter(r => !inField.has(r.franchise)
    && (!field || !r.conference || r.conference === conference)).map(r => ({
    bye: false, kind: '', franchise: r.franchise, name: r.name, division: r.division, rank: r.rank,
    record: r.record, pointsWords: r.pointsWords, pointsFor: r.pointsFor, mine: r.franchise === mine,
    inField: false }));
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

export type PulseTie = { teams: PulseSeedRow[]; step: string; detail: string };

/** tieGroups reads the standings' own reasons as ties: each reason two or more of these rows share,
 *  once, with the tie-breaker that split it. A reason every row shares is the table's first split by
 *  record, not a tie, so it is left out. */
export function tieGroups(standings: PulseStandings, rows: PulseSeedRow[]): PulseTie[] {
  const shared = new Map<string, PulseSeedRow[]>();
  for (const row of rows)
    for (const line of standings.rows.find(r => r.franchise === row.franchise)?.why ?? [])
      shared.set(line, [...(shared.get(line) ?? []), row]);
  return [...shared].filter(([, teams]) => teams.length > 1 && teams.length < rows.length)
    .map(([detail, teams]) => {
      const put = detail.lastIndexOf(' put ');
      return { teams, detail, step: put < 0 ? 'Franchise order'
        : detail.slice(detail.lastIndexOf('. ', put) + 1, put).trim() };
    });
}

export type PulseBracketSide = { words: string; seed?: number };
const ROUND_NAMES: Record<string, string> = { wildcard: 'WILD CARD', division: 'DIVISIONAL',
  conference: 'CONF FINAL', championship: 'THE FINAL' };

/** bracketRounds reads the league's bracket shape a week at a time for each side, today's seeds in
 *  their places. With more than one side, the last week is the final they share. */
export function bracketRounds(bracket: PulseBracket) {
  const weeks = [...new Set(bracket.shape.map(n => n.week))].sort((a, b) => a - b);
  const shared = bracket.fields.length > 1 ? weeks.pop() : undefined;
  const heading = (week: number) => {
    const kind = (bracket.shape.find(n => n.week === week)?.id ?? '').split('_')[0];
    return `${ROUND_NAMES[kind] ?? kind.toUpperCase()} · WK ${week}`;
  };
  const fields = bracket.fields.map(field => {
    const side = (seed?: number, from?: string): PulseBracketSide => seed === undefined
      ? { words: from ?? '—' }
      : { seed, words: field.seeds.find(s => s.seed === seed)?.name ?? `seed ${seed}` };
    return { conference: field.conference, rounds: weeks.map(week => ({ heading: heading(week),
      games: bracket.shape.filter(n => n.week === week).map(n => ({ id: n.id,
        home: side(n.homeSeed, n.homeFrom), away: side(n.awaySeed, n.awayFrom) })) })) };
  });
  return { fields, final: shared === undefined ? undefined
    : { heading: heading(shared), sides: bracket.fields.map(f => `${f.conference} champion`) } };
}
