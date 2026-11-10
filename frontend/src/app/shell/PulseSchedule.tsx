import { useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import { pointsLabel, type PulseGame } from '../data/nativePulse';
import { readPulseSeason, usePulseRead, type NativePulse, type PulseSeason } from './pulseReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import { PulseIdle } from './PulseNow';

/** ScheduleDrawer is the season's published schedule: the league's grid around this week, or my
 *  team's whole road. It reads only while it is open. */
export function ScheduleDrawer({ open, onClose, reading, mine, port }: { open: boolean;
  onClose: () => void; reading: NativePulse; mine: string; port: HostedPort }) {
  const [team, setTeam] = useState(false);
  return <div className={`bdrawer${open ? ' open' : ''}`}>
    <div className="bd-head">
      <h2>Schedule</h2>
      <span className="sub">weeks 1–{reading.lastWeek || '—'} · regular season</span>
      <div className="dtab">
        <PulseAct active={!team} onAction={() => setTeam(false)}>LEAGUE</PulseAct>
        <PulseAct active={team} onAction={() => setTeam(true)}>MY TEAM</PulseAct>
      </div>
      <PulseAct className="x" onAction={onClose}>ESC ✕</PulseAct>
    </div>
    <div className="dbody">{open && <Body reading={reading} mine={mine} port={port} team={team} />}</div>
  </div>;
}

type Cell = { opponent: string; away: boolean; result: string; score: string };

/** cellOf is one franchise's week: who it plays and, once the week has a result, how it went. */
function cellOf(season: PulseSeason, franchise: string, week: number, nameOf: (id: string) => string):
  Cell | undefined {
  const fixture = season.schedule.weeks.find(w => w.week === week)?.games
    .find(g => g.home === franchise || g.away === franchise);
  if (!fixture) return undefined;
  const away = fixture.away === franchise;
  const game: PulseGame | undefined = season.played[week]?.games
    .find(g => g.home === fixture.home && g.away === fixture.away);
  const [own, theirs] = away ? [game?.awayPoints, game?.homePoints] : [game?.homePoints, game?.awayPoints];
  const decided = own !== undefined && theirs !== undefined;
  return { opponent: nameOf(away ? fixture.home : fixture.away), away,
    result: !decided ? '' : game?.winner === franchise ? 'W' : game?.winner ? 'L' : 'T',
    score: decided ? `${pointsLabel(own)} – ${pointsLabel(theirs)}` : '' };
}

function Body({ reading, mine, port, team }: { reading: NativePulse; mine: string; port: HostedPort;
  team: boolean }) {
  const now = reading.week ?? 0;
  const season = usePulseRead<PulseSeason>(JSON.stringify(['pulse-season', reading.league, reading.season, now]),
    () => readPulseSeason(reading.league, reading.season, now, port));
  if (!season.reading) return <PulseIdle words={season.error ? `The schedule is unavailable · ${season.error}`
    : 'Reading the schedule…'} />;
  const nameOf = (id: string) => reading.seats.find(s => s.franchise === id)?.name ?? id;
  const weeks = season.reading.schedule.weeks.filter(w => w.games.length).map(w => w.week);
  if (team) return <div className="scroll"><table className="ct"><thead><tr>
    <th>WK</th><th>OPPONENT</th><th>RESULT</th><th>SCORE</th></tr></thead><tbody>
    {weeks.map(week => {
      const c = cellOf(season.reading!, mine, week, nameOf);
      return <tr key={week}><td>WK {week}</td><td>{c ? `${c.away ? '@ ' : ''}${c.opponent}` : 'bye'}</td>
        <td>{c?.result || (week === now ? 'NOW' : '–')}</td><td>{c?.score}</td></tr>;
    })}</tbody></table></div>;
  // The mock's window: last week, this week and the five after it.
  const shown = weeks.filter(w => w >= now - 1).slice(0, 7);
  return <div className="scroll">
    <div className="grid-sched" style={{ gridTemplateColumns: `170px repeat(${shown.length},1fr)` }}>
      <div className="h">FRANCHISE</div>
      {shown.map(w => <div className="h" key={w}>WK {w}</div>)}
      {reading.seats.flatMap(seat => [
        <div key={seat.franchise} className={seat.franchise === mine ? 'me' : ''}>{seat.name}</div>,
        ...shown.map(w => {
          const c = cellOf(season.reading!, seat.franchise, w, nameOf);
          const tone = c?.result === 'W' ? 'w' : c?.result ? 'l' : w === now ? 'n' : '';
          return <div key={`${seat.franchise}-${w}`}
            className={`${tone}${seat.franchise === mine && w === now ? ' me' : ''}`}>
            {c ? `${c.result ? `${c.result} ` : ''}${c.away ? '@ ' : ''}${c.opponent}` : 'bye'}</div>;
        })])}
    </div>
  </div>;
}
