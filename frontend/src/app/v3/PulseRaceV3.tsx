// Desktop v3 League Pulse · Race, after the mock's pulse race screen (R.standings, R.power, R.bracket,
// R.road and the Race tool bar). Every row is the league's own read: standings, the playoff bracket,
// the odds read's power ranking and the schedule. The mock's Dynasty and Blend power views, League
// headlines and Why it moved have no read behind them, so they are not drawn.
import { useState, type CSSProperties, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { percent, type PulseBracket, type PulseOdds, type PulseStandings } from '../data/nativePulse';
import { bracketRounds, conferencesOf, divisionGroups, seedRows, tieGroups, tieStake,
  type PulseSeedRow } from '../shell/pulseModel';
import type { NativePulse } from '../shell/pulseReads';
import { BDrawer } from './pulseDrawers';

export type RacePull = 'tb' | 'bk';
const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;

/** ladder is the mock's segmented bar: v (0 to 1) of n cells lit. */
function Ladder({ v, n, cls }: { v: number; n: number; cls?: string }) {
  const lit = Math.round(Math.max(0, Math.min(1, v)) * n);
  return <span className="lad">{Array.from({ length: n }, (_, i) =>
    <i key={i} className={i < lit ? `on ${cls ?? ''}`.trim() : ''} />)}</span>;
}
function Panel({ title, badge, children, foot, fb, style }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: string; fb?: ReactNode; style?: CSSProperties }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" /></div>
    <div className="pb">{fb}{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}
  </section>;
}
const share = (r?: { wins: number; losses: number; ties: number }) =>
  r && r.wins + r.losses + r.ties ? (r.wins + r.ties / 2) / (r.wins + r.losses + r.ties) : 0;
const oddsOf = (odds: PulseOdds | undefined, id: string) => odds?.rows.find(r => r.franchise === id);

export function PulseRaceV3({ reading, mine, badge, pull, onPull }: { reading: NativePulse; mine: string;
  badge: ReactNode; pull?: RacePull; onPull: (p: RacePull) => void }) {
  const conferences = conferencesOf(reading.bracket);
  const standings = reading.standings;
  const own = standings?.rows.find(r => r.franchise === mine)?.conference;
  const [view, setView] = useState<string>('seeds');
  const side = conferences.includes(view) ? view
    : [own, conferences[0]].find(c => c && conferences.includes(c)) ?? '';
  const rows = standings ? seedRows(standings, reading.bracket, side, mine) : [];
  const close = () => { if (pull) onPull(pull); };
  return <>
    <div className="tbar">
      <HQAct className="btn sm" expanded={pull === 'tb'} onAction={() => onPull('tb')}>Tiebreaks</HQAct>
      <HQAct className="btn sm" expanded={pull === 'bk'} onAction={() => onPull('bk')}>Full bracket</HQAct>
      <span className="m sp">{reading.week ? `week ${reading.week} · ` : ''}{reading.lastWeek
        ? `seeds settle after week ${reading.lastWeek}` : reading.weekNote}</span>
    </div>
    <div className="lay" style={{ gridTemplateColumns: 'minmax(0,1.1fr) minmax(0,1fr) minmax(0,1.1fr)',
      gridTemplateRows: 'minmax(0,1.5fr) minmax(0,1fr)' }}>
      <Standings reading={reading} standings={standings} rows={rows} view={view} setView={setView}
        conferences={conferences} mine={mine} badge={badge} />
      <Power odds={reading.odds} mine={mine} badge={badge} errors={reading.errors} />
      <Bracket bracket={reading.bracket} standings={standings} side={side} week={reading.week}
        badge={badge} />
      <Road reading={reading} mine={mine} badge={badge} />
    </div>
    {pull === 'tb' && <TiebreakDrawer standings={standings} rows={rows} side={side} close={close} />}
    {pull === 'bk' && <BracketDrawer bracket={reading.bracket} close={close} />}
  </>;
}

const STAND = '24px minmax(0,1fr) 44px 60px 54px';
function Standings({ reading, standings, rows, view, setView, conferences, mine, badge }: {
  reading: NativePulse; standings?: PulseStandings; rows: PulseSeedRow[]; view: string;
  setView: (v: string) => void; conferences: string[]; mine: string; badge: ReactNode }) {
  const views = [['seeds', 'Seeds'], ['divisions', 'Divisions'],
    ...(conferences.length > 1 ? conferences.map(c => [c, `${c} side`]) : [])];
  const fb = <div className="fbar">{views.map(([k, t]) => <HQAct key={k} className="chip"
    active={view === k || (k === 'seeds' && conferences.includes(view))} onAction={() => setView(k)}>
    {t}</HQAct>)}</div>;
  const line = (r: PulseStandings['rows'][number], n: number, seeded: boolean) =>
    <div key={r.franchise} className={`row${r.franchise === mine ? ' me' : ''}`} title={r.why.join(' ')}
      style={{ gridTemplateColumns: STAND }}>
      <span className="m">{n}</span><span className="ell">{r.name}</span><span className="v">{r.record}</span>
      <span className="m" style={{ textAlign: 'right' }}>{r.pointsFor.toFixed(1)}</span>
      <Ladder v={share(r)} n={6} cls={seeded ? 'g' : ''} /></div>;
  const byId = (id: string) => standings?.rows.find(r => r.franchise === id);
  const places = (reading.bracket?.perConference ?? 0) * Math.max(1, conferences.length);
  return <Panel title="Standings" badge={badge} fb={fb} style={{ gridRow: '1/3' }}
    foot={standings ? `${standings.rows.length} teams${places ? ` · top ${places} make the playoffs` : ''}`
      : undefined}>
    {!standings ? <St>{reading.standingsError ? `The standings are unavailable · ${reading.standingsError}`
      : 'Reading the standings…'}</St>
      : view === 'divisions' ? divisionGroups(standings).map(g => <div key={g.division}>
        <div className="sh">{g.division ? g.division.toUpperCase()
          : 'NO DIVISION SET'} · {g.rows.length}</div>
        {g.rows.map(r => line(r, r.rank, false))}</div>)
      : !rows.length ? <St>{standings.words}</St>
      : rows.map((s, i) => { const r = byId(s.franchise); return r ? line(r, i + 1, s.inField) : null; })}
  </Panel>;
}

function Power({ odds, mine, badge, errors }: { odds?: PulseOdds; mine: string; badge: ReactNode;
  errors: string[] }) {
  const rows = (odds?.rows ?? []).filter(r => r.power).sort((a, b) => (a.power ?? 0) - (b.power ?? 0));
  const top = Math.max(...rows.map(r => r.powerScore), 0.0001);
  return <Panel title="Power rankings" badge={badge} foot={odds?.powerWords}
    fb={<div className="fbar"><HQAct className="chip" active>This season</HQAct></div>}>
    {!rows.length ? <St>{odds ? `The power ranking is unavailable · ${odds.powerWords}`
      : errors.length ? `The power ranking is unavailable · ${errors.join(' · ')}`
      : 'Reading the power ranking…'}</St>
      : rows.map(r => <div key={r.franchise} className={`row${r.franchise === mine ? ' me' : ''}`}
        title={`${r.record} · expected ${r.expected.toFixed(1)} · playoffs ${percent(r.playoffOdds)}`}
        style={{ gridTemplateColumns: '24px minmax(0,1fr) 40px 110px' }}>
        <span className="m">{r.power}</span><span className="ell">{r.name}</span>
        <span className="v">{(r.powerScore * 100).toFixed(1)}</span>
        <Ladder v={r.powerScore / top} n={14} /></div>)}
  </Panel>;
}

function Bracket({ bracket, standings, side, week, badge }: { bracket?: PulseBracket;
  standings?: PulseStandings; side: string; week?: number; badge: ReactNode }) {
  const sides = bracket ? bracketRounds(bracket) : undefined;
  const field = sides?.fields.find(f => f.conference === side) ?? sides?.fields[0];
  const record = (seed?: number) => {
    const id = bracket?.fields.find(f => f.conference === field?.conference)?.seeds
      .find(s => s.seed === seed)?.franchise;
    return id ? standings?.rows.find(r => r.franchise === id)?.record ?? '' : '';
  };
  const gm = (a: { words: string; seed?: number }, b: { words: string; seed?: number }, key: string) =>
    <div className="gm" key={key}>
      <span><b>{a.words}</b><i className="m">{record(a.seed)}</i></span>
      <span>{b.words}<i className="m">{record(b.seed)}</i></span></div>;
  return <Panel title="Playoff picture and bracket" badge={badge}
    style={{ gridColumn: '3', gridRow: '1/3' }}
    foot={!bracket ? undefined : bracket.projected ? `projected · seeds as of week ${week ?? '—'}`
      : bracket.words}>
    {!field ? <St>{bracket?.words || 'Reading the playoffs…'}</St>
      : <div className="brk">{field.rounds.map(r => <div key={r.heading}>
        {r.games.map(g => gm(g.home, g.away, g.id))}</div>)}
        {sides?.final && <div>{gm({ words: sides.final.sides[0] ?? 'winner' },
          { words: sides.final.sides[1] ?? 'winner' }, 'final')}</div>}</div>}
  </Panel>;
}

function Road({ reading, mine, badge }: { reading: NativePulse; mine: string; badge: ReactNode }) {
  const name = (id: string) => reading.seats.find(s => s.franchise === id)?.name ?? id;
  const opp = (id: string) => reading.standings?.rows.find(r => r.franchise === id);
  const ahead = reading.fixtures.filter(g => !reading.week || g.week >= reading.week).slice(0, 6);
  const chance = (week: number, id: string) => {
    const g = reading.odds?.games.find(x => x.week === week && (x.home === mine || x.away === mine)
      && (x.home === id || x.away === id));
    return g ? (g.home === mine ? g.homeWin : 1 - g.homeWin) : undefined;
  };
  return <Panel title="My road" badge={badge} foot="bar = your win chance">
    {!mine ? <St>You have no team in this league, so there is no road to show.</St>
      : reading.scheduleError ? <St>{`The schedule is unavailable · ${reading.scheduleError}`}</St>
      : !ahead.length ? <St>{reading.fixtures.length ? 'Every week on your schedule has been played.'
        : reading.scheduleNote || 'No schedule is published for your team yet.'}</St>
      : ahead.map(g => {
        const w = g.opponent ? chance(g.week, g.opponent) : undefined;
        return <div key={g.week} className="row"
          style={{ gridTemplateColumns: '50px minmax(0,1fr) 50px 80px' }}>
          <span className="m">WK {g.week}</span>
          <span className="ell">{g.opponent ? `${g.venue === 'Away' ? '@ ' : 'vs '}${name(g.opponent)}`
            : 'Bye'}</span>
          <span className="m">{g.opponent ? opp(g.opponent)?.record ?? '' : ''}</span>
          {w === undefined ? <span /> : <Ladder v={w} n={8} cls={w >= 0.5 ? 'g' : 'a'} />}</div>;
      })}
  </Panel>;
}

function TiebreakDrawer({ standings, rows, side, close }: { standings?: PulseStandings;
  rows: PulseSeedRow[]; side: string; close: () => void }) {
  const ties = standings ? tieGroups(standings, rows) : [];
  return <BDrawer title="Tiebreaks" sub={`${side ? `${side} side · ` : ''}how ties resolve today`}
    close={close}
    side={standings && <span className="d mono" style={{ fontSize: 10, lineHeight: 1.5 }}>
      Tie-break order: {standings.criteria.join(' → ')}.
      {standings.noData.length ? ` ${standings.noData.join(' ')}` : ''}</span>}>
    {!ties.length ? <St>{standings ? 'No two franchises on this side are level on record.'
      : 'Reading the standings…'}</St>
      : ties.map(t => <div key={t.detail} className="row" title={t.detail}
        style={{ gridTemplateColumns: '110px minmax(0,1fr) auto auto' }}>
        <span className="m">{t.teams[0].record} · {t.teams.length} LEVEL</span>
        <span className="ell">{t.teams.map(x => `${x.name} ${x.seed ? `#${x.seed}` : 'out'}`)
          .join(t.split ? ' ▸ ' : ' = ')}</span>
        <span className="m">{t.split ? `WON ON ${t.step.toUpperCase()}${t.edge ? ` · ${t.edge}` : ''}`
          : 'STILL LEVEL'}</span>
        {tieStake(t) ? <span className="tag am">{tieStake(t)}</span> : <span />}</div>)}
  </BDrawer>;
}

function BracketDrawer({ bracket, close }: { bracket?: PulseBracket; close: () => void }) {
  const sides = bracket ? bracketRounds(bracket) : undefined;
  return <BDrawer title="Full bracket" close={close}
    sub={!bracket ? 'reading the playoffs' : !bracket.projected ? bracket.words
      : `projected from today's seeds · ${bracket.perConference} per side, ${bracket.byeSeeds} bye`}>
    {!sides?.fields.length ? <St>{bracket?.words || 'Reading the playoffs…'}</St>
      : <div className="brk" style={{ padding: 10 }}>{sides.fields.flatMap(f => f.rounds.map(r =>
        <div key={`${f.conference}${r.heading}`}><div className="sh">
          {f.conference ? `${f.conference.toUpperCase()} · ` : ''}{r.heading}</div>
          {r.games.map(g => <div className="gm" key={g.id}>
            <span><b>{g.home.words}</b><i className="m">{g.home.seed ? `#${g.home.seed}` : ''}</i></span>
            <span>{g.away.words}<i className="m">{g.away.seed ? `#${g.away.seed}` : ''}</i></span></div>)}
        </div>))}
        {sides.final && <div><div className="sh">{sides.final.heading}</div><div className="gm">
          {sides.final.sides.map(s => <span key={s}>{s}</span>)}</div></div>}</div>}
  </BDrawer>;
}
