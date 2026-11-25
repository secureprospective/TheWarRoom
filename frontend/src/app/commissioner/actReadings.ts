import { hostedWorkflow } from './api';
import type { NativeRequest } from '../commands/native';
import type { FieldKind } from './actTable';
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
  principal?: string;
};
export type HostedActReadings = Pick<ActReadings, 'league' | 'revision' | 'season' | 'options' | 'principal'>;
const unique = (items: Option[]) => Array.from(new Map(items.map(item => [item.value, item])).values());

// Only the readings a command needs are opened. A private reading's refusal cannot disable an unrelated act.
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
