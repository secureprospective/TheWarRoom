import { controlAPI } from '../control/api';
import { report } from '../reports/contract';
import { parseRules } from '../commissioner/rules';
import { nativeHQCap, nativeHQSettings } from '../commissioner/hqReading';
import { array, id, integer, object, requiredText, text } from '../data/parseValues';
import { personalWire, swapPersonalTargets } from '../offers/api';
import { parsePersonal, type Entry } from '../personal/contract';
import { poolProvider } from '../pool/provider';
import { parseHostedWaivers } from '../commissioner/hostedMarkets';
import { hostedWorkflow } from '../commissioner/api';
import { newKey } from '../commissioner/forms';
import { parseReceipt } from '../commissioner/proxy';
import { parsePreview, type Preview } from '../commissioner/preview';
import type { NativeRequest } from '../commands/native';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import { parseNativeValues, type LeaguePlayerValue } from '../data/nativeValues';
import type { WarRoomContext } from './warroomReads';
import type { BidFormState, Plan } from '../pool/contract';
import { planForm } from '../pool/state';
import { unavailable } from './hqModel';
export type WarRoomMarketPlayer = { id: string; name: string; position: string; age?: string; string?:
  string; points?: string;
  minimum?: string; minimumCents?: number; value?: string };
export type WarRoomOwnBid = { player: string; name: string; contract: string; state: string;
  kind?: 'plan' | 'placed'; id?: string; bidId?: string; post?: string; plan?: Plan };
export type WarRoomMarketReading = { players?: WarRoomMarketPlayer[]; bids?: WarRoomOwnBid[];
  bidRevision?: number; capRoom?: number; writable?: boolean; football?: string;
  targets?: Entry[]; maxYears?: number; errors: string[] };
export function parseWarRoomWatchList(value: unknown, league: string): string[] {
  const r = object(value, 'Watched players', ['league', 'players']);
  if (r.league !== league) throw new Error('Watched players belong to another league');
  return array(r.players, 'Watched players', (v, p) => id(v, p, true));
}
function marketFact(value?: string): string {
  const fact = value?.trim();
  return !fact || fact === '—' ? unavailable : fact;
}
function hostedMarketPlayer(player: WarRoomMarketPlayer, value?: LeaguePlayerValue): WarRoomMarketPlayer {
  const heldAge = marketFact(player.age);
  return { ...player, name: marketFact(player.name), position: marketFact(player.position),
    age: heldAge !== unavailable ? heldAge : value?.ageKnown ? value.age.toFixed(1) : unavailable,
    string: marketFact(player.string), points: marketFact(player.points),
    value: value?.available ? value.dynastyPPG.toFixed(1) : unavailable,
    minimum: player.minimumCents === undefined ? unavailable : marketFact(player.minimum) };
}
export type WarRoomBidReading = { bids: WarRoomOwnBid[]; revision: number };
export function parseWarRoomBidReading(value: unknown, league: string, mine: string): WarRoomBidReading {
  const r = object(value, 'Bids', ['league', 'franchise', 'revision', 'season', 'mode', 'activeBids',
    'activeLimit', 'players', 'awards', 'notes']);
  if (r.league !== league || r.franchise !== mine) throw new Error('Bids belong to another franchise');
  const bids = array(r.players, 'Bid players', (v, p): WarRoomOwnBid | undefined => {
    const row = object(v, p, ['player', 'name', 'mode', 'modeWords', 'leader', 'bids', 'yourBid', 'youLead',
      'lateAt', 'lateWords', 'lateNow', 'closesAt', 'closes']);
    if (!row.yourBid) return undefined;
    const bid = object(row.yourBid, 'Your bid', ['id', 'franchise', 'team', 'contract', 'points',
      'placedAt', 'placed']);
    if (bid.franchise !== mine) throw new Error('Private bid scope changed');
    if (typeof row.youLead !== 'boolean') throw new Error('Unreadable own bid state');
    return { player: text(row.player, p), name: text(row.name, p), contract: text(bid.contract, p),
      state: 'PLACED · PRIVATE', bidId: requiredText(bid.id, 'Your bid ID'), kind: 'placed' as const };
  }).filter((b): b is WarRoomOwnBid => b !== undefined);
  return { bids, revision: integer(r.revision, 'Bid version') };
}
export async function readWarRoomBids(c: WarRoomContext): Promise<WarRoomBidReading> {
  if (!c.native || c.native.owner === false) throw new Error('Owner bids are unavailable');
  const native = c.native;
  const { data } = await (native.port ?? hostedWorkflow)({ command: 'bids.read', mode: 'read',
    arguments: { league: native.league, franchise: c.mine } });
  return parseWarRoomBidReading(data, native.league, c.mine);
}
export async function readWarRoomCap(c: WarRoomContext): Promise<number | undefined> {
  if (!c.native || c.native.owner === false) return undefined;
  const native = c.native;
  const { data } = await (native.port ?? hostedWorkflow)({ command: 'native.cap.check', mode: 'read',
    arguments: { league: native.league, viewerZone: Intl.DateTimeFormat().resolvedOptions().timeZone } });
  return nativeHQCap(data, native.league, c.mine)[0]?.room;
}
export async function readWarRoomMarket(c: WarRoomContext): Promise<WarRoomMarketReading> {
  const result: WarRoomMarketReading = { errors: [] };
  const apply = async (fn: () => Promise<void>) => { try { await fn(); } catch (e) {
    result.errors.push(String(e)); console.warn('War Room market read failed:', e); } };
  if (c.native) {
    const native = c.native;
    const port = native.port ?? hostedWorkflow;
    let season: number | undefined;
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
        season = roster.season;
        result.players = roster.market; result.writable = r.writable; result.football = r.football;
      }),
      apply(async () => { if (native.owner === false) return;
        const bids = await readWarRoomBids(c);
        result.bids = bids.bids; result.bidRevision = bids.revision; }),
      apply(async () => { if (native.owner === false) return;
        const { data } = await port({ command: 'player.watch.read', mode: 'read',
          arguments: { league: native.league } });
        result.targets = parseWarRoomWatchList(data, native.league).map((id, index) => ({
          kind: 'watch', entity: 'player', id, text: '', at: '', order: index + 1,
        })); }),
      apply(async () => { result.capRoom = native.reading?.runway?.[0]?.room ?? await readWarRoomCap(c); }),
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
    let playerValues: Map<string, LeaguePlayerValue> | undefined;
    await apply(async () => {
      if (!season || !result.players?.length) return;
      const { data } = await port({ command: 'value.read', mode: 'read', arguments: {
        league: native.league, franchise: '', season } });
      const values = parseNativeValues(data, native.league, '', season);
      playerValues = new Map(values.players.map(value => [value.player, value]));
    });
    if (result.players) result.players = result.players.map(player =>
      hostedMarketPlayer(player, playerValues?.get(player.id)));
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
export type WarRoomMarketAction =
  | { type: 'draft'; form: BidFormState }
  | { type: 'revise'; bid: WarRoomOwnBid; form: BidFormState }
  | { type: 'place'; player: WarRoomMarketPlayer; salaryCents: number; years: number;
      revision: number; maxYears: number; capRoom?: number; writable: boolean; football: string }
  | { type: 'raise'; bid: WarRoomOwnBid; salaryCents: number; years: number;
      revision: number; maxYears: number; capRoom?: number; writable: boolean; football: string }
  | { type: 'confirm'; request: NativeRequest; football: string };
export type WarRoomMarketActionResult =
  | { kind: 'plan'; plan: Plan }
  | { kind: 'review'; request: NativeRequest; preview: Preview; football: string }
  | { kind: 'recorded'; receipt: ReturnType<typeof parseReceipt>; bids?: WarRoomBidReading;
      capRoom?: number; refreshError?: string };
function validatePrivateBidPreview(request: NativeRequest, league: string) {
  const args = request.arguments as unknown as { league?: unknown; command?: unknown; visibility?: unknown };
  if (request.mode !== 'preview' || !(['bid.place', 'bid.raise'] as string[])
    .includes(request.command) || args.league !== league || args.command !== request.command ||
    args.visibility !== 'private') throw new Error('Only a reviewed private bid can be confirmed');
}
function validateHostedContract(salaryCents: number, years: number, maxYears: number,
  capRoom?: number) {
  if (!Number.isSafeInteger(salaryCents) || salaryCents <= 0)
    throw new Error('Enter a bid greater than zero');
  if (!Number.isSafeInteger(years) || years < 1 || years > maxYears)
    throw new Error('Choose a contract length within the league limit');
  if (capRoom === undefined) throw new Error('Cap room is unavailable; bid was not changed');
  if (salaryCents > capRoom) throw new Error('Bid exceeds your cap room');
}
export async function warRoomMarketAction(c: WarRoomContext, action: WarRoomMarketAction):
  Promise<WarRoomMarketActionResult> {
  if (!c.native) {
    if (action.type === 'draft') return { kind: 'plan', plan: await poolProvider.draft(c.mine, action.form) };
    if (action.type === 'revise' && action.bid.plan) return { kind: 'plan', plan:
      await poolProvider.revise(action.bid.plan.id, c.mine, action.form) };
    throw new Error('This bid action is not available in local mode');
  }
  const native = c.native;
  const port = native.port ?? hostedWorkflow;
  if (native.owner === false) throw new Error('Bid actions require your franchise owner seat');
  if (action.type === 'confirm') {
    validatePrivateBidPreview(action.request, native.league);
    if (!action.football) throw new Error('Football evidence is unavailable; refresh before confirming');
    const request = { ...action.request, mode: 'execute' } as NativeRequest;
    const response = await port(request, action.football);
    const receipt = parseReceipt(response.data);
    if (receipt.league !== native.league) throw new Error('Bid receipt belongs to another league');
    const [bids, cap] = await Promise.allSettled([readWarRoomBids(c), readWarRoomCap(c)]);
    const refreshError = bids.status === 'rejected' || cap.status === 'rejected' ?
      'Your bid was recorded, but its current status or cap room could not be refreshed. ' +
        'Refresh before trying again.' : undefined;
    return { kind: 'recorded', receipt,
      ...(bids.status === 'fulfilled' ? { bids: bids.value } : {}),
      ...(cap.status === 'fulfilled' ? { capRoom: cap.value } : {}),
      ...(refreshError ? { refreshError } : {}) };
  }
  let command: 'bid.place' | 'bid.raise';
  let bid: Record<string, unknown>;
  if (action.type === 'place') {
    validateHostedContract(action.salaryCents, action.years, action.maxYears, action.capRoom);
    command = 'bid.place';
    bid = { player: action.player.id, franchise: c.mine,
      salaryCents: action.salaryCents, years: action.years };
  } else if (action.type === 'raise') {
    validateHostedContract(action.salaryCents, action.years, action.maxYears, action.capRoom);
    if (!action.bid.bidId) throw new Error('Your recorded bid is unavailable; refresh before revising');
    command = 'bid.raise';
    bid = { bid: action.bid.bidId, franchise: c.mine, salaryCents: action.salaryCents, years: action.years };
  } else {
    throw new Error('A hosted bid command is required');
  }
  if (!action.writable || !action.football)
    throw new Error('This league is read-only or football evidence is unavailable; refresh before bidding');
  const request = { command, mode: 'preview', arguments: { league: native.league, command,
    args: { bid }, reason: '', expected: action.revision, request: newKey(), effective: 'now',
    visibility: 'private' } } as NativeRequest;
  const response = await port(request, action.football);
  if (!response.writable || response.football !== action.football)
    throw new Error('League authority changed; refresh before confirming');
  const preview = parsePreview(response.data);
  if (preview.target.league !== native.league || preview.target.version !== action.revision)
    throw new Error('Bid preview is for a different league version; refresh before confirming');
  return { kind: 'review', request, preview, football: action.football };
}
export async function changeWarRoomTarget(c: WarRoomContext, id: string, watched: boolean,
  withPlayer?: string) {
  if (c.native) {
    const port = c.native.port ?? hostedWorkflow;
    if (withPlayer) await port({ command: 'player.watch.order', mode: 'execute', arguments: {
      league: c.native.league, player: id, with: withPlayer,
    } });
    else await port({ command: 'player.watch.set', mode: 'execute', arguments: {
      league: c.native.league, player: id, watched: !watched,
    } });
    return;
  }
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
