import { useContractRead } from '../contracts/Contracts';
import { hqEvents } from './hqReads';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { parsePulseBracket, parsePulseDirectory, parsePulseLineup, parsePulseOdds, parsePulseResults,
  parsePulseStandings, parsePulseWeek, type PulseBracket, type PulseFixture,
  type PulseLineupPlayer, type PulseOdds, type PulseResults, type PulseSeat, type PulseStandings } from
  '../data/nativePulse';
import { parseWarRoomProjections } from '../data/nativeProjections';
import type { Node } from './nodes';

export type PulseScope = { league: string; mine: string; port?: HostedPort;
  leave: (node?: Node) => void; embedded?: boolean; tabs?: HTMLElement | null };

/** NativePulse is every league-wide fact League Pulse reads. Each region reads one field and says
 *  plainly when the read it needs has not arrived. */
export type NativePulse = {
  league: string; name: string; season: number; revision: number;
  seats: PulseSeat[];
  week?: number; weekNote: string; lastWeek: number;
  fixtures: PulseFixture[];
  results?: PulseResults;
  standings?: PulseStandings;
  bracket?: PulseBracket;
  /** Win chance, power and playoff odds from the league's own results, schedule and values. */
  odds?: PulseOdds;
  errors: string[];
};

/** PulseMatchup is what one matchup adds to the week's scores: each starter's NFL team and whether
 *  that player's game has locked, plus the projections the league holds for them. */
export type PulseMatchup = { lineups: Record<string, PulseLineupPlayer[]>;
  projections: Record<string, number>; errors: string[] };

export function usePulseRead<T>(key: string, read: (publish: (value: T) => void) => Promise<T>) {
  return useContractRead<T>(key, async publish => {
    try { return await read(publish); } catch (error) {
      console.warn('League Pulse read failed:', error); throw error;
    }
  }, hqEvents);
}

export async function readNativePulse(league: string, mine: string, port: HostedPort = hostedWorkflow,
  publish?: (reading: NativePulse) => void): Promise<NativePulse> {
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const directory = await port({ command: 'native.directory.read', mode: 'read',
    arguments: { league } });
  const held = parsePulseDirectory(directory.data, league);
  const result: NativePulse = { league, name: held.name, season: held.season,
    revision: held.revision, seats: held.seats, weekNote: '', lastWeek: 0, fixtures: [], errors: [] };
  const apply = async (read: () => Promise<void>) => {
    try { await read(); } catch (error) {
      result.errors.push(String(error)); console.warn('League Pulse read failed:', error);
    } finally { publish?.({ ...result, errors: [...result.errors] }); }
  };
  publish?.({ ...result });
  await apply(async () => {
    const r = await port({ command: 'week.read', mode: 'read', arguments: { league, franchise: mine,
      season: held.season, localZone: zone, at: new Date().toISOString() } });
    const week = parsePulseWeek(r.data, league, mine, held.season);
    result.fixtures = week.games;
    result.lastWeek = week.games.reduce((high, g) => Math.max(high, g.week), 0);
    result.week = week.week ?? week.nextWeek;
    result.weekNote = week.note || week.message;
  });
  await Promise.all([
    apply(async () => {
      const r = await port({ command: 'standings.read', mode: 'read',
        arguments: { league, season: held.season } });
      result.standings = parsePulseStandings(r.data, league, held.season);
    }),
    apply(async () => {
      const r = await port({ command: 'bracket.read', mode: 'read',
        arguments: { league, season: held.season } });
      result.bracket = parsePulseBracket(r.data, league, held.season);
    }),
    apply(async () => {
      const r = await port({ command: 'odds.read', mode: 'read',
        arguments: { league, season: held.season } });
      result.odds = parsePulseOdds(r.data, league, held.season);
    }),
  ]);
  return result;
}

/** readPulseWeekResults scores one week. It is its own read so moving between weeks never reloads
 *  the standings, the playoff field or the directory. */
export async function readPulseWeekResults(league: string, season: number, week: number,
  port: HostedPort = hostedWorkflow): Promise<PulseResults> {
  const r = await port({ command: 'results.read', mode: 'read',
    arguments: { league, season, week } });
  return parsePulseResults(r.data, league, week);
}

export async function readPulseMatchup(league: string, season: number, week: number,
  franchises: readonly string[], starters: Record<string, readonly string[]>,
  port: HostedPort = hostedWorkflow, publish?: (reading: PulseMatchup) => void): Promise<PulseMatchup> {
  const out: PulseMatchup = { lineups: {}, projections: {}, errors: [] };
  const apply = async (read: () => Promise<void>) => {
    try { await read(); } catch (error) {
      out.errors.push(String(error)); console.warn('League Pulse matchup read failed:', error);
    }
  };
  await Promise.all(franchises.map(franchise => apply(async () => {
    const r = await port({ command: 'lineup.read', mode: 'read',
      arguments: { league, franchise, week, at: '' } });
    out.lineups[franchise] = parsePulseLineup(r.data, league, franchise, week);
  })));
  // Lineups show at once; projections wait for the value engine's first run after launch.
  publish?.({ ...out, lineups: { ...out.lineups }, errors: [...out.errors] });
  const players = [...new Set(franchises.flatMap(f => [...(starters[f] ?? [])]))];
  await apply(async () => {
    for (let offset = 0; offset < players.length; offset += 120) {
      const ids = players.slice(offset, offset + 120);
      const r = await port({ command: 'projections.read', mode: 'read',
        arguments: { league, players: ids.join(','), season, week } });
      for (const p of parseWarRoomProjections(r.data, league, season, week)) {
        if (!ids.includes(p.player)) throw new Error('Projection player scope changed');
        if (p.available) out.projections[p.player] = p.median;
      }
    }
  });
  return out;
}
