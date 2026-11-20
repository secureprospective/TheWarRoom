import { useEffect, useMemo, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import type { HostedPort } from '../commissioner/hostedRoom';
import { parseJobs } from '../commissioner/jobs';
import type { PulseActivity, PulseResults, PulseStandingRow } from '../data/nativePulse';
import { HQAct } from '../shell/HQAct';
import type { LeagueChoice } from '../shell/leaguePicker';
import { PhoneSheet } from '../shell/PhoneAppFrame';
import { readNativePulse, readPulseActivity, readPulseSeason, readPulseWeekResults,
  type NativePulse, type PulseSeason } from '../shell/pulseReads';
import { ProblemReport } from '../control/ProblemReport';
import { readPlayerStatus, statusNote, statusWords, type PlayerStatus } from './playerStatus';
import './LeagueTab.css';

export type LeagueView = 'st' | 'mu' | 'sc' | 'ac';
type Part<T> = { value?: T; error?: string };
export type LeagueJobs = { open: number; now: number };
export type LeagueReading = { pulse: Part<NativePulse>; season: Part<PulseSeason>;
  activity: Part<PulseActivity>; jobs?: Part<LeagueJobs>; status?: Part<LeagueStatus> };
/** LeagueStatus is the NFL player status read narrowed to players on this league's rosters. */
export type LeagueStatus = { read: PlayerStatus; names: Map<string, string> };
export type LeagueRosterPlayer = { id: string; name: string; position: string };
export const NOT_YET = 'Not available yet';
const message = (cause: unknown) => cause instanceof Error ? cause.message : String(cause);
async function part<T>(read: () => Promise<T>): Promise<Part<T>> {
  try { return { value: await read() }; } catch (cause) { return { error: message(cause) }; }
}

/** readLeague takes the existing League Pulse reads (directory, week, standings, bracket, schedule,
 *  results, public activity) and, for a commissioner, the hosted jobs read. Each fails on its own. */
export async function readLeague(league: string, mine: string, commissioner: boolean,
  port: HostedPort = hostedWorkflow): Promise<LeagueReading> {
  const pulse = await part(() => readNativePulse(league, mine, port, undefined, true, false));
  const p = pulse.value;
  const through = Math.max(p?.standings?.through ?? 0, (p?.week ?? 1) - 1);
  const [season, activity, jobs, status] = await Promise.all([
    p ? part(() => readPulseSeason(league, p.season, through, port))
      : Promise.resolve({ error: pulse.error }),
    part(() => readPulseActivity(league, port)),
    commissioner ? part(async () => {
      const r = await port({ command: 'commish.jobs.read', mode: 'read', arguments: { league } });
      const jobs = parseJobs(r.data).jobs.filter(job => job.state !== 'unbuilt');
      return { open: jobs.length, now: jobs.filter(job => job.state === 'live').length };
    }) : Promise.resolve(undefined),
    part(async () => {
      const [read, assets] = await Promise.all([readPlayerStatus(league, port),
        port({ command: 'native.assets.read', mode: 'read', arguments: { league } })]);
      const names = new Map(parseHostedRoster(assets.data).players.map(p => [p.id, p.name]));
      return { read, names };
    }),
  ]);
  return { pulse, season, activity, status, ...(jobs ? { jobs } : {}) };
}

/** readTeamRoster lists one franchise's players from the existing native.assets.read. */
export async function readTeamRoster(league: string, franchise: string,
  port: HostedPort = hostedWorkflow): Promise<LeagueRosterPlayer[]> {
  const r = parseHostedRoster((await port({ command: 'native.assets.read', mode: 'read',
    arguments: { league } })).data);
  const positions = new Map(r.identities.map(i => [i.player, i.position ?? '']));
  return r.players.filter(p => p.franchise === franchise)
    .map(p => ({ id: p.id, name: p.name, position: positions.get(p.id) ?? '' }));
}

type Props = { league?: LeagueChoice; port?: HostedPort;
  read?: (league: string, mine: string, commissioner: boolean) => Promise<LeagueReading>;
  readWeek?: (league: string, season: number, week: number) => Promise<PulseResults>;
  readRoster?: (league: string, franchise: string) => Promise<LeagueRosterPlayer[]>;
  onControlRoom?: () => void; onTrade?: () => void; onWatch?: () => void;
  /** onOpenJobs reports the commissioner's open job count, for the League tab badge. */
  onOpenJobs?: (count: number) => void };

export function LeagueTab({ league, port = hostedWorkflow, read, readWeek, readRoster, onControlRoom,
  onTrade, onWatch, onOpenJobs }: Props) {
  const commissioner = league?.role === 'Commissioner' || league?.role === 'Co-commissioner';
  const mine = league?.franchise ?? '';
  const [reading, setReading] = useState<LeagueReading>();
  const [view, setView] = useState<LeagueView>('st');
  const [team, setTeam] = useState<PulseStandingRow>();
  const [reporting, setReporting] = useState(false);
  useEffect(() => {
    if (!league) return;
    let live = true;
    setReading(undefined);
    void (read ?? ((l, m, c) => readLeague(l, m, c, port)))(league.id, mine, commissioner)
      .then(r => { if (live) setReading(r); })
      .catch(cause => { if (live) setReading({ pulse: { error: message(cause) }, season: {},
        activity: {} }); });
    return () => { live = false; };
  }, [league, mine, commissioner, port, read]);
  const openJobs = reading?.jobs?.value?.open ?? 0;
  useEffect(() => { onOpenJobs?.(openJobs); }, [openJobs, onOpenJobs]);
  if (!league) return <p className="lg5-empty" role="status">No league selected.</p>;
  if (!reading) return <p className="lg5-empty" role="status">Reading the league…</p>;
  const pulse = reading.pulse.value;
  const jobs = reading.jobs;
  return <div className="lg5">
    {commissioner && <HQAct className={`lg5-cr${jobs?.value?.now ? ' hot' : ''}`}
      onAction={onControlRoom} label="Control Room">
      <span><b>Control Room</b><em>{jobs?.value ? jobs.value.open
        ? `${jobs.value.open} open job${jobs.value.open > 1 ? 's' : ''}`
          + (jobs.value.now ? ` · ${jobs.value.now} needs you now` : '')
        : 'Nothing waiting on you' : jobs?.error ?? NOT_YET}</em></span>
      <i className={`n${jobs?.value?.open ? '' : ' z'}`}>{jobs?.value ? jobs.value.open || '✓' : '–'}</i>
    </HQAct>}
    <div className="lg5-seg" role="tablist" aria-label="League views">
      {([['st', 'Table'], ['mu', 'Scores'], ['sc', 'Schedule'], ['ac', 'Activity']] as const)
        .map(([id, label]) => <HQAct key={id} role="tab" selected={view === id}
          className={view === id ? 'on' : ''} onAction={() => setView(id)}>{label}</HQAct>)}
    </div>
    {!pulse ? <p className="lg5-empty" role="alert">{reading.pulse.error || NOT_YET}</p>
      : view === 'st' ? <Table pulse={pulse} mine={mine} onTeam={setTeam} />
        : view === 'mu' ? <Scores pulse={pulse} mine={mine} season={reading.season.value} onWatch={onWatch}
          readWeek={readWeek ?? ((l, s, w) => readPulseWeekResults(l, s, w, port))} />
          : view === 'sc' ? <Schedule pulse={pulse} mine={mine} season={reading.season} />
            : <Activity activity={reading.activity} status={reading.status} />}
    <HQAct className="lg5-key lg5-big lg5-report" onAction={() => setReporting(true)}>
      Report a problem</HQAct>
    {reporting && <ProblemReport league={league.id} route={{ node: 'pulse', workspace: 'league' }}
      port={port} onClose={() => setReporting(false)} />}
    {team && pulse && <TeamSheet team={team} mine={mine} commissioner={commissioner} league={league.id}
      status={reading.status?.value?.read}
      readRoster={readRoster ?? ((l, f) => readTeamRoster(l, f, port))} onClose={() => setTeam(undefined)}
      onTrade={onTrade && (() => { setTeam(undefined); onTrade(); })}
      onSeat={onControlRoom && (() => { setTeam(undefined); onControlRoom(); })} />}
  </div>;
}
export default LeagueTab;

/** The playoff line comes from the published bracket: the size of the overall field, or the
 *  per-conference count. No league size is guessed. */
function cutOf(pulse: NativePulse, conference: boolean): number | undefined {
  const b = pulse.bracket;
  if (!b) return undefined;
  if (conference) return b.perConference || undefined;
  if (b.fields.length === 1) return b.fields[0].seeds.length || undefined;
  return undefined;
}

/** streakClass colours a W streak green and an L streak red, as the mock does. */
export const streakClass = (streak?: string) => streak?.startsWith('W') ? 'w'
  : streak?.startsWith('L') ? 'l' : undefined;

function Table({ pulse, mine, onTeam }: { pulse: NativePulse; mine: string;
  onTeam: (row: PulseStandingRow) => void }) {
  const rows = useMemo(() => pulse.standings?.rows ?? [], [pulse.standings]);
  const conferences = [...new Set(rows.map(r => r.conference).filter(Boolean))];
  const divisions = [...new Set(rows.map(r => r.division).filter(Boolean))];
  const [split, setSplit] = useState('all');
  if (!pulse.standings) return <p className="lg5-empty" role="alert">{pulse.standingsError || NOT_YET}</p>;
  if (!rows.length) return <p className="lg5-empty" role="status">
    {pulse.standings.words || 'No standings yet.'}</p>;
  const grid = (list: PulseStandingRow[], cut?: number) => <section className="lg5-grp">
    <table className="st5"><thead><tr><th>#</th><th>Team</th><th>W-L</th><th>PF</th><th>STRK</th></tr></thead>
      <tbody>{list.map((r, i) => <tr key={r.franchise}
        className={`${r.franchise === mine ? 'me' : ''}${cut && i === cut ? ' cut' : ''}`}>
        <td>{i + 1}</td>
        <td><HQAct className="st5-team" label={`${r.name}: owner and record`} onAction={() => onTeam(r)}>
          {r.name}</HQAct></td>
        <td>{r.ties ? `${r.wins}-${r.losses}-${r.ties}` : `${r.wins}-${r.losses}`}</td>
        <td>{r.pointsFor.toFixed(0)}</td>
        <td className={streakClass(r.streak)}>{r.streak || '–'}</td></tr>)}</tbody></table>
  </section>;
  const chips = conferences.length > 1 || divisions.length > 1 ? <div className="lg5-chips">
    {[['all', 'Overall'], ...conferences.map(c => [c, `${c} conf.`]),
      ...(divisions.length > 1 ? [['div', 'Divisions']] : [])].map(([id, label]) =>
      <HQAct key={id} className={split === id ? 'on' : ''} selected={split === id}
        onAction={() => setSplit(id)}>{label}</HQAct>)}
  </div> : null;
  if (split === 'div') return <>{chips}
    {divisions.map(d => {
      const list = rows.filter(r => r.division === d);
      const title = [list[0]?.conference, d].filter(Boolean).join(' · ');
      return <div key={d}><div className="lg5-lbl">{title}</div>
        {grid(list)}</div>;
    })}
    <p className="lg5-note">{pulse.bracket?.words || NOT_YET}</p></>;
  const conference = split !== 'all';
  const list = conference ? rows.filter(r => r.conference === split) : rows;
  const cut = cutOf(pulse, conference);
  return <>{chips}{grid(list, cut)}
    <p className="lg5-note">{cut ? conference ? `Top ${cut} in each conference make the playoffs.`
      : `Top ${cut} make the playoffs (dashed line).` : pulse.bracket?.words || `Playoff line: ${NOT_YET}.`}
    {' '}Tap a team for its roster.</p></>;
}

function Scores({ pulse, mine, season, readWeek, onWatch }: { pulse: NativePulse; mine: string;
  season?: PulseSeason; onWatch?: () => void;
  readWeek: (league: string, season: number, week: number) => Promise<PulseResults> }) {
  const current = pulse.week;
  const last = Math.max(pulse.lastWeek, season?.schedule.weeks.reduce((h, w) => Math.max(h, w.week), 0) ?? 0);
  const [week, setWeek] = useState(current ?? 1);
  const [results, setResults] = useState<Part<PulseResults>>();
  const upcoming = current !== undefined && week > current;
  useEffect(() => {
    if (upcoming) return;
    let live = true;
    setResults(undefined);
    readWeek(pulse.league, pulse.season, week).then(value => { if (live) setResults({ value }); },
      cause => { if (live) setResults({ error: message(cause) }); });
    return () => { live = false; };
  }, [pulse.league, pulse.season, week, upcoming, readWeek]);
  if (current === undefined) return <p className="lg5-empty" role="status">{pulse.weekNote || NOT_YET}</p>;
  const names = new Map(pulse.seats.map(s => [s.franchise, s.name]));
  const r = results?.value;
  const state = upcoming ? 'UPCOMING' : r?.published ? 'FINAL' : week === current ? 'LIVE' : 'FINAL';
  const mineFirst = <T extends { home: string; away: string }>(list: T[]) => [...list].sort((x, y) =>
    Number(y.home === mine || y.away === mine) - Number(x.home === mine || x.away === mine));
  let body;
  if (upcoming) {
    const games = season?.schedule.weeks.find(w => w.week === week)?.games;
    body = !games?.length ? <p className="lg5-empty" role="status">{NOT_YET}</p>
      : mineFirst(games).map(g => <div key={g.home + g.away}
        className={`lg5-mu${g.home === mine || g.away === mine ? ' me' : ''}`}>
        <span>{names.get(g.home) ?? g.home}</span><span>{names.get(g.away) ?? g.away}</span>
        <em>{g.home === mine || g.away === mine ? 'YOUR MATCHUP · ' : ''}WEEK {week}</em></div>);
  } else if (!results) body = <p className="lg5-empty" role="status">Reading week {week}…</p>;
  else if (!r) body = <p className="lg5-empty" role="alert">{results.error}</p>;
  else if (!r.games.length) body = <p className="lg5-empty" role="status">{r.words || 'No results yet.'}</p>;
  else body = mineFirst(r.games).map(g => {
    const me = g.home === mine || g.away === mine;
    const a = g.homePoints ?? g.homeProvisional, b = g.awayPoints ?? g.awayProvisional;
    const lead = (a ?? 0) >= (b ?? 0);
    const score = (n?: number) => n === undefined ? '–' : n.toFixed(1);
    const content = <>
      <span className={lead ? 'lead' : 'trail'}>{g.homeName}<b>{score(a)}</b></span>
      <span className={lead ? 'trail' : 'lead'}>{g.awayName}<b>{score(b)}</b></span>
      <em>{me ? 'YOUR MATCHUP · ' : ''}
        {state === 'LIVE' && onWatch ? 'LIVE · TAP TO WATCH' : state}</em></>;
    return state === 'LIVE' && onWatch ? <HQAct key={g.home} className={`lg5-mu${me ? ' me' : ''}`}
      label={`${g.homeName} v ${g.awayName}: watch live`} onAction={onWatch}>{content}</HQAct>
      : <div key={g.home} className={`lg5-mu${me ? ' me' : ''}`}>{content}</div>;
  });
  return <>
    <div className="lg5-wk">
      <HQAct className="lg5-key" label="Previous week" disabled={week <= 1}
        onAction={() => setWeek(w => Math.max(1, w - 1))}>‹</HQAct>
      <b>WEEK {week} · {state}</b>
      <HQAct className="lg5-key" label="Next week" disabled={week >= last}
        onAction={() => setWeek(w => Math.min(last, w + 1))}>›</HQAct>
    </div>
    <section className="lg5-grp">{body}</section></>;
}

function Schedule({ pulse, mine, season }: { pulse: NativePulse; mine: string; season: Part<PulseSeason> }) {
  const s = season.value;
  if (!s) return <p className="lg5-empty" role="alert">{season.error || pulse.scheduleError || NOT_YET}</p>;
  const names = new Map(pulse.seats.map(seat => [seat.franchise, seat.name]));
  const weeks = s.schedule.weeks.filter(w => w.games.some(g => g.home === mine || g.away === mine));
  if (!mine || !weeks.length) return <p className="lg5-empty" role="status">
    {pulse.scheduleNote || 'No published games for your team.'}</p>;
  const all = s.schedule.weeks.map(w => w.week);
  const low = Math.min(...all), high = Math.max(...all);
  return <><p className="lg5-note">{names.get(mine) ?? mine} · regular season, weeks {low}–{high}</p>
    <section className="lg5-grp">{weeks.map(w => {
      const g = w.games.find(x => x.home === mine || x.away === mine)!;
      const opp = g.home === mine ? g.away : g.home;
      const played = s.played[w.week]?.games.find(x => x.home === g.home && x.away === g.away);
      const now = w.week === pulse.week;
      const my = played && (played.home === mine ? played.homePoints : played.awayPoints);
      const their = played && (played.home === mine ? played.awayPoints : played.homePoints);
      const final = played?.winner !== undefined && my !== undefined && their !== undefined;
      const won = final && played!.winner === mine, tie = final && my === their;
      const verb = final && !now ? tie ? 'tied' : won ? 'beat' : 'lost to' : 'vs';
      const kickoff = pulse.fixtures.find(f => f.week === w.week)?.kickoff;
      return <div key={w.week} className={`sc5${now ? ' now' : ''}`}><i>WK {w.week}</i>
        <span>{verb} {names.get(opp) ?? opp}</span>
        {final && !now ? <b className={tie ? '' : won ? 'w' : 'l'}>
          {tie ? 'T' : won ? 'W' : 'L'} {my!.toFixed(0)}–{their!.toFixed(0)}</b>
          : now ? <b className="lv">NOW</b> : <b>{kickoff ?? ''}</b>}</div>;
    })}</section></>;
}

function Activity({ activity, status }: { activity: Part<PulseActivity>; status?: Part<LeagueStatus> }) {
  const [filter, setFilter] = useState<'all' | 'lg' | 'pl'>('all');
  const events = activity.value?.events ?? [];
  return <>
    <div className="lg5-chips">{([['all', 'All'], ['lg', 'League'], ['pl', 'Player status']] as const)
      .map(([id, label]) => <HQAct key={id} className={filter === id ? 'on' : ''} selected={filter === id}
        onAction={() => setFilter(id)}>{label}</HQAct>)}</div>
    <section className="lg5-grp">
      {filter !== 'pl' && (activity.error ? <p className="lg5-empty" role="alert">{activity.error}</p>
        : !events.length ? <p className="lg5-empty" role="status">No league activity yet.</p>
          : events.map(e => <div key={e.id} className="ac"><em>{when(e.when)} · {e.kind.toUpperCase()}</em>
            <p>{e.summary}</p></div>))}
      {filter !== 'lg' && <PlayerStatusRows status={status} />}
    </section></>;
}

function PlayerStatusRows({ status }: { status?: Part<LeagueStatus> }) {
  const s = status?.value;
  if (!s) return <div className="ac"><em>STATUS</em><p>Player status: {status?.error || NOT_YET}</p></div>;
  const rows = [...s.read.facts.values()].filter(f => s.names.has(f.player));
  const note = statusNote(s.read);
  return <>
    {note && <div className="ac"><em>STATUS</em><p>{note}</p></div>}
    {s.read.state !== 'unavailable' && !rows.length && <div className="ac"><em>STATUS</em>
      <p>No injury designations for rostered players.</p></div>}
    {rows.map(f => <div key={f.player} className="ac"><em>STATUS</em>
      <p>{s.names.get(f.player)} · {statusWords(f)}</p></div>)}
  </>;
}

const when = (stamp: string) => new Date(stamp).toLocaleString(undefined, { weekday: 'short',
  hour: 'numeric', minute: '2-digit' }).toUpperCase();

function TeamSheet({ team, mine, commissioner, league, status, readRoster, onTrade, onSeat, onClose }: {
  team: PulseStandingRow; mine: string; commissioner: boolean; league: string; status?: PlayerStatus;
  readRoster: (league: string, franchise: string) => Promise<LeagueRosterPlayer[]>;
  onTrade?: () => void; onSeat?: () => void; onClose: () => void }) {
  const [roster, setRoster] = useState<Part<LeagueRosterPlayer[]>>();
  useEffect(() => {
    let live = true;
    readRoster(league, team.franchise).then(value => { if (live) setRoster({ value }); },
      cause => { if (live) setRoster({ error: message(cause) }); });
    return () => { live = false; };
  }, [league, team.franchise, readRoster]);
  const own = team.franchise === mine;
  return <PhoneSheet title={team.name} id="league-team-sheet" onClose={onClose}>
    <div className="lg5-sheet">
      <p className="lg5-note">{team.record} · {team.pointsFor.toFixed(0)} points for
        {' · streak '}{team.streak || NOT_YET}
        {[team.conference, team.division].filter(Boolean).map(part => ` · ${part}`).join('')}
      </p>
      <p className="lg5-note">Owner: {NOT_YET}</p>
      <section className="lg5-grp" aria-label="Roster">
        {!roster ? <p className="lg5-empty" role="status">Reading the roster…</p>
          : roster.error ? <p className="lg5-empty" role="alert">{roster.error}</p>
            : !roster.value!.length ? <p className="lg5-empty" role="status">No players on this roster.</p>
              : roster.value!.map(p => <div key={p.id} className="lg5-pk"><span><b>{p.name}</b>
                <em>{p.position || '–'}{status?.facts.get(p.id)
                  ? ` · ${statusWords(status.facts.get(p.id)!)}` : ''}</em></span>
                <b title={`Projection: ${NOT_YET}`}>–</b></div>)}
      </section>
      {!own && onTrade && <HQAct className="lg5-key lg5-big" onAction={onTrade}>Propose a trade</HQAct>}
      {!own && commissioner && onSeat && <HQAct className="lg5-key lg5-big" onAction={onSeat}>
        Manage this seat (Control Room)</HQAct>}
    </div>
  </PhoneSheet>;
}
