// Desktop v3 Control Room, Picks tab, after the mock's control.picks: every pick in the league with
// year / round / traded filters and a team search, plus the one real pick action (correct an owner).
// Reads come from the same native.assets.read the old picks body uses; actions are the old act tray.
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { onHostedTopics } from '../data/hostedReads';
import { createReadQuery } from '../data/query';
import { hostedCommandAvailable, type NativeRequest } from '../commands/native';
import { HostedCommissionerActTray } from '../commissioner/CommissionerActTray';
import { actTable, type ActDefinition } from '../commissioner/actTable';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedRoster, type HostedRoster } from '../commissioner/hostedReadings';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { ControlBodyProps } from './ControlRoomV3';

type PickRow = HostedRoster['picks'][number];
export type PickFilters = { year: string; round: string; trade: string; search: string };

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

function createPicksQuery(league: string, port: HostedPort = hostedWorkflow) {
  return createReadQuery('commissioner.TargetNativeAssets', {
    initial: (): { picks?: HostedRoster } => ({}),
    read: async (scope: string): Promise<HostedRoster> => {
      const response = await port({ command: 'native.assets.read', mode: 'read',
        arguments: { league: scope } } as NativeRequest);
      const picks = parseHostedRoster(response.data);
      if (picks.league !== league || scope !== league)
        throw new Error('Picks belong to another league.');
      return picks;
    },
    accept: (picks: HostedRoster) => ({ picks }),
  });
}

/** A franchise's label, or its id when the read does not name it. */
export function franchiseName(reading: Pick<HostedRoster, 'franchises'>, franchise: string): string {
  return reading.franchises.find(team => team.value === franchise)?.label ?? franchise;
}

/** How the pick came to its owner. */
export function pickHow(pick: PickRow): string {
  return pick.correction ? 'Correction' : pick.compensatory ? 'Compensatory'
    : pick.owner !== pick.original ? 'Traded' : 'Original';
}

/** The pick's state as the old body words it. */
export function pickStatus(pick: PickRow): 'USED' | 'SKIPPED' | 'ACTIVE' {
  return pick.player ? 'USED' : pick.skipped ? 'SKIPPED' : 'ACTIVE';
}

/** The pick number as round.slot, e.g. 1.04. */
export function pickNumber(pick: PickRow): string {
  return `${pick.round}.${String(pick.slot).padStart(2, '0')}`;
}

/** The next draft year: the first pick season after the league season, else the last one. */
export function defaultPickYear(reading: Pick<HostedRoster, 'picks' | 'season'> | undefined): string {
  const seasons = [...new Set(reading?.picks.map(pick => pick.season) ?? [])].sort((a, b) => a - b);
  return String(seasons.find(season => season > (reading?.season ?? 0)) ?? seasons.at(-1) ?? 'all');
}

/** The picks that pass the filters, in draft order. `year` is a season or 'all'. */
export function filterPicks(reading: HostedRoster, f: PickFilters): PickRow[] {
  const search = f.search.trim().toLowerCase();
  return reading.picks.filter(pick => {
    if (f.year !== 'all' && pick.season !== Number(f.year)) return false;
    if (f.round !== 'all' && pick.round !== Number(f.round)) return false;
    if (f.trade === 'traded' && (pick.owner === pick.original || pick.correction || pick.compensatory))
      return false;
    if (!search) return true;
    const names = `${franchiseName(reading, pick.original)} ${franchiseName(reading, pick.owner)} `
      + `${pick.original} ${pick.owner}`;
    return names.toLowerCase().includes(search);
  }).sort((a, b) => a.season - b.season || a.round - b.round || a.slot - b.slot);
}

const movePick = actTable.find((definition): definition is ActDefinition =>
  definition.command === 'correct.pick');
export const PICK_CONDITION_EMPTY = '—';
export const PICK_ACTIONS_NOTE = "Only correcting a pick's owner is available here; "
  + 'pick conditions, grants, swaps, forfeits, and extra picks cannot be edited here yet.';

export function CtrlPicksV3({ c, league, port, badge }: ControlBodyProps) {
  const room = c.room;
  const roomState = c.roomState;
  const query = useMemo(() => createPicksQuery(league, port), [league, port]);
  const state = query.use();
  const [year, setYear] = useState('next');
  const [round, setRound] = useState('all');
  const [trade, setTrade] = useState('all');
  const [team, setTeam] = useState('');
  const [definition, setDefinition] = useState<ActDefinition>();

  useEffect(() => {
    const unsubscribe = onHostedTopics(topics => {
      if (!topics.some(topic => topic.league === league && topic.kind === 'league')) return;
      query.invalidate();
      void query.load([league], { overlap: 'after' });
    });
    void query.load([league]);
    return () => { unsubscribe(); query.dispose(); };
  }, [league, query]);

  const refresh = () => { query.invalidate(); void query.load([league]); };
  const reading = state.picks;
  const seasons = [...new Set(reading?.picks.map(pick => pick.season) ?? [])].sort((a, b) => a - b);
  const rounds = [...new Set(reading?.picks.map(pick => pick.round) ?? [])].sort((a, b) => a - b);
  const selectedYear = year === 'next' ? defaultPickYear(reading) : year;
  const rows = reading ? filterPicks(reading, { year: selectedYear, round, trade, search: team }) : [];
  const cols = '56px 48px minmax(0,1fr) minmax(0,1fr) minmax(0,1fr) 96px 80px';
  const canChange = room.canChange() && !roomState.review && !roomState.pending;

  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.8fr) minmax(0,1fr)', gridTemplateRows: 'minmax(0,1fr)' }}>
    <Panel title="Picks" badge={badge} tag={<>
      {reading && <span className="tag o">{rows.length} of {reading.picks.length}</span>}
      <HQAct className="btn sm" disabled={state.pending} onAction={refresh}>Refresh picks</HQAct></>}
      foot={reading ? `${rows.length} of ${reading.picks.length} picks` : 'Reading picks…'}>
      <div className="fbar">
        <select aria-label="Filter picks by year" value={selectedYear}
          onChange={event => setYear(event.target.value)}>
          <option value="all">All years</option>
          {seasons.map(season => <option key={season} value={season}>{season}</option>)}
        </select>
        <select aria-label="Filter picks by round" value={round}
          onChange={event => setRound(event.target.value)}>
          <option value="all">All rounds</option>
          {rounds.map(value => <option key={value} value={value}>Round {value}</option>)}
        </select>
        <HQAct className="chip" selected={trade === 'traded'}
          onAction={() => setTrade(trade === 'traded' ? 'all' : 'traded')}>Traded</HQAct>
        <input aria-label="Search picks by team" type="search" placeholder="Find a team" value={team}
          onChange={event => setTeam(event.target.value)} />
      </div>
      {state.error && <St><span role="alert">Picks unavailable. {state.error}</span></St>}
      {!reading && !state.error && <St><span role="status">Reading the league's picks…</span></St>}
      {reading && <>
        <div className="sec">PICKS · {rows.length}</div>
        <div className="row" style={{ gridTemplateColumns: cols }}>
          <span className="m">YEAR</span><span className="m">PICK</span><span className="m">ORIGINAL</span>
          <span className="m">OWNED BY</span><span className="m">CONDITION</span>
          <span className="m">HOW</span><span className="m">STATUS</span></div>
        {rows.map(pick => <div key={`${pick.season}:${pick.round}:${pick.original}:${pick.slot}`}
          className="row" style={{ gridTemplateColumns: cols }}>
          <span className="m">{pick.season}</span><b className="m">{pickNumber(pick)}</b>
          <span className="ell">{franchiseName(reading, pick.original)}</span>
          <span className="ell">{franchiseName(reading, pick.owner)}</span>
          <span className="m ell" title="Pick conditions are not tracked here yet.">{PICK_CONDITION_EMPTY}</span>
          <span className="m">{pickHow(pick)}</span>
          <span className="tag o">{pickStatus(pick)}</span></div>)}
        {rows.length === 0 && <St>No picks match these filters.</St>}
      </>}
    </Panel>
    <Panel title="Pick actions" badge={badge} foot="every pick change is in the ledger">
      {movePick && hostedCommandAvailable(movePick.command) && (
        <HQAct className="btn sm" disabled={!canChange} onAction={() => setDefinition(movePick)}>
          Correct a pick owner</HQAct>
      )}
      <St><span role="status">{PICK_ACTIONS_NOTE}</span></St>
      {definition && <>
        <HQAct className="btn sm" disabled={roomState.pending}
          onAction={() => { room.cancel(); setDefinition(undefined); }}>Close pick correction</HQAct>
        <HostedCommissionerActTray definition={definition} league={league} room={room} port={port} />
      </>}
    </Panel>
  </div></div>;
}
