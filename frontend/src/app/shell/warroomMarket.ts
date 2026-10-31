import { array, integer, object, text } from '../data/parseValues';
import { personalWire, swapPersonalTargets } from '../offers/api';
import { parsePersonal, type Entry } from '../personal/contract';
import { poolProvider } from '../pool/provider';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import type { WarRoomContext } from './warroomReads';
import { unavailable } from './hqModel';
export type WarRoomMarketPlayer = { id: string; name: string; position: string; age?: string;
  minimum?: string; value?: string };
export type WarRoomOwnBid = { player: string; name: string; contract: string; state: string };
export type WarRoomMarketReading = { players?: WarRoomMarketPlayer[]; bids?: WarRoomOwnBid[];
  targets?: Entry[]; errors: string[] };
export function parseWarRoomOwnBids(value: unknown, league: string, mine: string): WarRoomOwnBid[] {
  const r = object(value, 'Bids', ['league', 'franchise', 'revision', 'season', 'mode', 'activeBids',
    'activeLimit', 'players', 'awards', 'notes']);
  if (r.league !== league || r.franchise !== mine) throw new Error('Bids belong to another franchise');
  return array(r.players, 'Bid players', (v, p) => {
    const row = object(v, p, ['player', 'name', 'mode', 'modeWords', 'leader', 'bids', 'yourBid', 'youLead',
      'lateAt', 'lateWords', 'lateNow', 'closesAt', 'closes']);
    if (!row.yourBid) return undefined;
    const bid = object(row.yourBid, 'Your bid', ['id', 'franchise', 'team', 'contract', 'points',
      'placedAt', 'placed']);
    if (bid.franchise !== mine) throw new Error('Private bid scope changed');
    if (typeof row.youLead !== 'boolean') throw new Error('Unreadable own bid state');
    return { player: text(row.player, p), name: text(row.name, p), contract: text(bid.contract, p),
      state: row.youLead ? 'HIGH BID' : 'OUTBID' };
  }).filter((b): b is WarRoomOwnBid => b !== undefined);
}
export async function readWarRoomMarket(c: WarRoomContext): Promise<WarRoomMarketReading> {
  const result: WarRoomMarketReading = { errors: [] };
  const apply = async (fn: () => Promise<void>) => { try { await fn(); } catch (e) {
    result.errors.push(String(e)); } };
  if (c.native) {
    const native = c.native;
    const port = native.port ?? hostedWorkflow;
    await Promise.all([
      apply(async () => { const r = await port({ command: 'native.assets.read', mode: 'read', arguments: {
        league: native.league } });
        const roster = parseHostedRoster(r.data);
        if (roster.league !== native.league) throw new Error('Pool belongs to another league');
        result.players = roster.pool.map(p => ({ id: p.value, name: p.label, position: unavailable }));
      }),
      apply(async () => { const r = await port({ command: 'bids.read', mode: 'read', arguments: {
        league: native.league, franchise: c.mine } }); result.bids = parseWarRoomOwnBids(r.data, native.league, c.mine); }),
    ]);
    return result;
  }
  await Promise.all([
    apply(async () => { const pool = await poolProvider.pool(c.mine, { position: '', sort: 'name',
      desc: false });
      if (pool.status === 'failed' || pool.status === 'not_wired') throw new Error(pool.absenceReason ??
        'Pool unavailable');
      result.players = pool.value.rows.map(p => ({ id: p.id, name: p.name, position: p.position,
        age: p.age, minimum: p.minimum.text, value: p.engineText }));
    }),
    apply(async () => { const personal = parsePersonal(await personalWire.read(c.mine));
      result.targets = personal.entries.filter(e => e.kind === 'watch' && e.entity === 'player')
        .sort((a, b) => a.order - b.order); }),
    apply(async () => { const bids = await poolProvider.plans(c.mine);
      if (bids.status === 'failed' || bids.status === 'not_wired') throw new Error(bids.absenceReason ??
        'Bid plans unavailable');
      result.bids = bids.value.plans.filter(p => p.franchiseId === c.mine).map(p => ({ player: p.playerId,
        name: p.player, contract: p.contract, state: p.status || p.state }));
    }),
  ]);
  return result;
}
export async function changeWarRoomTarget(c: WarRoomContext, id: string, watched: boolean,
  withPlayer?: string) {
  if (c.native) throw new Error('Hosted private targets are not available yet');
  if (withPlayer) await swapPersonalTargets(c.mine, id, withPlayer);
  else await personalWire.change(c.mine, 'watch', { id, entity: 'player', action: watched ? 'remove' :
    'add' });
}
export function warRoomTargetLanes(c: WarRoomContext, reading?: WarRoomMarketReading) {
  const lanes = new Map<string, { entry: Entry; player: WarRoomMarketPlayer }[]>();
  for (const entry of reading?.targets ?? []) {
    const identity = c.snapshot.players.value.find(p => p.id === entry.id);
    const player = reading?.players?.find(p => p.id === entry.id) ?? { id: entry.id,
      name: identity?.name ?? entry.id, position: identity?.position ?? unavailable };
    const lane = lanes.get(player.position) ?? [];
    lane.push({ entry, player }); lanes.set(player.position, lane);
  }
  return lanes;
}
export function parseWarRoomProjections(value: unknown, league: string, season: number, week: number) {
  const r = object(value, 'Projections', ['league', 'season', 'week', 'players', 'notes']);
  if (r.league !== league || r.season !== season || r.week !== week)
    throw new Error('Projections belong to another league or week');
  return array(r.players, 'Player projections', (v, p) => {
    const s = object(v, p, ['player', 'position', 'floor', 'median', 'ceiling', 'games', 'available', 'words']);
    const points = (v: unknown) => { if (typeof v !== 'number' || !Number.isFinite(v)) throw new
      Error('Invalid projection'); return v; };
    if (typeof s.available !== 'boolean') throw new Error('Unreadable projection availability');
    return { player: text(s.player, p), position: s.position === undefined ? undefined : text(s.position,
      p), floor: points(s.floor), median: points(s.median), ceiling: points(s.ceiling), games:
      integer(s.games, p), available: s.available, words: text(s.words, p) };
  });
}
