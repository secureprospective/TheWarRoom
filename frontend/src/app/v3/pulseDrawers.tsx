// League Pulse's pull-up drawers in the v3 look: the mock's bottom drawer (.bdrawer, a header bar,
// a table and a summary beside it), drawn in the frame's #layer like the mock's Layer.open. The rows
// are League Pulse's own reads: the chosen matchup's starters, the season schedule, stat changes.
import { useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { HQAct } from '../shell/HQAct';
import { pointsLabel, type PulseScore } from '../data/nativePulse';
import { bySlot, changePoints, changeVerdict, type StatChange } from '../shell/pulseModel';
import type { StarterSides } from '../shell/PulseNow';
import { cellOf } from '../shell/PulseSchedule';
import { readPulseSeason, usePulseRead, type NativePulse, type PulseSeason } from '../shell/pulseReads';
import { boxLine } from '../shell/statLine';
import type { HostedPort } from '../commissioner/hostedRoom';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;

export function BDrawer({ title, sub, tabs, close, children, side, tall }: { title: string; sub: string;
  tabs?: ReactNode; close: () => void; children: ReactNode; side?: ReactNode; tall?: boolean }) {
  const layer = typeof document === 'undefined' ? null : document.getElementById('layer');
  const body = <div className="bdrawer" role="dialog" aria-label={title}>
    <div className="dh"><h2>{title.toUpperCase()}</h2><span className="sub">{sub}</span>
      {tabs && <div className="fbar" style={{ display: 'flex', border: 0, padding: 0 }}>{tabs}</div>}
      <HQAct className="x" onAction={close}>CLOSE ✕</HQAct></div>
    <div style={{ display: 'grid', gridTemplateColumns: side ? '1.6fr 1fr' : '1fr',
      gridTemplateRows: tall ? 'minmax(0,420px)' : 'minmax(0,300px)' }}>
      <div style={{ overflow: 'auto', minHeight: 0 }}>{children}</div>
      {side && <div className="db" style={{ borderLeft: '2px solid var(--black)', overflow: 'auto' }}>
        {side}</div>}
    </div>
  </div>;
  return layer ? createPortal(body, layer) : body;
}

const BOX_COLS = '40px minmax(0,1.3fr) repeat(3,minmax(0,1fr)) 64px 50px';

/** Matchup box: the chosen matchup's fantasy box score, one side at a time. */
export function BoxDrawer({ sides, week, left, right, close }: { sides: StarterSides; week?: number;
  left?: PulseScore; right?: PulseScore; close: () => void }) {
  const [them, setThem] = useState(false);
  const rows = [...(them ? sides.right : sides.left)].sort((a, b) => bySlot(a.slot, b.slot));
  const name = (r: boolean) => sides.mine ? (r ? 'THEM' : 'YOU') : (r ? sides.rightName : sides.leftName)
    .toUpperCase();
  const score = (s?: PulseScore) => pointsLabel(s?.points ?? s?.provisional);
  return <BDrawer title="Matchup box" sub={`${sides.leftName} vs ${sides.rightName} · week ${week ?? '—'}`}
    close={close} tabs={<>
      <HQAct className="chip" active={!them} onAction={() => setThem(false)}>{name(false)}</HQAct>
      <HQAct className="chip" active={them} onAction={() => setThem(true)}>{name(true)}</HQAct></>}
    side={<>
      <div className="row" style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
        <span className="ell">{sides.leftName}</span><span className="v">{score(left)}</span></div>
      <div className="row" style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
        <span className="ell">{sides.rightName}</span><span className="v">{score(right)}</span></div>
      <span className="d mono" style={{ fontSize: 10, lineHeight: 1.5 }}>Points are the league's own scoring
        of each starter's line. A starter still to play shows his projection.</span></>}>
    {!rows.length ? <St>No lineup is recorded for this side this week.</St>
      : <>
        <div className="sh" style={{ display: 'grid', gridTemplateColumns: BOX_COLS, gap: 8 }}>
          <span>SLOT</span><span>PLAYER</span><span>LINE 1</span><span>LINE 2</span><span>TD / TO</span>
          <span>STATE</span><span style={{ textAlign: 'right' }}>PTS</span></div>
        {rows.map(r => <div key={r.player} className="row" title={r.words}
          style={{ gridTemplateColumns: BOX_COLS }}>
          <span className="pos">{r.slot}</span><span className="ell">{r.name} <small>{r.team}</small></span>
          {boxLine(r.line, r.slot).map((w, i) => <span key={i} className="m ell">{w}</span>)}
          <span className="m">{r.final ? 'done' : r.points !== undefined ? 'playing'
            : r.projection !== undefined ? `proj ${r.projection.toFixed(1)}` : 'to play'}</span>
          <span className="v">{pointsLabel(r.points)}</span></div>)}</>}
  </BDrawer>;
}

/** Schedule: the league grid from last week through five weeks ahead, or one team's season. */
export function ScheduleDrawer({ reading, mine, port, close }: { reading: NativePulse; mine: string;
  port: HostedPort; close: () => void }) {
  const [team, setTeam] = useState(false);
  const now = reading.week ?? 0;
  const season = usePulseRead<PulseSeason>(
    JSON.stringify(['pulse-season', reading.league, reading.season, now]),
    () => readPulseSeason(reading.league, reading.season, now, port));
  const nameOf = (id: string) => reading.seats.find(s => s.franchise === id)?.name ?? id;
  const s = season.reading;
  const weeks = s ? s.schedule.weeks.filter(w => w.games.length).map(w => w.week) : [];
  const shown = weeks.filter(w => w >= now - 1).slice(0, 7);
  const cell = (franchise: string, w: number) => {
    const c = s && cellOf(s, franchise, w, nameOf);
    return c ? `${c.result ? `${c.result} ` : ''}${c.away ? '@ ' : ''}${c.opponent}` : 'bye';
  };
  return <BDrawer title="Schedule" sub={`weeks 1–${reading.lastWeek || '—'} · regular season`} close={close}
    tabs={<><HQAct className="chip" active={!team} onAction={() => setTeam(false)}>LEAGUE</HQAct>
      <HQAct className="chip" active={team} disabled={!mine} onAction={() => setTeam(true)}>
        MY TEAM</HQAct></>}>
    {!s ? <St>{season.error ? `The schedule is unavailable · ${season.error}` : 'Reading the schedule…'}</St>
      : !weeks.length ? <St>No league schedule has been published yet.</St>
      : team ? <table className="t"><thead><tr><th>WK</th><th>OPPONENT</th><th>RESULT</th><th>SCORE</th>
        </tr></thead><tbody>{weeks.map(w => {
          const c = cellOf(s, mine, w, nameOf);
          return <tr key={w}><td>WK {w}</td><td>{c ? `${c.away ? '@ ' : ''}${c.opponent}` : 'bye'}</td>
            <td>{c?.result || (w === now ? 'NOW' : '–')}</td><td>{c?.score}</td></tr>;
        })}</tbody></table>
      : <table className="t"><thead><tr><th>FRANCHISE</th>{shown.map(w => <th key={w}>
        {w === now ? `WK ${w} · NOW` : `WK ${w}`}</th>)}</tr></thead><tbody>
        {reading.seats.map(seat => <tr key={seat.franchise} className={seat.franchise === mine ? 'me' : ''}>
          <td className="w">{seat.name}</td>
          {shown.map(w => <td key={w} className="w d2">{cell(seat.franchise, w)}</td>)}</tr>)}
      </tbody></table>}
  </BDrawer>;
}

/** Stat changes: which starters the final statistics re-scored, and whether it moved the result. */
export function ChangesDrawer({ sides, week, changes, left, right, close }: { sides: StarterSides;
  week?: number; changes: StatChange[]; left?: PulseScore; right?: PulseScore; close: () => void }) {
  const margin = left?.points !== undefined && right?.points !== undefined ? left.points - right.points
    : undefined;
  const names = { left: sides.leftName, right: sides.rightName, mine: sides.mine };
  const who = (c: StatChange) => sides.mine ? (c.left ? 'YOU' : 'THEM') : c.left ? names.left : names.right;
  return <BDrawer title="Stat changes" sub={`last live line vs final stats · week ${week ?? '—'}`}
    close={close} side={<span className="d mono" style={{ fontSize: 10, lineHeight: 1.5 }}>Compares the last
      live line this app held with the final stats, for the starters in this matchup. It is not the
      complete list of NFL stat corrections.</span>}>
    {!changes.length
      ? <St>The final stats scored every starter in this matchup the same as the last live line.</St>
      : <table className="t"><thead><tr><th>WK</th><th>PLAYER</th><th>STAT</th><th>WAS</th><th>NOW</th>
        <th>PTS Δ</th><th>RESULT</th></tr></thead><tbody>
        {changes.map(c => <tr key={c.player}><td>{week}</td><td className="w">{c.name}</td><td>{c.stats}</td>
          <td>{changePoints(c.was)}</td><td>{changePoints(c.now)}</td>
          <td>{changePoints(c.delta, true)} {who(c)}</td>
          <td className="w">{changeVerdict(c, margin, Boolean(left?.recorded || right?.recorded), names)}</td>
        </tr>)}</tbody></table>}
  </BDrawer>;
}
