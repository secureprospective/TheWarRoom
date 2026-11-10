import { array, integer, object, text } from './parseValues';

// League points cross the wire as leaguerules.Decimal: ten-thousandths of a point.
const SCALE = 10000;
function points(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value))
    throw new Error(`${path}: expected a points value`);
  return value / SCALE;
}
function optionalPoints(value: unknown, path: string): number | undefined {
  return value === undefined || value === null ? undefined : points(value, path);
}
/** pointsLabel writes a score the way the mock does: one decimal place. */
export function pointsLabel(value: number | undefined): string {
  return value === undefined ? '–' : value.toFixed(1);
}

export type PulseStarter = { player: string; name: string; position?: string; words: string;
  /** What the starter recorded this week, by league stat id, in whole units. */
  line?: Record<string, number>;
  points?: number; provisional?: number; source?: string; fieldSources?: Record<string, string> };
export type PulseScore = { franchise: string; name: string; status: string; points?: number;
  provisional?: number;
  words: string; starters: PulseStarter[]; adjustment: number; lineupStatus: string;
  lineupWords: string; flagged: boolean; zeroed?: string };
export type PulseGame = { week: number; home: string; away: string; homeName: string; awayName: string;
  status: string; homePoints?: number; awayPoints?: number; homeProvisional?: number;
  awayProvisional?: number; winner?: string; words: string;
  why: string[] };
export type PulseResults = { league: string; season: number; week: number; revision: number;
  published: boolean; words: string; games: PulseGame[]; bye?: string; byeName?: string;
  scores: PulseScore[]; missed: { franchise: string; name: string; flagged: boolean }[] };

function parseLine(value: unknown, path: string): Record<string, number> | undefined {
  if (value === undefined) return undefined;
  return Object.fromEntries(Object.entries(object(value, `${path}.line`, Object.keys(Object(value))))
    .map(([stat, quantity]) => [stat, points(quantity, `${path}.line.${stat}`)]));
}
function parseStarters(value: unknown): PulseStarter[] {
  return array(value, 'Starters', (v, p) => {
    const s = object(v, p, ['player', 'name', 'position', 'words', 'line', 'points',
      'provisional', 'source', 'fieldSources']);
    return { player: text(s.player, p), name: text(s.name, p),
      position: s.position === undefined ? undefined : text(s.position, p),
      words: text(s.words, p), line: parseLine(s.line, p), points: optionalPoints(s.points, p),
      provisional: optionalPoints(s.provisional, p),
      source: s.source === undefined ? undefined : text(s.source, p),
      fieldSources: parseFieldSources(s.fieldSources) };
  });
}
function parseScores(value: unknown): PulseScore[] {
  return array(value, 'Week scores', (v, p) => {
    const s = object(v, p, ['recordedResult', 'franchise', 'name', 'status', 'points', 'provisional',
      'words', 'starters', 'adjustment', 'changes', 'lineup', 'zeroed']);
    const lineup = object(s.lineup, 'Lineup verdict', ['status', 'words', 'problems', 'flagged',
      'penalty']);
    if (typeof lineup.flagged !== 'boolean') throw new Error('Unreadable lineup flag');
    return { franchise: text(s.franchise, p), name: text(s.name, p), status: text(s.status, p),
      points: optionalPoints(s.points, p), provisional: optionalPoints(s.provisional, p),
      words: text(s.words, p), starters: parseStarters(s.starters),
      adjustment: points(s.adjustment, p), lineupStatus: text(lineup.status, p),
      lineupWords: text(lineup.words, p), flagged: lineup.flagged,
      zeroed: s.zeroed === undefined ? undefined : text(s.zeroed, p) };
  });
}
export function parsePulseResults(value: unknown, league: string, week: number): PulseResults {
  const r = object(value, 'Week results', ['league', 'season', 'week', 'revision', 'published',
    'words', 'games', 'bye', 'byeName', 'scores', 'missed']);
  if (typeof r.published !== 'boolean') throw new Error('Unreadable schedule publication');
  if (r.league !== league || r.week !== week)
    throw new Error('Results belong to another league or week');
  return { league: text(r.league, 'League'), season: integer(r.season, 'Season', 1920),
    week: integer(r.week, 'Week', 1), revision: integer(r.revision, 'Revision'),
    published: r.published, words: text(r.words, 'Results'),
    games: array(r.games, 'Games', (v, p) => {
      const g = object(v, p, ['week', 'home', 'away', 'homeName', 'awayName', 'status', 'homePoints',
        'awayPoints', 'homeProvisional', 'awayProvisional', 'winner', 'words', 'why']);
      return { week: integer(g.week, p, 1), home: text(g.home, p), away: text(g.away, p),
        homeName: text(g.homeName, p), awayName: text(g.awayName, p), status: text(g.status, p),
        homePoints: optionalPoints(g.homePoints, p), awayPoints: optionalPoints(g.awayPoints, p),
        homeProvisional: optionalPoints(g.homeProvisional, p),
        awayProvisional: optionalPoints(g.awayProvisional, p),
        winner: g.winner === undefined ? undefined : text(g.winner, p), words: text(g.words, p),
        why: array(g.why, 'Reasons', (v, p) => text(v, p)) };
    }),
    bye: r.bye === undefined ? undefined : text(r.bye, 'Bye'),
    byeName: r.byeName === undefined ? undefined : text(r.byeName, 'Bye team'),
    scores: parseScores(r.scores),
    missed: array(r.missed, 'Missed lineups', (v, p) => {
      const m = object(v, p, ['franchise', 'name', 'flagged']);
      if (typeof m.flagged !== 'boolean') throw new Error('Unreadable missed-lineup flag');
      return { franchise: text(m.franchise, p), name: text(m.name, p), flagged: m.flagged };
    }) };
}

export type PulseStandingRow = { rank: number; franchise: string; name: string; division: string;
  conference: string;
  wins: number; losses: number; ties: number; record: string; pointsFor: number;
  pointsWords: string; why: string[] };
export type PulseStandings = { league: string; name: string; season: number; revision: number;
  status: string; words: string; through: number; decided: number; pending: number;
  criteria: string[]; noData: string[]; rows: PulseStandingRow[]; overrideWords?: string };

export function parsePulseStandings(value: unknown, league: string, season: number): PulseStandings {
  const r = object(value, 'Standings', ['league', 'name', 'season', 'revision', 'status', 'words',
    'through', 'decided', 'pending', 'criteria', 'noData', 'computed', 'override', 'changes']);
  if (r.league !== league || r.season !== season)
    throw new Error('Standings belong to another league or season');
  const rows = (v: unknown) => array(v, 'Standings rows', (v, p) => {
    const s = object(v, p, ['rank', 'franchise', 'name', 'division', 'conference', 'wins', 'losses', 'ties',
      'record', 'computedRecord', 'pointsFor', 'pointsWords', 'why']);
    return { rank: integer(s.rank, p, 1), franchise: text(s.franchise, p), name: text(s.name, p),
      division: s.division === undefined ? '' : text(s.division, p),
      conference: s.conference === undefined ? '' : text(s.conference, p), wins: integer(s.wins, p),
      losses: integer(s.losses, p), ties: integer(s.ties, p), record: text(s.record, p),
      pointsFor: points(s.pointsFor, p), pointsWords: text(s.pointsWords, p),
      why: array(s.why, 'Reasons', (v, p) => text(v, p)) };
  });
  const override = r.override == null ? undefined
    : object(r.override, 'Standings ruling', ['words', 'rows', 'order', 'records']);
  return { league: text(r.league, 'League'), name: text(r.name, 'League name'),
    season: integer(r.season, 'Season', 1920), revision: integer(r.revision, 'Revision'),
    status: text(r.status, 'Status'), words: text(r.words, 'Standings'),
    through: integer(r.through, 'Through week'), decided: integer(r.decided, 'Decided'),
    pending: integer(r.pending, 'Pending'),
    criteria: array(r.criteria, 'Tie-breakers', (v, p) => text(v, p)),
    noData: array(r.noData, 'Unapplied tie-breakers', (v, p) => text(v, p)),
    rows: rows(override?.rows ?? r.computed),
    overrideWords: override === undefined ? undefined : text(override.words, 'Ruling') };
}

export type PulseSeed = { seed: number; franchise: string; name: string; division: string;
  rank: number; kind: string; bye: boolean; byHand: boolean };
export type PulseBracket = { league: string; season: number; revision: number; status: string;
  words: string; startWeek: number; finalWeek: number; perConference: number; byeSeeds: number;
  started: boolean; fields: { conference: string; seeds: PulseSeed[] }[];
  /** The fields are today's standings read as seeds, not a set playoff field. */
  projected: boolean;
  /** The bracket each side plays: a game names a seed, or says in words who reaches it. */
  shape: { id: string; week: number; homeSeed?: number; awaySeed?: number; homeFrom?: string;
    awayFrom?: string }[] };

export function parsePulseBracket(value: unknown, league: string, season: number): PulseBracket {
  const r = object(value, 'Playoffs', ['league', 'name', 'season', 'revision', 'status', 'words',
    'preset', 'presets', 'rules', 'started', 'startKnown', 'fields', 'projected', 'rounds', 'champion', 'title',
    'history', 'changes']);
  if (r.league !== league || r.season !== season)
    throw new Error('Playoffs belong to another league or season');
  if (typeof r.started !== 'boolean') throw new Error('Unreadable playoff start');
  const fields = (value: unknown) => array(value, 'Playoff fields', (v, p) => {
    const f = object(v, p, ['conference', 'seeds']);
    return { conference: text(f.conference, p), seeds: array(f.seeds, 'Seeds', (v, p) => {
      const s = object(v, p, ['seed', 'franchise', 'name', 'division', 'rank', 'kind', 'bye',
        'byHand']);
      if (typeof s.bye !== 'boolean' || typeof s.byHand !== 'boolean')
        throw new Error('Unreadable seed');
      return { seed: integer(s.seed, p, 1), franchise: text(s.franchise, p), name: text(s.name, p),
        division: text(s.division, p), rank: integer(s.rank, p, 1), kind: text(s.kind, p),
        bye: s.bye, byHand: s.byHand };
    }) };
  });
  const set = fields(r.fields);
  const rules = object(r.rules, 'Playoff rules', ['words', 'perConference', 'divisionWinners',
    'wildCards', 'byeSeeds', 'seeding', 'reseed', 'startWeek', 'finalWeek', 'homeBonus', 'shape']);
  return { league: text(r.league, 'League'), season: integer(r.season, 'Season', 1920),
    revision: integer(r.revision, 'Revision'), status: text(r.status, 'Status'),
    words: text(r.words, 'Playoffs'), startWeek: integer(rules.startWeek, 'Playoff start'),
    finalWeek: integer(rules.finalWeek, 'Playoff final'),
    perConference: integer(rules.perConference, 'Playoff places'),
    byeSeeds: integer(rules.byeSeeds, 'Bye seeds'), started: r.started,
    shape: array(rules.shape, 'Bracket shape', (v, p) => {
      const n = object(v, p, ['id', 'week', 'homeSeed', 'awaySeed', 'homeFrom', 'awayFrom']);
      const seed = (s: unknown) => s === undefined ? undefined : integer(s, p, 1);
      const from = (s: unknown) => s === undefined ? undefined : text(s, p);
      return { id: text(n.id, p), week: integer(n.week, p, 1), homeSeed: seed(n.homeSeed),
        awaySeed: seed(n.awaySeed), homeFrom: from(n.homeFrom), awayFrom: from(n.awayFrom) };
    }),
    fields: set.length ? set : fields(r.projected), projected: !set.length };
}

export type PulseSchedule = { season: number;
  weeks: { week: number; games: { home: string; away: string }[]; bye?: string }[] };
export function parsePulseSchedule(value: unknown, league: string): PulseSchedule {
  const r = object(value, 'Schedule', ['league', 'revision', 'season', 'weeks']);
  if (r.league !== league) throw new Error('The schedule belongs to another league');
  return { season: integer(r.season, 'Season', 1920), weeks: array(r.weeks, 'Weeks', (v, p) => {
    const w = object(v, p, ['week', 'games', 'bye']);
    return { week: integer(w.week, p, 1), bye: w.bye === undefined ? undefined : text(w.bye, p),
      games: array(w.games, 'Games', (g, q) => {
        const game = object(g, q, ['id', 'home', 'away']);
        return { home: text(game.home, q), away: text(game.away, q) };
      }) };
  }) };
}

export type PulseFixture = { week: number; opponent: string; venue: string; kickoff?: string };
export type PulseWeek = { league: string; franchise: string; season: number; revision: number;
  message: string; games: PulseFixture[]; week?: number; nextWeek?: number; note: string };

function parseFixtures(value: unknown): PulseFixture[] {
  return array(value, 'Fixtures', (v, p) => {
    const g = object(v, p, ['week', 'game', 'opponent', 'venue', 'kickoff']);
    return { week: integer(g.week, p, 1),
      opponent: g.opponent === undefined ? '' : text(g.opponent, p), venue: text(g.venue, p),
      kickoff: g.kickoff == null ? undefined : text(g.kickoff, p) };
  });
}
export function parsePulseWeek(value: unknown, league: string, franchise: string,
  season: number): PulseWeek {
  const r = object(value, 'League week', ['schedule', 'current', 'nextWeek', 'message']);
  const s = object(r.schedule, 'Franchise schedule', ['league', 'franchise', 'season', 'revision',
    'message', 'games']);
  if (s.league !== league || s.franchise !== franchise || s.season !== season)
    throw new Error('The week belongs to another franchise');
  const current = r.current == null ? undefined
    : object(r.current, 'This week', ['week', 'game', 'opponent', 'venue', 'kickoff']);
  return { league: text(s.league, 'League'), franchise: text(s.franchise, 'Franchise'),
    season: integer(s.season, 'Season', 1920), revision: integer(s.revision, 'Revision'),
    message: s.message === undefined ? '' : text(s.message, 'Schedule note'),
    games: parseFixtures(s.games),
    week: current === undefined ? undefined : integer(current.week, 'This week', 1),
    nextWeek: r.nextWeek === undefined ? undefined : integer(r.nextWeek, 'Next week', 1),
    note: r.message === undefined ? '' : text(r.message, 'Week note') };
}

export type PulseSeat = { franchise: string; name: string; abbreviation: string; division: string };
export type PulseDirectory = { league: string; name: string; revision: number; season: number;
  seats: PulseSeat[] };

export function parsePulseDirectory(value: unknown, league: string): PulseDirectory {
  const r = object(value, 'League directory', ['league', 'name', 'revision', 'rosterSeason',
    'source', 'franchises']);
  if (r.league !== league) throw new Error('The directory belongs to another league');
  return { league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'Revision'), season: integer(r.rosterSeason, 'Season', 1920),
    seats: array(r.franchises, 'Franchises', (v, p) => {
      const s = object(v, p, ['franchise', 'name', 'abbreviation', 'owner', 'caretaker', 'division']);
      return { franchise: text(s.franchise, p), name: text(s.name, p),
        abbreviation: text(s.abbreviation, p),
        division: s.division === undefined ? '' : text(s.division, p) };
    }) };
}

/** PulseLineupPlayer is what a lineup read adds to a week score: the NFL team and whether the
 *  player's game has locked. Points come from the week score, never from here. */
export type PulseLineupPlayer = { player: string; team: string; locked: boolean };

export function parsePulseLineup(value: unknown, league: string, franchise: string,
  week: number): PulseLineupPlayer[] {
  const r = object(value, 'Lineup', ['league', 'name', 'revision', 'season', 'week', 'franchise',
    'franchiseName', 'zone', 'set', 'corrected', 'words', 'reason', 'starters', 'bench', 'verdict',
    'rules']);
  if (r.league !== league || r.franchise !== franchise || r.week !== week)
    throw new Error('The lineup belongs to another franchise or week');
  return array(r.starters, 'Starters', (v, p) => {
    const s = object(v, p, ['player', 'name', 'position', 'team', 'locked', 'locksAt', 'words']);
    if (typeof s.locked !== 'boolean') throw new Error('Unreadable lock state');
    return { player: text(s.player, p), team: s.team === undefined ? '' : text(s.team, p),
      locked: s.locked };
  });
}

/** PulseOdds is the league's own win chance, power and playoff odds (odds.read). Each part carries the words
 *  that say why it is missing; a missing part is undefined, never zero. */
export type PulseOddsRow = { franchise: string; name: string; rank: number; record: string;
  power?: number; powerScore: number; expected: number; playoffOdds?: number };
export type PulseOddsGame = { week: number; home: string; away: string; homeName: string; awayName: string;
  homeWin: number };
export type PulseOdds = { league: string; season: number; revision: number; status: string; words: string;
  powerWords: string; gameWords: string; oddsWords: string; week: number; runs: number; seed: number;
  rows: PulseOddsRow[]; games: PulseOddsGame[] };

function share(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0 || value > 1)
    throw new Error(`${path}: expected a chance between 0 and 1`);
  return value;
}
function finite(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error(`${path}: expected a number`);
  return value;
}

export function parsePulseOdds(value: unknown, league: string, season: number): PulseOdds {
  const r = object(value, 'Odds', ['league', 'name', 'season', 'revision', 'status', 'words', 'powerWords',
    'gameWords', 'oddsWords', 'week', 'runs', 'seed', 'rows', 'games']);
  if (r.league !== league || r.season !== season) throw new Error('Odds belong to another league or season');
  return { league: text(r.league, 'League'), season: integer(r.season, 'Season', 1920),
    revision: integer(r.revision, 'Revision'), status: text(r.status, 'Status'), words: text(r.words, 'Odds'),
    powerWords: text(r.powerWords, 'Power'), gameWords: text(r.gameWords, 'Win chance'),
    oddsWords: text(r.oddsWords, 'Playoff odds'), week: integer(r.week, 'Week'),
    runs: integer(r.runs, 'Runs'),
    seed: integer(r.seed, 'Seed'),
    rows: array(r.rows, 'Odds rows', (v, p) => {
      const s = object(v, p, ['franchise', 'name', 'rank', 'record', 'power', 'powerScore', 'expected',
        'playoffOdds']);
      const power = integer(s.power, p);
      return { franchise: text(s.franchise, p), name: text(s.name, p), rank: integer(s.rank, p, 1),
        record: text(s.record, p), power: power || undefined, powerScore: share(s.powerScore, p),
        expected: finite(s.expected, p),
        playoffOdds: s.playoffOdds == null ? undefined : share(s.playoffOdds, p) };
    }),
    games: array(r.games, 'Odds games', (v, p) => {
      const g = object(v, p, ['week', 'home', 'away', 'homeName', 'awayName', 'homeWin']);
      return { week: integer(g.week, p, 1), home: text(g.home, p), away: text(g.away, p),
        homeName: text(g.homeName, p), awayName: text(g.awayName, p), homeWin: share(g.homeWin, p) };
    }) };
}

/** percent writes a chance as the screens show it: a whole percent. */
export function percent(chance: number | undefined): string {
  return chance === undefined ? '—' : `${Math.round(chance * 100)}%`;
}

/** myChance is a franchise's chance in its game of the odds week, with its opponent's. */
export function myChance(odds: PulseOdds | undefined, franchise: string):
  { mine: number; theirs: number; week: number } | undefined {
  const g = odds?.games.find(x => x.home === franchise || x.away === franchise);
  if (!g) return undefined;
  const mine = g.home === franchise ? g.homeWin : 1 - g.homeWin;
  return { mine, theirs: 1 - mine, week: g.week };
}

function parseFieldSources(value: unknown): Record<string, string> | undefined {
  if (value === undefined) return undefined;
  if (value === null || typeof value !== 'object' || Array.isArray(value))
    throw new Error('Unreadable field sources');
  return Object.fromEntries(Object.entries(value).map(([field, owner]) => [field, text(owner, field)]));
}
