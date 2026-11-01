import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import { CONTRACT_STATUSES, POSITIONS, ROSTER_STATUSES, type Position, type Snapshot, type Provenance }
  from '../data/contract';
import { parseRules, type Rules } from './rules';
import { parseHostedLineup, parseHostedRoster, parseHostedReserves } from './hostedReadings';
import { parseNativeMoves } from '../data/nativeMoves';
import type { HQMove } from '../shell/HQMoves';
import type { HQCap, HQSettings } from '../shell/hqModel';
import type { HQLineup } from '../shell/HQBoard';
import type { HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';
import { selectProvider } from '../data/provider';
import { ordinal } from '../shell/hqPlayerData';
import { parseWarRoomProjections } from '../data/nativeProjections';

export function nativeHQStanding(rules: Rules | undefined,
  rows: { franchise: string; division: string }[], mine: string): string | undefined {
  const raw = rules?.sections.flatMap(s => s.rows).find(r =>
    r.field === 'league.alignment' && r.state === 'Set')?.current;
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return undefined;
  const alignment = new Map(Object.entries(raw).map(([id, value]) => [id, text(value, 'Division')]));
  const name = alignment.get(mine);
  // Older seeds contain only ledger identifiers, not display names.
  if (!name || /^division[._]\d+$/i.test(name)) return undefined;
  const group = rows.filter(r => alignment.get(r.franchise) === name);
  const index = group.findIndex(r => r.franchise === mine);
  return index < 0 ? undefined : `${ordinal(index + 1)} ${name}`;
}
export function nativeHQSettings(rules: Rules): HQSettings {
  const rows = rules.sections.flatMap(s => s.rows);
  const current = (field: string) => rows.find(r => r.field === field && r.state === 'Set')?.current;
  const count = (field: string) => current(field) == null ? undefined : integer(current(field), field);
  const bounds = (value: unknown) => {
    if (value == null) return {};
    const r = object(value, 'Slot bounds', ['Min', 'Max']);
    return { min: r.Min == null ? undefined : integer(r.Min, 'Minimum'), max: r.Max == null ? undefined :
      integer(r.Max, 'Maximum') };
  };
  const reserveExcluded = current('roster.reserveExcluded');
  if (reserveExcluded != null && typeof reserveExcluded !== 'boolean')
    throw new Error('Unreadable reserve roster rule');
  const positions = current('lineup.positions');
  const groups = current('lineup.groups');
  const group = groups == null ? {} : object(groups, 'Lineup groups', ['offense', 'defense', 'total']);
  const positionMap = positions == null ? {} : object(positions, 'Lineup positions', ['QB', 'RB', 'WR',
    'TE', 'PK', 'K', 'DT', 'DE', 'LB', 'CB', 'S', 'OL']);
  return { reserveExcluded: reserveExcluded == null ? undefined : reserveExcluded as boolean,
    active: bounds(current('roster.limits')).max, ir: count('ir.slots'), taxi: count('taxi.slots'),
    starters: bounds(group.total).max, offence: bounds(group.offense).min, defence: bounds(group.defense).min,
    positions: Object.entries(positionMap).map(([code, value]) => { const b = bounds(value);
      return { code: code === 'PK' ? 'K' : code, min: b.min ?? 0, max: b.max };
        }) };
}
export function nativeHQAssets(value: unknown, league: string): Snapshot {
  const held = parseHostedRoster(value);
  if (held.league !== league) throw new Error('Roster belongs to another league');
  const raw = value as { snapshot: { players: unknown[] } };
  const players = array(raw.snapshot.players, 'Roster contracts', (v, p) => {
    const r = v as Record<string, unknown>;
    return { id: text(r.id, p), franchise: text(r.franchise, p), salary: integer(r.salaryCents, p),
      // An imported league may hold no contract status (MFL left it blank);
      // the MFL mirror reads blank as FLAG.
      expiration: integer(r.expirationYear, p),
      contractStatus: choice(r.contractStatus === '' ? 'FLAG' : r.contractStatus, p, CONTRACT_STATUSES),
      rosterStatus: choice(r.status, p, ROSTER_STATUSES) };
  });
  const provenance: Provenance = { source: 'League ledger', kind: 'live', freshness: { state: 'live',
    fetchedAt: '', note: 'Recorded league assets' } };
  return { league: { value: { season: held.season, franchiseCount: held.franchises.length }, provenance },
    franchises: { value: held.franchises.map(f => ({ id: f.value, name: f.label })), provenance },
    rosters: { value: held.franchises.map(f => ({ franchiseId: f.value, players: players.filter(p =>
      p.franchise === f.value).map(p => ({ id: p.id, salary: p.salary, contractStatus: p.contractStatus,
      rosterStatus: p.rosterStatus,
      ...(p.expiration ? { yearsRemaining: Math.max(0, p.expiration - held.season + 1) } : {}) })) })),
        provenance },
    players: { value: held.players.map(p => ({ id: p.id, name: p.name })), provenance },
    eligibility: { players: {}, feeds: { injuries: provenance, players: provenance, rosters: provenance } } };
}
export function nativeHQCap(value: unknown, league: string, mine: string): HQCap[] {
  const r = object(value, 'Cap check', ['corrections', 'league', 'name', 'revision', 'rosterSeason',
    'rulesSeason', 'seeded', 'source', 'counting', 'rules', 'franchises']);
  if (r.league !== league) throw new Error('Cap belongs to another league');
  const franchises = array(r.franchises, 'Franchise caps', (v, p) => {
    const f = object(v, p, ['exception', 'capCents', 'salaryCents', 'activeCents', 'irCents', 'taxiCents',
      'adjustmentCents', 'committedCents', 'roomCents', 'quantumCents', 'mode', 'deadline', 'words',
      'enforcement', 'franchise', 'name', 'abbreviation', 'charges', 'runway', 'contracts']);
    return { franchise: text(f.franchise, p), runway: array(f.runway, 'Runway', (v, p) => {
      const y = object(v, p, ['season', 'committedCents', 'deadCents', 'capCents', 'roomCents']);
      const amount = (key: string) => y[key] === null ? undefined : integer(y[key], key,
        Number.MIN_SAFE_INTEGER);
      return { season: integer(y.season, p),
        ...(y.season === r.rulesSeason ? { ir: f.irCents === null ? undefined : integer(f.irCents,
          'IR pay') } : {}),
        committed: amount('committedCents'), dead:
        amount('deadCents'), max: amount('capCents'), room: amount('roomCents') };
    }) };
  });
  return franchises.find(f => f.franchise === mine)?.runway ?? [];
}
export function nativeHQMoves(value: unknown, league: string, mine: string): { moves: HQMove[]; reason:
  string } {
  const reading = parseNativeMoves(value);
  if (reading.league !== league || reading.franchise !== mine)
    throw new Error('Moves belong to another franchise');
  return { reason: reading.reason, moves: reading.moves.map(m => ({
    id: m.correlationId, state: m.state, intent: m.spec.intent, at: m.audit.at(-1)?.at ?? '',
    note: m.audit.at(-1)?.note ?? '', players: m.spec.subject.players, native: true,
  })) };

}
export type NativeHQ = { revision: number; snapshot: Snapshot; settings: HQSettings; leagueName?: string;
  lineup?: HQLineup; contracts?: Map<string, { season: number; cents: number }[]>;
  check?: string; lock?: string; runway?: HQCap[]; moves?: HQMove[]; moveReason?: string;
  reserveNotes?: Map<string, string>; record?: string; standing?: string; week?: number; weekNote?: string;
  projections?: Map<string, number>; errors: string[] };
export async function readNativeHQ(league: string, mine: string, port: HostedPort = hostedWorkflow):
  Promise<NativeHQ> {
  const assets = await port({ command: 'native.assets.read', mode: 'read', arguments: { league } });
  const snapshot = nativeHQAssets(assets.data, league);
  const season = snapshot.league.value.season;
  const revision = integer((assets.data as { revision: unknown }).revision, 'Roster revision');
  const sameRevision = (value: unknown) => {
    if (integer((value as { revision: unknown }).revision, 'Reading revision') !== revision)
      throw new Error('League changed during the read; refresh before using this region');
  };
  let leagueRules: Rules | undefined;
  let standingRows: { franchise: string; division: string }[] = [];
  const result: NativeHQ = { revision, snapshot, settings: { positions: [] }, errors: [] };
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const apply = async (read: () => Promise<void>) => { try { await read(); } catch (e) {
    result.errors.push(String(e)); console.warn('Franchise HQ read failed:', e); } };
  await Promise.all([
    apply(async () => { const r = await port({ command: 'commish.rules.read', mode: 'read', arguments: {
      league } });
      sameRevision(r.data);
      const rules = parseRules(r.data); if (rules.league !== league) throw new
        Error('Rules belong to another league');
      leagueRules = rules; result.settings = nativeHQSettings(rules); result.leagueName = rules.name; }),
    apply(async () => { const r = await port({ command: 'native.cap.check', mode: 'read', arguments: {
      league, viewerZone: zone } }); sameRevision(r.data); result.runway = nativeHQCap(r.data, league,
        mine);
      const caps = (r.data as { franchises: unknown[] }).franchises;
      const cap = caps.find(v => (v as { franchise: string }).franchise === mine) as {
        contracts?: unknown } | undefined;
      result.contracts = new Map(array(cap?.contracts ?? [], 'Contract schedules', (v, p) => {
        const row = object(v, p, ['player', 'years']);
        return [text(row.player, p), array(row.years, 'Annual salaries', (v, p) => {
          const y = object(v, p, ['season', 'cents']);
          return { season: integer(y.season, p), cents: integer(y.cents, p) };
        })] as const;
      })); }),
    apply(async () => { const r = await port({ command: 'native.reserve.check', mode: 'read', arguments: {
      league } });
      sameRevision(r.data);
      if ((r.data as { league: unknown }).league !== league) throw new Error('Reserve scope changed');
      const reserves = parseHostedReserves(r.data).find(r => r.franchise === mine);
      result.reserveNotes = new Map(reserves?.players.map(p => [p.player, p.words]));
    }),
    apply(async () => { const r = await port({ command: 'moves.read', mode: 'read', arguments: { league,
      franchise: mine } }); const moves = nativeHQMoves(r.data, league, mine); result.moves = moves.moves;
      result.moveReason = moves.reason; }),
    apply(async () => { const r = await port({ command: 'standings.read', mode: 'read', arguments: {
      league, season } });
      const raw = object(r.data, 'Standings', ['league', 'name', 'season', 'revision', 'status', 'words',
        'through', 'decided', 'pending', 'criteria', 'noData', 'computed', 'override', 'changes']);
      sameRevision(r.data);
      if (raw.league !== league || raw.season !== season)
        throw new Error('Standings belong to another season or league');
      const override = raw.override == null ? undefined : object(raw.override, 'Standing override',
        ['words', 'rows', 'order', 'records']);
      const rows = array(override?.rows ?? raw.computed, 'Standings rows', (v, p) => { const row = v as
        Record<string, unknown>;
        return { franchise: text(row.franchise, p), rank: integer(row.rank, p), record: text(row.record,
          p), division: row.division === undefined ? '' : text(row.division, p) }; });
      standingRows = rows;
      result.record = rows.find(r => r.franchise === mine)?.record; }),
    apply(async () => { const r = await port({ command: 'week.read', mode: 'read', arguments: { league,
      franchise: mine, season, localZone: zone, at: new Date().toISOString() } });
      const raw = object(r.data, 'Native week', ['schedule', 'current', 'nextWeek', 'message']);
      sameRevision(raw.schedule);
      const schedule = object(raw.schedule, 'Franchise schedule', ['league', 'franchise', 'season',
        'revision', 'message', 'games']);
      if (schedule.league !== league || schedule.franchise !== mine || schedule.season !== season) throw new
        Error('Week belongs to another franchise');
      const current = raw.current == null ? undefined : object(raw.current, 'Current week', ['week',
        'game', 'opponent', 'venue', 'kickoff']);
      result.week = current ? integer(current.week, 'Current week', 1) : raw.nextWeek === undefined ?
        undefined : integer(raw.nextWeek, 'Next week', 1);
      if (result.week === undefined) {
        const clock = await selectProvider().clock();
        if (clock.provenance.freshness.state !== 'fail' && clock.value.season === season)
          result.week = clock.value.week;
        const message = text(raw.message ?? schedule.message ??
          'No current scheduled week.', 'Week message');
        result.weekNote = result.week ? `Viewing week ${result.week} · ${message}` :
          `Current NFL week is unknown · ${message}`;
      }
    }),
  ]);
  result.standing = nativeHQStanding(leagueRules, standingRows, mine);
  if (result.week) await apply(async () => {
    const r = await port({ command: 'lineup.read', mode: 'read', arguments: { league, franchise: mine,
      week: result.week!, at: '' } });
    sameRevision(r.data);
    const held = parseHostedLineup(r.data);
    if (held.league !== league || held.franchise !== mine || held.week !== result.week || held.season !==
      season) throw new
      Error('Lineup belongs to another scope');
    const raw = r.data as { starters: { player: string; team?: string; locksAt?: string }[]; bench: {
      player: string; team?: string; locksAt?: string }[] };
    const players = [...held.starters, ...held.bench];
    for (const p of snapshot.players.value) { const identity = players.find(i => i.player === p.id);
      if (identity) { p.name = identity.name;
        const position = identity.position === 'PK' ? 'K' : identity.position;
        p.position = POSITIONS.includes(position as Position) ? position as Position : undefined;
        p.team = [...raw.starters, ...raw.bench].find(i => i.player === p.id)?.team; } }
    const locks = [...raw.starters, ...raw.bench].flatMap(p => p.locksAt ? [timestamp(p.locksAt,
      'Kickoff')] : []).filter(at => Date.parse(at) > Date.now()).sort();
    result.lock = locks[0]; result.check = held.verdict;
    result.lineup = { franchise: mine, week: held.week, starterCount: result.settings.starters ??
      held.starters.length,
      starters: held.starters.map(p => ({ id: p.player, position: p.position === 'PK' ? 'K' : p.position
        })), bench: held.bench.map(p => ({ id: p.player, position: p.position === 'PK' ? 'K' : p.position })),
      check: { legal: false, full: false, problems: [] }, provenance: snapshot.rosters.provenance,
        rulesSource: snapshot.league.provenance };
  });
  if (result.week) await apply(async () => {
    const players = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.map(p => p.id) ?? [];
    const values = new Map<string, number>();
    const seen = new Set<string>();
    for (let offset = 0; offset < players.length; offset += 120) {
      const ids = players.slice(offset, offset + 120);
      const r = await port({ command: 'projections.read', mode: 'read', arguments: { league,
        players: ids.join(','), season, week: result.week! } });
      const projections = parseWarRoomProjections(r.data, league, season, result.week!);
      for (const p of projections) {
        if (!ids.includes(p.player) || seen.has(p.player)) throw new Error('Projection player scope changed');
        seen.add(p.player);
        if (p.available) values.set(p.player, p.median);
      }
    }
    result.projections = values;
  });
  return result;
}

export type NativeCut = { player: string; revision: number; season: number; deadNow?: number;
  deadNext?: number; roomAfter?: number; before?: number; after?: number; reason: string };
export function parseNativeCut(value: unknown, league: string, player: string): NativeCut {
  const r = object(value, 'Cut quote', ['league', 'revision', 'season', 'player', 'name', 'franchise',
    'action', 'actionName', 'allowed', 'reason', 'needs', 'years', 'salaryCents', 'salaries',
    'deadMoney', 'reliefCents', 'cap', 'basis', 'notes', 'source']);
  if (r.league !== league || r.player !== player || r.action !== 'cut')
    throw new Error('Cut quote belongs to another player or action');
  if (typeof r.allowed !== 'boolean') throw new Error('Unreadable cut verdict');
  const season = integer(r.season, 'Quote season');
  const dead = array(r.deadMoney, 'Dead money', (v, p) => {
    const row = object(v, p, ['season', 'cents']);
    return { season: integer(row.season, p), cents: integer(row.cents, p) };
  });
  const priced = r.allowed || dead.length > 0;
  const cap = r.cap == null ? undefined : object(r.cap, 'Cut cap', ['beforeCents', 'afterCents',
    'changeCents', 'roomAfterCents', 'words']);
  const amount = (key: string) => cap?.[key] == null ? undefined
    : integer(cap[key], key, Number.MIN_SAFE_INTEGER);
  const sum = (year: number) => priced ? dead.filter(d => d.season === year)
    .reduce((n, d) => n + d.cents, 0) : undefined;
  return { player, revision: integer(r.revision, 'Quote revision'), season,
    deadNow: sum(season), deadNext: sum(season + 1), roomAfter: amount('roomAfterCents'),
    before: amount('beforeCents'), after: amount('afterCents'), reason: text(r.reason, 'Cut reason') };
}
export async function readNativeCut(league: string, mine: string, player: string,
  port: HostedPort = hostedWorkflow): Promise<NativeCut> {
  const response = await port({ command: 'contract.quote', mode: 'read', arguments: { league,
    franchise: mine, player, action: 'cut', viewerZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    years: 0, level: 0, moveCents: 0, amountCents: 0, reason: '' } });
  return parseNativeCut(response.data, league, player);
}
