import { array, integer, object, text } from './parseValues';
export function parseWarRoomProjections(value: unknown, league: string, season: number, week: number) {
  const r = object(value, 'Projections', ['league', 'season', 'week', 'players', 'notes']);
  if (r.league !== league || r.season !== season || r.week !== week)
    throw new Error('Projections belong to another league or week');
  return array(r.players, 'Player projections', (v, p) => {
    const s = object(v, p, ['player', 'position', 'floor', 'median', 'ceiling', 'games',
      'available', 'words']);
    const points = (v: unknown) => { if (typeof v !== 'number' || !Number.isFinite(v)) throw new
      Error('Invalid projection'); return v; };
    if (typeof s.available !== 'boolean') throw new Error('Unreadable projection availability');
    return { player: text(s.player, p), position: s.position === undefined ? undefined : text(s.position,
      p), floor: points(s.floor), median: points(s.median), ceiling: points(s.ceiling), games:
      integer(s.games, p), available: s.available, words: text(s.words, p) };
  });
}
