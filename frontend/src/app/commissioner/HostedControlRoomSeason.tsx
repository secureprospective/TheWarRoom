import { useEffect, useMemo, useState } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { createReadQuery } from '../data/query';
import { parsePulseBracket, parsePulseDirectory } from '../data/nativePulse';
import { array, integer, object, text } from '../data/parseValues';
import { hostedCommandAvailable } from '../commands/native';
import { HQAct } from '../shell/HQAct';
import { HostedBackupNow } from './HostedBackupNow';
import type { HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';
import { actTable, type ActDefinition } from './actTable';

const seasonActions: readonly [string, string][] = [
  ['schedule.generate', 'Generate schedule'], ['schedule.edit', 'Edit a game'],
  ['schedule.publish', 'Publish schedule'], ['bracket.build', 'Build playoff bracket'],
  ['bracket.edit', 'Edit playoff matchup'], ['bracket.seed', 'Set playoff seed'],
  ['bracket.advance', 'Advance playoff team'], ['title.set', 'Crown champion'],
  ['season.rollover', 'Roll over season'],
];

function createSeasonQuery(league: string, port: HostedPort) {
  return createReadQuery('commissioner.TargetCommissionerSchedule', {
    initial: () => ({} as { season?: number; schedule?: { draft: number; published: number; games: number };
      bracket?: { words: string };
      windows?: { id: string; opens: string[]; start: string; live: boolean }[] }),
    read: async (scope: string) => {
      const directoryResponse = await port({ command: 'native.directory.read', mode: 'read',
        arguments: { league: scope } });
      const directory = parsePulseDirectory(directoryResponse.data, league);
      const [scheduleResponse, bracketResponse, calendarResponse] = await Promise.all([
        port({ command: 'commish.schedule.read', mode: 'read', arguments: { league: scope } }),
        port({ command: 'bracket.read', mode: 'read', arguments: {
          league: scope, season: directory.season,
        } }),
        port({ command: 'calendar.read', mode: 'read', arguments: { league: scope, season: directory.season,
          franchise: directory.seats[0]?.franchise ?? '', action: 'lineup',
          localZone: Intl.DateTimeFormat().resolvedOptions().timeZone } }),
      ]);
      const schedule = object(scheduleResponse.data, 'Schedule',
        ['league', 'revision', 'draft', 'published']);
      if (schedule.league !== league
        || integer(schedule.revision, 'Schedule revision') !== directory.revision)
        throw new Error('Schedule belongs to another league revision.');
      const countWeeks = (value: unknown, label: string) => array(value, label, (row, path) => {
        const week = object(row, path, ['week', 'games', 'bye']);
        integer(week.week, `${path}.week`, 1);
        return array(week.games, `${path}.games`, (game, gamePath) => {
          const parsed = object(game, gamePath, ['id', 'home', 'away']);
          return { home: text(parsed.home, gamePath), away: text(parsed.away, gamePath) };
        });
      });
      const draft = countWeeks(schedule.draft, 'Draft weeks');
      const published = countWeeks(schedule.published, 'Published weeks');
      const bracket = parsePulseBracket(bracketResponse.data, league, directory.season);
      if (bracket.revision !== directory.revision)
        throw new Error('Playoffs belong to another league revision.');
      const calendar = object(calendarResponse.data, 'Calendar', ['league', 'revision', 'clock', 'decision']);
      if (calendar.league !== league
        || integer(calendar.revision, 'Calendar revision') !== directory.revision)
        throw new Error('Calendar belongs to another league revision.');
      const clock = object(calendar.clock, 'Action calendar', ['windows', 'exceptions', 'tradeDeadline',
        'secondsToTradeDeadline']);
      const windows = array(clock.windows, 'Calendar windows', (row, path) => {
        const actionWindow = object(row, path, ['id', 'opens', 'start', 'close', 'live', 'secondsToClose']);
        if (typeof actionWindow.live !== 'boolean') throw new Error(`${path}.live: expected boolean`);
        return { id: text(actionWindow.id, `${path}.id`),
          opens: array(actionWindow.opens, `${path}.opens`, text),
          start: text(actionWindow.start, `${path}.start`), live: actionWindow.live };
      });
      return { season: directory.season, schedule: { draft: draft.length, published: published.length,
        games: published.reduce((total, games) => total + games.length, 0) },
        bracket: { words: bracket.words }, windows };
    },
    accept: reading => reading,
  });
}

export function HostedControlRoomSeason({ league, port = hostedWorkflow, onTool }: {
  league: string; port?: HostedPort; onTool(definition: ActDefinition): void;
}) {
  const query = useMemo(() => createSeasonQuery(league, port), [league, port]);
  const state = query.use();
  const [backingUpForRollover, setBackingUpForRollover] = useState(false);
  useEffect(() => {
    void query.load([league]);
    return () => query.dispose();
  }, [league, query]);
  const available = seasonActions.filter(([command]) => command !== 'season.rollover'
    && hostedCommandAvailable(command));
  const open = (command: string) => {
    const definition = actTable.find(action => action.command === command);
    if (definition) onTool(definition);
  };
  return <div className="cr-col hosted-season" aria-label="Season">
    {state.error && <FailureAlert
      message={state.error}
      code={state.errorCode}
      league={league}
      screen="Control Room · Season"
      route="control/season"
      port={port}
    />}
    {!state.schedule && !state.error && <p role="status">Reading season from the league server…</p>}
    {state.schedule && <>
      <section className="cr-panel" aria-label="League calendar">
        <h3>League calendar</h3>
        {state.windows?.length ? <ul>{state.windows.map(actionWindow => <li key={actionWindow.id}>
          {actionWindow.opens.join(', ')} · {actionWindow.start}
          {actionWindow.live ? ' · Live' : ''}
          {hostedCommandAvailable('calendar.window.move') && <HQAct
            onAction={() => open('calendar.window.move')}>Move window</HQAct>}
        </li>)}</ul> : <p>No action windows in the league calendar.</p>}
      </section>
      <section className="cr-panel" aria-label="Schedule">
        <h3>Schedule · {state.season}</h3>
        <p>{state.schedule.published} published weeks · {state.schedule.games} games
          · {state.schedule.draft} draft weeks</p>
        <div>{available.filter(([command]) => command.startsWith('schedule.')).map(([command, label]) =>
          <HQAct key={command} onAction={() => open(command)}>{label}</HQAct>)}</div>
      </section>
      <section className="cr-panel" aria-label="Playoffs and rollover">
        <h3>Playoffs & rollover</h3>
        <p>{state.bracket?.words}</p>
        <div>{available.filter(([command]) => command.startsWith('bracket.') || command === 'title.set')
          .map(([command, label]) => (
            <HQAct key={command} onAction={() => open(command)}>{label}</HQAct>
          ))}</div>
      </section>
      <section className="cr-panel" aria-label="Season rollover">
        <h3>Rollover</h3>
        <p>Rollover starts with a fresh verified backup. After it finishes, review the rollover plan
          and confirm that change separately.</p>
        {hostedCommandAvailable('season.rollover') && <HQAct disabled={backingUpForRollover}
          onAction={() => setBackingUpForRollover(true)}>Roll over season</HQAct>}
        {backingUpForRollover && <>
          <HQAct onAction={() => setBackingUpForRollover(false)}>Cancel rollover</HQAct>
          <HostedBackupNow league={league} port={port} autoPreview onDone={() => {
            setBackingUpForRollover(false); open('season.rollover');
          }} />
        </>}
      </section>
      <section className="cr-panel" aria-label="Divisions">
        <h3>Divisions</h3>
        <p>This part of the Control Room is not on the server app yet.</p>
      </section>
    </>}
  </div>;
}
