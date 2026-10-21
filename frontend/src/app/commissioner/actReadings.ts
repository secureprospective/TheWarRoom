export { nativeBindings } from './api';
import { nativeBindings } from './api';
import type { commissioner } from './api';
import type { FieldKind } from './actTable';
import { readCompetitionOptions } from './competitionReadings';
import { leagueState } from './proxy';


export type Option = { value: string; label: string };
export type ActReadings = {
  league: string; revision: number; season: number; options: Partial<Record<FieldKind, Option[]>>;
  assets: commissioner.NativeAssets; principal?: string;
};
const unique = (items: Option[]) => Array.from(new Map(items.map(item => [item.value, item])).values());

// Only the readings a command needs are opened. A private reading's refusal cannot disable an unrelated act.
export async function readActData(league: string, kinds: readonly FieldKind[], season?: number,
  franchise = '', command = ''): Promise<ActReadings> {
  const app = nativeBindings();
  const people = leagueState.peek().people;
  const viewer = franchise || (people?.league === league ? people.yourFranchise : '');
  const [assets, directory] = await Promise.all([
    app.TargetNativeAssets(league), app.TargetNativeDirectory(league)]);
  const year = season ?? directory.rosterSeason;
  const ledger = leagueState.peek().ledger;
  if (assets.league !== league || directory.league !== league || assets.revision !== directory.revision)
    throw new Error('The league changed while reading its assets. Reopen the change.');
  const options: ActReadings['options'] = {
    franchise: directory.franchises.map(f => ({ value: f.franchise, label: f.name })),
    player: unique([...(assets.snapshot?.players ?? []), ...(assets.snapshot?.pool ?? [])]
      .map(p => ({ value: 'id' in p ? p.id : p.player, label: p.name || 'Unnamed player' }))),
    pick: (assets.snapshot?.picks ?? []).filter(p => p.original).map(p => ({
      value: JSON.stringify({ season: p.season, round: p.round, original: p.original }),
      label: `${p.season}, round ${p.round}, ${directory.franchises.find(f =>
        f.franchise === p.original)?.name ?? 'Original franchise'}`,
    })),
    entry: (ledger?.league === league ? ledger.entries : []).map(e => ({ value: e.id,
      label: `${e.what} — ${new Date(e.when).toLocaleString()}` })),
  };
  options.franchiseOrAll = [{ value: '*', label: 'All franchises' }, ...(options.franchise ?? [])];
  options.scope = [
    ...(options.franchise ?? []).map(o => ({ value: JSON.stringify({ franchise: o.value }),
      label: `Franchise: ${o.label}` })),
    ...(options.player ?? []).map(o => ({ value: JSON.stringify({ player: o.value }),
      label: `Player: ${o.label}` })),
    ...(options.entry ?? []).map(o => ({ value: JSON.stringify({ transaction: o.value }),
      label: `Transaction: ${o.label}` })),
  ];
  const marketCommands = ['waivers.process', 'waivers.award.override', 'bids.process',
    'bids.award.override', 'trade.review', 'trade.appeal.rule', 'trade.override'];
  const market = marketCommands.includes(command);
  if (market) {
    const reading = await app.TargetCommissionerMarket(league, command);
    if (reading.league !== league || reading.revision !== assets.revision)
      throw new Error('The league changed while reading its market. Reopen the change.');
    const choices = (rows: commissioner.MarketChoice[]) => rows.map(r => ({ value: r.id, label: r.words }));
    options.window = choices(reading.windows);
    options.claim = choices(reading.claims);
    options.bidPlayer = choices(reading.players);
    options.bid = choices(reading.bids);
    options.trade = choices(reading.trades);
  }
  if (!market && kinds.includes('trade')) {
    const trades = await app.TargetNativeTrades(league, viewer);
    options.trade = trades.offers.map(t => ({ value: t.id, label: t.words }));
  }
  if (!market && (kinds.includes('bid') || kinds.includes('bidPlayer'))) {
    const bids = await app.TargetNativeBids(league, viewer);
    options.bidPlayer = bids.players.map(p => ({ value: p.player, label: p.name }));
    options.bid = unique(bids.players.flatMap(p => [p.yourBid, p.leader].flatMap(b =>
      b ? [{ value: b.id, label: `${p.name} — ${b.team}: ${b.contract}` }] : [])));
  }
  if (!market && (kinds.includes('window') || kinds.includes('claim'))) {
    const waivers = await app.TargetNativeWaivers(league, viewer, new Date().toISOString());
    options.window = waivers.players.map(p => ({ value: p.window, label: `${p.name} — ${p.closes}` }));
    options.claim = waivers.claims.map(c => ({ value: c.id,
      label: `${options.player?.find(p => p.value === c.player)?.label ?? 'Player'} — ${c.status}` }));
  }
  if (kinds.includes('item')) {
    const queue = await app.TargetStatQueue(league, year);
    options.item = queue.open.map(i => ({ value: i.id, label: `${i.name}, week ${i.week}: ${i.words}` }));
  }
  await readCompetitionOptions(app, league, kinds, year, assets.revision, options);
  const principal = kinds.includes('self') ? await app.TargetCommissionerActor() : undefined;
  return { league, revision: assets.revision, season: year, options, assets, principal };
}
