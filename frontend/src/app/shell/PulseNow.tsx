import { useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import { pointsLabel, type PulseResults } from '../data/nativePulse';
import { readPulseMatchup, usePulseRead, type NativePulse, type PulseMatchup } from './pulseReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import { applyFilter, filterCounts, groupRows, pulseRows, scoreOf, starterRows, PULSE_FILTERS,
  type PulseFilter, type PulseRow, type PulseStarterRow } from './pulseModel';

/** PulseIdle is the calm empty state: what is missing, and what will fill it. */
export function PulseIdle({ words }: { words: string }) {
  return <div className="idle" role="status">{words}</div>;
}

const OFFENCE = ['QB', 'RB', 'WR', 'TE', 'K', 'PK', 'OL'];
type Reading<T> = { reading?: T; pending?: boolean; error?: string };

export function PulseNow({ reading, mine, week, results, league, port }: {
  reading: NativePulse; mine: string; week?: number; results: Reading<PulseResults>;
  league: string; port: HostedPort;
}) {
  const [side, setSide] = useState<'mu' | 'nfl'>('mu');
  const [filter, setFilter] = useState<PulseFilter>('all');
  const [picked, pick] = useState<string>();
  const current = week === reading.week;
  const rows = results.reading ? pulseRows(results.reading, mine, reading.seats, current) : [];
  const selected = rows.find(r => r.key === picked) ?? rows.find(r => r.mine) ?? rows[0];
  return <section className="ws cols on" id="ws-live">
    <div className="pcol">
      <div className="ch">
        <div className="seg">
          <PulseAct selected={side === 'mu'} onAction={() => setSide('mu')}>MATCHUPS</PulseAct>
          <PulseAct selected={side === 'nfl'} onAction={() => setSide('nfl')}>NFL GAMES</PulseAct>
        </div>
        <span>{week ? `week ${week}` : 'no week'}</span>
      </div>
      {side === 'nfl'
        ? <PulseIdle words={'NFL game scores, clocks and possession are not held in this league yet. '
          + 'They fill when a live NFL scoring feed is read for an app league.'} />
        : <MatchupList rows={rows} filter={filter} onFilter={setFilter} results={results}
            selected={selected} onPick={key => pick(key)} />}
    </div>
    <div className="pcol fieldwrap">
      <div className="follow">
        <span className="h">FOLLOWING</span>
        <span className="src">following a game needs the NFL scoreboard read</span>
      </div>
      <div className="gh">
        <span className="src">The NFL game behind a starter is not held yet.</span>
      </div>
      <div>
        <PulseIdle words={'The field, the drive and the down come from a live NFL game read. '
          + 'This league holds none yet.'} />
      </div>
      <div className="spine">
        <div className="ch">
          <b>{selected?.mine ? 'MY MATCHUP · STARTERS' : 'MATCHUP · STARTERS'}</b>
          <span>{selected ? `${selected.game.awayName} · ${selected.game.homeName}` : ''}</span>
        </div>
        <Starters reading={reading} week={week} results={results.reading} row={selected}
          league={league} port={port} />
      </div>
    </div>
    <div className="pcol">
      <div className="ch"><b>IN THIS GAME</b><span className="src">needs the NFL read</span></div>
      <PulseIdle words={'Who else in the league is in an NFL game fills when the NFL scoreboard is '
        + 'read for an app league.'} />
      <div className="ch" style={{ borderTop: '2px solid var(--black)' }}>
        <b>PLAY-BY-PLAY</b><span className="src">needs the NFL read</span>
      </div>
      <PulseIdle words={'Plays arrive with a live NFL play feed. No plays are held for this league.'} />
    </div>
  </section>;
}

function MatchupList({ rows, filter, onFilter, results, selected, onPick }: {
  rows: PulseRow[]; filter: PulseFilter; onFilter: (f: PulseFilter) => void;
  results: Reading<PulseResults>; selected?: PulseRow; onPick: (key: string) => void;
}) {
  const counts = filterCounts(rows);
  const shown = applyFilter(rows, filter);
  return <>
    <div className="fbar">
      {PULSE_FILTERS.map(f => <PulseAct key={f.key} className="mbtn" selected={filter === f.key}
        onAction={() => onFilter(f.key)}>{f.label}<small>{counts[f.key]}</small></PulseAct>)}
    </div>
    <div className="scroll">
      {!results.reading
        ? <PulseIdle words={results.error ? `The week's games are unavailable · ${results.error}`
          : results.pending ? 'Reading the week…' : 'No week is read yet'} />
        : !results.reading.published
          ? <PulseIdle words={`${results.reading.words}. Games appear here as soon as the league's `
            + 'schedule is published.'} />
        : !rows.length ? <PulseIdle words={results.reading.words} />
        : !shown.length ? <PulseIdle words="no games match this filter" />
        : groupRows(shown).map(group => <div key={group.heading}>
          <div className="ghd">{group.heading}</div>
          {group.rows.map(row => <GameRow key={row.key} row={row}
            selected={row.key === selected?.key} onPick={onPick} />)}
        </div>)}
    </div>
  </>;
}

function GameRow({ row, selected, onPick }: { row: PulseRow; selected: boolean;
  onPick: (key: string) => void }) {
  const awayAhead = row.decided && row.away! >= row.home!;
  return <PulseAct className={`game${selected ? ' sel' : ''}${row.mine ? ' mine' : ''}`}
    label={row.game.words} onAction={() => onPick(row.key)}>
    <div className={`tm${awayAhead ? ' w' : ''}`}>
      <span>{row.game.awayName}</span><b>{pointsLabel(row.away)}</b>
    </div>
    <div className={`tm${row.decided && !awayAhead ? ' w' : ''}`}>
      <span>{row.game.homeName}</span><b>{pointsLabel(row.home)}</b>
    </div>
    <div className="side">
      <span className={`st${row.status === 'LIVE' ? ' live' : row.status === 'FINAL' ? ' fin' : ''}`}>
        {row.status}</span>
      <span className="src">{row.margin === undefined ? '' : `by ${row.margin.toFixed(1)}`}</span>
    </div>
    <div className="wpl" />
  </PulseAct>;
}

function Starters({ reading, week, results, row, league, port }: {
  reading: NativePulse; week?: number; results?: PulseResults; row?: PulseRow; league: string;
  port: HostedPort;
}) {
  const away = row?.game.away;
  const home = row?.game.home;
  const awayScore = scoreOf(results, away ?? '');
  const homeScore = scoreOf(results, home ?? '');
  const starters = (franchise?: string) => scoreOf(results, franchise ?? '')?.starters
    .map(s => s.player) ?? [];
  const matchup = usePulseRead<PulseMatchup>(
    JSON.stringify(['pulse-matchup', league, week, away, home]),
    () => week && away && home
      ? readPulseMatchup(league, reading.season, week, [away, home],
        { [away]: starters(away), [home]: starters(home) }, port)
      : Promise.reject(new Error('Choose a matchup to read its lineups.')));
  if (!row) return <PulseIdle words="No game is read for this week yet." />;
  if (!awayScore?.starters.length && !homeScore?.starters.length)
    return <PulseIdle words={`${row.game.words} Starters appear once both franchises have a lineup `
      + 'recorded for this week.'} />;
  const left = starterRows(awayScore, matchup.reading?.lineups[away ?? ''],
    matchup.reading?.projections ?? {});
  const right = starterRows(homeScore, matchup.reading?.lineups[home ?? ''],
    matchup.reading?.projections ?? {});
  const pairs = Array.from({ length: Math.max(left.length, right.length) },
    (_unused, i) => [left[i], right[i]] as const);
  const offence = pairs.filter(([a, b]) => OFFENCE.includes((a ?? b)?.slot ?? ''));
  const defence = pairs.filter(pair => !offence.includes(pair));
  return <div className="mym">
    <div className="mh">
      <span className="a">{row.game.awayName.toUpperCase()}</span>
      <span className="t" style={{ textAlign: 'right', paddingRight: '10px' }}>
        {pointsLabel(awayScore?.points)}</span>
      <span className="t" />
      <span className="t" style={{ textAlign: 'left', paddingLeft: '10px' }}>
        {pointsLabel(homeScore?.points)}</span>
      <span className="b">{row.game.homeName.toUpperCase()}</span>
    </div>
    {[['OFFENCE', offence], ['DEFENCE', defence]].flatMap(([label, group]) =>
      (group as (readonly [PulseStarterRow?, PulseStarterRow?])[]).length === 0 ? [] : [
        <div className="mgrp" key={label as string}>{label as string}</div>,
        ...(group as (readonly [PulseStarterRow?, PulseStarterRow?])[]).map(([a, b], i) =>
          <StarterRow key={`${label as string}-${a?.player ?? i}-${b?.player ?? i}`} away={a}
            home={b} />),
      ])}
    {matchup.error && <p role="status" className="src">
      Starter teams and projections are not read · {matchup.error}</p>}
  </div>;
}

function StarterRow({ away, home }: { away?: PulseStarterRow; home?: PulseStarterRow }) {
  const cell = (p?: PulseStarterRow, right = false) => <div className={right ? 'p r' : 'p'}>
    <div className="nm">
      {right && <small>{p?.team || '—'}</small>}
      <span>{p?.name ?? '—'}</span>
      {!right && <small>{p?.team || '—'}</small>}
    </div>
    <div className={`ln${p?.projection === undefined ? ' wait' : ''}`}>
      {p === undefined ? '' : p.projection === undefined ? 'no projection held'
        : `proj ${p.projection.toFixed(1)}`}
    </div>
  </div>;
  const score = (p?: PulseStarterRow, other?: PulseStarterRow, right = false) => {
    const lead = p?.points !== undefined && (other?.points === undefined || p.points >= other.points);
    return <span className={`s${right ? ' r' : ''} ${p?.points === undefined ? 'tbd'
      : lead ? 'up' : 'dn'}`}>{pointsLabel(p?.points)}</span>;
  };
  const chip = (p?: PulseStarterRow, right = false) => <span
    className={`chipc ${p?.locked ? 'F' : 'P'}${right ? ' r' : ''}`}>
    {p === undefined ? '' : p.locked ? 'LOCKED' : 'OPEN'}</span>;
  return <div className="mr">
    {cell(away)}
    {chip(away)}
    <span className="ball2 off" />
    {score(away, home)}
    <span className="ps">{away?.slot ?? home?.slot ?? '—'}</span>
    {score(home, away, true)}
    <span className="ball2 off" />
    {chip(home, true)}
    {cell(home, true)}
  </div>;
}
