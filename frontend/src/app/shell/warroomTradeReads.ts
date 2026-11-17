import { ageAt } from '../cards/format';
import { array, choice, object, text } from '../data/parseValues';
import type { TradeAsset } from '../data/contract';
import { targetTradeBuilderRead, targetTradeBuilderPreview } from '../trades/api';
import { parseBuilder } from './tradeBuilderState';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import { nativeHQAssets } from '../commissioner/hqReading';
import { parsePreview } from '../commissioner/preview';
import { newKey } from '../commissioner/forms';
import { parseReceipt } from '../commissioner/proxy';
import type { WarRoomContext } from './warroomReads';
import { finite } from './hqPlayerData';
import { money, rosterContract } from './hqModel';
export type WarRoomAsset = TradeAsset & { age?: number;
  pick?: { season: number; round: number; original: string } };
export type WarRoomCatalog = { teams: { id: string; name: string; assets: WarRoomAsset[] }[];
  revision?: number };
export type WarRoomDeal = { partner: string; give: string[]; get: string[] };
export type WarRoomDeltaSide = { franchise: string; perWeek: number; thisWeek: number;
  before: { player: string; position: string; ppg: number }[];
  after: { player: string; position: string; ppg: number }[];
  changes: { kind: 'enters' | 'leaves' | 'benched' | 'promoted'; player: string; position: string }[];
  unprojected: string[] };
export function parseWarRoomDelta(value: unknown) {
  const r = object(value, 'Lineup delta', ['league', 'season', 'week', 'you', 'them', 'notes']);
  const side = (v: unknown): WarRoomDeltaSide => {
    const s = object(v, 'Side', ['franchise', 'before', 'after', 'perWeek', 'thisWeek', 'changes',
      'unprojected']);
    const starters = (v: unknown) => array(v, 'Starters', (v, p) => {
      const s = object(v, p, ['player', 'position', 'ppg']);
      return { player: text(s.player, p), position: text(s.position, p), ppg: finite(s.ppg, p) };
    });
    return { franchise: text(s.franchise, 'Franchise'), before: starters(s.before), after: starters(s.after),
      perWeek: finite(s.perWeek, 'Per week'), thisWeek: finite(s.thisWeek, 'This week'),
      unprojected: array(s.unprojected, 'Unprojected players', text),
      changes: array(s.changes, 'Changes', (v, p) => { const c = object(v, p, ['kind', 'player', 'position']);
        return { kind: choice(c.kind, p, ['enters', 'leaves', 'benched', 'promoted']),
          player: text(c.player, p), position: text(c.position, p) }; }) };
  };
  return { league: text(r.league, 'League'), season: finite(r.season, 'Season'), week: finite(r.week,
    'Week'), you: side(r.you), them: side(r.them), notes: array(r.notes, 'Notes', text) };
}
export async function readWarRoomCatalog(c: WarRoomContext): Promise<WarRoomCatalog> {
  if (!c.native) {
    const r = parseBuilder(await targetTradeBuilderRead(c.mine));
    if (!r.value || r.status !== 'ok') throw new Error(r.absenceReason ?? 'No trade catalog');
    return { teams: r.value.teams.map(t => ({ ...t, assets: t.assets.map(a => ({ ...a,
      age: ageAt(c.snapshot.players.value.find(p => p.id === a.token)?.birthdate, new Date()) })) })) };
  }
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'native.assets.read', mode: 'read',
    arguments: { league: c.native.league } });
  const roster = parseHostedRoster(data);
  const snapshot = nativeHQAssets(data, c.native.league);
  if (roster.league !== c.native.league) throw new Error('Trade catalog scope changed');
  return { revision: roster.revision, teams: roster.franchises.map(f => ({ id: f.value, name: f.label,
    assets: [...roster.players.filter(p => p.franchise === f.value).map(p => {
      const contract = snapshot.rosters.value.find(r => r.franchiseId === f.value)?.players.find(r =>
        r.id === p.id);
      const identity = roster.identities.find(i => i.player === p.id && i.franchise === f.value)
        ?? c.snapshot.players.value.find(i => i.id === p.id);
      return { token: p.id, name: identity?.name ?? p.name,
        position: identity?.position,
        contract: rosterContract(contract) };
    }), ...roster.picks.filter(p => p.owner === f.value && !p.used).map(p => ({ token:
      `pick:${p.season}:${p.round}:${p.original}`, name: `${p.season} round ${p.round}`, pick: p }))] })) };
}
export async function readWarRoomPreview(c: WarRoomContext, catalog: WarRoomCatalog, deal: WarRoomDeal) {
  if (!c.native) {
    const r = parseBuilder(await targetTradeBuilderPreview({ mine: c.mine, other: deal.partner,
      give: deal.give, get: deal.get, rationale: '', action: 'preview', id: '' }));
    if (!r.value?.preview || r.status !== 'ok') throw new Error(r.absenceReason ?? 'No trade preview');
    return { reasons: r.value.preview.reasons, teams: r.value.preview.teams };
  }
  const native = c.native;
  const assets = (tokens: string[], from: string, to: string) => tokens.map(token =>
    savedTradeAsset(token, from, to));
  const response = await (native.port ?? hostedWorkflow)({ command: 'trade.offer', mode: 'preview',
    arguments: { league: native.league, command: 'trade.offer', expected: catalog.revision ?? 0,
      request: newKey(), reason: '', effective: 'now', visibility: 'private', args: { trade: {
        franchise: c.mine, parties: [c.mine, deal.partner], assets: [...assets(deal.give, c.mine,
          deal.partner), ...assets(deal.get, deal.partner, c.mine)] } } } });
  const preview = parsePreview(response.data);
  if (preview.target.league !== native.league) throw new Error('Trade preview belongs to another league');
  return { reasons: preview.blockers, teams: [{ team: c.mine, figures: preview.fields.map(f => ({
    label: f.label, before: f.before, after: f.after, change: f.when })), roster: [] }] };
}
export async function readWarRoomDelta(c: WarRoomContext, catalog: WarRoomCatalog, deal: WarRoomDeal) {
  if (!c.native || !c.week || !deal.partner || !(deal.give.length + deal.get.length)) return undefined;
  const players = (tokens: string[], team: string) => tokens.filter(token => !catalog.teams.find(t =>
    t.id === team)?.assets.find(a => a.token === token)?.pick);
  const give = players(deal.give, c.mine), get = players(deal.get, deal.partner);
  if (!give.length && !get.length) return undefined;
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'lineupdelta.read', mode: 'read',
    arguments: { league: c.native.league, franchise: c.mine, partner: deal.partner, give, get,
      season: c.snapshot.league.value.season, week: c.week } });
  const r = parseWarRoomDelta(data);
  if (r.league !== c.native.league || r.season !== c.snapshot.league.value.season || r.week !== c.week ||
    r.you.franchise !== c.mine || r.them.franchise !== deal.partner)
    throw new Error('Lineup delta belongs to another proposal scope');
  return r;
}

export async function readWarRoomPartner(c: WarRoomContext, partner: string) {
  if (!partner) return undefined;
  if (!c.native) {
    const { raceAPI } = await import('../race/api');
    const { standingFromReport } = await import('./hqPlayerData');
    const race = await raceAPI.read(c.week, partner, partner, 'topn');
    return { record: standingFromReport(race.standings).record,
      room: c.snapshot.franchises.value.find(f => f.id === partner)?.capRoom };
  }
  const native = c.native;
  const port = native.port ?? hostedWorkflow;
  const [standings, cap] = await Promise.all([
    port({ command: 'standings.read', mode: 'read', arguments: { league: native.league,
      season: c.snapshot.league.value.season } }),
    port({ command: 'native.cap.check', mode: 'read', arguments: { league: native.league,
      viewerZone: Intl.DateTimeFormat().resolvedOptions().timeZone } }),
  ]);
  const { parseHostedStandings } = await import('../commissioner/hostedMarkets');
  const { nativeHQCap } = await import('../commissioner/hqReading');
  const checked = parseHostedStandings(standings.data);
  if (checked.league !== native.league) throw new Error('Partner standings belong to another league');
  const rows = checked.override?.rows ?? checked.computed;
  const row = rows.find(r => r.franchise === partner);
  return { record: row?.record, room: nativeHQCap(cap.data, native.league, partner)[0]?.room };
}

function savedTradeAsset(token: string, from: string, to: string) {
  if (!token.startsWith('pick:')) return { from, to, player: token };
  const [, season, round, original, ...extra] = token.split(':');
  const parsedSeason = Number(season), parsedRound = Number(round);
  if (extra.length || !Number.isInteger(parsedSeason) || !Number.isInteger(parsedRound) ||
    !original || parsedSeason < 1 || parsedRound < 1)
    throw new Error('This saved asset is no longer available');
  return { from, to, season: parsedSeason, round: parsedRound, original };
}

export async function executeWarRoomOffer(c: WarRoomContext, catalog: WarRoomCatalog,
  deal: WarRoomDeal, request = newKey()) {
  if (!c.native || c.native.owner === false || !deal.partner ||
    (!deal.give.length && !deal.get.length))
    throw new Error('Choose a partner and at least one asset');
  const team = (id: string) => catalog.teams.find(t => t.id === id);
  const transfer = (tokens: string[], from: string, to: string) => tokens.map(token => {
    const asset = team(from)?.assets.find(a => a.token === token);
    if (!asset) throw new Error('This asset is no longer on the selected shelf');
    return asset.pick ? { from, to, ...asset.pick } : { from, to, player: token };
  });
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'trade.offer', mode: 'execute',
    arguments: { league: c.native.league, command: 'trade.offer', expected: catalog.revision ?? 0,
      request, reason: '', effective: 'now', visibility: 'league', args: { trade: {
        franchise: c.mine, parties: [c.mine, deal.partner], assets: [...transfer(deal.give, c.mine,
          deal.partner), ...transfer(deal.get, deal.partner, c.mine)] } } } });
  const receipt = parseReceipt(data);
  if (receipt.league !== c.native.league) throw new Error('Trade receipt belongs to another league');
  const result = await (c.native.port ?? hostedWorkflow)({ command: 'trades.read', mode: 'read',
    arguments: { league: c.native.league, viewer: c.mine } });
  const trades = (await import('../commissioner/hostedMarkets')).parseHostedTrades(result.data);
  if (trades.league !== c.native.league || trades.franchise !== c.mine)
    throw new Error('Trade offers belong to another viewer');
  if (!trades.offers.some(offer => offer.id === receipt.entry))
    throw new Error('The offer was recorded, but is not present in the refreshed trade reading');
  return receipt;
}
