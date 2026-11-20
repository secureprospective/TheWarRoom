import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { array, choice, integer, object, text, timestamp } from '../data/parseValues';

// The member-safe NFL player status read (nfl.player-status.read): public injury designation,
// body part and expected return week. Freshness is the server's own state word.
export type PlayerStatusFact = { player: string; designation: string; bodyPart: string; returnWeek?: number };
export type PlayerStatus = { league: string; state: 'available' | 'stale' | 'unavailable';
  fetchedAt?: string; facts: Map<string, PlayerStatusFact> };

export function parsePlayerStatus(v: unknown, league: string): PlayerStatus {
  const r = object(v, 'NFL player status', ['league', 'state', 'season', 'fetchedAt', 'players']);
  if (text(r.league, 'League') !== league) throw new Error('Player status belongs to another league');
  const state = choice(r.state, 'State', ['available', 'stale', 'unavailable'] as const);
  const fetchedAt = r.fetchedAt === null || r.fetchedAt === undefined ? ''
    : timestamp(r.fetchedAt, 'Fetched');
  const facts = array(r.players, 'Players', (p, path) => {
    const o = object(p, path, ['player', 'designation', 'bodyPart', 'expectedReturnWeek']);
    return { player: text(o.player, `${path}.player`),
      designation: text(o.designation, `${path}.designation`),
      bodyPart: o.bodyPart === undefined ? '' : text(o.bodyPart, `${path}.bodyPart`),
      ...(o.expectedReturnWeek === undefined ? {}
        : { returnWeek: integer(o.expectedReturnWeek, `${path}.expectedReturnWeek`, 1) }) };
  });
  return { league, state, ...(fetchedAt ? { fetchedAt } : {}),
    facts: new Map(facts.map(f => [f.player, f])) };
}

export async function readPlayerStatus(league: string,
  port: HostedPort = hostedWorkflow): Promise<PlayerStatus> {
  const r = await port({ command: 'nfl.player-status.read', mode: 'read', arguments: { league } });
  return parsePlayerStatus(r.data, league);
}

/** statusWords is one fact as the phone shows it: "Out · hamstring · back week 12". */
export const statusWords = (f: PlayerStatusFact) => [f.designation, f.bodyPart,
  f.returnWeek ? `back week ${f.returnWeek}` : ''].filter(Boolean).join(' · ');

/** statusNote is the read's own words when it holds no current facts. */
export const statusNote = (s: PlayerStatus) => s.state === 'unavailable'
  ? 'Player status: the NFL injury report is not available right now.'
  : s.state === 'stale' ? 'Player status may be out of date.' : '';
