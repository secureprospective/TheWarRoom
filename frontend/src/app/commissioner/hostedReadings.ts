import { array, choice, integer, object, text, timestamp } from '../data/parseValues';

const flag = (v: unknown): boolean => {
  if (typeof v !== 'boolean') throw new Error('Unreadable league status.');
  return v;
};
export type HostedPlayer = { player: string; name: string; position: string; locked: boolean; words: string };
export type HostedLineup = { league: string; franchise: string; revision: number; season: number;
  week: number; words: string; verdict: string; starters: HostedPlayer[]; bench: HostedPlayer[] };
export function parseHostedLineup(v: unknown): HostedLineup {
  const r = object(v, 'Lineup', ['league', 'name', 'revision', 'season', 'week', 'franchise',
    'franchiseName', 'zone', 'set', 'corrected', 'words', 'reason', 'starters', 'bench', 'verdict', 'rules']);
  const player = (v: unknown, p: string): HostedPlayer => {
    const r = object(v, p, ['player', 'name', 'position', 'team', 'locked', 'locksAt', 'words']);
    if (r.locksAt !== undefined) timestamp(r.locksAt, p);
    return { player: text(r.player, p), name: text(r.name, p), position: text(r.position, p),
      locked: flag(r.locked), words: text(r.words, p) };
  };
  const verdict = object(r.verdict, 'Lineup verdict', ['status', 'problems', 'words', 'penalty', 'flagged']);
  return { league: text(r.league, 'League'), franchise: text(r.franchise, 'Franchise'),
    revision: integer(r.revision, 'Version'), season: integer(r.season, 'Season'), week: integer(r.week, 'Week'),
    words: text(r.words, 'Lineup'), verdict: text(verdict.words, 'Verdict'),
    starters: array(r.starters, 'Starters', player), bench: array(r.bench, 'Bench', player) };
}
export type HostedRoster = { revision: number; season: number;
  players: { id: string; name: string; franchise: string; bucket: 'active' | 'ir' | 'taxi' }[] };
export function parseHostedRoster(v: unknown): HostedRoster {
  const r = object(v, 'Roster', ['corrections', 'league', 'revision', 'seeded', 'entry', 'source', 'snapshot']);
  if (!flag(r.seeded)) throw new Error('Ask your commissioner to record the starting rosters.');
  const s = object(r.snapshot, 'Roster snapshot', ['standingsOrder', 'season', 'champion', 'champions',
    'conferences', 'rosterAt', 'draftAt', 'futureAt', 'franchises', 'players', 'picks', 'adjustments', 'pool']);
  return { revision: integer(r.revision, 'Roster version'), season: integer(s.season, 'Roster season'),
    players: array(s.players, 'Roster players', (v, p) => {
      const r = object(v, p, ['acquiredVia', 'correction', 'name', 'rookieRound', 'pending', 'restructured',
        'id', 'franchise', 'status', 'salaryCents', 'expirationYear', 'contractStatus', 'contractInfo']);
      const status = choice(r.status, p, ['ROSTER', 'IR', 'TAXI_SQUAD'] as const);
      return { id: text(r.id, p), name: r.name ? text(r.name, p) : 'Unnamed player — ask your commissioner',
        franchise: text(r.franchise, p), bucket: status === 'IR' ? 'ir' : status === 'TAXI_SQUAD' ? 'taxi' : 'active' };
    }) };
}
export type HostedReserves = { franchise: string; words: string[]; players: { player: string; words: string }[] }[];
export function parseHostedReserves(v: unknown): HostedReserves {
  const r = object(v, 'Reserves', ['corrections', 'league', 'name', 'revision', 'rosterSeason',
    'rulesSeason', 'seeded', 'source', 'rules', 'franchises']);
  return array(r.franchises, 'Reserve squads', (v, p) => {
    const r = object(v, p, ['franchise', 'name', 'abbreviation', 'ir', 'taxi', 'players']);
    const words = [r.ir, r.taxi].map(v => {
      const r = object(v, p, ['used', 'slots', 'judged', 'words']);
      return text(r.words, p);
    });
    return { franchise: text(r.franchise, p), words, players: array(r.players, p, (v, p) => {
      const r = object(v, p, ['player', 'name', 'squad', 'standing', 'words']);
      return { player: text(r.player, p), words: text(r.words, p) };
    }) };
  });
}
