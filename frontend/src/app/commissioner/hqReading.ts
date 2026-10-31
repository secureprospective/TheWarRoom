import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import { CONTRACT_STATUSES, POSITIONS, ROSTER_STATUSES, type Position, type Snapshot, type Provenance } from '../data/contract';
import { parseRules, type Rules } from './rules';
import { parseHostedLineup, parseHostedRoster } from './hostedReadings';
import { parseRailReceipt } from '../shell/railReceipt';
import type { HQMove } from '../shell/HQMoves';
import type { HQCap, HQSettings } from '../shell/hqModel';
import type { HQLineup } from '../shell/HQBoard';
import type { HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';

export function nativeHQSettings(rules: Rules): HQSettings {
  const rows = rules.sections.flatMap(s => s.rows);
  const current = (field: string) => rows.find(r => r.field === field && r.state === 'Set')?.current;
  const count = (field: string) => current(field) == null ? undefined : integer(current(field), field);
  const bounds = (value: unknown) => {
    if (value == null) return {};
    const r = object(value, 'Slot bounds', ['Min', 'Max']);
    return { min: r.Min == null ? undefined : integer(r.Min, 'Minimum'), max: r.Max == null ? undefined : integer(r.Max, 'Maximum') };
  };
  const positions = current('lineup.positions');
  const groups = current('lineup.groups');
  const group = groups == null ? {} : object(groups, 'Lineup groups', ['offense', 'defense', 'total']);
  const positionMap = positions == null ? {} : object(positions, 'Lineup positions', ['QB', 'RB', 'WR', 'TE', 'PK', 'K', 'DT', 'DE', 'LB', 'CB', 'S', 'OL']);
  return { active: bounds(current('roster.limits')).max, ir: count('ir.slots'), taxi: count('taxi.slots'),
    starters: bounds(group.total).max, offence: bounds(group.offense).min, defence: bounds(group.defense).min,
    positions: Object.entries(positionMap).map(([code, value]) => { const b = bounds(value);
      return { code: code === 'PK' ? 'K' : code, min: b.min ?? 0, max: b.max ?? Number.MAX_SAFE_INTEGER }; }) };
}
export function nativeHQAssets(value: unknown, league: string): Snapshot {
  const held = parseHostedRoster(value);
  if (held.league !== league) throw new Error('Roster belongs to another league');
  const raw = value as { snapshot: { players: unknown[] } };
  const players = array(raw.snapshot.players, 'Roster contracts', (v, p) => {
    const r = v as Record<string, unknown>;
    return { id: text(r.id, p), franchise: text(r.franchise, p), salary: integer(r.salaryCents, p),
      expiration: integer(r.expirationYear, p), contractStatus: choice(r.contractStatus, p, CONTRACT_STATUSES),
      rosterStatus: choice(r.status, p, ROSTER_STATUSES) };
  });
  const provenance: Provenance = { source: 'League ledger', kind: 'live', freshness: { state: 'live', fetchedAt: '', note: 'Recorded league assets' } };
  return { league: { value: { season: held.season, franchiseCount: held.franchises.length }, provenance },
    franchises: { value: held.franchises.map(f => ({ id: f.value, name: f.label })), provenance },
    rosters: { value: held.franchises.map(f => ({ franchiseId: f.value, players: players.filter(p => p.franchise === f.value).map(p => ({ id: p.id, salary: p.salary, contractStatus: p.contractStatus, rosterStatus: p.rosterStatus,
      ...(p.expiration ? { yearsRemaining: Math.max(0, p.expiration - held.season + 1) } : {}) })) })), provenance },
    players: { value: held.players.map(p => ({ id: p.id, name: p.name })), provenance },
    eligibility: { players: {}, feeds: { injuries: provenance, players: provenance, rosters: provenance } } };
}
export function nativeHQCap(value: unknown, league: string, mine: string): HQCap[] {
  const r = object(value, 'Cap check', ['corrections', 'league', 'name', 'revision', 'rosterSeason', 'rulesSeason', 'seeded', 'source', 'counting', 'rules', 'franchises']);
  if (r.league !== league) throw new Error('Cap belongs to another league');
  const franchises = array(r.franchises, 'Franchise caps', (v, p) => {
    const f = object(v, p, ['exception', 'capCents', 'salaryCents', 'activeCents', 'irCents', 'taxiCents', 'adjustmentCents', 'committedCents', 'roomCents', 'quantumCents', 'mode', 'deadline', 'words', 'enforcement', 'franchise', 'name', 'abbreviation', 'charges', 'runway']);
    return { franchise: text(f.franchise, p), runway: array(f.runway, 'Runway', (v, p) => {
      const y = object(v, p, ['season', 'committedCents', 'deadCents', 'capCents', 'roomCents']);
      const amount = (key: string) => y[key] === null ? undefined : integer(y[key], key, Number.MIN_SAFE_INTEGER);
      return { season: integer(y.season, p), committed: amount('committedCents'), dead: amount('deadCents'), max: amount('capCents'), room: amount('roomCents') };
    }) };
  });
  return franchises.find(f => f.franchise === mine)?.runway ?? [];
}
export function nativeHQMoves(value: unknown, league: string, mine: string): { moves: HQMove[]; reason: string } {
  const r = object(value, 'Native moves', ['league', 'franchise', 'planned', 'reason', 'moves']);
  if (r.league !== league || r.franchise !== mine) throw new Error('Moves belong to another franchise');
  const move = (v: unknown, p: string): HQMove => {
    const receipt = parseRailReceipt(v, p);
    const raw = v as { spec: { leagueId: unknown; subject: unknown } };
    if (receipt.franchiseId !== mine || raw.spec.leagueId !== league) throw new Error('Move scope changed');
    const subject = object(raw.spec.subject, 'Move subject', ['players', 'picks']);
    return { id: receipt.id, state: receipt.state, intent: receipt.rail.spec.intent,
      at: receipt.history.at(-1)?.at ?? '', note: receipt.history.at(-1)?.note ?? '',
      players: array(subject.players, p, text), native: true };
  };
  return { moves: [...array(r.planned, 'Planned moves', move), ...array(r.moves, 'Recorded moves', move)], reason: text(r.reason, 'Native plans') };
}
export type NativeHQ = { snapshot: Snapshot; settings: HQSettings; leagueName?: string; lineup?: HQLineup;
  check?: string; lock?: string; runway?: HQCap[]; moves?: HQMove[]; moveReason?: string;
  record?: string; standing?: string; week?: number; errors: string[] };
export async function readNativeHQ(league: string, mine: string, port: HostedPort = hostedWorkflow): Promise<NativeHQ> {
  const assets = await port({ command: 'native.assets.read', mode: 'read', arguments: { league } });
  const snapshot = nativeHQAssets(assets.data, league);
  const season = snapshot.league.value.season;
  const result: NativeHQ = { snapshot, settings: { positions: [] }, errors: [] };
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const apply = async (read: () => Promise<void>) => { try { await read(); } catch (e) { result.errors.push(String(e)); } };
  await Promise.all([
    apply(async () => { const r = await port({ command: 'commish.rules.read', mode: 'read', arguments: { league } });
      const rules = parseRules(r.data); if (rules.league !== league) throw new Error('Rules belong to another league');
      result.settings = nativeHQSettings(rules); result.leagueName = rules.name; }),
    apply(async () => { const r = await port({ command: 'native.cap.check', mode: 'read', arguments: { league, viewerZone: zone } }); result.runway = nativeHQCap(r.data, league, mine); }),
    apply(async () => { const r = await port({ command: 'moves.read', mode: 'read', arguments: { league, franchise: mine } }); const moves = nativeHQMoves(r.data, league, mine); result.moves = moves.moves; result.moveReason = moves.reason; }),
    apply(async () => { const r = await port({ command: 'standings.read', mode: 'read', arguments: { league, season } });
      const raw = object(r.data, 'Standings', ['league', 'name', 'season', 'revision', 'status', 'words', 'through', 'decided', 'pending', 'criteria', 'noData', 'computed', 'override', 'changes']);
      if (raw.league !== league) throw new Error('Standings belong to another league');
      const override = raw.override == null ? undefined : object(raw.override, 'Standing override', ['words', 'rows', 'order', 'records']);
      const rows = array(override?.rows ?? raw.computed, 'Standings rows', (v, p) => { const row = v as Record<string, unknown>;
        return { franchise: text(row.franchise, p), rank: integer(row.rank, p), record: text(row.record, p), division: row.division === undefined ? '' : text(row.division, p) }; });
      const row = rows.find(r => r.franchise === mine); if (row) { result.record = row.record; result.standing = `${row.rank} · ${row.division}`; } }),
    apply(async () => { const r = await port({ command: 'week.read', mode: 'read', arguments: { league, franchise: mine, season, localZone: zone, at: new Date().toISOString() } });
      const raw = object(r.data, 'Native week', ['schedule', 'current', 'nextWeek', 'message']);
      const schedule = raw.schedule as { league?: unknown; franchise?: unknown };
      if (schedule.league !== league || schedule.franchise !== mine) throw new Error('Week belongs to another franchise');
      const current = raw.current == null ? undefined : object(raw.current, 'Current week', ['week', 'game', 'opponent', 'venue', 'kickoff']);
      result.week = current ? integer(current.week, 'Current week', 1) : raw.nextWeek === undefined ? undefined : integer(raw.nextWeek, 'Next week', 1);
    }),
  ]);
  if (result.week) await apply(async () => {
    const r = await port({ command: 'lineup.read', mode: 'read', arguments: { league, franchise: mine, week: result.week!, at: '' } });
    const held = parseHostedLineup(r.data);
    if (held.league !== league || held.franchise !== mine || held.week !== result.week) throw new Error('Lineup belongs to another scope');
    const raw = r.data as { starters: { player: string; team?: string; locksAt?: string }[]; bench: { player: string; team?: string; locksAt?: string }[] };
    const players = [...held.starters, ...held.bench];
    for (const p of snapshot.players.value) { const identity = players.find(i => i.player === p.id);
      if (identity) { p.name = identity.name; p.position = POSITIONS.includes(identity.position as Position) ? identity.position as Position : undefined;
        p.team = [...raw.starters, ...raw.bench].find(i => i.player === p.id)?.team; } }
    const locks = [...raw.starters, ...raw.bench].flatMap(p => p.locksAt ? [timestamp(p.locksAt, 'Kickoff')] : []).filter(at => Date.parse(at) > Date.now()).sort();
    result.lock = locks[0]; result.check = held.verdict;
    result.lineup = { franchise: mine, week: held.week, starterCount: result.settings.starters ?? held.starters.length,
      starters: held.starters.map(p => ({ id: p.player, position: p.position === 'PK' ? 'K' : p.position })), bench: held.bench.map(p => ({ id: p.player, position: p.position === 'PK' ? 'K' : p.position })),
      check: { legal: false, full: false, problems: [] }, provenance: snapshot.rosters.provenance, rulesSource: snapshot.league.provenance };
  });
  return result;
}
