import { useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import { hostedWorkflow } from '../commissioner/api';
import { pointsLabel, type PulseResults } from '../data/nativePulse';
import { readNativePulse, readPulseWeekResults, usePulseRead, type NativePulse, type PulseScope }
  from './pulseReads';
import { scoreOf } from './pulseModel';
import { PulseIdle, PulseNow } from './PulseNow';
import { PulseRace } from './PulseRace';
import './pulse.css';

export type PulseTab = 'now' | 'race';

export default function NativeLeaguePulse(scope: PulseScope & { tab?: PulseTab }) {
  const port = scope.port ?? hostedWorkflow;
  const tab: PulseTab = scope.tab ?? 'now';
  const [chosen, chooseWeek] = useState<number>();
  const league = usePulseRead<NativePulse>(JSON.stringify(['pulse', scope.league, scope.mine]),
    publish => readNativePulse(scope.league, scope.mine, port, publish));
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
          week={week} onWeek={chooseWeek} />
        {!reading ? <section className="ws cols on">
          <PulseIdle words={league.error ? `League Pulse is unavailable · ${league.error}`
            : `Reading ${scope.league}…`} /></section>
        : tab === 'race' ? <PulseRace reading={reading} mine={scope.mine} />
        : <PulseNow reading={reading} mine={scope.mine} week={week} results={results}
            league={scope.league} port={port} />}
        <div className="tray">
          <span className="msg">{reading?.standings?.words ?? reading?.weekNote ?? ''}</span>
          <span className="sp" />
          <span className="src">reads only · nothing here changes the league</span>
        </div>
      </main>
    </div>
  </div>;
  return body;
}

function Strip({ reading, results, pending, error, mine, week, onWeek }: {
  reading?: NativePulse; results?: PulseResults; pending: boolean; error?: string; mine: string;
  week?: number; onWeek: (week: number) => void;
}) {
  const last = reading?.lastWeek ?? 0;
  const decided = results?.games.filter(g => g.status !== 'No result yet').length ?? 0;
  const waiting = (results?.games.length ?? 0) - decided;
  const mineScore = scoreOf(results, mine);
  const game = results?.games.find(g => g.home === mine || g.away === mine);
  const opponent = game ? (game.home === mine ? game.away : game.home) : undefined;
  const theirs = opponent === undefined ? undefined : scoreOf(results, opponent);
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
        : <>
          <span><b>{decided}</b> with a result</span>
          <span><b>{waiting}</b> still to come</span>
          <span className="src">NFL game clocks are not held yet</span>
        </>}
    </div>
    <div className="fresh">
      scores: <b>the league ledger</b><br />
      feed policy: ESPN live · Sleeper gaps · nflverse final · commissioner overrides<br />
      revision {reading?.revision ?? '—'} · {results?.words ?? 'no week read'}
    </div>
    <div className="mybox">
      <span className="t me">YOU</span>
      <span className="s">{pointsLabel(mineScore?.points)}</span>
      <span className="wpm" title="win probability is not held yet" />
      <span className="s">{pointsLabel(theirs?.points)}</span>
      <span className="t th">{theirs ? theirs.name.slice(0, 3).toUpperCase() : '—'}</span>
    </div>
  </section>;
}
