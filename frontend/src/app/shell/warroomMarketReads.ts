import { controlAPI } from '../control/api';
import { report } from '../reports/contract';
import { parseRules } from '../commissioner/rules';
import { nativeHQSettings } from '../commissioner/hqReading';
import { array, object, text } from '../data/parseValues';
import { personalWire, swapPersonalTargets } from '../offers/api';
import { parsePersonal, type Entry } from '../personal/contract';
import { poolProvider } from '../pool/provider';
import { parseHostedWaivers } from '../commissioner/hostedMarkets';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import type { WarRoomContext } from './warroomReads';
import type { Plan } from '../pool/contract';
import { unavailable } from './hqModel';
export type WarRoomMarketPlayer = { id: string; name: string; position: string; age?: string; string?:
  string; points?: string;
  minimum?: string; minimumCents?: number; value?: string };
export type WarRoomOwnBid = { player: string; name: string; contract: string; state: string;
  kind?: 'plan' | 'placed'; id?: string; post?: string; plan?: Plan };
export type WarRoomMarketReading = { players?: WarRoomMarketPlayer[]; bids?: WarRoomOwnBid[];
  targets?: Entry[]; maxYears?: number; errors: string[] };
export function parseWarRoomOwnBids(value: unknown, league: string, mine: string): WarRoomOwnBid[] {
  const r = object(value, 'Bids', ['league', 'franchise', 'revision', 'season', 'mode', 'activeBids',
    'activeLimit', 'players', 'awards', 'notes']);
  if (r.league !== league || r.franchise !== mine) throw new Error('Bids belong to another franchise');
  return array(r.players, 'Bid players', (v, p): WarRoomOwnBid | undefined => {
    const row = object(v, p, ['player', 'name', 'mode', 'modeWords', 'leader', 'bids', 'yourBid', 'youLead',
      'lateAt', 'lateWords', 'lateNow', 'closesAt', 'closes']);
    if (!row.yourBid) return undefined;
    const bid = object(row.yourBid, 'Your bid', ['id', 'franchise', 'team', 'contract', 'points',
      'placedAt', 'placed']);
    if (bid.franchise !== mine) throw new Error('Private bid scope changed');
    if (typeof row.youLead !== 'boolean') throw new Error('Unreadable own bid state');
    return { player: text(row.player, p), name: text(row.name, p), contract: text(bid.contract, p),
      state: row.mode === 'blind' ? 'PLACED · PRIVATE' : row.youLead ? 'HIGH BID' : 'OUTBID',
      kind: 'placed' as const };
  }).filter((b): b is WarRoomOwnBid => b !== undefined);
}
export async function readWarRoomMarket(c: WarRoomContext): Promise<WarRoomMarketReading> {
  const result: WarRoomMarketReading = { errors: [] };
  const apply = async (fn: () => Promise<void>) => { try { await fn(); } catch (e) {
    result.errors.push(String(e)); console.warn('War Room market read failed:', e); } };
  if (c.native) {
    const native = c.native;
    const port = native.port ?? hostedWorkflow;
    await Promise.all([
      apply(async () => {
        if (native.reading) { result.maxYears = native.reading.settings.maxYears; return; }
        const r = await port({ command: 'commish.rules.read', mode: 'read', arguments: { league:
          native.league } });
        const rules = parseRules(r.data);
        if (rules.league !== native.league) throw new Error('Rules belong to another league');
        result.maxYears = nativeHQSettings(rules).maxYears;
      }),
      apply(async () => { const r = await port({ command: 'native.assets.read', mode: 'read', arguments: {
        league: native.league } });
        const roster = parseHostedRoster(r.data);
        if (roster.league !== native.league) throw new Error('Pool belongs to another league');
        result.players = roster.market;

      }),
      apply(async () => { if (native.owner === false) return;
        const r = await port({ command: 'bids.read', mode: 'read', arguments: {
        league: native.league, franchise: c.mine } });
        result.bids = parseWarRoomOwnBids(r.data, native.league,
          c.mine); }),
    ]);
    if (native.owner !== false) await apply(async () => {
      const { data } = await port({ command: 'waivers.read', mode: 'read', arguments: {
        league: native.league, franchise: c.mine, at: new Date().toISOString() } });
      const waivers = parseHostedWaivers(data);
      if (waivers.league !== native.league || waivers.franchise !== c.mine)
        throw new Error('Waivers belong to another franchise');
      for (const p of waivers.players) if (!result.players?.some(player => player.id === p.player))
        (result.players ??= []).push({ id: p.player, name: p.name, position: unavailable });
    });
    return result;
  }
  await Promise.all([
    apply(async () => {
      const settings = report(await controlAPI.settings(), 'Settings');
      const raw = settings.rows.find(r => r.cells[0] === 'Maximum new contract years')?.cells[1];
      const years = Number(raw);
      if (Number.isSafeInteger(years) && years > 0) result.maxYears = years;
    }),
    apply(async () => { await poolProvider.refresh(false);
      const pool = await poolProvider.pool(c.mine, { position: '', sort: 'name',
      desc: false });
      if (pool.status === 'failed' || pool.status === 'not_wired') throw new Error(pool.absenceReason ??
        'Pool unavailable');
      result.players = pool.value.rows.map(p => ({ id: p.id, name: p.name, position: p.position,
        age: p.age, string: p.string, points: p.points, minimum: p.minimum.text, minimumCents:
          p.minimum.cents, value: p.engineText }));
    }),
    apply(async () => { const personal = parsePersonal(await personalWire.read(c.mine));
      result.targets = personal.entries.filter(e => e.kind === 'watch' && e.entity === 'player')
        .sort((a, b) => a.order - b.order); }),
    apply(async () => { const bids = await poolProvider.plans(c.mine);
      if (bids.status === 'failed' || bids.status === 'not_wired') throw new Error(bids.absenceReason ??
        'Bid plans unavailable');
      result.bids = bids.value.plans.filter(p => p.franchiseId === c.mine).map(p => ({ player: p.playerId,
        name: p.player, contract: p.contract, state: p.status || p.state, kind: 'plan', id: p.id, post:
          p.post, plan: p }));
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
export { parseWarRoomProjections } from '../data/nativeProjections';
