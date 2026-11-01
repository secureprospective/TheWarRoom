import { array, integer, object, text } from './parseValues';

const finite = (value: unknown, path: string): number => {
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error(`${path}: invalid value`);
  return value;
};
const flag = (value: unknown, path: string): boolean => {
  if (typeof value !== 'boolean') throw new Error(`${path}: invalid availability`);
  return value;
};
export type LeaguePlayerValue = { player: string; franchise: string; position?: string; nowPPG: number;
  dynastyPPG: number; age: number; ageKnown: boolean; available: boolean; words: string };
export type AgeBand = { from: number; to: number; open: boolean; value: number; players: number };
export function parseNativeValues(value: unknown, league: string, franchise: string, season: number) {
  const r = object(value, 'League values', ['league', 'season', 'franchise', 'players', 'best', 'age',
    'notes']);
  if (r.league !== league || r.franchise !== franchise || r.season !== season)
    throw new Error('Values belong to another league, franchise or season');
  const b = object(r.best, 'Best N', ['franchise', 'starterN', 'value', 'age', 'ageKnown', 'rank', 'of',
    'available', 'words']);
  if (b.franchise !== franchise) throw new Error('Best N belongs to another franchise');
  const best = { starterN: integer(b.starterN, 'Starter count'), value: finite(b.value, 'Best N'),
    age: finite(b.age, 'Age'), ageKnown: flag(b.ageKnown, 'Age'), rank: integer(b.rank, 'Rank'),
    of: integer(b.of, 'Franchises'), available: flag(b.available, 'Best N'), words: text(b.words, 'Best N') };
  if (best.available && (best.starterN < 1 || best.rank < 1 || best.rank > best.of))
    throw new Error('Invalid Best N rank or starter count');
  const seen = new Set<string>();
  const players = array(r.players, 'Player values', (v, path): LeaguePlayerValue => {
    const p = object(v, path, ['player', 'franchise', 'position', 'nowPPG', 'dynastyPPG', 'age',
      'ageKnown', 'available', 'words']);
    const player = text(p.player, path), owner = text(p.franchise, path);
    if (!player || seen.has(player) || (franchise && owner !== franchise))
      throw new Error('Player value has a duplicate identity or wrong owner');
    seen.add(player);
    return { player, franchise: owner, position: p.position === undefined ? undefined : text(p.position,
      path), nowPPG: finite(p.nowPPG, path), dynastyPPG: finite(p.dynastyPPG, path),
      age: finite(p.age, path), ageKnown: flag(p.ageKnown, path), available: flag(p.available, path),
      words: text(p.words, path) };
  });
  const age = array(r.age, 'Value by age', (v, path): AgeBand => {
    const b = object(v, path, ['from', 'to', 'open', 'value', 'players']);
    return { from: integer(b.from, path), to: integer(b.to, path), open: flag(b.open, path),
      value: finite(b.value, path), players: integer(b.players, path) };
  });
  return { best, players, age, notes: array(r.notes, 'Value notes', text) };
}
export type LeagueValues = ReturnType<typeof parseNativeValues>;
