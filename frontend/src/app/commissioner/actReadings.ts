export { nativeBindings } from './api';
import { nativeBindings } from './api';
import type { commissioner, market } from './api';
import { hostedWorkflow } from './api';
import type { NativeRequest } from '../commands/native';
import type { FieldKind } from './actTable';
import { readCompetitionOptions } from './competitionReadings';
import { array, integer, object, text } from '../data/parseValues';
import { leagueState, parseLedger } from './proxy';
import { parsePeople } from './people';
import { parseHostedMarket } from './hostedMarkets';
import { parseHostedRoster } from './hostedReadings';
import { parsePulseBracket } from '../data/nativePulse';
import { newKey } from './forms';
import type { HostedPort } from './hostedRoom';

export type Option = { value: string; label: string };
export type ActReadings = {
  league: string; revision: number; season: number; options: Partial<Record<FieldKind, Option[]>>;
  assets: commissioner.NativeAssets; principal?: string;
};
export type HostedActReadings = Pick<ActReadings, 'league' | 'revision' | 'season' | 'options' | 'principal'>;
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
    franchise: directory.franchises.map(f => {
      const side = assets.snapshot?.sides?.[f.franchise];
      return { value: f.franchise, label: side ? `${f.name} — ${side}` : f.name };
    }),
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
  // The deputy order lists every current co-commissioner once; the people reading holds who they are.
  options.deputies = (people?.league === league ? people.roles : [])
    .filter(r => r.role === 'co-commissioner').map(r => ({ value: r.person, label: r.person }));
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
    'bids.award.override', 'trade.review', 'trade.hold', 'trade.appeal.rule', 'trade.override'];
  const market = marketCommands.includes(command);
  if (market) {
    const reading = await app.TargetCommissionerMarket(league, command);
    if (reading.league !== league || reading.revision !== assets.revision)
      throw new Error('The league changed while reading its market. Reopen the change.');
    const choices = (rows: market.Choice[]) => rows.map(r => ({ value: r.id, label: r.words }));
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
    const queue = await app.TargetStatQueue(league, year, '');
    options.item = queue.open.map(i => ({ value: i.id, label: `${i.name}, week ${i.week}: ${i.words}` }));
  }
  await readCompetitionOptions(app, league, kinds, year, assets.revision, options);
  const principal = kinds.includes('self') ? await app.TargetCommissionerActor() : undefined;
  return { league, revision: assets.revision, season: year, options, assets, principal };
}

const hostedRead = async (port: HostedPort, command: string, args: Record<string, unknown>) => {
  const request = { command, mode: 'read', arguments: args } as NativeRequest;
  return (await port(request)).data;
};

async function readHostedCompetitionOptions(league: string, revision: number, kinds: readonly FieldKind[],
  port: HostedPort, options: HostedActReadings['options'], season: number): Promise<number> {
  if (kinds.includes('scheduleGame'))
    await readHostedSchedule(league, revision, port, options);
  const bracketKinds: FieldKind[] = ['bracketNode', 'bracketGame', 'bracketSlot', 'preset', 'shape'];
  if (kinds.some(kind => bracketKinds.includes(kind)))
    await readHostedBracket(league, revision, season, port, options);
  return season;
}

async function readHostedSchedule(league: string, revision: number, port: HostedPort,
  options: HostedActReadings['options']): Promise<void> {
  const schedule = object(await hostedRead(port, 'commish.schedule.read', { league }), 'Schedule',
    ['league', 'revision', 'draft', 'published']);
  if (schedule.league !== league || integer(schedule.revision, 'Schedule revision') !== revision)
    throw new Error('The schedule changed. Reopen the change.');
  const parseWeeks = (value: unknown, label: string) => array(value, label, (row, path) => {
    const week = object(row, path, ['week', 'games', 'bye']);
    const weekNumber = integer(week.week, `${path}.week`, 1);
    const games = array(week.games, `${path}.games`, (item, gamePath) => {
      const game = object(item, gamePath, ['id', 'home', 'away']);
      return { id: text(game.id, gamePath), home: text(game.home, gamePath),
        away: text(game.away, gamePath) };
    });
    return { week: weekNumber, games };
  });
  const name = (id: string) => options.franchise?.find(team => team.value === id)?.label
    ?? 'Unnamed team';
  const draft = parseWeeks(schedule.draft, 'Draft schedule');
  const published = parseWeeks(schedule.published, 'Published schedule');
  options.scheduleGame = [
    ...draft.flatMap(week => week.games.map(game => ({
      value: JSON.stringify({ week: week.week, id: game.id, published: false }),
      label: `Draft week ${week.week}: ${name(game.home)} v ${name(game.away)}`,
    }))),
    ...published.flatMap(week => week.games.map(game => ({
      value: JSON.stringify({ week: week.week, id: game.id, published: true }),
      label: `Published week ${week.week}: ${name(game.home)} v ${name(game.away)}`,
    }))),
  ];
}

async function readHostedBracket(league: string, revision: number, season: number, port: HostedPort,
  options: HostedActReadings['options']): Promise<void> {
  const raw = await hostedRead(port, 'bracket.read', { league, season });
  const bracket = parsePulseBracket(raw, league, season);
  if (bracket.revision !== revision) throw new Error('The playoffs changed. Reopen the change.');
  const value = object(raw, 'Playoffs', ['league', 'name', 'season', 'revision', 'status', 'words',
    'preset', 'presets', 'rules', 'started', 'startKnown', 'fields', 'projected', 'rounds', 'champion',
    'title', 'history', 'changes']);
  const presets = array(value.presets, 'Saved playoff layouts', text);
  options.preset = [
    { value: 'league_rules', label: 'League playoff rules' },
    { value: 'legacy_nfl', label: 'Legacy NFL playoff layout' },
    ...presets.map(preset => ({ value: preset, label: `Saved layout: ${preset}` })),
  ];
  options.shape = bracket.shape.length
    ? [{ value: JSON.stringify(bracket.shape), label: 'Current playoff layout' }] : [];
  options.bracketSlot = bracketSlots(bracket.fields);
  options.bracketNode = [];
  options.bracketGame = [];
  const rounds = parseBracketRounds(value.rounds);
  for (const round of rounds) for (const game of round.games) {
    const label = `${round.name}, week ${game.week}: ${game.home} v ${game.away}`;
    options.bracketGame.push({ value: game.id, label });
    options.bracketNode.push({ value: game.homeSlot, label: `${label} — home team` },
      { value: game.awaySlot, label: `${label} — away team` });
  }
}

function bracketSlots(fields: ReturnType<typeof parsePulseBracket>['fields']): Option[] {
  const slots = new Map<string, Option>();
  for (const field of fields) for (const team of field.seeds) {
    slots.set(String(team.seed), { value: String(team.seed),
      label: `Seed ${team.seed} in the chosen team’s conference` });
  }
  return [...slots.values()];
}

function parseBracketRounds(value: unknown) {
  return array(value, 'Bracket rounds', (row, path) => {
    const round = object(row, path, ['round', 'week', 'name', 'games']);
    const name = text(round.name, path);
    const week = integer(round.week, path, 1);
    const games = array(round.games, `${path}.games`, (item, gamePath) => {
      const game = object(item, gamePath, ['id', 'node', 'conference', 'week', 'homeSlot',
        'awaySlot', 'home', 'away', 'status', 'homePoints', 'awayPoints', 'homeBonus',
        'homeTotal', 'awayTotal', 'winner', 'byHand', 'words']);
      const sideName = (side: unknown, label: string) => {
        if (side === undefined) return 'Team to be decided';
        const team = object(side, label, ['franchise', 'name', 'seed', 'rank', 'byHand']);
        return `${text(team.name, label)} (seed ${integer(team.seed, label, 1)})`;
      };
      return { id: text(game.id, gamePath), week: integer(game.week, gamePath, 1),
        homeSlot: text(game.homeSlot, gamePath), awaySlot: text(game.awaySlot, gamePath),
        home: sideName(game.home, `${gamePath}.home`), away: sideName(game.away, `${gamePath}.away`) };
    });
    return { name, week, games };
  });
}

/** Reads only the held options required by a hosted commissioner form. The server admits each read. */
export async function readHostedActData(league: string, revision: number, kinds: readonly FieldKind[],
  command: string, port: HostedPort = hostedWorkflow, requestedSeason?: number): Promise<HostedActReadings> {
  const options: HostedActReadings['options'] = {};
  const needsTeams = command.startsWith('bracket.') || kinds.some(kind => ['franchise',
    'franchiseOrAll', 'order', 'deputies', 'scope', 'scheduleGame', 'item'].includes(kind));
  let season = requestedSeason ?? 0;
  if (needsTeams) {
    const value = object(await hostedRead(port, 'native.directory.read', { league }), 'League directory',
      ['league', 'name', 'revision', 'rosterSeason', 'source', 'franchises']);
    if (value.league !== league || integer(value.revision, 'Directory revision') !== revision)
      throw new Error('The league changed while reading commissioner choices. Reopen the change.');
    season = requestedSeason ?? integer(value.rosterSeason, 'Roster season');
    options.franchise = array(value.franchises, 'Franchises', (row, path) => {
      const team = object(row, path, ['franchise', 'name', 'abbreviation', 'owner', 'caretaker', 'division']);
      return { value: text(team.franchise, path), label: text(team.name, path) };
    });
    options.franchiseOrAll = [{ value: '*', label: 'All franchises' }, ...options.franchise];
    options.order = options.franchise;
    options.scope = options.franchise.map(team => ({ value: JSON.stringify({ franchise: team.value }),
      label: `Franchise: ${team.label}` }));
  }
  const needsAssets = kinds.some(kind => ['player', 'pick', 'scope', 'starters'].includes(kind));
  if (needsAssets) {
    const roster = parseHostedRoster(await hostedRead(port, 'native.assets.read', { league }));
    if (roster.league !== league || roster.revision !== revision)
      throw new Error('The league changed while reading commissioner assets. Reopen the change.');
    season = roster.season;
    const players = unique([
      ...roster.players.map(player => ({ value: player.id, label: player.name })),
      ...roster.pool.map(player => ({ value: player.value, label: player.label })),
    ]);
    options.player = players;
    options.pick = roster.picks.filter(pick => pick.original).map(pick => {
      const original = options.franchise?.find(team => team.value === pick.original)?.label
        ?? 'Original franchise';
      return {
        value: JSON.stringify({ season: pick.season, round: pick.round, original: pick.original,
          slot: pick.slot }),
        label: `${pick.season}, round ${pick.round}, slot ${pick.slot}: ${original}`,
      };
    });
    options.scope = [...(options.scope ?? []), ...players.map(player => ({
      value: JSON.stringify({ player: player.value }), label: `Player: ${player.label}`,
    }))];
  }
  if (kinds.includes('entry')) {
    const ledger = parseLedger(await hostedRead(port, 'commish.ledger.read', { league }));
    if (ledger.league !== league || ledger.revision !== revision)
      throw new Error('The league changed while reading the ledger. Reopen the change.');
    const entries = command === 'commish.correction.revert'
      ? ledger.entries.filter(entry => entry.undo !== '' && !entry.replacedAt)
      : ledger.entries;
    options.entry = entries.map(entry => ({ value: entry.id, label: entry.what }));
    options.scope = [...(options.scope ?? []), ...options.entry.map(entry => ({
      value: JSON.stringify({ transaction: entry.value }), label: `Transaction: ${entry.label}`,
    }))];
  }
  if (kinds.includes('deputies')) {
    const people = parsePeople(await hostedRead(port, 'commish.people.read', { league }));
    if (people.league !== league || people.revision !== revision)
      throw new Error('The league changed while reading commissioner roles. Reopen the change.');
    options.deputies = people.roles.filter(role => role.role === 'co-commissioner')
      .map(role => ({ value: role.person, label: role.person }));
  }
  const marketCommands = ['waivers.process', 'waivers.award.override', 'bids.process',
    'bids.award.override', 'trade.review', 'trade.hold', 'trade.appeal.rule', 'trade.override',
    'trade.recuse', 'trade.appeal'];
  const marketFields: FieldKind[] = ['window', 'claim', 'bid', 'bidPlayer', 'trade'];
  if (marketCommands.includes(command) && kinds.some(kind => marketFields.includes(kind))) {
    const market = parseHostedMarket(await hostedRead(port, 'commish.market.read', { league, command }));
    if (market.league !== league || market.revision !== revision)
      throw new Error('The league changed while reading commissioner choices. Reopen the change.');
    Object.assign(options, market.options);
  }
  season = await readHostedCompetitionOptions(league, revision, kinds, port, options, season);
  if (kinds.includes('item')) {
    const queue = object(await hostedRead(port, 'stats.queue.read', { league, season,
      request: newKey() }), 'Stat queue', ['league', 'season', 'revision', 'words', 'open', 'decided']);
    if (queue.league !== league || integer(queue.revision, 'Stat queue revision') !== revision
      || integer(queue.season, 'Stat queue season') !== season)
      throw new Error('The statistics queue changed. Reopen the change.');
    options.item = array(queue.open, 'Open statistics changes', (row, path) => {
      const item = object(row, path, ['id', 'player', 'name', 'season', 'week', 'changes', 'points',
        'flip', 'flipWords', 'words', 'arrived', 'sha256']);
      const label = `${text(item.name, path)}, week ${integer(item.week, path)}: ${text(item.words, path)}`;
      return { value: text(item.id, path), label };
    });
  }
  if (kinds.includes('self')) {
    const principal = text(await hostedRead(port, 'commish.actor.read', {}), 'Commissioner identity');
    return { league, revision, season, options, principal };
  }
  return { league, revision, season, options };
}
