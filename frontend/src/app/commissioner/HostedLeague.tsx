import { useEffect, useMemo, useState } from 'react';
import { Act } from '../commands/Act';
import { surfaceActions } from './surfaceActions';
import { newKey } from './forms';
import { HoldToFire } from './HoldToFire';
import { GovernedGlyph } from '../governed/GovernedGlyph';
import { localTime } from '../clock/localTime';
import { createHostedRoom, type HostedRoom } from './hostedRoom';
import type { NativeArguments, NativeRequest } from '../commands/native';
import type { HostedResponse } from './api';
import './hostedLeague.css';

type Tab = 'lineup' | 'roster' | 'draft' | 'rules';
const labels: Record<Tab, string> = { lineup: 'Set lineup', roster: 'Roster and reserves', draft: 'Draft room', rules: 'League rules' };
const bucketNames = { active: 'Active roster', ir: 'Injured reserve', taxi: 'Taxi squad' };
export default function HostedLeague({ league, franchise, name, role, port }: {
  league: string; franchise: string; name: string; role: string;
  port?: (request: NativeRequest, football?: string) => Promise<HostedResponse>;
}) {
  const room = useMemo(() => createHostedRoom(league, franchise, port), [league, franchise, port]);
  const state = room.use();
  const [tab, setTab] = useState<Tab>('lineup');
  const [week, setWeek] = useState(1);
  const [starters, setStarters] = useState<string[]>([]);
  const [player, setPlayer] = useState('');
  const [bucket, setBucket] = useState<'active' | 'ir' | 'taxi'>('ir');
  const [rookie, setRookie] = useState('');
  const [reason, setReason] = useState('');
  useEffect(() => { void room.read(); return room.dispose; }, [room]);
  useEffect(() => { setStarters(state.reading?.lineup.starters.map(p => p.player) || []); }, [state.reading]);
  const reading = state.reading;
  const disabled = !room.canChange();
  const command = (
    id: 'commish.lineup.set' | 'roster.reserve' | 'draft.timer' | 'draft.skip',
    args: NativeArguments<'commish.lineup.set'>['args'] | NativeArguments<'roster.reserve'>['args'] | NativeArguments<'draft.timer'>['args'], visibility: 'league' | 'private', summary: string,
  ) => {
    if (!reading) return;
    const request = { command: id, mode: 'preview', arguments: { league, command: id, args, reason,
      expected: id === 'commish.lineup.set' ? reading.lineup.revision
        : id.startsWith('draft.') ? reading.draft.revision : reading.roster.revision,
      request: newKey(), effective: 'now', visibility } } as NativeRequest;
    void room.review(request, summary);
  };
  async function act(value: string) {
    if (value === 'refresh') { await room.read(week); return; }
    if (value === 'cancel') { room.cancel(); return; }
    if (value === 'confirm') { await room.confirm(); return; }
    if (!reading) return;
    if (value === 'lineup') command('commish.lineup.set', { lineup: { season: reading.lineup.season,
      franchise, week: reading.lineup.week, starters } }, 'private', `Set your week ${reading.lineup.week} lineup with ${starters.length} starters`);
    if (value === 'reserve') {
      const p = reading.roster.players.find(p => p.id === player && p.franchise === franchise);
      if (p) command('roster.reserve', { waiver: { player, franchise, bucket } }, 'league', `Move ${p.name} to ${bucketNames[bucket]}`);
    }
    if (value === 'pick') {
      const p = reading.draft.available.find(p => p.id === rookie);
      if (p) await room.review({ command: 'draft.pick', mode: 'execute', arguments: {
        league, franchise, player: rookie, season: reading.draft.season,
        expected: reading.draft.revision, request: newKey() } }, `Draft ${p.name}`);
    }
    if (value === 'pause' || value === 'resume') command('draft.timer', { draft: {
      season: reading.draft.season, act: { timer: value } } }, 'league', `${value === 'pause' ? 'Pause' : 'Resume'} the draft clock`);
    if (value === 'skip') command('draft.skip', { draft: { season: reading.draft.season, act: {} } }, 'league', 'Skip the pick on the clock');
  }
  useEffect(() => surfaceActions.register('hosted-league', args => {
    if (args.action === 'view' && args.value in labels) {
      setTab(args.value as Tab); room.cancel();
    } else if (args.action === 'check') void act(args.value);
  }));
  const action = (value: string, label: string, off = disabled) => <Act variant="button"
    verb="commissioner.surface" args={{ screen: 'hosted-league', action: 'check', value }}
    disabled={off} label={label}>{label}</Act>;
  const changed = () => room.cancel();
  const own = reading?.roster.players.filter(p => p.franchise === franchise) || [];
  const reserves = reading?.reserves.find(r => r.franchise === franchise);
  const clock = reading?.draft.onClock;
  const commissioner = role === 'Commissioner' || role === 'Co-commissioner';
  return <section className="hosted-league" aria-label="Hosted franchise" aria-busy={state.pending}>
    <h2>{name || 'Your franchise'}</h2>
    <p>{reading?.label || 'Checking where your league is managed…'}</p>
    <p role="status">{state.stale ? 'Read-only saved view — not connected. No changes can be made.'
      : state.pending ? 'Checking the server…' : 'Connected to your league server. The server records every change.'}</p>
    {reading && <p>Last read: {localTime(new Date(reading.readAt).toISOString())}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.message && <p role="status">{state.message}</p>}
    {action('refresh', state.pending ? 'Refreshing…' : 'Refresh league', state.pending)}
    <nav aria-label="Franchise tasks">{(Object.keys(labels) as Tab[]).map(id => <Act key={id}
      variant="button" verb="commissioner.surface" args={{ screen: 'hosted-league', action: 'view', value: id }}
      label={labels[id]} current={tab === id} disabled={state.pending}>{labels[id]}</Act>)}</nav>
    {!reading && !state.pending && <p>Refresh to open your lineup, roster and draft. If access is refused, sign in again and ask your commissioner to check your seat.</p>}
    {reading && tab === 'lineup' && <>
      <h3>Week {reading.lineup.week} lineup</h3>
      <label>Week to read<select value={week} disabled={state.pending} onChange={e => { setWeek(Number(e.target.value)); changed(); }}>
        {Array.from({ length: 22 }, (_, i) => <option key={i} value={i + 1}>Week {i + 1}</option>)}
      </select></label><p>Choose a week, then refresh. Your changes below stay on this screen until confirmed.</p>
      <p>{reading.lineup.words}</p><p>{reading.lineup.verdict}</p>
      <fieldset disabled={disabled || Boolean(state.review)}><legend>Choose your starters</legend>
        {[...reading.lineup.starters, ...reading.lineup.bench].map(p => <label className="hosted-starter" key={p.player}>
          <input type="checkbox" checked={starters.includes(p.player)} disabled={p.locked}
            onChange={e => { setStarters(s => e.target.checked ? [...s, p.player] : s.filter(id => id !== p.player)); changed(); }} />
          <span>{p.name} · {p.position}<small>{p.words}</small></span>
        </label>)}
      </fieldset>
      {own.length === 0 && <p>No players are recorded for your franchise. Ask your commissioner to check the starting roster.</p>}
      {action('lineup', 'Check lineup', disabled || starters.length === 0)}
    </>}
    {reading && tab === 'roster' && <>
      <h3>Roster and reserves</h3><p>{reserves?.words.join(' · ')}</p>
      <ul>{own.map(p => <li key={p.id}><strong>{p.name}</strong> · {bucketNames[p.bucket]}
        <small>{reserves?.players.find(v => v.player === p.id)?.words}</small></li>)}</ul>
      <label>Player to move<select value={player} disabled={disabled} onChange={e => { setPlayer(e.target.value); changed(); }}>
        <option value="">Choose a player</option>{own.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
      </select></label>
      <label>Move to<select value={bucket} disabled={disabled} onChange={e => { setBucket(e.target.value as typeof bucket); changed(); }}>
        {(Object.keys(bucketNames) as (keyof typeof bucketNames)[]).map(b => <option key={b} value={b}>{bucketNames[b]}</option>)}
      </select></label>
      <p>The server checks eligibility, available spots, calendar, roster size and salary cap. An unknown fact needs a commissioner ruling; it is not permission.</p>
      {action('reserve', 'Check reserve move', disabled || !player)}
    </>}
    {reading && tab === 'draft' && <>
      <h3>{reading.draft.season} rookie draft</h3><p>{reading.draft.words}</p>
      {clock && <p>{reading.draft.owners.find(o => o.id === clock.owner)?.name || 'Another franchise'} is on the clock.
        {clock.paused ? ' Clock paused.' : ` Pick due ${localTime(clock.deadline)}.`}</p>}
      <p>Refresh for other owners’ picks and clock changes. The server rechecks whose turn it is when you confirm.</p>
      <ol>{reading.draft.picks.map((p, i) => <li key={i}>Round {p.round}, pick {p.slot}: {p.player
        ? reading.draft.players[p.player]?.name || 'Recorded rookie'
        : p.skipped ? 'Skipped' : 'Not selected'} · {reading.draft.owners.find(o => o.id === p.owner)?.name || 'League franchise'}</li>)}</ol>
      <label>Rookie<select value={rookie} disabled={disabled} onChange={e => { setRookie(e.target.value); changed(); }}>
        <option value="">Choose a rookie</option>{reading.draft.available.map(p => <option key={p.id} value={p.id}>{p.name} · {p.position} {p.team}</option>)}
      </select></label>
      {action('pick', 'Review rookie pick', disabled || !rookie || clock?.owner !== franchise || clock.paused)}
      {commissioner && <><h3>Commissioner draft controls</h3>
        <label>Reason for the ruling<textarea value={reason} disabled={disabled} onChange={e => { setReason(e.target.value); changed(); }} /></label>
        <div className="native-actions">{action(clock?.paused ? 'resume' : 'pause', clock?.paused ? 'Check resume' : 'Check pause', disabled || !clock || !reason.trim())}
          {action('skip', 'Check skip', disabled || !clock || !reason.trim())}</div></>}
    </>}
    {reading && tab === 'rules' && <><h3>Rules in force</h3><p>Your commissioner controls these rules. Ask them for a ruling if a value is missing.</p>
      {reading.rules.sections.filter(s => s.rows.some(r => /^(lineup|roster|ir|taxi|draft)\./.test(r.field))).map(s => <section key={s.label}>
        <h4>{s.label}</h4><dl>{s.rows.filter(r => /^(lineup|roster|ir|taxi|draft)\./.test(r.field)).map(r => <div key={r.field}>
          <dt>{r.label}{r.governed && <GovernedGlyph governed={r.governed} subject={r.label} />}</dt><dd>{r.value}</dd>
        </div>)}</dl></section>)}</>}
    {state.review && <Review room={room} />}
  </section>;
}

function Review({ room }: { room: HostedRoom }) {
  const { review, pending } = room.use();
  if (!review) return null;
  return <section aria-label="Confirm league change" className="hosted-review"><h3>{review.summary}</h3>
    <p>This changes your hosted league, not a local plan. Nothing is queued while offline.</p>
    {review.preview?.fields.map((field, i) => <p key={i}><strong>{field.label}</strong>: {field.before} → {field.after}. {field.when}</p>)}
    {review.preview?.blockers.map((b, i) => <p role="alert" key={i}>{b}</p>)}
    {!review.preview?.blockers.length && <HoldToFire label="Confirm league change" body={JSON.stringify(review.request)}
      disabled={pending || !room.canChange()} onConfirm={() => void room.confirm()} />}
    <Act variant="button" verb="commissioner.surface" args={{ screen: 'hosted-league', action: 'check', value: 'cancel' }}
      label="Cancel change" disabled={pending}>Cancel change</Act>
  </section>;
}
