import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { onHostedTopics } from '../data/hostedReads';
import { myChance, percent, pointsLabel } from '../data/nativePulse';
import { gameTime } from '../now/LiveGame';
import type { Game } from '../now/liveContract';
import { clockWords, followKey, useHostedGames } from '../shell/hostedGames';
import { unavailable } from '../shell/hqModel';
import { HQAct } from '../shell/HQAct';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import { PhoneSheet } from '../shell/PhoneAppFrame';
import { espnTeam, pairStarters, pulseRows, scoreOf, starterRows, type PulseRow, type PulseStarterRow }
  from '../shell/pulseModel';
import { readPulseMatchup, type PulseMatchup } from '../shell/pulseReads';
import { compactLine } from '../shell/statLine';
import './LiveTab.css';

type BriefReader = (league: string, mine: string, port: HostedPort) => Promise<HomeBrief>;
const readLive: BriefReader = (league, mine, port) => readThinHomeBrief(league, mine, port);
type Subscribe = (load: () => void) => () => void;
const topicsFor = (league: string): Subscribe => load =>
  onHostedTopics(topics => { if (topics.some(topic => topic.league === league)) load(); });

/** LiveTab is the phone Live page of mobile mock v1. Every value is one existing read: the league
 *  brief Today's "Live now" uses (results, odds, my lineups) and League Pulse's NFL scoreboard. */
export function LiveTab({ league, port = hostedWorkflow, readBrief = readLive, subscribe }: {
  league?: LeagueChoice; port?: HostedPort; readBrief?: BriefReader; subscribe?: Subscribe;
}) {
  if (!league) return <div className="live6" data-testid="live-tab">
    <p role="status" className="live6__note">{unavailable} · No admitted league.</p></div>;
  return <LiveBody key={league.id} league={league} port={port} readBrief={readBrief}
    subscribe={subscribe ?? topicsFor(league.id)} />;
}

type Sheet = { kind: 'matchups' } | { kind: 'player'; player?: PulseStarterRow; name: string };

function LiveBody({ league, port, readBrief, subscribe }: { league: LeagueChoice; port: HostedPort;
  readBrief: BriefReader; subscribe: Subscribe }) {
  const mine = league.franchise ?? '';
  const changes = useRef(subscribe); changes.current = subscribe;
  const onChange = useCallback<Subscribe>(load => changes.current(load), []);
  const live = useHostedGames(league.id, port, onChange);
  const [brief, setBrief] = useState<HomeBrief>();
  const [failure, setFailure] = useState('');
  const [tick, setTick] = useState(0);
  const [picked, pick] = useState<string>();
  const [box, setBox] = useState<'nfl' | 'lg'>('nfl');
  const [sheet, setSheet] = useState<Sheet>();
  const boxes = useRef<HTMLDivElement>(null);
  useEffect(() => onChange(() => setTick(n => n + 1)), [onChange]);
  useEffect(() => {
    let active = true;
    if (!mine) return;
    readBrief(league.id, mine, port)
      .then(found => { if (active) { setBrief(found); setFailure(''); } })
      .catch(cause => { if (active) setFailure(cause instanceof Error ? cause.message : String(cause)); });
    return () => { active = false; };
  }, [league.id, mine, port, readBrief, tick]);

  const results = brief?.results;
  const current = results !== undefined && results.week === brief?.pulse?.week;
  const games = useMemo(() => current ? live.view?.games ?? [] : [], [current, live.view]);
  const rows = results ? pulseRows(results, mine, brief?.pulse?.seats ?? [], current, games) : [];
  const selected = rows.find(r => r.key === picked) ?? rows.find(r => r.mine) ?? rows[0];
  const sides = useSides(brief, selected, league.id, port, mine);
  const followed = current ? live.view?.followed ?? undefined : undefined;
  const myRows = starterRows(scoreOf(results, mine), brief?.matchup?.lineups[mine],
    brief?.matchup?.projections ?? {});
  const toBoxes = (which: 'nfl' | 'lg') => {
    setBox(which); boxes.current?.scrollIntoView?.({ behavior: 'smooth', block: 'start' });
  };
  const words = !mine ? 'No team of yours in this league'
    : failure ? `${unavailable} · ${failure}`
      : !brief ? 'Reading…'
        : brief.errors.results ? `${unavailable} · ${brief.errors.results}` : '';

  return <div className="live6" data-testid="live-tab">
    <section className="live6__card live6__mym" aria-label="My matchup">
      <div className="live6__head"><b>My matchup{results ? ` · wk ${results.week}` : ''}</b>
        <span>
          <HQAct className="live6__link" label="All matchups" disabled={!rows.length}
            onAction={() => setSheet({ kind: 'matchups' })}>All matchups</HQAct>
          <HQAct className="live6__link" label="Box score" disabled={!rows.length}
            onAction={() => { pick(undefined); toBoxes('lg'); }}>Box score</HQAct>
        </span></div>
      {words ? <p role={failure || brief?.errors.results ? 'alert' : 'status'}
        className="live6__note">{words}</p>
        : <Matchup brief={brief!} mine={mine} />}
      {!words && myRows.length > 0 && <div className="live6__myp">
        {myRows.map(p => {
          const g = gameOf(p, games);
          return <HQAct key={p.player} className={`live6__pl${g && g.id === followed?.id ? ' is-on' : ''}`}
            label={`Watch ${p.name}'s game`} onAction={() => {
              if (g) live.follow(followKey(g));
              setSheet({ kind: 'player', player: p, name: p.name });
            }}>
            <i className="live6__pos">{p.slot}</i><span>{p.name}</span>
            {onField(p, g) && <i className="live6__onf" title="on the field now" aria-label="on the field" />}
            <b>{pointsLabel(p.points)}</b>
          </HQAct>;
        })}
      </div>}
    </section>

    <nav className="live6__strip" aria-label="NFL games"><b className="live6__strip-l">NFL</b>
      <div className="live6__strip-s">
        {!games.length && <span className="live6__strip-w">{!current
          ? 'NFL games show for the week being played'
          : live.error ? `${unavailable} · ${live.error}` : !live.view ? 'Reading the NFL scoreboard…'
            : 'No NFL games are held for this week'}</span>}
        {games.map(g => <HQAct key={g.id} className={`live6__gc${g.id === followed?.id ? ' is-on' : ''}`}
          label={`Follow ${g.away.abbreviation} at ${g.home.abbreviation}`}
          onAction={() => live.follow(followKey(g))}>
          <span>{g.away.abbreviation}<b>{g.status === 'pre' ? '' : g.away.score}</b></span>
          <span>{g.home.abbreviation}<b>{g.status === 'pre' ? '' : g.home.score}</b></span>
          <em>{clockWords(g)}</em>
        </HQAct>)}
      </div></nav>
    <nav className="live6__strip" aria-label="League matchups"><b className="live6__strip-l">LEAGUE</b>
      <div className="live6__strip-s">
        {!rows.length && <span className="live6__strip-w">{words || results?.words
          || 'No matchups are read for this week'}</span>}
        {rows.map(r => <HQAct key={r.key} className={`live6__mc${r.key === selected?.key ? ' is-on' : ''}`
          + `${r.mine ? ' is-me' : ''}`} label={r.game.words || `${r.game.awayName} at ${r.game.homeName}`}
          onAction={() => { pick(r.key); setBox('lg'); }}>
          <span>{r.game.awayName}<b>{pointsLabel(r.away)}</b></span>
          <span>{r.game.homeName}<b>{pointsLabel(r.home)}</b></span>
          <em>{r.mine ? 'YOURS · ' : ''}{r.status}</em>
        </HQAct>)}
      </div></nav>

    <FieldState game={followed} words={!current ? 'Live games show for the week being played.'
      : live.error && !live.view ? `${unavailable} · ${live.error}`
        : !live.view ? 'Reading the NFL scoreboard…'
          : 'Pick a game in the NFL strip to follow it.'} />
    <div className={`live6__fresh${live.error ? ' is-re' : ''}`} role="status">
      {live.error && live.view ? `Reconnecting… showing the last held board · ${live.error}`
        : live.view ? `updated ${live.view.asOf || 'time not supplied'}${live.view.source
          ? ` · ${live.view.source}` : ''}` : ''}
    </div>

    <div className="live6__boxes" ref={boxes}>
      <div className="live6__seg" role="tablist" aria-label="Box scores">
        <HQAct role="tab" selected={box === 'nfl'} className={box === 'nfl' ? 'is-on' : ''}
          onAction={() => setBox('nfl')}>NFL box score</HQAct>
        <HQAct role="tab" selected={box === 'lg'} className={box === 'lg' ? 'is-on' : ''}
          onAction={() => setBox('lg')}>Matchup box score</HQAct>
      </div>
      <section className="live6__panel" aria-label={box === 'nfl' ? 'NFL box score' : 'Matchup box score'}>
        {box === 'nfl' ? <NflBox game={followed} /> : <MatchupBox row={selected} sides={sides} />}
      </section>
    </div>

    <section className="live6__card" aria-label="Play by play">
      <div className="live6__head"><b>Play by play</b><span className="live6__m">newest first</span></div>
      <Plays game={followed} />
    </section>

    {sheet?.kind === 'matchups' && <PhoneSheet id="live6-matchups-title"
      title={`Week ${results?.week ?? '—'} matchups`} onClose={() => setSheet(undefined)}>
      {rows.map(r => {
        const awayUp = (r.away ?? 0) >= (r.home ?? 0);
        return <HQAct key={r.key} className={`live6__row${r.key === selected?.key ? ' is-on' : ''}`}
          label={`${r.game.awayName} ${pointsLabel(r.away)}, ${r.game.homeName} ${pointsLabel(r.home)}`}
          onAction={() => { pick(r.key); setSheet(undefined); toBoxes('lg'); }}>
          <span className={awayUp ? 'is-up' : ''}>{r.game.awayName}</span>
          <b className={awayUp ? 'is-up' : ''}>{pointsLabel(r.away)}</b>
          <span className={awayUp ? '' : 'is-up'}>{r.game.homeName}</span>
          <b className={awayUp ? '' : 'is-up'}>{pointsLabel(r.home)}</b>
          {r.mine && <em>YOUR MATCHUP</em>}
        </HQAct>;
      })}
    </PhoneSheet>}
    {sheet?.kind === 'player' && <PhoneSheet id="live6-player-title" title={sheet.name}
      onClose={() => setSheet(undefined)}>
      <PlayerDetail player={sheet.player} game={gameOf(sheet.player, games)} />
    </PhoneSheet>}
  </div>;
}

function Matchup({ brief, mine }: { brief: HomeBrief; mine: string }) {
  const game = brief.results?.games.find(g => g.home === mine || g.away === mine);
  if (!game) return <p role="status" className="live6__note">
    {brief.results?.bye === mine ? 'Bye week' : 'No game this week'}</p>;
  const home = game.home === mine;
  const score = (side: 'home' | 'away') => game[`${side}Points`] ?? game[`${side}Provisional`];
  const chance = myChance(brief.pulse?.odds, mine);
  const held = chance && chance.week === brief.results?.week ? chance.mine : undefined;
  return <div className="live6__mys">
    <div className="live6__t is-me"><span>{home ? game.homeName : game.awayName} · YOU</span>
      <b>{pointsLabel(score(home ? 'home' : 'away'))}</b></div>
    <div className="live6__wp">
      <div className="live6__bar">
        <i style={{ width: held === undefined ? 0 : `${Math.round(held * 100)}%` }} /></div>
      <div className="live6__wpl"><b>{held === undefined ? 'WIN —' : `${percent(held)} WIN`}</b>
        {held === undefined && <span>{brief.errors.pulse ?? 'No odds for this week'}</span>}
      </div>
    </div>
    <div className="live6__t is-r"><span>{home ? game.awayName : game.homeName}</span>
      <b>{pointsLabel(score(home ? 'away' : 'home'))}</b></div>
  </div>;
}

type Sides = { left: PulseStarterRow[]; right: PulseStarterRow[]; leftName: string; rightName: string;
  leftPoints?: number; rightPoints?: number; words: string };

/** useSides is the chosen matchup's two lineups: my game's from the brief, any other from the same
 *  matchup read League Pulse uses. */
function useSides(brief: HomeBrief | undefined, row: PulseRow | undefined, league: string,
  port: HostedPort, mine: string): Sides {
  const [read, setRead] = useState<{ key: string; matchup?: PulseMatchup; error?: string }>();
  const results = brief?.results;
  const own = row?.mine ? brief?.matchup : undefined;
  useEffect(() => {
    if (!row || !results || !brief?.pulse || own) return;
    let active = true;
    const { away, home } = row.game;
    const starters = (id: string) => scoreOf(results, id)?.starters.map(s => s.player) ?? [];
    readPulseMatchup(league, brief.pulse.season, results.week, [away, home],
      { [away]: starters(away), [home]: starters(home) }, port)
      .then(matchup => { if (active) setRead({ key: row.key, matchup }); })
      .catch(cause => { if (active) setRead({ key: row.key, error: String(cause?.message ?? cause) }); });
    return () => { active = false; };
  }, [row, results, brief?.pulse, own, league, port]);
  if (!row) return { left: [], right: [], leftName: '', rightName: '',
    words: 'No game is read for this week yet.' };
  const matchup = own ?? (read?.key === row.key ? read.matchup : undefined);
  const flip = row.game.home === mine;
  const [l, r] = flip ? [row.game.home, row.game.away] : [row.game.away, row.game.home];
  const side = (id: string) => starterRows(scoreOf(results, id), matchup?.lineups[id],
    matchup?.projections ?? {});
  const total = (id: string) => { const s = scoreOf(results, id); return s?.points ?? s?.provisional; };
  return { left: side(l), right: side(r), leftPoints: total(l), rightPoints: total(r),
    leftName: flip ? row.game.homeName : row.game.awayName,
    rightName: flip ? row.game.awayName : row.game.homeName,
    words: read?.key === row.key && read.error ? `${unavailable} · ${read.error}` : '' };
}

function MatchupBox({ row, sides }: { row?: PulseRow; sides: Sides }) {
  if (!row) return <p className="live6__note">{sides.words}</p>;
  const pairs = pairStarters(sides.left, sides.right).flatMap(g => g.pairs);
  const leftUp = (sides.leftPoints ?? 0) >= (sides.rightPoints ?? 0);
  return <>
    <div className="live6__bhead">
      <div className={leftUp ? 'is-lead' : ''}><span>{sides.leftName}</span>
        <b>{pointsLabel(sides.leftPoints)}</b></div>
      <em>{row.status}</em>
      <div className={`is-r${leftUp ? '' : ' is-lead'}`}><span>{sides.rightName}</span>
        <b>{pointsLabel(sides.rightPoints)}</b></div>
    </div>
    {sides.words && <p role="alert" className="live6__note">{sides.words}</p>}
    {!pairs.length ? <p className="live6__note">{row.game.words || 'No lineup is recorded for this week.'}</p>
      : pairs.map((pair, i) => <div key={`${pair.slot}-${i}`} className="live6__pair">
        <span>{pair.left?.name ?? ''}</span><b>{pointsLabel(pair.left?.points)}</b>
        <i>{pair.slot}</i>
        <b>{pointsLabel(pair.right?.points)}</b><span className="is-r">{pair.right?.name ?? ''}</span>
      </div>)}
  </>;
}

function NflBox({ game }: { game?: Game }) {
  if (!game) return <p className="live6__note">The box score shows for the NFL game you follow.</p>;
  if (!game.box.length) return <p className="live6__note">{game.status === 'pre'
    ? 'The game has not kicked off.' : 'No box score is held for this game yet.'}</p>;
  return <>
    <div className="live6__bhead"><b>{game.away.abbreviation} {game.away.score}</b><em>{clockWords(game)}</em>
      <b className="is-r">{game.home.score} {game.home.abbreviation}</b></div>
    {game.box.map(b => <div key={`${b.team}-${b.category}`} className="live6__nbox">
      <div className="live6__nbox-h"><i>{b.team}</i>{b.category}</div>
      {b.rows.map(r => <div key={r.name} className="live6__nrow"><span>{r.name}</span>
        <em>{r.stats.map((v, i) => `${v} ${b.labels[i] ?? ''}`.trim()).join(' · ')}</em></div>)}
    </div>)}
  </>;
}

function FieldState({ game, words }: { game?: Game; words: string }) {
  if (!game) return <section className="live6__card" aria-label="NFL live">
    <p className="live6__note">{words}</p></section>;
  const s = game.state;
  const known = game.status === 'in' && s !== null && s.toEndzone >= 0
    && [game.home.id, game.away.id].includes(s.possession);
  const offense = known ? (s.possession === game.home.id ? game.home : game.away) : undefined;
  const defense = offense === game.home ? game.away : game.home;
  const ballOn = known && offense ? s.toEndzone > 50 ? `${offense.abbreviation} ${100 - s.toEndzone}`
    : s.toEndzone === 50 ? '50' : `${defense.abbreviation} ${s.toEndzone}` : '—';
  const down = known && s.down > 0 ? `${ordinal(s.down)} & ${s.distance}` : '';
  return <section className="live6__live" aria-label="NFL live">
    <div className="live6__score">
      <div className="live6__side"><b>{game.away.abbreviation}</b><strong>{game.away.score}</strong></div>
      <div className="live6__mid"><b>{clockWords(game)}</b>{down && <span className="live6__dd">{down}</span>}
        {known && <span>{ballOn}</span>}</div>
      <div className="live6__side is-r"><strong>{game.home.score}</strong>
        <b>{game.home.abbreviation}</b></div>
    </div>
    <div className="live6__head live6__pad"><b>NFL live</b>
      <span className="live6__m">
        {game.away.abbreviation} at {game.home.abbreviation} · {gameTime(game)}</span></div>
    <div className="live6__sit">
      <div><span>Down</span><b>{known && s.down > 0 ? ordinal(s.down).toUpperCase() : '—'}</b></div>
      <div><span>To go</span><b>{known && s.down > 0 ? s.distance : '—'}</b></div>
      <div><span>Ball on</span><b>{ballOn}</b></div>
      <div><span>To goal</span><b>{known ? `${s.toEndzone} YD` : '—'}</b></div>
    </div>
    {!known && <p className="live6__note">{game.status === 'pre' ? gameTime(game)
      : game.status === 'post' ? 'Final' : 'Field position not supplied'}</p>}
  </section>;
}

function Plays({ game }: { game?: Game }) {
  if (!game) return <p className="live6__note">Plays show for the game you follow.</p>;
  if (!game.plays.length) return <p className="live6__note">{game.status === 'pre'
    ? 'The game has not kicked off.' : 'No plays are held for this game yet.'}</p>;
  const abbr = (id?: string) => id === game.home.id ? game.home.abbreviation
    : id === game.away.id ? game.away.abbreviation : '·';
  return <ol className="live6__plays">{[...game.plays].sort((a, b) => b.sequence - a.sequence).slice(0, 60)
    .map(p => <li key={p.id}><span className="live6__m">Q{p.period} {p.clock}</span>
      <b>{abbr(p.start?.possession)}</b><span>{p.text}</span>
      <em>{p.awayScore}-{p.homeScore}</em></li>)}</ol>;
}

function PlayerDetail({ player, game }: { player?: PulseStarterRow; game?: Game }) {
  if (!player) return <p className="live6__note">Not available yet.</p>;
  const line = compactLine(player.line, player.slot);
  return <div className="live6__detail">
    <p><b>{player.slot}</b> · {player.team ?? 'NFL team not held'}</p>
    <p className="live6__big">{pointsLabel(player.points)} <small>pts</small></p>
    <p>{line.length ? line.join(' · ') : player.projection !== undefined && !player.final
      ? `proj ${player.projection.toFixed(1)}` : 'No stat line yet'}</p>
    <p className="live6__m">{game
      ? `${game.away.abbreviation} at ${game.home.abbreviation} · ${gameTime(game)}`
      : 'NFL game not on this week\'s board'}</p>
    {player.words && <p className="live6__m">{player.words}</p>}
  </div>;
}

function gameOf(p: PulseStarterRow | undefined, games: Game[]) {
  const club = espnTeam(p?.team);
  return club ? games.find(g => g.away.abbreviation === club || g.home.abbreviation === club) : undefined;
}
/** onField: the starter's club has the ball (offense only; the read holds no defensive snaps). */
function onField(p: PulseStarterRow, g?: Game) {
  if (!g || g.status !== 'in' || !g.state?.possession) return false;
  const club = g.home.abbreviation === espnTeam(p.team) ? g.home : g.away;
  return g.state.possession === club.id && !['DEF', 'DST', 'K'].includes(p.slot);
}
function ordinal(n: number) { return `${n}${['th', 'st', 'nd', 'rd'][n] ?? 'th'}`; }

export default LiveTab;
