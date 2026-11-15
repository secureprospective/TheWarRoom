import { useEffect, useMemo, useState } from 'react';
import { onHostedTopics } from '../data/hostedReads';
import { HQAct as PulseAct } from './HQAct';
import { hostedWorkflow } from '../commissioner/api';
import { myChance, percent, pointsLabel, type PulseResults } from '../data/nativePulse';
import { readNativePulse, readPulseWeekResults, usePulseRead, PulseChanges,
  type NativePulse, type PulseScope }
  from './pulseReads';
import { scoreOf } from './pulseModel';
import { PulseIdle, PulseNow, type NowDrawer } from './PulseNow';
import { PulseRace, type RaceDrawer } from './PulseRace';
import { PulseInspector } from './PulseInspector';
import { ScheduleDrawer } from './PulseSchedule';
import { usePulseLive } from './hqLive';
import type { LiveView } from '../now/liveContract';
import './pulse.css';

export type PulseTab = 'now' | 'race';
export const PULSE_FOLLOW_MS = 3000;

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
    const listeners = new Set<() => void>();
    return { onChange: (load: () => void) => {
      listeners.add(load);
      return () => { listeners.delete(load); };
    }, refresh: () => { for (const load of listeners) load(); } };
  }, []);
  // A published change for this league (scores, lineups, the ledger, the clock, values) rereads the open
  // Pulse once, PULSE_FOLLOW_MS after the first topic of a burst; Refresh stays for an impatient owner.
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const off = onHostedTopics(topics => {
      if (timer || !topics.some(topic => topic.league === scope.league)) return;
      timer = setTimeout(() => { timer = undefined; changes.refresh(); }, PULSE_FOLLOW_MS);
    });
    return () => { off(); clearTimeout(timer); };
  }, [changes, scope.league]);
  return <PulseChanges.Provider value={changes}>
    <PulseBody {...scope} refresh={changes.refresh} live={{ view: undefined, error: '', follow: () => {} }} />
  </PulseChanges.Provider>;
}

function PulseBody(scope: PulseScope & { tab?: PulseTab; live: ReturnType<typeof usePulseLive>;
  refresh?: () => void }) {
  const port = scope.port ?? hostedWorkflow;
  const tab: PulseTab = scope.tab ?? 'now';
  const [chosen, chooseWeek] = useState<number>();
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
    publish => readNativePulse(scope.league, scope.mine, port, publish, scope.serverTime));
  const live = scope.live;
  const reading = league.reading;
  const week = chosen ?? reading?.week;
  const results = usePulseRead<PulseResults>(
    JSON.stringify(['pulse-week', scope.league, reading?.season, week]),
    () => reading && week ? readPulseWeekResults(scope.league, reading.season, week, port)
      : Promise.reject(new Error('The league has not said which week it is playing.')));
  const modeClass = scope.serverTime ? ' hosted' : '';
  const body = <div className={`lp-v14${scope.embedded ? ' embedded' : ''}${modeClass}`}>
    <div className="shell">
      <main className="work">
        {scope.serverTime && <nav className="ws-tabs" aria-label="League Pulse">
          <PulseAct onAction={scope.back ?? (() => scope.leave())}>Back</PulseAct>
          <PulseAct role="tab" selected={tab === 'now'} onAction={() => scope.navigate?.('now')}>
            Now</PulseAct>
          <PulseAct role="tab" selected={tab === 'race'} onAction={() => scope.navigate?.('race')}>
            Race</PulseAct>
        </nav>}
        <Strip reading={reading} results={results.reading} pending={Boolean(league.pending)}
          error={league.error ?? (week ? results.error : undefined)} mine={scope.mine}
          week={week} onWeek={chooseWeek} live={live} hosted={scope.serverTime} />
        {!reading ? <section className="ws cols on">
          <PulseIdle words={league.error ? `League Pulse is unavailable · ${league.error}`
            : `Reading ${scope.league}…`} /></section>
        : tab === 'race' ? <PulseRace reading={reading} mine={scope.mine}
            drawer={drawer === 'tb' || drawer === 'bk' ? drawer : undefined}
              onClose={() => setDrawer(undefined)} />
        : <PulseNow reading={reading} mine={scope.mine} week={week} results={results}
            league={scope.league} port={port} live={live}
            viewer={scope.serverTime
              ? { mine: scope.mine, commissioner: scope.commissioner ?? false } : undefined}
            drawer={drawer === 'box' || drawer === 'corr' ? drawer : undefined}
            onClose={() => setDrawer(undefined)} onChanges={setChanges} />}
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
          <span className="msg">{reading?.standings?.words ?? reading?.weekNote ?? ''}</span>
          <span className="sp" />
          <span className="src">reads only · nothing here changes the league</span>
        </div>
      </main>
      {reading && <PulseInspector reading={reading} results={results.reading} franchise={scope.mine}
        mine={scope.mine} week={week} leave={scope.leave} />}
    </div>
  </div>;
  return body;
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
          <span className="src">{hosted ? 'Live NFL games are coming next.'
            : nflClockSummary(undefined, live.error)}</span>
        </>}
    </div>
    <div className="fresh">
      scores: <b>{scoreSources(results)}</b><br />
      {hosted ? 'Scores follow the league server as it publishes them'
        : <>revision {reading?.revision ?? '—'} · {results?.words ?? 'no week read'}</>}
    </div>
    <div className="mybox">
      <span className="t me">YOU</span>
      <span className="s">{pointsLabel(mineScore?.points ?? mineScore?.provisional)}</span>
      <span className="wpm" title={win === undefined ? 'win probability shows while the game is open'
        : `win probability ${percent(win)}`}>
          {win !== undefined && <i style={{ width: percent(win) }} />}</span>
      <span className="s">{pointsLabel(theirs?.points ?? theirs?.provisional)}</span>
      <span className="t th">{theirs ? theirs.name.slice(0, 3).toUpperCase() : '—'}</span>
    </div>
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
  return [...sources].sort().join(' · ') || 'source not reported';
}

export function nflClockSummary(view?: LiveView, error = ''): string {
  if (error) return `NFL scoreboard unavailable · ${error}`;
  if (!view) return 'Reading the NFL scoreboard…';
  const playing = view.games.filter(g => g.status === 'in');
  return playing.length ? playing.map(g => `${g.away.abbreviation}–${g.home.abbreviation} `
    + `${g.period > 4 ? 'OT' : `Q${g.period}`} ${g.clock}`).join(' · ')
    : 'No NFL game in progress';
}
