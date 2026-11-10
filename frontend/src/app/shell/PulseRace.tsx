import { useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import type { NativePulse } from './pulseReads';
import { conferencesOf, divisionGroups, seedRows, type PulseSeedRow } from './pulseModel';
import { PulseIdle } from './PulseNow';
import { BracketDrawer, TiebreakDrawer } from './PulseRaceDrawers';
import { percent, type PulseFixture, type PulseOdds, type PulseStandings } from '../data/nativePulse';

export type RaceDrawer = 'tb' | 'bk';

export function PulseRace({ reading, mine, drawer, onClose }: { reading: NativePulse; mine: string;
  drawer?: RaceDrawer; onClose: () => void }) {
  const conferences = conferencesOf(reading.bracket);
  const [conference, setConference] = useState<string>();
  const [view, setView] = useState<'seed' | 'div'>('seed');
  const [mode, setMode] = useState<'path' | 'power'>('path');
  const standings = reading.standings;
  const own = standings?.rows.find(r => r.franchise === mine)?.conference;
  const side = [conference, own, conferences[0]].find(name => name && conferences.includes(name)) ?? '';
  return <><section className="ws cols on" id="ws-race">
    <div className="pcol">
      <div className="ch">
        <b>STANDINGS</b>
        <div style={{ display: 'flex', gap: '8px' }}>
          <div className="seg">
            <PulseAct selected={view === 'seed'} onAction={() => setView('seed')}>SEEDS</PulseAct>
            <PulseAct selected={view === 'div'} onAction={() => setView('div')}>DIVISIONS</PulseAct>
          </div>
          {conferences.length > 1 && <div className="seg">
            {conferences.map(name => <PulseAct key={name} selected={side === name}
              onAction={() => setConference(name)}>{name.toUpperCase()}</PulseAct>)}
          </div>}
        </div>
      </div>
      {!standings
        ? <PulseIdle words={reading.errors.length
          ? `The standings are unavailable · ${reading.errors.join(' · ')}`
          : 'Reading the standings…'} />
        : view === 'div' ? <Divisions standings={standings} mine={mine} />
        : <Seeds reading={reading} standings={standings} side={side} mine={mine} />}
      <div id="road" style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column',
        borderTop: '2px solid var(--black)' }}>
        <Road reading={reading} mine={mine} />
      </div>
    </div>
    <div className="pcol" id="racec" style={{ minHeight: 0, display: 'flex',
      flexDirection: 'column' }}>
      <div className="ch">
        <div style={{ display: 'flex', gap: '14px', alignItems: 'center' }}>
          <b>{mode === 'path' ? 'PLAYOFF PATH' : 'POWER'}</b>
          <div className="seg">
            <PulseAct selected={mode === 'path'} onAction={() => setMode('path')}>
              PLAYOFF PATH</PulseAct>
            <PulseAct selected={mode === 'power'} onAction={() => setMode('power')}>POWER</PulseAct>
          </div>
        </div>
        <span className="mode">{reading.bracket?.words || ''}</span>
      </div>
      {mode === 'power'
        ? <Power reading={reading} mine={mine} />
        : <Path reading={reading} side={side} mine={mine} />}
    </div>
  </section>
    <TiebreakDrawer open={drawer === 'tb'} onClose={onClose} standings={standings} side={side}
      rows={standings ? seedRows(standings, reading.bracket, side, mine) : []} />
    <BracketDrawer open={drawer === 'bk'} onClose={onClose} bracket={reading.bracket} />
  </>;
}

/** oddsOf is one franchise's row in the league's odds read. */
const oddsOf = (odds: PulseOdds | undefined, franchise: string) =>
  odds?.rows.find(r => r.franchise === franchise);

function Power({ reading, mine }: { reading: NativePulse; mine: string }) {
  const odds = reading.odds;
  const rows = (odds?.rows ?? []).filter(r => r.power).sort((x, y) => (x.power ?? 0) - (y.power ?? 0));
  if (!rows.length) return <PulseIdle words={odds ? `The power ranking is unavailable · ${odds.powerWords}`
    : reading.errors.length ? `The power ranking is unavailable · ${reading.errors.join(' · ')}`
      : 'Reading the power ranking…'} />;
  return <>
    <div className="scroll"><table className="ct pwr">
      <thead><tr><th>PWR</th><th>FRANCHISE</th><th>W–L–T</th><th>POWER SCORE</th><th>EXP PTS</th>
        <th>ODDS</th></tr></thead>
      <tbody>{rows.map(r => <tr key={r.franchise} className={r.franchise === mine ? 'sel' : undefined}>
        <td className="sd">{r.power}</td><td><span>{r.name}</span></td><td>{r.record}</td>
        <td>{Math.round(r.powerScore * 100)}</td><td>{r.expected.toFixed(1)}</td>
        <td>{percent(r.playoffOdds)}</td></tr>)}</tbody>
    </table></div>
    <div className="drawer-note">POWER = this season: {odds?.powerWords}. Lineup value is the league's own
      on-field-now values; results are all-play when any week was all-played, else points for.
      Not standings.</div>
  </>;
}

function Divisions({ standings, mine }: { standings: PulseStandings; mine: string }) {
  return <table className="ct stand">
    <thead><tr><th>#</th><th>FRANCHISE</th><th>W–L–T</th><th>POINTS FOR</th></tr></thead>
    <tbody>
      {divisionGroups(standings).flatMap(group => [
        <tr className="dvh" key={`h-${group.division}`}>
          <td colSpan={4}>{group.division ? group.division.toUpperCase() : 'NO DIVISION SET'}
            <span>{group.rows.length} franchises</span></td>
        </tr>,
        ...group.rows.map(row => <tr key={row.franchise}
          className={row.franchise === mine ? 'me' : undefined}>
          <td className="sd">{row.rank}</td>
          <td><span>{row.name}</span></td>
          <td>{row.record}</td>
          <td title={row.pointsWords}>{row.pointsFor.toFixed(2)}</td>
        </tr>),
      ])}
    </tbody>
  </table>;
}

function Seeds({ reading, standings, side, mine }: { reading: NativePulse;
  standings: PulseStandings; side: string; mine: string }) {
  const rows = seedRows(standings, reading.bracket, side, mine);
  const places = reading.bracket?.perConference ?? 0;
  if (!rows.length) return <PulseIdle words={standings.words} />;
  return <table className="ct stand">
    <thead><tr><th>SEED</th><th>FRANCHISE</th><th>W–L–T</th><th>POINTS FOR</th><th>PWR</th><th>ODDS</th></tr>
    </thead>
    <tbody>
      {rows.map((row, index) => <tr key={row.franchise}
        className={`${row.mine ? 'me' : ''} ${index === places - 1 ? 'cut' : ''}`.trim()
          || undefined}>
        <td className="sd">{row.seed ?? '—'}</td>
        <td><span>{row.name}</span>
          <span className="div">{row.division.toUpperCase().replace(`${side.toUpperCase()} `, '')}
            {row.bye ? ' ★ BYE' : ''}</span></td>
        <td>{row.record}</td>
        <td title={row.pointsWords}>{row.pointsFor?.toFixed(2) ?? '—'}</td>
        <td>{oddsOf(reading.odds, row.franchise)?.power ?? '—'}</td>
        <td title={reading.odds?.oddsWords}>{percent(oddsOf(reading.odds, row.franchise)?.playoffOdds)}</td>
      </tr>)}
    </tbody>
  </table>;
}

function Road({ reading, mine }: { reading: NativePulse; mine: string }) {
  const name = (franchise: string) =>
    reading.seats.find(s => s.franchise === franchise)?.name ?? franchise;
  const record = (franchise: string) =>
    reading.standings?.rows.find(r => r.franchise === franchise)?.record ?? '';
  const ahead = reading.fixtures.filter(g => !reading.week || g.week >= reading.week);
  const shown: PulseFixture[] = ahead.slice(0, 6);
  return <>
    <div className="ch"><b>MY ROAD{shown.length
      ? ` · WKS ${shown[0].week}–${shown[shown.length - 1].week}` : ''}</b>
      <span>opponent record</span></div>
    {!shown.length
      ? <PulseIdle words={reading.fixtures.length
        ? 'Every week on this franchise’s schedule has been played.'
        : reading.weekNote || 'No schedule is published for this franchise yet.'} />
      : <>
        {shown.map(game => <div className="rem" key={game.week}>
          <span className="w">WK {game.week}</span>
          <span><span>{game.opponent ? name(game.opponent) : 'Bye'}</span>
            {game.week === reading.week && <span className="tag2">NOW</span>}
            <span className="src" style={{ marginLeft: '6px' }}>{game.venue.toLowerCase()}</span>
          </span>
          <span className="r">{game.opponent ? record(game.opponent) : ''}</span>
        </div>)}
        <div className="drawer-note" style={{ marginTop: 'auto' }}>
          Opponent strength is an all-play measure this league does not hold yet.
          {reading.bracket
            ? ` Playoffs run weeks ${reading.bracket.startWeek}–${reading.bracket.finalWeek}.` : ''}
        </div>
      </>}
  </>;
}

function Path({ reading, side, mine }: { reading: NativePulse; side: string; mine: string }) {
  const bracket = reading.bracket;
  if (!bracket) return <PulseIdle words={reading.errors.length
    ? `The playoff field is unavailable · ${reading.errors.join(' · ')}` : 'Reading the playoffs…'} />;
  const field = bracket.fields.find(f => f.conference === side);
  const seat = field?.seeds.find(s => s.franchise === mine);
  const rows: PulseSeedRow[] = reading.standings
    ? seedRows(reading.standings, bracket, side, mine) : [];
  return <>
    <div className="bigs">
      <div><b>{seat ? seat.seed : '—'}</b><span>MY SEED TODAY{seat?.bye ? ' · bye' : ''}</span></div>
      <div title={reading.odds?.oddsWords}><b>{percent(oddsOf(reading.odds, mine)?.playoffOdds)}</b>
        <span>MAKE PLAYOFFS · sim</span></div>
      <div><b>{bracket.byeSeeds || '—'}</b><span>FIRST-ROUND BYES</span></div>
      <div><b>{bracket.startWeek || '—'}</b><span>PLAYOFFS START · WEEK</span></div>
    </div>
    <div className="odds">
      <div className="okey">
        <span>{field ? `PLAYOFF FIELD · ${side.toUpperCase()} SIDE` : 'PLAYOFF FIELD'}</span>
        <span>on the bubble below the line</span>
      </div>
      {!rows.length ? <PulseIdle words={bracket.words || 'No playoff field is set yet.'} />
        : rows.map(row => {
          const chance = oddsOf(reading.odds, row.franchise)?.playoffOdds;
          return <div key={row.franchise} title={row.kind || undefined}
            className={`orow ${row.mine ? 'me' : ''} ${row.inField ? '' : 'out'}`.trim()}>
            <span className="sn">{row.seed ? `SEED ${row.seed}${row.bye ? ' · BYE' : ''}` : 'OUT'}</span>
            <span className="tn">{row.name}</span>
            <span className="rc">{row.record}</span>
            <span className="ob">
              <i style={{ width: chance === undefined ? 0 : `${Math.round(chance * 100)}%` }} />
            </span>
            <span className="op">{percent(chance)}</span>
          </div>;
        })}
    </div>
    <div className="drawer-note" style={{ marginTop: 'auto' }}>
      {bracket.words || `Seeded from the standings under the league’s own tie-breakers.`}
      {' '}Playoff odds: {reading.odds?.oddsWords
        ?? (reading.errors.length ? 'unavailable' : 'still reading')}.
      A bye chance, a magic number and week-by-week scenarios are not read yet.
    </div>
  </>;
}
