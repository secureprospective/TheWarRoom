// Desktop v3 League Pulse, after TheWarRoom-mock-v3-FINAL.html (screens.js pulse.now and its later
// overrides): tab strip, Matchups board, the followed NFL game's field, the big matchup card with both
// starting lineups, In this game, the play tracker and the pull tabs. The reads, live updates and the
// matchup model are League Pulse's own (NativeLeaguePulse, pulseModel, hostedGames); only the look is v3.
import { useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { percent, pointsLabel, type PulseOdds, type PulseResults } from '../data/nativePulse';
import { applyFilter, espnTeam, pulseRows, PULSE_FILTERS, scoreOf, statChanges,
  type PulseFilter, type PulseRow,
  type PulseStarterRow } from '../shell/pulseModel';
import { useStarterSides, type StarterSides } from '../shell/PulseNow';
import { clockWords } from '../shell/hostedGames';
import { fieldPositions } from '../now/LiveGame';
import { compactLine } from '../shell/statLine';
import type { Game, LiveView } from '../now/liveContract';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { PulseTab } from '../shell/pulseReads';
import type { NativePulse } from '../shell/pulseReads';
import { Icon } from './icons';
import { BoxDrawer, ChangesDrawer, ScheduleDrawer } from './pulseDrawers';

type Reading<T> = { reading?: T; pending?: boolean; error?: string };
export type PulsePull = 'box' | 'sched' | 'corr';

/** seg is the mock's seven-segment number: the digits over a faint 8.8 ghost. */
export function Seg({ v, size }: { v: string; size: number }) {
  return <span className="seg" style={{ fontSize: size }} data-g={v.replace(/\d/g, '8')}>
    <span>{v}</span></span>;
}
const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
const Lamp = ({ live, words }: { live: boolean; words: string }) =>
  <span className="lv"><i className={`lamp${live ? ' live' : ''}`} /><span>{words}</span></span>;

/** The screen's tab strip: title, Now and Race, and the league's badge on the right. */
export function PulseTabs({ tab, navigate, badge, kind }: { tab: PulseTab; navigate?: (t: PulseTab) => void;
  badge: ReactNode; kind: string }) {
  return <div className="ws-tabs"><span className="ttl"><Icon n="pulse" />LEAGUE PULSE</span>
    {(['now', 'race'] as const).map(t => <HQAct key={t} className="tb" selected={tab === t}
      onAction={() => navigate?.(t)}>{t === 'now' ? 'Now' : 'Race'}</HQAct>)}
    <div className="tools"><span className="d mono" style={{ fontSize: 10, letterSpacing: '.08em' }}>
      {badge} {kind}</span></div></div>;
}

export function PulseNowV3({ reading, mine, week, results, league, port, live, viewer, badge, pull,
  onPull }: {
  reading: NativePulse; mine: string; week?: number; results: Reading<PulseResults>; league: string;
  port: HostedPort; live: { view?: LiveView; error?: string };
  viewer?: { mine: string; commissioner: boolean }; badge: ReactNode;
  pull?: PulsePull; onPull: (p: PulsePull) => void;
}) {
  const [filter, setFilter] = useState<PulseFilter>('all');
  const [picked, pick] = useState<string>();
  const current = week === reading.week;
  const games = current ? live.view?.games ?? [] : [];
  const followed = current ? live.view?.followed ?? undefined : undefined;
  const rows = results.reading ? pulseRows(results.reading, mine, reading.seats, current, games) : [];
  const selected = rows.find(r => r.key === picked) ?? rows.find(r => r.mine) ?? rows[0];
  const sides = useStarterSides(reading, week, results.reading, selected, league, port, mine, viewer);
  const odds = week === reading.odds?.week ? reading.odds : undefined;
  const anyLive = rows.some(r => r.status === 'LIVE');
  const changeList = statChanges(sides.left, sides.right), changes = changeList.length;
  const left = scoreOf(results.reading, sides.leftId ?? '');
  const right = scoreOf(results.reading, sides.rightId ?? '');
  const close = () => { if (pull) onPull(pull); };
  return <>
    <div className="pnow">
      <Board rows={rows} filter={filter} setFilter={setFilter} selected={selected} pick={pick} odds={odds}
        results={results} badge={badge} live={anyLive} week={week} />
      <div className="pcenter">
        <NflPanel game={followed} error={current ? live.error : undefined} reading={!live.view} />
        <MatchupCard row={selected} sides={sides} results={results.reading} odds={odds} games={games}
          badge={badge}
          back={rows.some(r => r.mine) && !selected?.mine ? () => pick(undefined) : undefined} />
      </div>
      <div className="pright">
        <InThisGame game={followed} sides={sides} badge={badge} />
        <PlayTracker game={followed} scoring={live.view?.scoring ?? []} badge={badge} />
      </div>
    </div>
    <div className="pulls">
      <HQAct className="pull" expanded={pull === 'box'} onAction={() => onPull('box')}>▴ MATCHUP BOX</HQAct>
      <HQAct className="pull" expanded={pull === 'sched'} onAction={() => onPull('sched')}>
        ▴ SCHEDULE</HQAct>
      <HQAct className="pull" expanded={pull === 'corr'} onAction={() => onPull('corr')}>
        {`▴ STAT CHANGES${changes ? ` ${changes}` : ''}`}</HQAct>
    </div>
    {pull === 'box' && <BoxDrawer sides={sides} week={week} left={left} right={right} close={close} />}
    {pull === 'sched' && <ScheduleDrawer reading={reading} mine={mine} port={port} close={close} />}
    {pull === 'corr' && <ChangesDrawer sides={sides} week={week} changes={changeList} left={left}
      right={right} close={close} />}
  </>;
}

function Board({ rows, filter, setFilter, selected, pick, odds, results, badge, live, week }: {
  rows: PulseRow[]; filter: PulseFilter; setFilter: (f: PulseFilter) => void; selected?: PulseRow;
  pick: (k: string) => void; odds?: PulseOdds; results: Reading<PulseResults>; badge: ReactNode;
  live: boolean; week?: number }) {
  const shown = applyFilter(rows, filter);
  const mineRow = shown.find(r => r.mine);
  const rest = shown.filter(r => !r.mine);
  const row = (r: PulseRow) => {
    const home = odds?.games.find(g => g.home === r.game.home && g.away === r.game.away)?.homeWin;
    const away = r.decided || home === undefined ? undefined : 1 - home;
    const tone = (p?: number) => p === undefined ? '' : p >= 0.65 ? ' g' : p <= 0.35 ? ' a' : '';
    return <HQAct key={r.key}
      className={`mrow${r.key === selected?.key ? ' sel' : ''}${r.mine ? ' mine' : ''}`}
      label={r.game.words} onAction={() => pick(r.key)}>
      <span className="ell">{r.game.awayName}</span><b className="mono">{pointsLabel(r.away)}</b>
      <span className={`w${tone(away)}`}>{away === undefined ? '' : percent(away)}</span>
      <span className="ell">{r.game.homeName}</span><b className="mono">{pointsLabel(r.home)}</b>
      <span className={`w${tone(home)}`}>
        {away === undefined || home === undefined ? '' : percent(home)}</span>
    </HQAct>;
  };
  const body = !results.reading
    ? <St>{results.error ? `The week's games are unavailable · ${results.error}`
      : results.pending ? 'Reading the week…' : 'No week is read yet.'}</St>
    : !results.reading.published ? <St>{`${results.reading.words}. Games appear here as soon as the `
      + "league's schedule is published."}</St>
    : !rows.length ? <St>{results.reading.words}</St>
    : !shown.length ? <St>No games match this filter.</St>
    : <>{mineRow && <><div className="sh">MY MATCHUP</div>{row(mineRow)}</>}
      <div className="sh">THE LEAGUE · {rest.length}</div>{rest.map(row)}</>;
  return <section className="pc panel sz-L" data-panel="board">
    <div className="ph">{badge}<b className="pt">Matchups</b><span className="sp" />
      <Lamp live={live} words={live ? 'LIVE' : week ? `WEEK ${week}` : 'NO WEEK'} /></div>
    <div className="fbar" style={{ display: 'flex' }}>{PULSE_FILTERS.map(f => <HQAct key={f.key}
      className="chip" active={filter === f.key} onAction={() => setFilter(f.key)}>{f.label}</HQAct>)}</div>
    <div className="pb" style={{ overflow: 'auto' }}>{body}</div>
  </section>;
}

const at = (yards: number) => `${7 + Math.max(0, Math.min(100, yards)) * 0.86}%`;
const ordinal = (n: number) => `${n}${['th', 'st', 'nd', 'rd'][n] ?? 'th'}`;

function NflPanel({ game, error, reading }: { game?: Game; error?: string; reading: boolean }) {
  if (!game) return <section className="pc panel sz-L" data-panel="nfl">
    <div className="ph"><b className="pt">NFL</b><span className="sp" />
      <span className="d mono" style={{ fontSize: 10 }}>FOLLOWING · no game</span></div>
    <div className="field"><div className="ez" /><div className="ez r" />
      <span className="dn">{error ? `NFL scoreboard unavailable · ${error}`
        : reading ? 'Reading the NFL scoreboard…'
        : 'The field shows while an NFL game is being played.'}</span></div>
  </section>;
  const s = game.state;
  const known = game.status === 'in' && s !== null && s.toEndzone >= 0
    && [game.home.id, game.away.id].includes(s.possession);
  const spot = known ? fieldPositions(s, s.possession === game.home.id) : undefined;
  const team = known ? (s.possession === game.home.id ? game.home : game.away).abbreviation : '';
  return <section className="pc panel sz-L" data-panel="nfl">
    <div className="ph"><b className="pt">
      {`${game.away.abbreviation} ${game.away.score} · ${game.home.abbreviation} ${game.home.score}`}</b>
      <span className="d mono" style={{ fontSize: 10 }}>{clockWords(game)}</span>
      <span className="sp" /><span className="d mono ell" style={{ fontSize: 10 }}>
        FOLLOWING · {game.last?.text ?? 'no play yet'}</span></div>
    <div className="field"><div className="ez">{game.away.abbreviation}</div>
      <div className="ez r">{game.home.abbreviation}</div>
      {spot && s && <><div className="los" style={{ left: at(spot.line) }} />
        {s.down > 0 && <div className="ftd" style={{ left: at(spot.first) }} />}
        <div className="ball" style={{ left: at(spot.line) }} /></>}
      <span className="dn">{known && s ? `${team} · ${s.down > 0 ? `${ordinal(s.down)} & ${s.distance}`
        : 'between downs'}` : clockWords(game)}</span></div>
  </section>;
}

/** onField: the starter's club is in a live game and has the ball (offense) or is defending (defense). */
function onField(p: PulseStarterRow, games: Game[], defense: boolean) {
  const club = espnTeam(p.team);
  const g = games.find(x => x.status === 'in'
    && (x.home.abbreviation === club || x.away.abbreviation === club));
  if (!g?.state?.possession) return false;
  const side = g.home.abbreviation === club ? g.home : g.away;
  return (g.state.possession === side.id) !== defense;
}
const DEFENSE = new Set(['DL', 'LB', 'DB', 'CB', 'S', 'DE', 'DT', 'IDP']);
const lineOf = (p: PulseStarterRow) => {
  const parts = compactLine(p.line, p.slot);
  return parts.length ? parts.join(' | ') : p.final ? '—'
    : p.projection !== undefined ? `proj ${p.projection.toFixed(1)}` : 'later';
};

function Lineup({ rows, games }: { rows: PulseStarterRow[]; games: Game[] }) {
  const firstDef = rows.findIndex(r => DEFENSE.has(r.slot));
  return <div>{rows.map((p, i) => <div key={p.player}
    className={`xr${i === firstDef && i > 0 ? ' defs' : ''}`}
    title={p.words}>
    <span className="pos">{p.slot}</span>
    <span className="nm ell">{p.name} <small>{p.team}</small>
      {onField(p, games, DEFENSE.has(p.slot)) && <i className="ball2" />}</span>
    <span className="ln ell">{lineOf(p)}</span><b className="mono">{pointsLabel(p.points)}</b>
  </div>)}</div>;
}

function MatchupCard({ row, sides, results, odds, games, badge, back }: { row?: PulseRow; sides: StarterSides;
  results?: PulseResults; odds?: PulseOdds; games: Game[]; badge: ReactNode; back?: () => void }) {
  const left = scoreOf(results, sides.leftId ?? ''), right = scoreOf(results, sides.rightId ?? '');
  const proj = (rs: PulseStarterRow[]) => rs.reduce((n, r) => n + (r.projection ?? 0), 0);
  const g = row && odds?.games.find(x => x.home === row.game.home && x.away === row.game.away);
  const leftHome = row?.game.home === sides.leftId;
  const lw = g && !row?.decided ? (leftHome ? g.homeWin : 1 - g.homeWin) : undefined;
  const pts = (s?: typeof left) => (s?.points ?? s?.provisional ?? 0).toFixed(1);
  const words = (w: number | undefined, rs: PulseStarterRow[]) =>
    `${w === undefined ? '' : `${percent(w)} · `}proj ${proj(rs).toFixed(1)}`;
  const empty = !left?.starters.length && !right?.starters.length;
  return <section className="pc panel sz-L pcard-center" data-panel="matchup">
    <div className="ph">{badge}<b className="pt">{sides.mine ? 'My matchup' : 'Matchup'}</b>
      <span className="sp" />
      {back && <HQAct className="btn sm" onAction={back}>MY MATCHUP ‹</HQAct>}</div>
    {!row ? <St>No game is read for this week yet.</St> : <>
      <div className="bigmu">
        <div><span className="nm">{sides.leftName}</span><Seg v={pts(left)} size={52} />
          <span className="w">{words(lw, sides.left)}</span></div>
        <div className="mid"><span className="chipq">{row.status}</span>
          <div className="wbar" style={{ margin: '6px 0', width: 180 }}>
            {lw !== undefined && <i style={{ width: percent(lw) }} />}</div></div>
        <div className="rt"><span className="nm">{sides.rightName}</span><Seg v={pts(right)} size={52} />
          <span className="w">{words(lw === undefined ? undefined : 1 - lw, sides.right)}</span></div>
      </div>
      {empty ? <St>{`${row.game.words} Starters appear once both franchises have a lineup recorded for `
        + 'this week.'}</St>
        
        : sides.private && !sides.left.length 
          ? <St>Lineups are private until kickoff. Scores stay public.</St>
        : <div className="pb xv" style={{ overflow: 'auto' }}>
          <Lineup rows={sides.left} games={games} /><Lineup rows={sides.right} games={games} /></div>}
      <div className="pf"><span>offense · defense groups ·{' '}
        <Icon n="play" /> football = on the field now</span></div>
    </>}
  </section>;
}

function InThisGame({ game, sides, badge }: { game?: Game; sides: StarterSides; badge: ReactNode }) {
  const teams = game ? [game.away.abbreviation, game.home.abbreviation] : [];
  const players = [...sides.left, ...sides.right].filter(r => teams.includes(espnTeam(r.team)));
  return <section className="pc panel sz-L" data-panel="ingame">
    <div className="ph">{badge}<b className="pt">In this game</b><span className="sp" />
      {game?.status === 'in' && <Lamp live words="LIVE" />}</div>
    <div className="pb">{!game ? <St>Follow an NFL game to see who in this matchup is playing in it.</St>
      : !players.length ? <St>{sides.reading ? 'Reading this matchup’s lineups…'
        : `No starter in this matchup plays in ${teams.join(' at ')}.`}</St>
      : players.map(p => <div key={p.player} className="row" title={p.words}
        style={{ gridTemplateColumns: '30px minmax(0,1fr) auto 44px' }}>
        <span className="pos">{p.slot}</span><span className="ell">{p.name}</span>
        <span className="m">{compactLine(p.line, p.slot).join(' · ')}</span>
        <span className="v">{pointsLabel(p.points)}</span></div>)}</div>
    <div className="pf"><span className="ell">follows the game on the field</span></div>
  </section>;
}

const TAG: [RegExp, string][] = [[/touchdown/i, 'TD'], [/field goal is good/i, 'FG'], [/safety/i, 'SAF'],
  [/sacked/i, 'SACK'], [/intercepted/i, 'INT'], [/fumbles/i, 'FUM']];

function PlayTracker({ game, scoring, badge }: { game?: Game; scoring: { id: string }[]; badge: ReactNode }) {
  const scored = new Set(scoring.map(p => p.id));
  const abbr = (id?: string) => id === game?.home.id ? game?.home.abbreviation
    : id === game?.away.id ? game?.away.abbreviation : '';
  const plays = game ? [...game.plays].sort((a, b) => b.sequence - a.sequence).slice(0, 60) : [];
  return <section className="pc panel sz-L" data-panel="pbp">
    <div className="ph">{badge}<b className="pt">Play tracker</b><span className="sp" />
      {game?.status === 'in' && <Lamp live words="LIVE" />}</div>
    <div className="pb">{!game ? <St>Plays show for the game you follow.</St>
      : !plays.length ? <St>{game.status === 'pre' ? 'The game has not kicked off.'
        : 'No plays are held for this game yet.'}</St>
      : plays.map(p => {
        const tag = TAG.find(([re]) => re.test(p.text))?.[1];
        return <div key={p.id} className={`pbp${scored.has(p.id) ? ' big' : ''}`}>
          <span className="m">Q{p.period} {p.clock}</span><b>{abbr(p.start?.possession) || '·'}</b>
          <span style={{ whiteSpace: 'normal' }}>{p.text}</span>
          {tag ? <span className="tag o">{tag}</span> : <span />}</div>;
      })}</div>
    <div className="pf"><span className="ell">{game ? `${game.away.abbreviation} @ ${game.home.abbreviation}`
      + ' · newest first' : 'newest first'}</span></div>
  </section>;
}
