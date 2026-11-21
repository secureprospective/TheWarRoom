import { HQAct as WarRoomAct } from '../shell/HQAct';
import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { Act } from '../commands/Act';
import { surfaceActions } from './surfaceActions';
import { newKey } from './forms';
import { Review } from './HostedReview';
import { HostedRules } from './HostedRules';
import { HostedCommissionerLeague } from './HostedCommissionerLeague';
import { localTime } from '../clock/localTime';
import { createHostedRoom, type HostedPort } from './hostedRoom';
import type { NativeArguments, NativeRequest } from '../commands/native';
import { hostedWorkflow, type HostedResponse } from './api';
import './hostedLeague.css';
import { hostedScreens } from '../data/hostedReads';
import { FailureAlert } from '../data/FailureAlert';
import { setFailureScreen } from '../data/errorCodes';
import { parseHostedRoster } from './hostedReadings';
import { HostedWorkflows, type HostedTask } from './HostedWorkflows';
import { ThinLeagueShell } from '../shell/ThinLeagueShell';
import type { PlayerCandidate, ThinScreenRoute } from '../shell/ThinPlayerSearch';
import type { HQTab } from '../shell/HQShell';
import type { PulseTab } from '../shell/NativeLeaguePulse';
import type { Node } from '../shell/nodes';

type Tab = 'lineup' | 'roster' | 'draft' | 'rules' | HostedTask;
export type HostedEntry = 'home' | 'hq' | 'war' | 'pulse' | 'league' | 'control'
  | 'lineup' | 'bids' | 'trades';
const entryTab = (entry?: HostedEntry): Tab => entry === 'control' ? 'rules'
  : entry === 'bids' || entry === 'trades' || entry === 'lineup' ? entry : 'lineup';
const pulseTab = (workspace: string): PulseTab => workspace === 'race' || workspace === 'schedule'
  || workspace === 'activity' ? workspace : 'now';
const labels: Record<Tab, string> = {
  lineup: 'Set lineup',
  roster: 'Roster and reserves',
  draft: 'Draft room',
  rules: 'League rules',
  bids: 'Free-agent bids',
  waivers: 'Waivers',
  trades: 'Trades',
  results: 'Results and standings',
  rollover: 'Next season',
};
const bucketNames = { active: 'Active roster', ir: 'Injured reserve', taxi: 'Taxi squad' };
const WarRoomNative = lazy(() => import('../shell/WarRoomNative'));
const NativeLeaguePulse = lazy(() => import('../shell/NativeLeaguePulse'));
const NativeFranchiseHQ = lazy(() => import('../shell/NativeFranchiseHQ'));
// A server restart remounts this tree behind the identity fence; the owner comes back to the same screen.
// A changed role or franchise is a different view and starts fresh.
function useRemembered<T>(key: string, initial: T, forceInitial = false): [T, (value: T) => void] {
  const [value, set] = useState<T>(() => forceInitial ? initial : hostedScreens.recall(key, initial));
  return [value, (next: T) => { hostedScreens.remember(key, next); set(next); }];
}
function entryRoute(entry: HostedEntry | undefined, hq: boolean | undefined,
  commissioner: boolean): ThinScreenRoute {
  if (entry === 'home') return { node: 'home', workspace: 'overview' };
  if (entry === 'league') return { node: 'pulse', workspace: 'race' };
  if (entry === 'hq') return { node: 'hq', workspace: 'lineup-and-roster' };
  if (entry === 'war') return { node: 'war', workspace: 'trade-desk' };
  if (entry === 'pulse') return { node: 'pulse', workspace: 'now' };
  if (entry === 'control') return { node: 'control', workspace: 'league' };
  if (entry) return { node: 'home', workspace: 'overview' };
  return commissioner ? { node: 'home', workspace: 'overview' }
    : hq ? { node: 'hq', workspace: 'lineup-and-roster' }
      : { node: 'home', workspace: 'overview' };
}
export default function HostedLeague(props: {
  league: string; franchise: string; name: string; role: string; port?: HostedPort; hq?: boolean;
  entry?: HostedEntry; onHome?: () => void;
}) {
  const commissioner = props.role === 'Commissioner' || props.role === 'Co-commissioner';
  const place = `${props.league}:${props.franchise}:${props.role}`;
  const initialRoute: ThinScreenRoute = entryRoute(props.entry, props.hq,
    !props.franchise && commissioner);
  const [route, setRoute] = useRemembered<ThinScreenRoute>(`${place}:thinRoute`, initialRoute,
    props.entry !== undefined);
  useEffect(() => {
    setFailureScreen(`${route.node} · ${route.workspace}`, `${route.node}/${route.workspace}`);
  }, [route.node, route.workspace]);
  const [playerSources, setPlayerSources] = useState<Record<string, PlayerCandidate[]>>({});
  const [inspectPlayer, setInspectPlayer] = useState<PlayerCandidate>();
  useEffect(() => {
    if (!props.franchise) return;
    let active = true;
    void (props.port ?? hostedWorkflow)({ command: 'native.assets.read', mode: 'read',
      arguments: { league: props.league } }).then(response => {
      const roster = parseHostedRoster(response.data);
      if (!active) return;
      publishLeaguePlayers([
        ...roster.players.map(player => ({ id: player.id, name: player.name })),
        ...roster.market.map(player => ({ id: player.id, name: player.name,
          position: player.position })),
        ...roster.pool.map(player => ({ id: player.value, name: player.label })),
      ]);
    }).catch(error => console.warn('League player search read failed:', error));
    return () => { active = false; };
  }, [props.league, props.port]);
  const publishPlayers = useCallback((source: string, players: PlayerCandidate[]) => {
    setPlayerSources(previous => ({ ...previous, [source]: players }));
  }, []);
  const players = useMemo(() => {
    const byId = new Map<string, PlayerCandidate>();
    Object.values(playerSources).flat().forEach(player => byId.set(player.id, player));
    return [...byId.values()];
  }, [playerSources]);
  const navigate = (next: ThinScreenRoute) => { setRoute(next); setInspectPlayer(undefined); };
  const publishLeaguePlayers = useCallback((found: PlayerCandidate[]) => publishPlayers('league', found),
    [publishPlayers]);
  const openPlayer = (player: PlayerCandidate) => {
    setInspectPlayer(player);
    setRoute({ node: 'war', workspace: 'valuations-and-pool' });
  };
  const publishHomePlayers = useCallback((found: PlayerCandidate[]) => publishPlayers('home', found),
    [publishPlayers]);
  const publishHQPlayers = useCallback((found: PlayerCandidate[]) => publishPlayers('hq', found),
    [publishPlayers]);
  const publishWarPlayers = useCallback((found: PlayerCandidate[]) => publishPlayers('war', found),
    [publishPlayers]);
  const publishPulsePlayers = useCallback((found: PlayerCandidate[]) => publishPlayers('pulse', found),
    [publishPlayers]);
  const openNode = (node?: Node) => navigate({ node: node ?? 'home',
    workspace: node === 'hq' ? 'lineup-and-roster' : node === 'war' ? 'trade-desk' :
      node === 'pulse' ? 'now' : node === 'control' ? 'league' : 'overview' });
  const screen = !props.franchise && commissioner && route.node !== 'war' ? 'commissioner'
    : route.node === 'control' && !commissioner ? 'home' : route.node;
  const page = screen === 'commissioner' || screen === 'control'
    ? <HostedCommissionerLeague league={props.league} port={props.port} embedded
        view={route.node === 'control' ? 'control' : 'home'}
        pulseTab={route.node === 'pulse' ? pulseTab(route.workspace) : 'now'}
        navigatePulse={workspace => setRoute({ node: 'pulse', workspace })}
        leave={openNode} onPlayer={openPlayer} onPlayers={publishPulsePlayers} />
    : screen === 'hq' ? <Suspense fallback={<p>Opening your franchise…</p>}>
        <NativeFranchiseHQ key={`${props.league}:${props.franchise}`} league={props.league}
          mine={props.franchise} name={props.name} port={props.port} embedded
          tab={route.workspace as HQTab} navigate={workspace => setRoute({ node: 'hq', workspace })}
          leave={openNode} onPlayers={publishHQPlayers} />
      </Suspense>
    : screen === 'war' ? <Suspense fallback={<p>Opening War Room…</p>}>
        <WarRoomNative league={props.league} mine={props.franchise} port={props.port} embedded
          owner={Boolean(props.franchise)} workspace={route.workspace}
          navigateWorkspace={workspace => setRoute({ node: 'war', workspace })}
          inspectPlayer={inspectPlayer} onPlayers={publishWarPlayers} leave={openNode} />
      </Suspense>
    : screen === 'pulse' ? <Suspense fallback={<p>Opening League Pulse…</p>}>
        <NativeLeaguePulse league={props.league} mine={props.franchise} port={props.port} serverTime
          embedded tab={pulseTab(route.workspace)}
          navigate={workspace => setRoute({ node: 'pulse', workspace })} leave={openNode}
          commissioner={commissioner} onPlayer={openPlayer} onPlayers={publishPulsePlayers} />
      </Suspense>
    : <HostedWorkRoom {...props} onPlayers={publishHomePlayers} onPlayer={openPlayer} />;
  return <ThinLeagueShell league={props.league} leagueName={props.name} route={route}
    commissioner={commissioner} players={players} navigate={navigate} openPlayer={openPlayer}
    onHome={props.onHome}>{page}</ThinLeagueShell>;
}
function HostedWorkRoom({
  league,
  franchise,
  name,
  role,
  port,
  onPlayers,
  onPlayer,
  entry,
}: {
  league: string;
  franchise: string;
  name: string;
  role: string;
  port?: (request: NativeRequest, football?: string) => Promise<HostedResponse>;
  onPlayers?: (players: PlayerCandidate[]) => void;
  onPlayer?: (player: PlayerCandidate) => void;
  entry?: HostedEntry;
}) {
  const room = useMemo(() => createHostedRoom(league, franchise, port), [league, franchise, port]);
  const state = room.use();
  const [tab, setTab] = useRemembered<Tab>(`${league}:${franchise}:${role}:tab`, entryTab(entry),
    entry !== undefined);
  const [week, setWeek] = useState(1);
  const [starters, setStarters] = useState<string[]>([]);
  const [player, setPlayer] = useState('');
  const [bucket, setBucket] = useState<'active' | 'ir' | 'taxi'>('ir');
  const [rookie, setRookie] = useState('');
  const [reason, setReason] = useState('');
  useEffect(() => {
    void room.refresh();
    return room.dispose;
  }, [room]);
  useEffect(() => {
    setStarters(state.reading?.lineup.starters.map((p) => p.player) || []);
  }, [state.reading]);
  const reading = state.reading;
  useEffect(() => {
    if (!reading) return;
    onPlayers?.([
      ...reading.roster.players.map(player => ({ id: player.id, name: player.name })),
      ...reading.roster.market.map(player => ({ id: player.id, name: player.name,
        position: player.position })),
    ]);
  }, [onPlayers, reading]);
  const disabled = !room.canChange();
  const command = (
    id: 'commish.lineup.set' | 'roster.reserve' | 'draft.timer' | 'draft.skip',
    args:
      | NativeArguments<'commish.lineup.set'>['args']
      | NativeArguments<'roster.reserve'>['args']
      | NativeArguments<'draft.timer'>['args'],
    visibility: 'league' | 'private',
    summary: string,
  ) => {
    if (!reading) return;
    const request = {
      command: id,
      mode: 'preview',
      arguments: {
        league,
        command: id,
        args,
        reason,
        expected:
          id === 'commish.lineup.set'
            ? reading.lineup.revision
            : id.startsWith('draft.')
              ? reading.draft.revision
              : reading.roster.revision,
        request: newKey(),
        effective: 'now',
        visibility,
      },
    } as NativeRequest;
    return room.review(request, summary);
  };
  async function act(value: string) {
    if (value === 'refresh') {
      await room.refresh(week);
      return;
    }
    if (value === 'cancel') {
      room.cancel();
      return;
    }
    if (value === 'confirm') {
      await room.confirm();
      return;
    }
    if (!reading) return;
    if (value === 'lineup')
      await command(
        'commish.lineup.set',
        {
          lineup: { season: reading.lineup.season, franchise, week: reading.lineup.week, starters },
        },
        'private',
        `Set your week ${reading.lineup.week} lineup with ${starters.length} starters`,
      );
    if (value === 'reserve') {
      const p = reading.roster.players.find((p) => p.id === player && p.franchise === franchise);
      if (p)
        await command(
          'roster.reserve',
          { waiver: { player, franchise, bucket } },
          'league',
          `Move ${p.name} to ${bucketNames[bucket]}`,
        );
    }
    if (value === 'pick') {
      const p = reading.draft.available.find((p) => p.id === rookie);
      if (p)
        await room.review(
          {
            command: 'draft.pick',
            mode: 'execute',
            arguments: {
              league,
              franchise,
              player: rookie,
              season: reading.draft.season,
              expected: reading.draft.revision,
              request: newKey(),
            },
          },
          `Draft ${p.name}`,
        );
    }
    if (value === 'pause' || value === 'resume')
      await command(
        'draft.timer',
        {
          draft: {
            season: reading.draft.season,
            act: { timer: value },
          },
        },
        'league',
        `${value === 'pause' ? 'Pause' : 'Resume'} the draft clock`,
      );
    if (value === 'skip' && reading.draft.onClock)
      await command(
        'draft.skip',
        {
          draft: {
            season: reading.draft.season,
            act: {
              round: reading.draft.onClock.round,
              original: reading.draft.onClock.original,
            },
          },
        },
        'league',
        'Skip the pick on the clock',
      );
  }
  useEffect(() =>
    surfaceActions.register('hosted-league', async (args) => {
      if (args.action === 'view' && args.value in labels) {
        setTab(args.value as Tab);
        room.cancel();
      } else if (args.action === 'check') {
        await act(args.value);
        if (room.peek().error) throw new Error(room.peek().error);
      }
    }),
  );
  const action = (value: string, label: string, off = disabled) => (
    <Act
      variant="button"
      verb="commissioner.surface"
      args={{ screen: 'hosted-league', action: 'check', value }}
      disabled={off}
      label={label}
    >
      {label}
    </Act>
  );
  const changed = () => room.cancel();
  const own = reading?.roster.players.filter((p) => p.franchise === franchise) || [];
  const reserves = reading?.reserves.find((r) => r.franchise === franchise);
  const clock = reading?.draft.onClock;
  const commissioner = role === 'Commissioner' || role === 'Co-commissioner';
  return (
    <section className="hosted-league" aria-label="Hosted franchise" aria-busy={state.pending}>
      <h2>{name || 'Your franchise'}</h2>
      <p>Your league server</p>
      <p>
        MFL is reference only. Refreshing MFL cannot replace this league’s roster, rules or recorded changes.
      </p>
      <p>{reading?.label || 'Checking where your league is managed…'}</p>
      <p role="status">
        {state.stale
          ? 'Stale — read-only saved view. Refresh to check your connection. No changes can be made.'
          : state.pending
            ? 'Checking the server…'
            : 'Connected to your league server. The server records every change.'}
      </p>
      {reading && <p>Last read: {localTime(new Date(reading.readAt).toISOString())}</p>}
      {state.error && <FailureAlert message={state.error} league={league} port={port} />}
      {state.message && <p role="status">{state.message}</p>}
      {action('refresh', state.pending ? 'Refreshing…' : 'Refresh league server', state.pending)}
      <nav aria-label="Franchise tasks">
        {(Object.keys(labels) as Tab[]).map((id) => (
          <Act
            key={id}
            variant="button"
            verb="commissioner.surface"
            args={{ screen: 'hosted-league', action: 'view', value: id }}
            label={labels[id]}
            current={tab === id}
            disabled={state.pending}
          >
            {labels[id]}
          </Act>
        ))}
      </nav>
      {!reading && !state.pending && (
        <p>
          Refresh to open your lineup, roster and draft. If access is refused, sign in again and ask
          your commissioner to check your seat.
        </p>
      )}
      {reading && tab === 'lineup' && (
        <>
          <h3>Week {reading.lineup.week} lineup</h3>
          <label>
            Week to read
            <select
              value={week}
              disabled={state.pending}
              onChange={(e) => {
                setWeek(Number(e.target.value));
                changed();
              }}
            >
              {Array.from({ length: 22 }, (_, i) => (
                <option key={i} value={i + 1}>
                  Week {i + 1}
                </option>
              ))}
            </select>
          </label>
          <p>
            Choose a week, then refresh. Your changes below stay on this screen until confirmed.
          </p>
          <p>{reading.lineup.words}</p>
          <fieldset disabled={disabled || !reading.lineup.available || Boolean(state.review)}>
            <legend>Choose your starters</legend>
            {[...reading.lineup.starters, ...reading.lineup.bench].map((p) => (
              <label className="hosted-starter" key={p.player}>
                <input
                  type="checkbox"
                  checked={starters.includes(p.player)}
                  disabled={p.locked}
                  onChange={(e) => {
                    setStarters((s) =>
                      e.target.checked ? [...s, p.player] : s.filter((id) => id !== p.player),
                    );
                    changed();
                  }}
                />
                <span>
                  {p.name} · {p.position}
                  <small>{p.words}</small>
                </span>
              </label>
            ))}
          </fieldset>
          {own.length === 0 && (
            <p>
              No players are recorded for your franchise. Ask your commissioner to check the
              starting roster.
            </p>
          )}
          {action(
            'lineup',
            'Check lineup',
            disabled || !reading.lineup.available || starters.length === 0,
          )}
        </>
      )}
      {reading && tab === 'roster' && (
        <>
          <h3>Roster and reserves</h3>
          <p>{reserves?.words.join(' · ')}</p>
          <ul>
            {own.map((p) => (
              <li key={p.id}>
                <WarRoomAct className="hosted-player-name" onAction={() => onPlayer?.({
                  id: p.id, name: p.name,
                })}>{p.name}</WarRoomAct> · {bucketNames[p.bucket]}
                <small>{reserves?.players.find((v) => v.player === p.id)?.words}</small>
              </li>
            ))}
          </ul>
          <label>
            Player to move
            <select
              value={player}
              disabled={disabled}
              onChange={(e) => {
                setPlayer(e.target.value);
                changed();
              }}
            >
              <option value="">Choose a player</option>
              {own.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Move to
            <select
              value={bucket}
              disabled={disabled}
              onChange={(e) => {
                setBucket(e.target.value as typeof bucket);
                changed();
              }}
            >
              {(Object.keys(bucketNames) as (keyof typeof bucketNames)[]).map((b) => (
                <option key={b} value={b}>
                  {bucketNames[b]}
                </option>
              ))}
            </select>
          </label>
          <p>
            The server checks eligibility, available spots, calendar, roster size and salary cap. An
            unknown fact needs a commissioner ruling; it is not permission.
          </p>
          {action('reserve', 'Check reserve move', disabled || !player)}
        </>
      )}
      {reading && tab === 'draft' && (
        <>
          <h3>{reading.draft.season} rookie draft</h3>
          <p>{reading.draft.words}</p>
          {clock && (
            <p>
              {reading.draft.owners.find((o) => o.id === clock.owner)?.name || 'Another franchise'}{' '}
              is on the clock.
              {clock.paused ? ' Clock paused.' : ` Pick due ${localTime(clock.deadline)}.`}
            </p>
          )}
          <p>
            Refresh for other owners’ picks and clock changes. The server rechecks whose turn it is
            when you confirm.
          </p>
          <ol>
            {reading.draft.picks.map((p, i) => (
              <li key={i}>
                Round {p.round}, pick {p.slot}:{' '}
                {p.player
                  ? reading.draft.players[p.player]?.name || 'Recorded rookie'
                  : p.skipped
                    ? 'Skipped'
                    : 'Not selected'}{' '}
                · {reading.draft.owners.find((o) => o.id === p.owner)?.name || 'League franchise'}
              </li>
            ))}
          </ol>
          <label>
            Rookie
            <select
              value={rookie}
              disabled={disabled}
              onChange={(e) => {
                setRookie(e.target.value);
                changed();
              }}
            >
              <option value="">Choose a rookie</option>
              {reading.draft.available.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} · {p.position} {p.team}
                </option>
              ))}
            </select>
          </label>
          {action(
            'pick',
            'Review rookie pick',
            disabled || !rookie || clock?.owner !== franchise || clock.paused,
          )}
          {commissioner && (
            <>
              <h3>Commissioner draft controls</h3>
              <label>
                Reason for the ruling
                <textarea
                  value={reason}
                  disabled={disabled}
                  onChange={(e) => {
                    setReason(e.target.value);
                    changed();
                  }}
                />
              </label>
              <div className="native-actions">
                {action(
                  clock?.paused ? 'resume' : 'pause',
                  clock?.paused ? 'Check resume' : 'Check pause',
                  disabled || !clock || !reason.trim(),
                )}
                {action('skip', 'Check skip', disabled || !clock || !reason.trim())}
              </div>
            </>
          )}
        </>
      )}
      {reading && ['bids', 'waivers', 'trades', 'results', 'rollover'].includes(tab) && (
        <HostedWorkflows
          key={`${tab}:${reading.roster.revision}`}
          task={tab as HostedTask}
          room={room}
          reading={reading}
          franchise={franchise}
          commissioner={commissioner}
        />
      )}
      {reading && tab === 'rules' &&
        <HostedRules rules={reading.rules} commissioner={commissioner} room={room} />}
      {state.review && tab !== 'rules' && <Review room={room} />}
    </section>
  );
}
