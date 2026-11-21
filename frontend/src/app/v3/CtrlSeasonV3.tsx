// Desktop v3 Control Room, Season tab: calendar windows, schedule summary, playoff picture,
// rollover with its backup step, divisions status, and the act tray its actions open, drawn with
// v3 panels and rows. Reads and action state come from useHostedSeason and useHostedControlRoom.
import { useState, type ReactNode } from 'react';
import { deadlineLabel } from '../clock/labels';
import { HQAct } from '../shell/HQAct';
import { hostedCommandAvailable } from '../commands/native';
import { HostedCommissionerActTray } from '../commissioner/CommissionerActTray';
import { HostedBackupNow } from '../commissioner/HostedBackupNow';
import { useHostedSeason } from '../commissioner/HostedControlRoomSeason';
import type { ControlBodyProps } from './ControlRoomV3';

export type SeasonWindow = NonNullable<ReturnType<typeof useHostedSeason>['state']['windows']>[number];
export type WindowFilter = 'all' | 'live';
export type ScheduleSummary = { draft: number; published: number; games: number };

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;

function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

export const DIVISIONS_UNAVAILABLE = 'Division alignment cannot be changed here yet.';

const SEASON_WINDOW_NAMES: Record<string, string> = {
  contracts: 'Contracts',
  cut_buyout: 'Cut / buyout',
  lineups: 'Lineups',
  open_fa: 'Open FA',
  resign: 'Re-sign',
  rfa_tenders: 'RFA tenders',
  rookie_draft: 'Rookie draft',
  ufa: 'UFA',
  waivers: 'Waivers',
};

function humanizeWindowId(id: string): string {
  if (id in SEASON_WINDOW_NAMES) return SEASON_WINDOW_NAMES[id];
  const words = id.trim().replaceAll('_', ' ');
  return words ? words.charAt(0).toUpperCase() + words.slice(1) : '';
}

/** Human-readable name for a calendar window, using its id so windows with empty `opens` stay named. */
export function seasonWindowName(actionWindow: Pick<SeasonWindow, 'id' | 'opens'>): string {
  const fromId = humanizeWindowId(actionWindow.id);
  if (fromId) return fromId;
  return actionWindow.opens.map(deadlineLabel).join(', ') || 'League window';
}

/** Actions opened by a calendar window, formatted for display (empty when none are configured). */
export function seasonWindowOpens(actionWindow: Pick<SeasonWindow, 'opens'>): string {
  return actionWindow.opens.map(deadlineLabel).join(', ');
}

/** Filters league calendar windows by all or live-only. */
export function filterSeasonWindows(
  windows: readonly SeasonWindow[] | undefined,
  filter: WindowFilter,
): SeasonWindow[] {
  if (!windows) return [];
  return filter === 'live' ? windows.filter(actionWindow => actionWindow.live) : [...windows];
}

/** Formats schedule counts as key/value rows for the Schedule panel. */
export function scheduleSummaryRows(
  schedule: ScheduleSummary | undefined,
): { label: string; value: number }[] {
  if (!schedule) return [];
  return [
    { label: 'Published weeks', value: schedule.published },
    { label: 'Games', value: schedule.games },
    { label: 'Draft weeks', value: schedule.draft },
  ];
}

/** Splits available season actions into schedule or playoffs groups. */
export function groupSeasonActions(
  available: readonly [string, string][],
  group: 'schedule' | 'playoffs',
): readonly [string, string][] {
  return available.filter(([command]) => group === 'schedule'
    ? command.startsWith('schedule.')
    : command.startsWith('bracket.') || command === 'title.set');
}

export function CtrlSeasonV3({ c, league, port, badge }: ControlBodyProps) {
  const { state, available, open, backingUpForRollover, setBackingUpForRollover } =
    useHostedSeason(league, port, c.openSeasonTool);
  const [windowFilter, setWindowFilter] = useState<WindowFilter>('all');
  const visibleWindows = filterSeasonWindows(state.windows, windowFilter);
  const scheduleRows = scheduleSummaryRows(state.schedule);
  const scheduleActs = groupSeasonActions(available, 'schedule');
  const playoffActs = groupSeasonActions(available, 'playoffs');
  const canMoveWindow = hostedCommandAvailable('calendar.window.move');
  const canRollover = hostedCommandAvailable('season.rollover');

  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.25fr) minmax(0,1fr)',
    gridTemplateRows: 'minmax(0,1fr) minmax(0,1fr)' }}>
    <Panel title="Season calendar" badge={badge}
      tag={state.windows ? <span className="tag o">{visibleWindows.length}</span> : null}
      foot="from the league server's calendar read">
      <div className="fbar">
        {([['all', 'All'], ['live', 'Live']] as const).map(([value, label]) =>
          <HQAct key={value} className="chip" selected={windowFilter === value}
            onAction={() => setWindowFilter(value)}>{label}</HQAct>)}
      </div>
      {state.error && <St><span role="alert">{state.error}</span></St>}
      {!state.schedule && !state.error &&
        <St><span role="status">Reading season from the league server…</span></St>}
      {state.schedule && <>
        <div className="sec">ACTION WINDOWS · {visibleWindows.length}</div>
        {visibleWindows.length === 0 && <St>{windowFilter === 'live'
          ? 'No live action windows right now.' : 'No action windows in the league calendar.'}</St>}
        {visibleWindows.map(actionWindow => {
          const opens = seasonWindowOpens(actionWindow);
          return <div key={actionWindow.id} className="row"
            style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto auto' }}>
            <span className="ell"><b>{seasonWindowName(actionWindow)}</b>
              {opens && <span className="d"> · {opens}</span>}</span>
            <span className="m">{actionWindow.start}</span>
            {actionWindow.live
              ? <span className="tag gr">Live</span>
              : <span className="tag">Scheduled</span>}
            <span className="acts">{canMoveWindow && <HQAct className="btn sm"
              onAction={() => open('calendar.window.move')}>Move window</HQAct>}</span>
          </div>;
        })}
      </>}
    </Panel>

    <Panel title="Schedule" badge={badge}
      tag={state.season !== undefined ? <span className="tag o">{state.season}</span> : null}
      foot="from the league server's schedule read">
      {state.error && <St><span role="alert">{state.error}</span></St>}
      {!state.schedule && !state.error &&
        <St><span role="status">Reading season from the league server…</span></St>}
      {state.schedule && <>
        <div className="sec">SCHEDULE · {state.season}</div>
        {scheduleRows.map(row => <div key={row.label} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
          <span className="k">{row.label}</span>
          <span className="v">{row.value}</span>
        </div>)}
        <div className="sec">ACTIONS</div>
        {scheduleActs.length > 0 ? <div className="fbar">
          {scheduleActs.map(([command, label]) =>
            <HQAct key={command} className="btn sm" onAction={() => open(command)}>{label}</HQAct>)}
        </div> : <St>No schedule actions available.</St>}
      </>}
    </Panel>

    <Panel title="Playoffs" badge={badge} foot="from the league server's bracket read">
      {state.error && <St><span role="alert">{state.error}</span></St>}
      {!state.schedule && !state.error &&
        <St><span role="status">Reading season from the league server…</span></St>}
      {state.schedule && <>
        <div className="sec">PLAYOFFS</div>
        <div className="row" style={{ height: 'auto', minHeight: 30, padding: '6px 10px',
          whiteSpace: 'normal', gridTemplateColumns: 'minmax(0,1fr)' }}>
          <span>{state.bracket?.words}</span>
        </div>
        <div className="sec">ACTIONS</div>
        {playoffActs.length > 0 ? <div className="fbar">
          {playoffActs.map(([command, label]) =>
            <HQAct key={command} className="btn sm" onAction={() => open(command)}>{label}</HQAct>)}
        </div> : <St>No playoff actions available.</St>}
      </>}
    </Panel>

    <Panel title="Rollover & divisions" badge={badge}
      foot="rollover starts with a fresh verified backup">
      {state.error && <St><span role="alert">{state.error}</span></St>}
      {!state.schedule && !state.error &&
        <St><span role="status">Reading season from the league server…</span></St>}
      {state.schedule && <>
        <div className="sec">ROLLOVER</div>
        <div className="row" style={{ height: 'auto', minHeight: 30, padding: '6px 10px',
          whiteSpace: 'normal', gridTemplateColumns: 'minmax(0,1fr)' }}>
          <span className="d">
            Rollover starts with a fresh verified backup. After it finishes, review the rollover plan
            and confirm that change separately.
          </span>
        </div>
        <div className="fbar">
          {canRollover && <HQAct className="btn sm" disabled={backingUpForRollover}
            onAction={() => setBackingUpForRollover(true)}>Roll over season</HQAct>}
          {backingUpForRollover && <HQAct className="btn sm"
            onAction={() => setBackingUpForRollover(false)}>Cancel rollover</HQAct>}
        </div>
        {backingUpForRollover && <div style={{ padding: '6px 10px' }}>
          <HostedBackupNow league={league} port={port} autoPreview onDone={() => {
            setBackingUpForRollover(false);
            open('season.rollover');
          }} />
        </div>}
        <div className="sec">DIVISIONS</div>
        <St>{DIVISIONS_UNAVAILABLE}</St>
      </>}
    </Panel>

    {c.definition && <Panel title={c.definition.label} badge={badge} style={{ gridColumn: '1 / -1' }}
      tag={<HQAct className="btn sm" onAction={() => c.navigateTo('season')}>Close</HQAct>}
      foot="preview and confirm before any change is written">
      <HostedCommissionerActTray definition={c.definition} league={league} room={c.room} port={port} />
    </Panel>}
  </div></div>;
}
