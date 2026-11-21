// Desktop v3 Control Room, Fix tab, after TheWarRoom-mock-v3-FINAL.html (screens.js control.fix): the fix
// actions, the exceptions in force and the recent fixes that can be undone. Reads and actions are the old
// room's own (c.fixState, c.openTool, the commissioner act tray); nothing here is invented.
import type { ReactNode } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct } from '../shell/HQAct';
import { hostedCommandAvailable } from '../commands/native';
import { actTable } from '../commissioner/actTable';
import { HostedCommissionerActTray } from '../commissioner/CommissionerActTray';
import type { HostedFixReadings } from '../commissioner/hostedControlRoomData';
import { memberDisplayName, type People } from '../commissioner/people';
import type { ControlBodyProps } from './ControlRoomV3';

const fixCommands = [
  'commish.stat.correct', 'commish.score.player', 'commish.score.franchise',
  'commish.lineup.correct', 'commish.player.position', 'commish.standings.override',
  'commish.standings.record.adjust', 'commish.contract.edit', 'commish.correction.revert',
] as const;

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

export type FixException = { key: string; what: string; franchise: string; ends: string; entry: string };

/** Active corrections with an end or a rule, plus calendar exceptions no correction already covers. */
export function fixExceptions(readings: HostedFixReadings | undefined): FixException[] {
  const corrections = readings?.corrections?.items.filter(item => item.until || item.rule) ?? [];
  const calendar = readings?.calendarExceptions ?? [];
  return [
    ...corrections.map(item => ({ key: item.entry, what: item.what,
      franchise: item.franchise ?? item.scope?.franchise ?? '',
      ends: item.until ? new Date(item.until).toLocaleDateString() : 'until cleared',
      entry: item.entry })),
    ...calendar.filter(exception => !corrections.some(item => {
      const franchise = item.franchise ?? item.scope?.franchise ?? '';
      return item.rule === `calendar.${exception.action}` && franchise === exception.franchise &&
        item.when === exception.start;
    })).map(exception => ({ key: `${exception.action}:${exception.franchise}:${exception.start}`,
      what: `${exception.action} outside its window`, franchise: exception.franchise,
      ends: exception.permanent ? 'until cleared' : new Date(exception.until).toLocaleDateString(),
      entry: '' })),
  ];
}

/** Ledger entries that can still be undone, newest first as read; at most `limit`. */
export function undoableEntries(readings: HostedFixReadings | undefined, limit = 10) {
  return readings?.ledger?.entries.filter(entry => entry.undo !== '' && !entry.replacedAt)
    .slice(0, limit) ?? [];
}

/** Resolves the actor display name on a recent-fix ledger row, falling back to the raw id. */
export function fixActorName(who: string, people?: Pick<People, 'seats'>): string {
  return memberDisplayName(who, people);
}

export function CtrlFixV3({ c, league, port, badge, people }: ControlBodyProps) {
  const readings = c.fixState.fixes;
  const { pending, error, errorCode } = c.fixState;
  const open = (next: Parameters<typeof c.openTool>[0], initial?: Parameters<typeof c.openTool>[3]) =>
    c.openTool(next, undefined, 'fix', initial);
  const definitions = fixCommands.map(command => actTable.find(row => row.command === command));
  const franchises = new Map((readings?.franchises ?? []).map(team => [team.franchise, team.name]));
  const exceptions = fixExceptions(readings);
  const recent = undoableEntries(readings);
  const allReversible = new Set(readings?.ledger?.entries
    .filter(entry => entry.undo !== '' && !entry.replacedAt).map(entry => entry.id) ?? []);
  const revert = actTable.find(row => row.command === 'commish.correction.revert');
  const canRevert = revert !== undefined && hostedCommandAvailable(revert.command);
  const hasExceptionRead = readings?.corrections !== undefined || readings?.calendarExceptions !== undefined;
  const completeExceptionRead = readings?.corrections !== undefined &&
    readings.calendarExceptions !== undefined;
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1fr) minmax(0,1.3fr)', gridTemplateRows: 'minmax(0,1fr)' }}>
    <Panel title="Fix something" badge={badge}
      foot={'Every fix is previewed with its affected league data before review and confirmation. '
        + 'Record the reason with the change; the server applies its safeguard.'}>
      {error && <St><FailureAlert message={error} code={errorCode} league={league}
        screen="Control Room · Fix" route="control/fix" port={port} /></St>}
      {definitions.map(act => act && <div key={act.command} className="row" data-fix-command={act.command}
        style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
        <span className="ell"><b>{act.label}</b> <span className="d">
          {act.fields.map(f => f.label).join(' · ') || 'Select an eligible ledger entry'}</span></span>
        <span className="acts">{hostedCommandAvailable(act.command)
          ? <HQAct className="btn sm" label={act.label} onAction={() => open(act)}>Open ▸</HQAct>
          : <span className="m">Not available yet.</span>}</span></div>)}
      {c.definition && c.openedTab === 'fix' && <HostedCommissionerActTray definition={c.definition}
        league={league} room={c.room} port={port} initial={c.initialValues} />}
    </Panel>
    <Panel title="Fix desk" badge={badge} foot="undo records a server correction">
      <div className="sec">EXCEPTIONS IN FORCE · {hasExceptionRead ? exceptions.length : '—'}</div>
      {readings?.correctionsError && <St><span role="status">{readings.correctionsError}</span></St>}
      {readings?.calendarError &&
        <St><span role="status">Calendar exceptions: {readings.calendarError}</span></St>}
      {!readings?.corrections && !readings?.correctionsError &&
        <St><span role="status">{pending ? 'Reading exceptions…' :
          'The exceptions have not been read yet.'}</span></St>}
      {exceptions.map(item => <div key={item.key} className="row"
        style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto auto' }}>
        <span className="ell">{item.what}</span>
        <span className="m">{(franchises.get(item.franchise) ?? item.franchise) || '—'}</span>
        <span className="m">{item.ends}</span>
        <span className="acts">{item.entry && canRevert && allReversible.has(item.entry) &&
          <HQAct className="btn sm" label={`End ${item.what} now`}
            onAction={() => open(revert, { entry: item.entry })}>End now</HQAct>}</span></div>)}
      {exceptions.length === 0 && completeExceptionRead && <St>None in force.</St>}
      <div className="sec">RECENT FIXES · {readings?.ledger ? recent.length : '—'}</div>
      {readings?.ledgerError && <St><span role="status">{readings.ledgerError}</span></St>}
      {!readings?.ledger && !readings?.ledgerError &&
        <St><span role="status">{pending ? 'Reading the ledger…' :
          'The ledger has not been read yet.'}</span></St>}
      {recent.map(entry => <div key={entry.id} className="row"
        style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto auto' }}>
        <span className="m">{new Date(entry.when).toLocaleDateString()}</span>
        <span className="ell">{entry.what}</span>
        <span className="m">{fixActorName(entry.who, people)}</span>
        <span className="acts">{canRevert && <HQAct className="btn sm" label={`Undo ${entry.what}`}
          onAction={() => open(revert, { entry: entry.id })}>Undo</HQAct>}</span></div>)}
      {readings?.ledger && recent.length === 0 && <St>Nothing on the ledger can be undone.</St>}
    </Panel>
  </div></div>;
}
