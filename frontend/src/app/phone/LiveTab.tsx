import { useCallback, useEffect, useMemo, useRef, useState, type RefObject } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { onHostedTopics } from '../data/hostedReads';
import { errorCodeFor, errorMessage } from '../data/errorCodes';
import { FailureAlert } from '../data/FailureAlert';
import { myChance, pointsLabel } from '../data/nativePulse';
import { gameTime } from '../now/LiveGame';
import type { Game } from '../now/liveContract';
import { clockWords, followKey, useHostedGames, useNow } from '../shell/hostedGames';
import { unavailable } from '../shell/hqModel';
import { HQAct } from '../shell/HQAct';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import { PhoneSheet } from '../shell/PhoneAppFrame';
import { espnTeam, pairStarters, pulseRows, scoreOf, starterRows, type PulseRow, type PulseStarterRow }
  from '../shell/pulseModel';
import { readPulseMatchup, type PulseMatchup } from '../shell/pulseReads';
import { compactLine } from '../shell/statLine';
import { LiveField, useReducedMotion } from './LiveField';
import { LivePregame, type PregameGuy } from './LivePregame';
import { useAnimatedValue } from './liveMotion';
import './LiveTab.css';

type BriefReader = (league: string, mine: string, port: HostedPort) => Promise<HomeBrief>;
const readLive: BriefReader = (league, mine, port) => readThinHomeBrief(league, mine, port);
type Subscribe = (load: () => void) => () => void;
const topicsFor = (league: string): Subscribe => load =>
  onHostedTopics(topics => { if (topics.some(topic => topic.league === league)) load(); });

/** LiveTab is the phone Live page of mobile mock v1. Every value is one existing read: the league
 *  brief Today's "Live now" uses (results, odds, my lineups) and League Pulse's NFL scoreboard. */
export function LiveTab({ league, port = hostedWorkflow, readBrief = readLive, subscribe, reduced }: {
  league?: LeagueChoice; port?: HostedPort; readBrief?: BriefReader; subscribe?: Subscribe;
  reduced?: boolean;
}) {
  if (!league) return <div className="live6" data-testid="live-tab">
    <p role="status" className="live6__note">{unavailable} · No admitted league.</p></div>;
  return <LiveBody key={league.id} league={league} port={port} readBrief={readBrief}
    subscribe={subscribe ?? topicsFor(league.id)} forcedReduced={reduced} />;
}

type Sheet = { kind: 'matchups' } | { kind: 'player'; player?: PulseStarterRow; name: string };

function LiveBody({ league, port, readBrief, subscribe, forcedReduced }: {
  league: LeagueChoice; port: HostedPort; readBrief: BriefReader; subscribe: Subscribe;
  forcedReduced?: boolean;
}) {
  const prefersReduced = useReducedMotion();
  const reduced = forcedReduced ?? prefersReduced;
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
  const now = useNow(30_000);
  const boxes = useRef<HTMLDivElement>(null);
  const field = useRef<HTMLElement>(null);
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
  const pregameGuys = useMemo(
    () => followed?.status === 'pre' ? matchupGuysInGame(followed, selected, sides, brief, mine) : [],
    [followed, selected, sides, brief, mine],
  );
  const preferKey = useMemo(() => {
    if (!games.length || games.some(g => g.status === 'in')) return '';
    const pre = games.filter(g => g.status === 'pre');
    if (!pre.length) return '';
    const featured = pre.find(g => matchupGuysInGame(g, selected, sides, brief, mine).length > 0);
    return followKey(featured ?? pre[0]);
  }, [games, selected, sides, brief, mine]);
  useEffect(() => {
    if (preferKey) live.prefer(preferKey);
  }, [preferKey, live.prefer]);
  const toBoxes = (which: 'nfl' | 'lg') => {
    setBox(which);
    boxes.current?.scrollIntoView?.({ behavior: reduced ? 'auto' : 'smooth', block: 'start' });
  };
  const words = !mine ? 'No team of yours in this league'
    : failure ? `${unavailable} · ${failure}`
      : !brief ? 'Reading…'
        : brief.errors.results ? `${unavailable} · ${brief.errors.results}` : '';

  return <div className={`live6${reduced ? ' rm' : ''}`} data-testid="live-tab">
    <section className="panel mym live6__mym" aria-label="My matchup">
      <div className="bhead live6__head">
        <HQAct className={`lbl live6__mine${selected && !selected.mine ? ' is-away' : ''}`}
          label="Show my matchup" disabled={!rows.length} onAction={() => pick(undefined)}>
          {selected && !selected.mine ? '‹ My matchup' : 'My matchup'}{results ? ` · wk ${results.week}` : ''}
        </HQAct>
        <span>
          <HQAct className="live6__link" label="All matchups" disabled={!rows.length}
            onAction={() => setSheet({ kind: 'matchups' })}>All matchups</HQAct>
          <HQAct className="live6__link" label="Box score" disabled={!rows.length}
            onAction={() => { pick(undefined); toBoxes('lg'); }}>Box score</HQAct>
        </span></div>
      {words ? (failure || brief?.errors.results)
        ? <FailureAlert
            message={failure || brief!.errors.results!}
            prefix={`${unavailable} · `}
            className="live6__note"
            league={league.id}
            leagueName={league.name}
            screen="Phone · Live"
            route="phone/live"
            port={port}
          />
        : <p role="status" className="live6__note">{words}</p>
        : <Matchup brief={brief!} mine={mine} focus={selected?.game} reduced={reduced} />}
    </section>

    <nav className="strip live6__strip" aria-label="NFL games"><b className="strip-l live6__strip-l">NFL</b>
      <div className="strip-s live6__strip-s">
        {!games.length && <span className="live6__strip-w">{!current
          ? 'NFL games show for the week being played'
          : live.error ? `${unavailable} · ${live.error}` : !live.view ? 'Reading the NFL scoreboard…'
            : 'No NFL games are held for this week'}</span>}
        {games.map(g => <HQAct key={g.id}
          className={`gc live6__gc${g.id === followed?.id ? ' on is-on' : ''}`}
          label={`Follow ${g.away.abbreviation} at ${g.home.abbreviation}`}
          onAction={() => live.follow(followKey(g))}>
          <span>{g.away.abbreviation}<b>{g.status === 'pre' ? '' : g.away.score}</b></span>
          <span>{g.home.abbreviation}<b>{g.status === 'pre' ? '' : g.home.score}</b></span>
          <em>{clockWords(g)}</em>
        </HQAct>)}
      </div></nav>
    <FieldState game={followed} anchor={field} reduced={reduced} guys={pregameGuys} now={now}
      fresh={live.heldAt === undefined ? '' : freshness(now - live.heldAt)}
      words={!current ? 'Live games show for the week being played.'
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
        {box === 'nfl' ? <NflBox game={followed} />
          : <MatchupBox row={selected} sides={sides} league={league.id} port={port} />}
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

/** Matchup is the score row for the matchup picked in All matchups, or mine when none is picked. */
function Matchup({ brief, mine, focus, reduced }: {
  brief: HomeBrief; mine: string; focus?: { home: string; away: string }; reduced: boolean;
}) {
  const games = brief.results?.games ?? [];
  const game = (focus && games.find(g => g.home === focus.home && g.away === focus.away))
    ?? games.find(g => g.home === mine || g.away === mine);
  const lead = game && (game.home === mine || game.away === mine) ? mine : game?.away ?? '';
  const home = game?.home === lead;
  const myId = game ? (home ? game.home : game.away) : '';
  const theirId = game ? (home ? game.away : game.home) : '';
  const score = (side: 'home' | 'away') =>
    game ? game[`${side}Points`] ?? game[`${side}Provisional`] : undefined;
  const myScore = score(home ? 'home' : 'away');
  const theirScore = score(home ? 'away' : 'home');
  const chance = myChance(brief.pulse?.odds, lead);
  const held = chance && chance.week === brief.results?.week ? chance.mine : undefined;
  const heldPct = held === undefined ? undefined : Math.round(held * 100);
  const ctx = `${brief.results?.league ?? ''}:${brief.results?.week ?? ''}:${myId}:${theirId}`;

  const myAnim = useAnimatedValue(myScore, `${ctx}:me`, { reduced, decimals: 1 });
  const theirAnim = useAnimatedValue(theirScore, `${ctx}:opp`, { reduced, decimals: 1 });
  const winAnim = useAnimatedValue(heldPct, `${ctx}:win`, { reduced, decimals: 0 });

  if (!game) return <p role="status" className="live6__note">
    {brief.results?.bye === mine ? 'Bye week' : 'No game this week'}</p>;

  const tagFor = (id: string, fallback: string) =>
    brief.pulse?.seats.find(s => s.franchise === id)?.abbreviation?.trim() || fallback;
  const myLabel = tagFor(myId, home ? game.homeName : game.awayName);
  const theirLabel = tagFor(theirId, home ? game.awayName : game.homeName);
  const projPace = projectedPair(brief, myId, theirId);

  return <div className="mys live6__mys">
    <div className={`t live6__t${lead === mine ? ' me is-me' : ''}`}>
      <span><span className="live6__nm">{myLabel}</span>{lead === mine ? ' · YOU' : ''}</span>
      <b key={myAnim.flashKey} className={myAnim.changing ? 'chg' : undefined}>
        {pointsLabel(myAnim.display)}</b>
    </div>
    <div className="wp live6__wp">
      <div className="wp-bar live6__bar" role="img"
        aria-label={heldPct === undefined ? 'Win chance unavailable' : `${heldPct}% win chance`}>
        <i style={{ width: heldPct === undefined ? '0%' : `${heldPct}%` }} />
      </div>
      <div className="wp-l live6__wpl">
        <b key={winAnim.flashKey} className={winAnim.changing ? 'chg' : undefined}>
          {winAnim.display === undefined ? 'WIN —' : `${winAnim.display}% WIN`}</b>
        {held === undefined
          ? <span>{brief.errors.pulse ?? 'No odds for this week'}</span>
          : projPace ? <span>{projPace}</span> : null}
      </div>
    </div>
    <div className="t r live6__t is-r">
      <span>{theirLabel}</span>
      <b key={theirAnim.flashKey} className={theirAnim.changing ? 'chg' : undefined}>
        {pointsLabel(theirAnim.display)}</b>
    </div>
  </div>;
}

/** projectedPair sums held starter projections for both sides only when every starter holds one. */
function projectedPair(brief: HomeBrief, myId: string, theirId: string): string {
  const proj = brief.matchup?.projections;
  if (!proj) return '';
  const totalFor = (id: string): number | undefined => {
    const starters = scoreOf(brief.results, id)?.starters ?? [];
    if (!starters.length) return undefined;
    let sum = 0;
    for (const s of starters) {
      const value = proj[s.player];
      if (value === undefined) return undefined;
      sum += value;
    }
    return sum;
  };
  const mine = totalFor(myId), theirs = totalFor(theirId);
  return mine === undefined || theirs === undefined
    ? '' : `PROJ ${Math.round(mine)}–${Math.round(theirs)}`;
}

type Sides = { left: PulseStarterRow[]; right: PulseStarterRow[]; leftName: string; rightName: string;
  leftPoints?: number; rightPoints?: number; words: string; error?: string; errorCode?: string };

/** useSides is the chosen matchup's two lineups: my game's from the brief, any other from the same
 *  matchup read League Pulse uses. */
function useSides(brief: HomeBrief | undefined, row: PulseRow | undefined, league: string,
  port: HostedPort, mine: string): Sides {
  const [read, setRead] = useState<{
    key: string; matchup?: PulseMatchup; error?: string; errorCode?: string;
  }>();
  const results = brief?.results;
  const own = row?.mine ? brief?.matchup : undefined;
  const rowKey = row?.key, away = row?.game.away, home = row?.game.home;
  const season = brief?.pulse?.season;
  const hasOwn = Boolean(own);
  useEffect(() => {
    if (!rowKey || !away || !home || !results || season === undefined || hasOwn) return;
    let active = true;
    const starters = (id: string) => scoreOf(results, id)?.starters.map(s => s.player) ?? [];
    readPulseMatchup(league, season, results.week, [away, home],
      { [away]: starters(away), [home]: starters(home) }, port)
      .then(matchup => { if (active) setRead({ key: rowKey, matchup }); })
      .catch(cause => {
        if (active) setRead({
          key: rowKey, error: errorMessage(cause), errorCode: errorCodeFor(cause),
        });
      });
    return () => { active = false; };
  }, [rowKey, away, home, results, season, hasOwn, league, port]);
  if (!row) return { left: [], right: [], leftName: '', rightName: '',
    words: 'No game is read for this week yet.' };
  const matchup = own ?? (read?.key === row.key ? read.matchup : undefined);
  const flip = row.game.home === mine;
  const [l, r] = flip ? [row.game.home, row.game.away] : [row.game.away, row.game.home];
  const side = (id: string) => starterRows(scoreOf(results, id), matchup?.lineups[id],
    matchup?.projections ?? {});
  const total = (id: string) => { const s = scoreOf(results, id); return s?.points ?? s?.provisional; };
  const hasError = read?.key === row.key && Boolean(read.error);
  return { left: side(l), right: side(r), leftPoints: total(l), rightPoints: total(r),
    leftName: flip ? row.game.homeName : row.game.awayName,
    rightName: flip ? row.game.awayName : row.game.homeName,
    words: '', error: hasError ? read?.error : '',
    errorCode: hasError ? read?.errorCode : undefined };
}

function MatchupBox({ row, sides, league, port }: {
  row?: PulseRow; sides: Sides; league: string; port: HostedPort;
}) {
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
    {sides.error && <FailureAlert
      message={sides.error}
      prefix={`${unavailable} · `}
      code={sides.errorCode}
      className="live6__note"
      league={league}
      screen="Phone · Live"
      route="phone/live"
      port={port}
    />}
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

/** freshness: the held board's age in words, from its read time. */
function freshness(ms: number) {
  const minutes = Math.floor(ms / 60_000);
  return minutes < 1 ? 'Updated just now' : `Updated ${minutes} min ago`;
}

function FieldState({ game, words, fresh, anchor, reduced, guys, now }: {
  game?: Game; words: string; fresh: string; anchor: RefObject<HTMLElement>; reduced: boolean;
  guys: PregameGuy[]; now: number;
}) {
  if (!game) return <section className="live6__card" aria-label="NFL live" ref={anchor}>
    <p className="live6__note">{words}</p></section>;
  if (game.status === 'pre') return <section className="live6__live" aria-label="NFL live" ref={anchor}>
    <LivePregame key={game.id} game={game} guys={guys} now={now} reduced={reduced} /></section>;
  return <section className="live6__live" aria-label="NFL live" ref={anchor}>
    <LiveField key={game.id} game={game} fresh={fresh} reduced={reduced} /></section>;
}

function matchupGuysInGame(
  game: Game, row: PulseRow | undefined, sides: Sides, brief: HomeBrief | undefined, mine: string,
): PregameGuy[] {
  if (!row) return [];
  const awayAb = game.away.abbreviation;
  const homeAb = game.home.abbreviation;
  const flip = row.game.home === mine;
  const [leftId, rightId] = flip ? [row.game.home, row.game.away] : [row.game.away, row.game.home];
  const tagOf = (id: string, fallback: string) =>
    brief?.pulse?.seats.find(s => s.franchise === id)?.abbreviation?.trim() || fallback;
  const scoreTeamMap = (id: string) => {
    const map = new Map<string, string>();
    for (const s of scoreOf(brief?.results, id)?.starters ?? []) {
      if (s.team) map.set(s.player, s.team);
    }
    return map;
  };
  const collect = (
    list: PulseStarterRow[], franchiseId: string, sideLabel: string, isMine: boolean,
  ): PregameGuy[] => {
    const fallbackTeams = scoreTeamMap(franchiseId);
    const out: PregameGuy[] = [];
    for (const p of list) {
      const clubCode = espnTeam(p.team ?? fallbackTeams.get(p.player));
      if (clubCode !== awayAb && clubCode !== homeAb) continue;
      out.push({
        id: p.player, name: p.name, position: p.slot, team: clubCode,
        side: sideLabel, mine: isMine, projection: p.projection,
      });
    }
    return out;
  };
  return [
    ...collect(sides.left, leftId, row.mine ? 'YOURS' : tagOf(leftId, sides.leftName), row.mine),
    ...collect(sides.right, rightId, tagOf(rightId, sides.rightName), false),
  ];
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

export default LiveTab;
