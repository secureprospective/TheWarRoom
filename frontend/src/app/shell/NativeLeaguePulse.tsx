import { useEffect, useMemo, useState } from 'react';
import { onHostedTopics } from '../data/hostedReads';
import { HQAct as PulseAct } from './HQAct';
import { hostedWorkflow } from '../commissioner/api';
import { myChance, percent, pointsLabel, type PulseResults } from '../data/nativePulse';
import { readNativePulse, readPulseOdds, readPulseWeekResults, usePulseRead, PulseChanges, PulseWaiting,
  readPulseActivity, type NativePulse, type PulseScope, type PulseTab }
  from './pulseReads';
import { scoreOf } from './pulseModel';
import { PulseIdle, PulseNow, type NowDrawer } from './PulseNow';
import { PulseRace, type RaceDrawer } from './PulseRace';
import { PulseInspector } from './PulseInspector';
import { PulseScheduleView, ScheduleDrawer } from './PulseSchedule';
import { usePulseLive } from './hqLive';
import { useHostedGames } from './hostedGames';
import type { LiveView } from '../now/liveContract';
import './pulse.css';
import { PulseNowV3, PulseTabs } from '../v3/PulseV3';

export type { PulseTab } from './pulseReads';
export const PULSE_FOLLOW_MS = 250;
export { useHostedGames };

export default function NativeLeaguePulse(scope: PulseScope & { tab?: PulseTab }) {
  if (!scope.serverTime) return <LocalPulse {...scope} />;
  return <HostedPulse {...scope} />;
}

function LocalPulse(scope: PulseScope & { tab?: PulseTab }) {
  const live = usePulseLive('');
  return <PulseBody {...scope} live={live} />;
}

function HostedPulse(scope: PulseScope & { tab?: PulseTab }) {
  const changes = useMemo(() => {
    const scores = new Set<() => void>();
    const league = new Set<() => void>();
    const odds = new Set<() => void>();
    const listen = (set: Set<() => void>) => (load: () => void) => {
      set.add(load);
      return () => { set.delete(load); };
    };
    const refresh = (onlyScores = false) => {
      for (const load of [...scores, ...(onlyScores ? [] : league)]) load();
    };
    return { onChange: listen(scores), onLeagueChange: listen(league), onOddsChange: listen(odds),
      refresh: () => refresh(), refreshScores: () => refresh(true),
      refreshOdds: () => { for (const load of odds) load(); } };
  }, []);
  // A published change for this league rereads the open Pulse once, PULSE_FOLLOW_MS after the first topic
  // of a burst. A burst of only scores topics (every few seconds while games are on) rereads the scores,
  // matchups and games, not the directory, week, standings or bracket. Odds follow only league topics:
  // neither live ticks, unrelated topics nor manual/error refreshes can change a result.
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    let onlyScores = true;
    let oddsChanged = false;
    const off = onHostedTopics(topics => {
      const mine = topics.filter(topic => topic.league === scope.league);
      if (mine.length === 0) return;
      onlyScores &&= mine.every(topic => topic.kind === 'scores');
      oddsChanged ||= mine.some(topic => topic.kind === 'league');
      if (timer) return;
      timer = setTimeout(() => {
        timer = undefined;
        if (onlyScores) changes.refreshScores(); else changes.refresh();
        if (oddsChanged) changes.refreshOdds();
        onlyScores = true;
        oddsChanged = false;
      }, PULSE_FOLLOW_MS);
    });
    return () => { off(); clearTimeout(timer); };
  }, [changes, scope.league]);
  return <PulseChanges.Provider value={changes}>
    <HostedBody key={`${scope.league}:${scope.mine}`} scope={scope} changes={changes} />
  </PulseChanges.Provider>;
}

/** HostedBody reads the server's NFL scoreboard alongside the league reads and rereads it on every
 *  Pulse refresh, so a scores topic moves the games list with the matchups. */
function HostedBody({ scope, changes }: { scope: PulseScope & { tab?: PulseTab };
  changes: { onChange: (load: () => void) => () => void; refresh: () => void } }) {
  const live = useHostedGames(scope.league, scope.port ?? hostedWorkflow, changes.onChange);
  return <PulseBody {...scope} refresh={changes.refresh} live={live} />;
}

function PulseBody(scope: PulseScope & { tab?: PulseTab;
  live: ReturnType<typeof usePulseLive> & { prefer?: (key: string) => void };
  refresh?: () => void }) {
  const port = scope.port ?? hostedWorkflow;
  const tab: PulseTab = scope.tab ?? 'now';
  const [chosen, chooseWeek] = useState<number>();
  const [inspectedFranchise, inspectFranchise] = useState(scope.mine);
  const [drawer, setDrawer] = useState<NowDrawer | 'sched' | RaceDrawer>();
  const [changes, setChanges] = useState(0);
  const pull = (key: NonNullable<typeof drawer>) => setDrawer(drawer === key ? undefined : key);
  useEffect(() => setDrawer(undefined), [tab]);
  useEffect(() => {
    if (!drawer) return;
    // Escape closes only the drawer: capture it before the shell's Escape closes the inspector too.
    const close = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      e.preventDefault(); e.stopPropagation(); setDrawer(undefined);
    };
    document.addEventListener('keydown', close, { capture: true });
    return () => document.removeEventListener('keydown', close, { capture: true });
  }, [drawer]);
  const league = usePulseRead<NativePulse>(JSON.stringify(['pulse', scope.league, scope.mine]),
    publish => readNativePulse(scope.league, scope.mine, port, publish, scope.serverTime, false), 'league');
  const season = league.reading?.season;
  const odds = usePulseRead(JSON.stringify(['pulse-odds', scope.league, season]),
    () => season ? readPulseOdds(scope.league, season, port)
      : Promise.reject(new PulseWaiting('The league has not said which season it is playing.')), 'odds');
  const live = scope.live;
  const reading = league.reading && { ...league.reading, odds: odds.reading,
    errors: [...league.reading.errors, ...(odds.error ? [odds.error] : [])] };
  const week = chosen ?? reading?.week;
  const results = usePulseRead<PulseResults>(
    JSON.stringify(['pulse-week', scope.league, reading?.season, week,
      week === undefined ? reading?.weekNote : '']),
    () => reading && week ? readPulseWeekResults(scope.league, reading.season, week, port)
      : Promise.reject(new PulseWaiting(reading?.weekNote ||
        'The league has not said which week it is playing.')));
  const errored = Boolean(league.error || (week && results.error) || live.error);
  useEffect(() => {
    if (!scope.serverTime || !scope.refresh || !errored) return;
    let timer: ReturnType<typeof setTimeout>;
    let attempt = 0;
    const retry = () => {
      const delay = [5000, 10000, 20000, 30000][Math.min(attempt++, 3)];
      timer = setTimeout(() => { scope.refresh?.(); retry(); }, delay);
    };
    retry();
    return () => clearTimeout(timer);
  }, [errored, scope.refresh, scope.serverTime]);
  if (scope.v3 && tab === 'now') return <>
    <PulseTabs tab={tab} navigate={scope.navigate} badge={scope.v3.badge} kind={scope.v3.kind} />
    <div className="wbody scr scr-pulse">{!reading
      ? <div className="st"><div className="eng">{league.error
        ? `League Pulse is unavailable · ${league.error}`
        : 'Reading League Pulse…'}</div></div>
      : <PulseNowV3 reading={reading} mine={scope.mine} week={week} results={results} league={scope.league}
        port={port} live={live} badge={scope.v3.badge} changes={changes}
        viewer={{ mine: scope.mine, commissioner: scope.commissioner ?? false }}
        pull={drawer === 'box' || drawer === 'sched' || drawer === 'corr' ? drawer : undefined}
        onPull={pull} />}</div>
  </>;
  const modeClass = scope.serverTime ? ' hosted' : '';
  const body = <div className={`lp-v14${scope.embedded ? ' embedded' : ''}${modeClass}`}>
    <div className="shell">
      <main className="work">
        {scope.serverTime && !scope.embedded && <nav className="ws-tabs" aria-label="League Pulse">
          <PulseAct onAction={scope.back ?? (() => scope.leave())}>Back</PulseAct>
          <PulseAct role="tab" selected={tab === 'now'} onAction={() => scope.navigate?.('now')}>
            Now</PulseAct>
          <PulseAct role="tab" selected={tab === 'race'} onAction={() => scope.navigate?.('race')}>
            Race</PulseAct>
        </nav>}
        {scope.serverTime && scope.embedded && <nav className="pulse-phone-tabs" role="tablist"
          aria-label="League overview">
          <PulseAct role="tab" selected={tab === 'race'} onAction={() => scope.navigate?.('race')}>
            Table</PulseAct>
          <PulseAct role="tab" selected={tab === 'schedule'}
            onAction={() => scope.navigate?.('schedule')}>Schedule</PulseAct>
          <PulseAct role="tab" selected={tab === 'activity'}
            onAction={() => scope.navigate?.('activity')}>Activity</PulseAct>
          <PulseAct role="tab" selected={tab === 'now'} onAction={() => scope.navigate?.('now')}>
            Live</PulseAct>
        </nav>}
        <Strip reading={reading} results={results.reading} pending={Boolean(league.pending)}
          error={league.error ?? (week ? results.error : undefined)} mine={scope.mine}
          week={week} onWeek={chooseWeek} live={live} hosted={scope.serverTime} />
        {!reading ? <section className="ws cols on">
          <PulseIdle words={league.error ? `League Pulse is unavailable · ${league.error}`
            : `Reading ${scope.league}…`} /></section>
        : tab === 'race' ? <PulseRace reading={reading} mine={scope.mine}
            onFranchise={scope.serverTime ? inspectFranchise : undefined}
            drawer={drawer === 'tb' || drawer === 'bk' ? drawer : undefined}
            onClose={() => setDrawer(undefined)} />
        : tab === 'schedule' ? scope.serverTime
          ? <PulseScheduleView reading={reading} mine={scope.mine} port={port} />
          : <section className="ws cols on">
            <PulseIdle words="The hosted league schedule is unavailable in this view." />
          </section>
        : tab === 'activity' ? scope.serverTime
          ? <PulseActivity league={scope.league} port={port} />
          : <section className="ws cols on">
            <PulseIdle words="League activity is unavailable in this view." />
          </section>
        : <PulseNow reading={reading} mine={scope.mine} week={week} results={results}
            league={scope.league} port={port} live={live}
            viewer={scope.serverTime
              ? { mine: scope.mine, commissioner: scope.commissioner ?? false } : undefined}
            drawer={drawer === 'box' || drawer === 'corr' ? drawer : undefined}
            onClose={() => setDrawer(undefined)} onChanges={setChanges} onPlayer={scope.onPlayer}
            onPlayers={scope.onPlayers} />}
        {reading && tab === 'now' && <ScheduleDrawer open={drawer === 'sched'} reading={reading}
          mine={scope.mine} port={port} onClose={() => setDrawer(undefined)} />}
        <div className="tray">
          {scope.refresh && <PulseAct onAction={scope.refresh}>Refresh</PulseAct>}
          {reading && tab === 'now' && <div className="pulls">
            <PulseAct className="pull" expanded={drawer === 'box'} onAction={() => pull('box')}>
              MATCHUP BOX ▴</PulseAct>
            <PulseAct className="pull" expanded={drawer === 'sched'} onAction={() => pull('sched')}>
              SCHEDULE ▴</PulseAct>
            <PulseAct className="pull" expanded={drawer === 'corr'} onAction={() => pull('corr')}>
              STAT CHANGES{changes ? <> <b>{changes}</b></> : null} ▴</PulseAct>
          </div>}
          {reading && tab === 'race' && <div className="pulls">
            <PulseAct className="pull" expanded={drawer === 'tb'} onAction={() => pull('tb')}>
              TIEBREAKERS ▴</PulseAct>
            <PulseAct className="pull" expanded={drawer === 'bk'} onAction={() => pull('bk')}>
              BRACKET ▴</PulseAct>
          </div>}
          <span className="msg">{reading?.standings?.words ?? reading?.weekNote ?? ''}
            {odds.error ? ' · Playoff odds: unavailable.' : ''}
            {scope.serverTime && errored ? ' · retrying' : ''}</span>
          <span className="sp" />
          <span className="src">reads only · nothing here changes the league</span>
        </div>
      </main>
      {reading && (scope.serverTime ? inspectedFranchise || scope.mine : scope.mine) &&
        <PulseInspector reading={reading} results={results.reading}
          franchise={scope.serverTime ? inspectedFranchise || scope.mine : scope.mine}
          mine={scope.mine} week={week} leave={scope.leave} />}
    </div>
  </div>;
  return body;
}

function PulseActivity({ league, port }: { league: string; port: NonNullable<PulseScope['port']> }) {
  const [filter, setFilter] = useState<'all' | 'league' | 'player-status'>('all');
  const activity = usePulseRead(JSON.stringify(['pulse-activity', league]),
    () => readPulseActivity(league, port), 'league');
  const events = activity.reading?.events ?? [];
  return <section className="ws cols on" id="ws-activity">
    <div className="pcol pulse-activity">
      <div className="ch"><b>LEAGUE ACTIVITY</b><span>Public events only</span></div>
      <div className="fbar pulse-activity-filters" role="tablist" aria-label="Activity filter">
        <PulseAct role="tab" selected={filter === 'all'} onAction={() => setFilter('all')}>All</PulseAct>
        <PulseAct role="tab" selected={filter === 'league'} onAction={() => setFilter('league')}>
          League</PulseAct>
        <PulseAct role="tab" selected={filter === 'player-status'}
          onAction={() => setFilter('player-status')}>Player status</PulseAct>
      </div>
      <div className="scroll pulse-activity-list" role="tabpanel">
        {filter === 'player-status'
          ? <PulseIdle words="Player status is unavailable · no member-safe status feed is published." />
          : !activity.reading ? <PulseIdle words={activity.error
            ? `League activity is unavailable · ${activity.error}`
            : 'Reading public league activity…'} />
          : !events.length ? <PulseIdle words="No public league activity has been recorded yet." />
          : <>
            <p className="pulse-activity-note">Only public league events are shown. Private bids and
              commissioner details are excluded.</p>
            {events.map(event => <article className="pulse-activity-event" key={event.id}>
              <div><b>{event.kind === 'notice' ? 'NOTICE' : event.kind === 'trade' ? 'TRADE' : 'ROSTER'}</b>
                <time dateTime={event.when}>{activityTime(event.when)}</time></div>
              <p>{event.summary}</p>
            </article>)}
          </>}
      </div>
    </div>
  </section>;
}

function activityTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? 'Time unavailable'
    : new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function Strip({ reading, results, pending, error, mine, week, onWeek, live, hosted }: {
  reading?: NativePulse; results?: PulseResults; pending: boolean; error?: string; mine: string;
  hosted?: boolean; week?: number; onWeek: (week: number) => void; live: ReturnType<typeof usePulseLive>;
}) {
  const last = reading?.lastWeek ?? 0;
  const decided = results?.games.filter(g => g.status !== 'No result yet').length ?? 0;
  const waiting = (results?.games.length ?? 0) - decided;
  const mineScore = scoreOf(results, mine);
  const game = results?.games.find(g => g.home === mine || g.away === mine);
  const opponent = game ? (game.home === mine ? game.away : game.home) : undefined;
  const theirs = opponent === undefined ? undefined : scoreOf(results, opponent);
  const chance = myChance(reading?.odds, mine);
  const win = chance && chance.week === week && !game?.winner ? chance.mine : undefined;
  return <section className="sline">
    <div className="wk">
      <PulseAct title="Previous week" disabled={!week || week <= 1}
        onAction={() => week && week > 1 ? onWeek(week - 1) : undefined}>◂</PulseAct>
      <b>{week ? `WEEK ${week}` : 'WEEK —'}</b>
      <PulseAct title="Next week" disabled={!week || !last || week >= last}
        onAction={() => week && last && week < last ? onWeek(week + 1) : undefined}>▸</PulseAct>
    </div>
    <div className="slate">
      {error ? <span>{error}</span>
        : !results ? <span>{pending ? 'Reading the week…' : 'The week is not read yet'}</span>
        : !results.published ? <span>{results.words}</span>
        : !waiting && decided ? <span>WEEK {week} · <b>FINAL</b> · all {decided} results locked</span>
        : week === reading?.week && live.view ? <NflSlate view={live.view} />
        : <>
          <span><b>{decided}</b> with a result</span>
          <span><b>{waiting}</b> still to come</span>
          <span className="src">{nflClockSummary(live.view, live.error)}</span>
        </>}
    </div>
    <div className="fresh">
      scores: <b>{scoreSources(results)}</b><br />
      {hosted ? 'Scores follow the league server as it publishes them'
        : <>revision {reading?.revision ?? '—'} · {results?.words ?? 'no week read'}</>}
    </div>
    {mine && <div className="mybox">
      <span className="t me">YOU</span>
      <span className="s">{pointsLabel(mineScore?.points ?? mineScore?.provisional)}</span>
      <span className="wpm" title={win === undefined ? 'win probability shows while the game is open'
        : `win probability ${percent(win)}`}>
          {win !== undefined && <i style={{ width: percent(win) }} />}</span>
      <span className="s">{pointsLabel(theirs?.points ?? theirs?.provisional)}</span>
      <span className="t th">{theirs ? theirs.name.slice(0, 3).toUpperCase() : '—'}</span>
    </div>}
  </section>;
}

/** NflSlate counts the week's NFL games the way the mock's strip does: live, final, to play. */
function NflSlate({ view }: { view: LiveView }) {
  const count = (status: string) => view.games.filter(g => g.status === status).length;
  return <>
    <span className="lv"><span className="dot blink" /><b>{count('in')}</b> NFL live</span>
    <span><b>{count('post')}</b> final</span>
    <span><b>{count('pre')}</b> to play</span>
  </>;
}

export function scoreSources(results?: PulseResults): string {
  const sources = new Set<string>();
  for (const score of results?.scores ?? []) for (const starter of score.starters) {
    if (starter.source) sources.add(starter.source);
    for (const owner of Object.values(starter.fieldSources ?? {})) sources.add(owner);
  }
  const evidence = results?.evidence;
  if (!sources.size && evidence?.source) sources.add(evidence.source);
  const label = [...sources].sort().join(' · ');
  if (!label) return 'source not reported';
  return label + (evidence?.stale ? ' · held' : '') + (evidence?.provisional ? ' · provisional' : '');
}

export function nflClockSummary(view?: LiveView, error = ''): string {
  if (error) return `NFL scoreboard unavailable · ${error}`;
  if (!view) return 'Reading the NFL scoreboard…';
  const playing = view.games.filter(g => g.status === 'in');
  return playing.length ? playing.map(g => `${g.away.abbreviation}–${g.home.abbreviation} `
    + `${g.period > 4 ? 'OT' : `Q${g.period}`} ${g.clock}`).join(' · ')
    : 'No NFL game in progress';
}
