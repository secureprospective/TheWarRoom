import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import {
  CONTRACT_STATUSES,
  POSITIONS,
  ROSTER_STATUSES,
  type Position,
  type Snapshot,
  type Provenance,
} from '../data/contract';
import { parseRules, type Rules } from './rules';
import { parseHostedLineup, parseHostedRoster, parseHostedReserves } from './hostedReadings';
import { parsePulseResults, recentPlayerForm, type PulsePlayerFormPoint, type PulseResults } from
  '../data/nativePulse';
import { parseNativeMoves } from '../data/nativeMoves';
import type { HQMove } from '../shell/HQMoves';
import type { HQCap, HQSettings } from '../shell/hqModel';
import type { HQLineup } from '../shell/HQBoard';
import type { HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';
import { selectProvider } from '../data/provider';
import { ordinal } from '../shell/hqPlayerData';
import { parseNativeValues, type LeagueValues } from '../data/nativeValues';
import { parseWarRoomProjections } from '../data/nativeProjections';
import { deadlineLabel } from '../clock/labels';
import { readRecentResults } from '../shell/pulseReads';
import { parseContractDecisions, type ContractDecisions } from '../shell/hqContractDecisions';

export function nativeHQStanding(
  rules: Rules | undefined,
  rows: { franchise: string; division: string }[],
  mine: string,
): string | undefined {
  const raw = rules?.sections
    .flatMap((s) => s.rows)
    .find((r) => r.field === 'league.alignment' && r.state === 'Set')?.current;
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return undefined;
  const alignment = new Map(Object.entries(raw).map(([id, value]) => [id, text(value, 'Division')]));
  const name = alignment.get(mine);
  // Older seeds contain only ledger identifiers, not display names.
  if (!name || /^division[._]\d+$/i.test(name)) return undefined;
  const group = rows.filter((r) => alignment.get(r.franchise) === name);
  const index = group.findIndex((r) => r.franchise === mine);
  return index < 0 ? undefined : `${ordinal(index + 1)} ${name}`;
}
export function nativeHQSettings(rules: Rules): HQSettings {
  const rows = rules.sections.flatMap((s) => s.rows);
  const current = (field: string) => rows.find((r) => r.field === field && r.state === 'Set')?.current;
  const count = (field: string) => (current(field) == null ? undefined : integer(current(field), field));
  const bounds = (value: unknown) => {
    if (value == null) return {};
    const r = object(value, 'Slot bounds', ['Min', 'Max']);
    return {
      min: r.Min == null ? undefined : integer(r.Min, 'Minimum'),
      max: r.Max == null ? undefined : integer(r.Max, 'Maximum'),
    };
  };
  const reserveExcluded = current('roster.reserveExcluded');
  if (reserveExcluded != null && typeof reserveExcluded !== 'boolean')
    throw new Error('Unreadable reserve roster rule');
  const positions = current('lineup.positions');
  const groups = current('lineup.groups');
  const group = groups == null ? {} : object(groups, 'Lineup groups', ['offense', 'defense', 'total']);
  const positionMap =
    positions == null
      ? {}
      : object(positions, 'Lineup positions', [
          'QB',
          'RB',
          'WR',
          'TE',
          'PK',
          'K',
          'DT',
          'DE',
          'LB',
          'CB',
          'S',
          'OL',
        ]);
  return {
    reserveExcluded: reserveExcluded == null ? undefined : (reserveExcluded as boolean),
    maxYears: count('contracts.maximumYears'),
    active: bounds(current('roster.limits')).max,
    ir: count('ir.slots'),
    taxi: count('taxi.slots'),
    starters: bounds(group.total).max,
    offence: bounds(group.offense).min,
    defence: bounds(group.defense).min,
    positions: Object.entries(positionMap).map(([code, value]) => {
      const b = bounds(value);
      return { code: code === 'PK' ? 'K' : code, min: b.min ?? 0, max: b.max };
    }),
  };
}
export function nativeHQAssets(value: unknown, league: string, held = parseHostedRoster(value)): Snapshot {
  if (held.league !== league) throw new Error('Roster belongs to another league');
  const raw = value as { snapshot: { players: unknown[] } };
  const players = array(raw.snapshot.players, 'Roster contracts', (v, p) => {
    const r = v as Record<string, unknown>;
    return {
      id: text(r.id, p),
      franchise: text(r.franchise, p),
      salary: r.salaryCents === null ? null : integer(r.salaryCents, p),
      // An imported league may hold no contract status (MFL left it blank);
      // the MFL mirror reads blank as FLAG.
      expiration: r.expirationYear === null ? null : integer(r.expirationYear, p),
      contractStatus: choice(r.contractStatus === '' ? 'FLAG' : r.contractStatus, p, CONTRACT_STATUSES),
      rosterStatus: choice(r.status, p, ROSTER_STATUSES),
    };
  });
  const provenance: Provenance = {
    source: 'League ledger',
    kind: 'live',
    freshness: { state: 'live', fetchedAt: '', note: 'Recorded league assets' },
  };
  return {
    league: { value: { season: held.season, franchiseCount: held.franchises.length }, provenance },
    franchises: { value: held.franchises.map((f) => ({ id: f.value, name: f.label })), provenance },
    rosters: {
      value: held.franchises.map((f) => ({
        franchiseId: f.value,
        players: players
          .filter((p) => p.franchise === f.value)
          .map((p) => ({
            id: p.id,
            salary: p.salary,
            contractStatus: p.contractStatus,
            rosterStatus: p.rosterStatus,
            ...(p.expiration ? { yearsRemaining: Math.max(0, p.expiration - held.season + 1) } : {}),
          })),
      })),
      provenance,
    },
    // Names and positions are football facts: the identity block carries them, and the recorded
    // roster name stands in when the app holds no identity for a player.
    players: {
      value: held.players.map((p) => {
        const identity = held.identities.find((i) => i.player === p.id && i.franchise === p.franchise);
        return {
          id: p.id,
          name: identity?.name ?? p.name,
          ...(identity?.position ? { position: identity.position } : {}),
        };
      }),
      provenance,
    },
    eligibility: { players: {}, feeds: { injuries: provenance, players: provenance, rosters: provenance } },
  };
}
export function nativeHQCap(value: unknown, league: string, mine: string): HQCap[] {
  const r = object(value, 'Cap check', [
    'corrections',
    'league',
    'name',
    'revision',
    'rosterSeason',
    'rulesSeason',
    'seeded',
    'source',
    'counting',
    'rules',
    'franchises',
  ]);
  if (r.league !== league) throw new Error('Cap belongs to another league');
  const franchises = array(r.franchises, 'Franchise caps', (v, p) => {
    const f = object(v, p, [
      'exception',
      'capCents',
      'salaryCents',
      'activeCents',
      'irCents',
      'taxiCents',
      'adjustmentCents',
      'committedCents',
      'roomCents',
      'quantumCents',
      'mode',
      'deadline',
      'words',
      'enforcement',
      'franchise',
      'name',
      'abbreviation',
      'charges',
      'runway',
      'contracts',
    ]);
    return {
      franchise: text(f.franchise, p),
      runway: array(f.runway, 'Runway', (v, p) => {
        const y = object(v, p, ['season', 'committedCents', 'deadCents', 'capCents', 'roomCents']);
        const amount = (key: string) =>
          y[key] === null ? undefined : integer(y[key], key, Number.MIN_SAFE_INTEGER);
        return {
          season: integer(y.season, p),
          ...(y.season === r.rulesSeason
            ? { ir: f.irCents === null ? undefined : integer(f.irCents, 'IR pay') }
            : {}),
          committed: amount('committedCents'),
          dead: amount('deadCents'),
          max: amount('capCents'),
          room: amount('roomCents'),
        };
      }),
    };
  });
  return franchises.find((f) => f.franchise === mine)?.runway ?? [];
}
export function nativeHQMoves(
  value: unknown,
  league: string,
  mine: string,
): { moves: HQMove[]; reason: string } {
  const reading = parseNativeMoves(value);
  if (reading.league !== league || reading.franchise !== mine)
    throw new Error('Moves belong to another franchise');
  return {
    reason: reading.reason,
    moves: reading.moves.map((m) => ({
      id: m.correlationId,
      state: m.state,
      intent: m.spec.intent,
      at: m.audit.at(-1)?.at ?? '',
      note: m.audit.at(-1)?.note ?? '',
      players: m.spec.subject.players,
      native: true,
    })),
  };
}
export type NativeHQ = {
  revision: number;
  snapshot: Snapshot;
  searchPlayers: { id: string; name: string; position?: Position }[];
  settings: HQSettings;
  leagueName?: string;
  lineup?: HQLineup;
  contracts?: Map<string, { season: number; cents: number }[]>;
  check?: string;
  lock?: string;
  runway?: HQCap[];
  moves?: HQMove[];
  moveReason?: string;
  reserveNotes?: Map<string, string>;
  reserveExperience?: Map<string, { experienceYears?: number; taxiYearsLeft?: number }>;
  record?: string;
  standing?: string;
  standingsNote?: string;
  week?: number;
  weekNote?: string;
  values?: LeagueValues;
  decisions?: { label: string; at?: string; live: boolean }[];
  contractDecisions?: ContractDecisions;
  projections?: Map<string, number>;
  thisWeek?: PulseResults;
  playerForm?: Record<string, PulsePlayerFormPoint[]>;
  /** The last six started weeks per player, for the phone's low–high range. */
  playerHistory?: Record<string, PulsePlayerFormPoint[]>;
  /** Each lineup player's kickoff lock time from lineup.read, past or future. */
  kickoffs?: Map<string, string>;
  formPending?: boolean;
  formError?: string;
  formNote?: string;
  errors: string[];
};
export const OWNER_ONLY_MOVES = 'Pending moves are visible only to this franchise\'s owner';
export async function readNativeHQ(
  league: string,
  mine: string,
  port: HostedPort = hostedWorkflow,
  publish?: (reading: NativeHQ) => void,
  owner = true,
  includeContractDecisions = false,
): Promise<NativeHQ> {
  const assets = await port({ command: 'native.assets.read', mode: 'read', arguments: { league } });
  const held = parseHostedRoster(assets.data);
  const snapshot = nativeHQAssets(assets.data, league, held);
  const season = snapshot.league.value.season;
  const revision = integer((assets.data as { revision: unknown }).revision, 'Roster revision');
  const sameRevision = (value: unknown) => {
    if (integer((value as { revision: unknown }).revision, 'Reading revision') !== revision)
      throw new Error('League changed during the read; refresh before using this region');
  };
  let leagueRules: Rules | undefined;
  let standingRows: { franchise: string; division: string }[] = [];
  let standingsThrough: number | undefined;
  let scheduleWeeks: number[] = [];
  const result: NativeHQ = { revision, snapshot, searchPlayers: [
    ...held.players.map(player => ({ id: player.id, name: player.name })),
    ...held.market.map(player => ({ id: player.id, name: player.name, position: player.position })),
  ], settings: { positions: [] }, formPending: Boolean(mine), errors: [] };
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const apply = async (read: () => Promise<void>, failed?: (words: string) => void) => {
    try {
      await read();
    } catch (e) {
      const words = e instanceof Error ? e.message : String(e);
      result.errors.push(words);
      failed?.(words);
      console.warn('Franchise HQ read failed:', e);
    } finally {
      publish?.({ ...result, errors: [...result.errors] });
    }
  };
  publish?.({ ...result });
  const values = apply(async () => {
    const r = await port({
      command: 'value.read',
      mode: 'read',
      arguments: { league, franchise: mine, season },
    });
    result.values = parseNativeValues(r.data, league, mine, season);
  });
  const regions = Promise.all([
    apply(async () => {
      const r = await port({
        command: 'commish.rules.read',
        mode: 'read',
        arguments: {
          league,
        },
      });
      sameRevision(r.data);
      const rules = parseRules(r.data);
      if (rules.league !== league) throw new Error('Rules belong to another league');
      leagueRules = rules;
      result.settings = nativeHQSettings(rules);
      result.leagueName = rules.name;
    }),
    apply(async () => {
      const r = await port({
        command: 'native.cap.check',
        mode: 'read',
        arguments: {
          league,
          viewerZone: zone,
        },
      });
      sameRevision(r.data);
      result.runway = nativeHQCap(r.data, league, mine);
      const caps = (r.data as { franchises: unknown[] }).franchises;
      const cap = caps.find((v) => (v as { franchise: string }).franchise === mine) as
        | {
            contracts?: unknown;
          }
        | undefined;
      result.contracts = new Map(
        array(cap?.contracts ?? [], 'Contract schedules', (v, p) => {
          const row = object(v, p, ['player', 'years']);
          return [
            text(row.player, p),
            array(row.years, 'Annual salaries', (v, p) => {
              const y = object(v, p, ['season', 'cents']);
              return { season: integer(y.season, p), cents: integer(y.cents, p) };
            }),
          ] as const;
        }),
      );
    }),
    apply(async () => {
      if (!owner || !includeContractDecisions || !mine) return;
      const r = await port({ command: 'contract.decisions.read', mode: 'read',
        arguments: { league, viewerZone: zone } });
      sameRevision(r.data);
      result.contractDecisions = parseContractDecisions(r.data, league, mine, revision);
    }),
    apply(async () => {
      const r = await port({
        command: 'native.reserve.check',
        mode: 'read',
        arguments: {
          league,
        },
      });
      sameRevision(r.data);
      if ((r.data as { league: unknown }).league !== league) throw new Error('Reserve scope changed');
      const reserves = parseHostedReserves(r.data).find((r) => r.franchise === mine);
      result.reserveNotes = new Map(reserves?.players.map((p) => [p.player, p.words]));
      result.reserveExperience = new Map(reserves?.players.map((p) => [p.player, {
        experienceYears: p.experienceYears, taxiYearsLeft: p.taxiYearsLeft,
      }]));
    }),
    apply(async () => {
      // Moves are owner-only on the server; viewing another franchise must not ask for them.
      if (!owner) { result.moves = []; result.moveReason = OWNER_ONLY_MOVES; return; }
      const r = await port({ command: 'moves.read', mode: 'read', arguments: { league, franchise: mine } });
      const moves = nativeHQMoves(r.data, league, mine);
      result.moves = moves.moves;
      result.moveReason = moves.reason;
    }),
    apply(async () => {
      const r = await port({
        command: 'standings.read',
        mode: 'read',
        arguments: {
          league,
          season,
        },
      });
      const raw = object(r.data, 'Standings', [
        'league',
        'name',
        'season',
        'revision',
        'status',
        'words',
        'through',
        'decided',
        'pending',
        'criteria',
        'noData',
        'computed',
        'override',
        'changes',
        'evidence',
      ]);
      sameRevision(r.data);
      if (raw.league !== league || raw.season !== season)
        throw new Error('Standings belong to another season or league');
      const override =
        raw.override == null
          ? undefined
          : object(raw.override, 'Standing override', ['words', 'rows', 'order', 'records']);
      const rows = array(override?.rows ?? raw.computed, 'Standings rows', (v, p) => {
        const row = v as Record<string, unknown>;
        return {
          franchise: text(row.franchise, p),
          rank: integer(row.rank, p),
          record: text(row.record, p),
          division: row.division === undefined ? '' : text(row.division, p),
        };
      });
      standingRows = rows;
      standingsThrough = integer(raw.through, 'Standings through');
      result.record = rows.find((r) => r.franchise === mine)?.record;
      result.standingsNote = raw.words === undefined ? undefined : text(raw.words, 'Standings words');
    }),
  ]);
  await apply(async () => {
      const r = await port({
        command: 'week.read',
        mode: 'read',
        arguments: { league, franchise: mine, season, localZone: zone, at: new Date().toISOString() },
      });
      const raw = object(r.data, 'Native week', ['schedule', 'current', 'nextWeek', 'message']);
      sameRevision(raw.schedule);
      const schedule = object(raw.schedule, 'Franchise schedule', [
        'league',
        'franchise',
        'season',
        'revision',
        'message',
        'games',
      ]);
      if (schedule.league !== league || schedule.franchise !== mine || schedule.season !== season)
        throw new Error('Week belongs to another franchise');
      scheduleWeeks = array(schedule.games, 'Schedule games', (v, p) => {
        const game = object(v, p, ['week', 'game', 'opponent', 'venue', 'kickoff']);
        return integer(game.week, p, 1);
      });
      const current =
        raw.current == null
          ? undefined
          : object(raw.current, 'Current week', ['week', 'game', 'opponent', 'venue', 'kickoff']);
      result.week = current
        ? integer(current.week, 'Current week', 1)
        : raw.nextWeek === undefined
          ? undefined
          : integer(raw.nextWeek, 'Next week', 1);
      if (result.week === undefined) {
        const clock = await selectProvider().clock();
        if (clock.provenance.freshness.state !== 'fail' && clock.value.season === season)
          result.week = clock.value.week;
        const message = text(raw.message ?? schedule.message ?? 'No current scheduled week.', 'Week message');
        result.weekNote = result.week
          ? `Viewing week ${result.week} · ${message}`
          : `Current NFL week is unknown · ${message}`;
      }
    });
  if (result.week)
    await apply(async () => {
      const r = await port({
        command: 'lineup.read',
        mode: 'read',
        arguments: { league, franchise: mine, week: result.week!, at: '' },
      });
      sameRevision(r.data);
      const held = parseHostedLineup(r.data);
      if (
        held.league !== league ||
        held.franchise !== mine ||
        held.week !== result.week ||
        held.season !== season
      )
        throw new Error('Lineup belongs to another scope');
      const raw = r.data as {
        starters: { player: string; team?: string; locksAt?: string }[];
        bench: {
          player: string;
          team?: string;
          locksAt?: string;
        }[];
      };
      const players = [...held.starters, ...held.bench];
      // The lineup read fills gaps only: a name or position already held for a player is the
      // app's own fact, and must not be replaced by the lineup's stand-in label.
      for (const p of snapshot.players.value) {
        const identity = players.find((i) => i.player === p.id);
        if (identity) {
          if (!p.name) p.name = identity.name;
          const position = identity.position === 'PK' ? 'K' : identity.position;
          if (!p.position && POSITIONS.includes(position as Position)) p.position = position as Position;
          p.team = [...raw.starters, ...raw.bench].find((i) => i.player === p.id)?.team;
        }
      }
      const locks = [...raw.starters, ...raw.bench]
        .flatMap((p) => (p.locksAt ? [timestamp(p.locksAt, 'Kickoff')] : []))
        .filter((at) => Date.parse(at) > Date.now())
        .sort();
      result.lock = locks[0];
      result.kickoffs = new Map([...raw.starters, ...raw.bench].flatMap((p) =>
        p.locksAt ? [[p.player, timestamp(p.locksAt, 'Kickoff')] as const] : []));
      result.check = held.verdict;
      result.lineup = {
        franchise: mine,
        week: held.week,
        starterCount: result.settings.starters ?? held.starters.length,
        starters: held.starters.map((p) => ({
          id: p.player,
          position: p.position === 'PK' ? 'K' : p.position,
        })),
        bench: held.bench.map((p) => ({ id: p.player, position: p.position === 'PK' ? 'K' : p.position })),
        check: { legal: false, full: false, problems: [] },
        provenance: snapshot.rosters.provenance,
        rulesSource: snapshot.league.provenance,
      };
    });
  const thisWeek = result.week ? apply(async () => {
    const r = await port({ command: 'results.read', mode: 'read',
      arguments: { league, season, week: result.week! } });
    sameRevision(r.data);
    const games = parsePulseResults(r.data, league, result.week!);
    if (games.season !== season) throw new Error('Results belong to another season');
    result.thisWeek = games;
  }) : Promise.resolve();
  if (result.week)
    await apply(async () => {
      const players =
        snapshot.rosters.value.find((r) => r.franchiseId === mine)?.players.map((p) => p.id) ?? [];
      const values = new Map<string, number>();
      const seen = new Set<string>();
      for (let offset = 0; offset < players.length; offset += 120) {
        const ids = players.slice(offset, offset + 120);
        const r = await port({
          command: 'projections.read',
          mode: 'read',
          arguments: { league, players: ids.join(','), season, week: result.week! },
        });
        const projections = parseWarRoomProjections(r.data, league, season, result.week!);
        for (const p of projections) {
          if (!ids.includes(p.player) || seen.has(p.player))
            throw new Error('Projection player scope changed');
          seen.add(p.player);
          if (p.available) values.set(p.player, p.median);
        }
      }
      result.projections = values;
    });
  await Promise.all([regions, values, thisWeek]);
  if (mine && scheduleWeeks.length) await apply(async () => {
    const through = Math.max(standingsThrough ?? 0, (result.week ?? 1) - 1);
    const history = await readRecentResults(league, season, scheduleWeeks, through, port);
    if (history.some(reading => reading.revision !== revision))
      throw new Error('League changed during the read; refresh before using this region');
    const players = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.map(p => p.id) ?? [];
    result.playerHistory = Object.fromEntries(players.map(player => [player,
      recentPlayerForm(history, player, 6)]));
    result.playerForm = Object.fromEntries(players.map(player => [player,
      result.playerHistory![player].slice(-5)]));
    result.formNote = history.map(reading => reading.words).filter(Boolean).join(' · ');
  }, words => { result.formError = words; });
  result.formNote ||= result.standingsNote;
  result.formPending = false;
  await apply(async () => {
    const r = await port({ command: 'calendar.read', mode: 'read', arguments: {
      league, season, franchise: mine, action: 'lineup',
      localZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    } });
    const calendar = object(r.data, 'Calendar', ['league', 'revision', 'clock', 'decision']);
    if (calendar.league !== league) throw new Error('Calendar belongs to another league');
    const clock = object(calendar.clock, 'Action calendar', ['windows', 'exceptions', 'tradeDeadline',
      'secondsToTradeDeadline']);
    result.decisions = array(clock.windows, 'Calendar windows', (v, p) => {
      const actionWindow = object(v, p, ['id', 'opens', 'start', 'close', 'live', 'secondsToClose']);
      if (typeof actionWindow.live !== 'boolean') throw new Error('Unreadable calendar window state');
      return { label: array(actionWindow.opens, p, text).map(deadlineLabel).join(' · '),
        at: timestamp(actionWindow.start, p),
        live: actionWindow.live };
    }).sort((a, b) => (a.at ?? '').localeCompare(b.at ?? ''));
    if (clock.tradeDeadline != null) result.decisions.push({ label: 'Trade deadline',
      at: timestamp(clock.tradeDeadline, 'Trade deadline'), live: false });
    result.decisions.sort((a, b) => (a.at ?? '').localeCompare(b.at ?? ''));
  });
  result.standing = nativeHQStanding(leagueRules, standingRows, mine);
  publish?.({ ...result, errors: [...result.errors] });
  return result;
}

export type NativeCut = {
  player: string;
  revision: number;
  season: number;
  deadNow?: number;
  deadNext?: number;
  roomAfter?: number;
  before?: number;
  after?: number;
  reason: string;
};
export function parseNativeCut(value: unknown, league: string, player: string): NativeCut {
  const r = object(value, 'Cut quote', [
    'league',
    'revision',
    'season',
    'player',
    'name',
    'franchise',
    'action',
    'actionName',
    'allowed',
    'reason',
    'needs',
    'years',
    'salaryCents',
    'salaries',
    'deadMoney',
    'reliefCents',
    'cap',
    'basis',
    'notes',
    'source',
  ]);
  if (r.league !== league || r.player !== player || r.action !== 'cut')
    throw new Error('Cut quote belongs to another player or action');
  if (typeof r.allowed !== 'boolean') throw new Error('Unreadable cut verdict');
  const season = integer(r.season, 'Quote season');
  const dead = array(r.deadMoney, 'Dead money', (v, p) => {
    const row = object(v, p, ['season', 'cents']);
    return { season: integer(row.season, p), cents: integer(row.cents, p) };
  });
  const cap =
    r.cap == null
      ? undefined
      : object(r.cap, 'Cut cap', ['beforeCents', 'afterCents', 'changeCents', 'roomAfterCents', 'words']);
  const amount = (key: string) =>
    cap?.[key] == null ? undefined : integer(cap[key], key, Number.MIN_SAFE_INTEGER);
  const priced =
    r.allowed ||
    dead.length > 0 ||
    (amount('beforeCents') !== undefined && amount('afterCents') !== undefined);
  const sum = (year: number) =>
    priced ? dead.filter((d) => d.season === year).reduce((n, d) => n + d.cents, 0) : undefined;
  return {
    player,
    revision: integer(r.revision, 'Quote revision'),
    season,
    deadNow: sum(season),
    deadNext: sum(season + 1),
    roomAfter: amount('roomAfterCents'),
    before: amount('beforeCents'),
    after: amount('afterCents'),
    reason: text(r.reason, 'Cut reason'),
  };
}
export async function readNativeCut(
  league: string,
  mine: string,
  player: string,
  port: HostedPort = hostedWorkflow,
): Promise<NativeCut> {
  const response = await port({
    command: 'contract.quote',
    mode: 'read',
    arguments: {
      league,
      franchise: mine,
      player,
      action: 'cut',
      viewerZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      years: 0,
      level: 0,
      moveCents: 0,
      amountCents: 0,
      reason: '',
    },
  });
  return parseNativeCut(response.data, league, player);
}

export type NativeWhatIfMove = {
  player: string; action: string; years: number; level: number;
  moveCents: number; amountCents: number; reason: string;
};
export type NativeWhatIfYear = {
  season: number; committedCents?: number; deadCents?: number; capCents?: number; roomCents?: number;
};
export type NativeWhatIfQuote = {
  player: string; name: string; action: string; actionName: string;
  allowed: boolean; priceable: boolean; reason: string;
  deadMoney: { season: number; cents: number }[]; salaries: { season: number; cents: number }[];
};
export type NativeWhatIfStack = {
  league: string; revision: number; season: number; before: NativeWhatIfYear[]; after: NativeWhatIfYear[];
  moves: NativeWhatIfQuote[];
};

function whatIfMoney(value: unknown, path: string): number | undefined {
  return value == null ? undefined : integer(value, path, Number.MIN_SAFE_INTEGER);
}
function whatIfAmounts(value: unknown, path: string) {
  return array(value, path, (item, at) => {
    const row = object(item, at, ['season', 'cents']);
    return { season: integer(row.season, `${at}.season`),
      cents: integer(row.cents, `${at}.cents`, Number.MIN_SAFE_INTEGER) };
  });
}
function whatIfYears(value: unknown): NativeWhatIfYear[] {
  return array(value, 'Cap runway', (item, path) => {
    const row = object(item, path, ['season', 'committedCents', 'deadCents', 'capCents', 'roomCents']);
    return { season: integer(row.season, `${path}.season`),
      committedCents: whatIfMoney(row.committedCents, `${path}.committedCents`),
      deadCents: whatIfMoney(row.deadCents, `${path}.deadCents`),
      capCents: whatIfMoney(row.capCents, `${path}.capCents`),
      roomCents: whatIfMoney(row.roomCents, `${path}.roomCents`) };
  });
}
function whatIfQuote(value: unknown, move: NativeWhatIfMove, league: string): NativeWhatIfQuote {
  const row = object(value, 'What-if move', [
    'league', 'revision', 'season', 'player', 'name', 'franchise', 'action', 'actionName', 'allowed',
    'reason', 'needs', 'years', 'salaryCents', 'salaries', 'deadMoney', 'reliefCents', 'cap', 'basis',
    'priceable',
    'notes', 'source',
  ]);
  if (row.league !== league || row.player !== move.player || row.action !== move.action
    || typeof row.allowed !== 'boolean' || typeof row.priceable !== 'boolean')
    throw new Error('What-if quote scope changed');
  const name = text(row.name, 'What-if player');
  text(row.franchise, 'What-if franchise');
  const actionName = text(row.actionName, 'What-if action');
  const reason = text(row.reason, 'What-if reason');
  text(row.source, 'What-if source');
  integer(row.revision, 'What-if revision');
  integer(row.season, 'What-if season');
  integer(row.years, 'What-if years');
  whatIfMoney(row.salaryCents, 'What-if salary');
  whatIfMoney(row.reliefCents, 'What-if relief');
  array(row.needs, 'What-if needs', text);
  array(row.basis, 'What-if basis', text);
  array(row.notes, 'What-if notes', text);
  const salaries = whatIfAmounts(row.salaries, 'What-if salaries');
  const deadMoney = whatIfAmounts(row.deadMoney, 'What-if dead money');
  if (row.cap != null) {
    const cap = object(row.cap, 'What-if cap', [
      'beforeCents', 'afterCents', 'changeCents', 'roomAfterCents', 'words',
    ]);
    whatIfMoney(cap.beforeCents, 'Cap before');
    whatIfMoney(cap.afterCents, 'Cap after');
    whatIfMoney(cap.changeCents, 'Cap change');
    whatIfMoney(cap.roomAfterCents, 'Cap room');
    text(cap.words, 'Cap words');
  }
  return {
    player: move.player, name, action: move.action, actionName, allowed: row.allowed,
    priceable: row.priceable, reason,
    deadMoney, salaries,
  };
}
export function parseNativeWhatIfStack(
  value: unknown, league: string, moves: NativeWhatIfMove[],
): NativeWhatIfStack {
  const row = object(value, 'What-if stack', ['league', 'revision', 'season', 'before', 'after', 'moves']);
  if (row.league !== league) throw new Error('What-if stack belongs to another league');
  let index = 0;
  const quotes = array(row.moves, 'What-if moves', item => {
    const move = moves[index++];
    if (!move) throw new Error('What-if move count changed');
    return whatIfQuote(item, move, league);
  });
  if (quotes.length !== moves.length) throw new Error('What-if move count changed');
  return {
    league, revision: integer(row.revision, 'What-if revision'),
    season: integer(row.season, 'What-if season'), before: whatIfYears(row.before),
    after: whatIfYears(row.after), moves: quotes,
  };
}
