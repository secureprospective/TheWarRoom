import { useEffect, useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import { hostedWorkflow } from '../commissioner/api';
import { myChance, percent, pointsLabel, type PulseResults } from '../data/nativePulse';
import { readNativePulse, readPulseWeekResults, usePulseRead, type NativePulse, type PulseScope }
  from './pulseReads';
import { scoreOf } from './pulseModel';
import { PulseIdle, PulseNow } from './PulseNow';
import { PulseRace } from './PulseRace';
import { PulseInspector } from './PulseInspector';
import { usePulseLive } from './hqLive';
import type { LiveView } from '../now/liveContract';
import './pulse.css';

export type PulseTab = 'now' | 'race';

export default function NativeLeaguePulse(scope: PulseScope & { tab?: PulseTab }) {
  const port = scope.port ?? hostedWorkflow;
  const tab: PulseTab = scope.tab ?? 'now';
  const [chosen, chooseWeek] = useState<number>();
  const [box, setBox] = useState(false);
  useEffect(() => {
    if (!box) return;
    const close = (e: KeyboardEvent) => { if (e.key === 'Escape') setBox(false); };
    window.addEventListener('keydown', close);
    return () => window.removeEventListener('keydown', close);
  }, [box]);
  const league = usePulseRead<NativePulse>(JSON.stringify(['pulse', scope.league, scope.mine]),
    publish => readNativePulse(scope.league, scope.mine, port, publish));
  const live = usePulseLive('');
  const reading = league.reading;
  const week = chosen ?? reading?.week;
  const results = usePulseRead<PulseResults>(
    JSON.stringify(['pulse-week', scope.league, reading?.season, week]),
    () => reading && week ? readPulseWeekResults(scope.league, reading.season, week, port)
      : Promise.reject(new Error('The league has not said which week it is playing.')));
  const body = <div className={`lp-v14${scope.embedded ? ' embedded' : ''}`}>
    <div className="shell">
      <main className="work">
        <Strip reading={reading} results={results.reading} pending={Boolean(league.pending)}
          error={league.error ?? (week ? results.error : undefined)} mine={scope.mine}
          week={week} onWeek={chooseWeek} live={live} />
        {!reading ? <section className="ws cols on">
          <PulseIdle words={league.error ? `League Pulse is unavailable · ${league.error}`
            : `Reading ${scope.league}…`} /></section>
        : tab === 'race' ? <PulseRace reading={reading} mine={scope.mine} />
        : <PulseNow reading={reading} mine={scope.mine} week={week} results={results}
            league={scope.league} port={port} live={live} box={box} onBox={setBox} />}
        <div className="tray">
          {reading && tab === 'now' && <div className="pulls">
            <PulseAct className="pull" expanded={box} onAction={() => setBox(!box)}>MATCHUP BOX ▴</PulseAct>
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

function Strip({ reading, results, pending, error, mine, week, onWeek, live }: {
  reading?: NativePulse; results?: PulseResults; pending: boolean; error?: string; mine: string;
  week?: number; onWeek: (week: number) => void; live: ReturnType<typeof usePulseLive>;
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
          <span className="src">{nflClockSummary(undefined, live.error)}</span>
        </>}
    </div>
    <div className="fresh">
      scores: <b>{scoreSources(results)}</b><br />
      revision {reading?.revision ?? '—'} · {results?.words ?? 'no week read'}
    </div>
    <div className="mybox">
      <span className="t me">YOU</span>
      <span className="s">{pointsLabel(mineScore?.points ?? mineScore?.provisional)}</span>
      <span className="wpm" title={win === undefined ? 'win probability shows while the game is open'
        : `win probability ${percent(win)}`}>{win !== undefined && <i style={{ width: percent(win) }} />}</span>
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
