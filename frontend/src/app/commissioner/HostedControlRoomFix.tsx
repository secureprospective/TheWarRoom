import { HQAct } from '../shell/HQAct';
import { hostedCommandAvailable } from '../commands/native';
import { actTable, type ActDefinition } from './actTable';
import { HostedCommissionerActTray } from './CommissionerActTray';
import type { ActValues } from './actBody';
import type { HostedFixReadings } from './hostedControlRoomData';
import { hostedControlRoomNotOnServer } from './hostedControlRoomData';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';

const fixCommands = [
  'commish.stat.correct', 'commish.score.player', 'commish.score.franchise',
  'commish.lineup.correct', 'commish.player.position', 'commish.standings.override',
  'commish.standings.record.adjust', 'commish.contract.edit', 'commish.correction.revert',
] as const;

export function HostedControlRoomFix({ league, room, port, readings, pending, error, definition, initial,
  onOpen }: {
  league: string; room: HostedLeagueRoom; port?: HostedPort; readings?: HostedFixReadings;
  pending: boolean; error: string; definition?: ActDefinition; initial?: ActValues;
  onOpen(definition: ActDefinition, initial?: ActValues): void;
}) {
  const definitions = fixCommands.map(command => actTable.find(row => row.command === command));
  const correctionExceptions = readings?.corrections?.items.filter(item => item.until || item.rule) ?? [];
  const calendarExceptions = readings?.calendarExceptions ?? [];
  const franchises = new Map((readings?.franchises ?? []).map(team => [team.franchise, team.name]));
  const exceptions = [
    ...correctionExceptions.map(item => ({ key: item.entry, what: item.what,
      franchise: item.franchise ?? item.scope?.franchise ?? '',
      ends: item.until ? new Date(item.until).toLocaleDateString() : 'until cleared',
      entry: item.entry })),
    ...calendarExceptions.filter(exception => !correctionExceptions.some(item => {
      const franchise = item.franchise ?? item.scope?.franchise ?? '';
      return item.rule === `calendar.${exception.action}` && franchise === exception.franchise &&
        item.when === exception.start;
    })).map(exception => ({ key: `${exception.action}:${exception.franchise}:${exception.start}`,
      what: `${exception.action} outside its window`, franchise: exception.franchise,
      ends: exception.permanent ? 'until cleared' : new Date(exception.until).toLocaleDateString(),
      entry: '' })),
  ];
  const reversible = new Set(readings?.ledger?.entries
    .filter(entry => entry.undo !== '' && !entry.replacedAt).map(entry => entry.id) ?? []);
  const recent = readings?.ledger?.entries
    .filter(entry => entry.undo !== '' && !entry.replacedAt).slice(0, 10) ?? [];
  const revert = actTable.find(row => row.command === 'commish.correction.revert');
  const exceptionFailure = readings?.correctionsError;
  const calendarFailure = readings?.calendarError;
  const ledgerFailure = readings?.ledgerError;
  const hasExceptionRead = readings?.corrections !== undefined || readings?.calendarExceptions !== undefined;
  const completeExceptionRead = readings?.corrections !== undefined &&
    readings.calendarExceptions !== undefined;

  return <div className="cr-col hosted-fix" data-testid="hosted-control-room-fix">
    {error && <p role="alert">{error}</p>}
    <div className="cr-fix">
      {definitions.map(act => act && <div key={act.command} className="hosted-fix-action"
        data-fix-command={act.command}>
        <HQAct disabled={!hostedCommandAvailable(act.command)} onAction={() => onOpen(act)}>
          <b>{act.label}</b>
          <span>{act.fields.map(field => field.label).join(' · ') ||
            'Select an eligible ledger entry'}</span>
          {!hostedCommandAvailable(act.command) && <i>{hostedControlRoomNotOnServer}</i>}
        </HQAct>
      </div>)}
    </div>
    <div className="cr-note">
      Every fix is previewed with its affected league data before review and confirmation.
      Record the reason with the change; the server applies its safeguard.</div>
    {definition && <HostedCommissionerActTray definition={definition} league={league} room={room}
      port={port} initial={initial} />}
    <div className="hosted-fix-tables">
      <section className="cr-panel" aria-label="Exceptions in force">
        <div className="cr-ph">EXCEPTIONS IN FORCE <span className="sp" />
          <span className="ct">active corrections and calendar exceptions</span></div>
        <div className="cr-pb">
          {exceptionFailure && <p role="status">{exceptionFailure}</p>}
          {calendarFailure && <p role="status">Calendar exceptions: {calendarFailure}</p>}
          {!readings?.corrections && !exceptionFailure &&
            <p role="status">{pending ? 'Reading exceptions…' :
              'The exceptions have not been read yet.'}</p>}
          {hasExceptionRead && <table className="cr-t">
            <thead><tr><th>WHAT</th><th>WHO</th><th>ENDS</th><th /></tr></thead>
            <tbody>
              {exceptions.map(item => <tr key={item.key}>
                <td>{item.what}</td>
                <td>{(franchises.get(item.franchise) ?? item.franchise) || '—'}</td>
                <td>{item.ends}</td>
                <td>{item.entry && revert && reversible.has(item.entry) &&
                  hostedCommandAvailable(revert.command) &&
                  <HQAct onAction={() => onOpen(revert, { entry: item.entry })}>END NOW</HQAct>}</td>
              </tr>)}
              {exceptions.length === 0 && completeExceptionRead &&
                <tr><td colSpan={4}>None in force.</td></tr>}
            </tbody>
          </table>}
        </div>
      </section>
      <section className="cr-panel" aria-label="Recent fixes">
        <div className="cr-ph">RECENT FIXES <span className="sp" />
          <span className="ct">undo records a server correction</span></div>
        <div className="cr-pb">
          {ledgerFailure && <p role="status">{ledgerFailure}</p>}
          {!readings?.ledger && !ledgerFailure &&
            <p role="status">{pending ? 'Reading the ledger…' : 'The ledger has not been read yet.'}</p>}
          {readings?.ledger && <table className="cr-t">
            <thead><tr><th>WHEN</th><th>FIX</th><th>BY</th><th /></tr></thead>
            <tbody>
              {recent.map(entry => <tr key={entry.id}>
                <td>{new Date(entry.when).toLocaleDateString()}</td>
                <td>{entry.what}</td>
                <td>{entry.who}</td>
                <td>{revert && hostedCommandAvailable(revert.command) &&
                  <HQAct onAction={() => onOpen(revert, { entry: entry.id })}>UNDO</HQAct>}</td>
              </tr>)}
              {recent.length === 0 && <tr><td colSpan={4}>Nothing on the ledger can be undone.</td></tr>}
            </tbody>
          </table>}
        </div>
      </section>
    </div>
  </div>;
}
