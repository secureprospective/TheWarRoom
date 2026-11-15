import { HQAct as WarRoomAct } from '../shell/HQAct';
import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { Act } from '../commands/Act';
import { surfaceActions } from './surfaceActions';
import { newKey } from './forms';
import { Review } from './HostedReview';
import { HostedRules } from './HostedRules';
import { HostedCommissionerLeague } from './HostedCommissionerLeague';
import { localTime } from '../clock/localTime';
import { createHostedRoom, type HostedPort } from './hostedRoom';
import type { NativeArguments, NativeRequest } from '../commands/native';
import type { HostedResponse } from './api';
import './hostedLeague.css';
import { HostedWorkflows, type HostedTask } from './HostedWorkflows';

type Tab = 'lineup' | 'roster' | 'draft' | 'rules' | HostedTask;
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
// Which screen an owner is on is not private league data: a server restart remounts this tree behind
// the identity fence (hostedReads), and the owner must come back to the screen they were reading. A
// changed role or franchise is a different view and starts fresh.
const remembered = new Map<string, unknown>();
function useRemembered<T>(key: string, initial: T): [T, (value: T) => void] {
  const [value, set] = useState<T>(() => (remembered.has(key) ? remembered.get(key) as T : initial));
  return [value, (next: T) => { remembered.set(key, next); set(next); }];
}
export default function HostedLeague(props: {
  league: string; franchise: string; name: string; role: string; port?: HostedPort; hq?: boolean;
}) {
  if (!props.franchise && (props.role === 'Commissioner' || props.role === 'Co-commissioner'))
    return <HostedCommissionerLeague league={props.league} port={props.port} />;
  return <HostedFranchise {...props} />;
}
function HostedFranchise(props: {
  league: string; franchise: string; name: string; role: string; port?: HostedPort; hq?: boolean;
}) {
  const place = `${props.league}:${props.franchise}:${props.role}`;
  const [showWar, setWar] = useRemembered(`${place}:war`, false);
  const [showHQ, setHQ] = useRemembered(`${place}:hq`, props.hq ?? false);
  const [showPulse, setPulse] = useRemembered(`${place}:pulse`, false);
  const [pulseTab, setPulseTab] = useRemembered<'now' | 'race'>(`${place}:pulseTab`, 'now');
  if (showPulse) return <Suspense fallback={<p>Opening League Pulse…</p>}>
    <NativeLeaguePulse league={props.league} mine={props.franchise} port={props.port} serverTime
      navigate={setPulseTab} back={() => setPulse(false)}
      commissioner={props.role === 'Commissioner' || props.role === 'Co-commissioner'} tab={pulseTab}
      leave={node => { setPulse(false); if (node === 'hq') { setHQ(true); setWar(false); }
        if (node === 'war') { setHQ(false); setWar(true); } }} />
  </Suspense>;
  if (showWar) return <Suspense fallback={<p>Opening War Room…</p>}><WarRoomNative
    league={props.league} mine={props.franchise} port={props.port}
    leave={node => { if (node === 'pulse') setPulse(true);
      else { setWar(false); if (node === 'hq') setHQ(true); } }}
    /></Suspense>;
  if (showHQ) return <Suspense fallback={<p>Opening your franchise…</p>}>
    <NativeFranchiseHQ key={`${props.league}:${props.franchise}`} league={props.league}
      mine={props.franchise} name={props.name} port={props.port}
      leave={node => { if (node === 'pulse') setPulse(true);
        else { setHQ(false); if (node === 'war') setWar(true); } }} />
  </Suspense>;
  return <><WarRoomAct onAction={() => setPulse(true)}>League Pulse</WarRoomAct>
    <WarRoomAct onAction={() => setWar(true)}>War Room</WarRoomAct><HostedWorkRoom {...props} /></>;
}
function HostedWorkRoom({
  league,
  franchise,
  name,
  role,
  port,
}: {
  league: string;
  franchise: string;
  name: string;
  role: string;
  port?: (request: NativeRequest, football?: string) => Promise<HostedResponse>;
}) {
  const room = useMemo(() => createHostedRoom(league, franchise, port), [league, franchise, port]);
  const state = room.use();
  const [tab, setTab] = useRemembered<Tab>(`${league}:${franchise}:${role}:tab`, 'lineup');
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
      {state.error && <p role="alert">{state.error}</p>}
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
                <strong>{p.name}</strong> · {bucketNames[p.bucket]}
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
