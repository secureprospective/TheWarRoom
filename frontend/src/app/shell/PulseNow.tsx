import { useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import { pointsLabel, type PulseResults } from '../data/nativePulse';
import { readPulseMatchup, usePulseRead, type NativePulse, type PulseMatchup } from './pulseReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import { fieldPositions, gameTime, kickoffText } from '../now/LiveGame';
import type { Game as LiveNflGame, LiveView as LiveNflView } from '../now/liveContract';
import { usePulseLive } from './hqLive';
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
  // The NFL board and the followed game are the same for every league; the matchup's own starters
  // say who is in a game, so the live read is asked for no franchise.
  const live = usePulseLive('');
  const current = week === reading.week;
  // Only the week being played can be live, and only while an NFL game is actually under way.
  const playing = Boolean(live.view?.games.some(g => g.status === 'in'));
  const rows = results.reading ? pulseRows(results.reading, mine, reading.seats, current, playing) : [];
  const selected = rows.find(r => r.key === picked) ?? rows.find(r => r.mine) ?? rows[0];
  const sides = useStarterSides(reading, week, results.reading, selected, league, port);
  const games = current ? live.view?.games ?? [] : [];
  const followed = current ? live.view?.followed ?? undefined : undefined;
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
        ? <NflList games={games} followed={followed} onFollow={live.follow}
            words={!current ? 'NFL games are shown for the week being played.'
              : live.error ? `The NFL scoreboard is unavailable · ${live.error}`
              : !live.view ? 'Reading the NFL scoreboard…' : 'No NFL games are held for this week.'} />
        : <MatchupList rows={rows} filter={filter} onFilter={setFilter} results={results}
            selected={selected} onPick={key => pick(key)} />}
    </div>
    <div className="pcol fieldwrap">
      <div className="follow">
        <span className="h">FOLLOWING</span>
        {games.filter(g => g.status === 'in' || g.id === followed?.id).slice(0, 4).map(g =>
          <PulseAct key={g.id} className={`fchip${g.id === followed?.id ? ' on' : ''}`}
            label={`Follow ${g.away.abbreviation} at ${g.home.abbreviation}`}
            onAction={() => live.follow(followKey(g))}>
            {g.away.abbreviation} <b>{g.away.score}</b> · {g.home.abbreviation} <b>{g.home.score}</b>
            <span className="q">{clockWords(g)}</span>
          </PulseAct>)}
        {!games.some(g => g.status === 'in') && !followed && <span className="src">
          {current ? 'no NFL game is being played right now' : 'live games show for the week being played'}
        </span>}
      </div>
      {followed ? <FollowedGame game={followed} /> : <>
        <div className="gh"><span className="src">Follow an NFL game to see its field.</span></div>
        <div><PulseIdle words={playing ? 'Pick a live game above or under NFL GAMES.'
          : 'The field, the drive and the down show while an NFL game is being played.'} /></div>
      </>}
      <div className="spine">
        <div className="ch">
          <b>{selected?.mine ? 'MY MATCHUP · STARTERS' : 'MATCHUP · STARTERS'}</b>
          <span>{selected ? `${selected.game.awayName} · ${selected.game.homeName}` : ''}</span>
        </div>
        <Starters results={results.reading} row={selected} sides={sides} />
      </div>
    </div>
    <div className="pcol">
      <div className="ch"><b>IN THIS GAME</b>
        <span className="src">{followed ? `${followed.away.abbreviation} at ${followed.home.abbreviation}`
          : 'follow a game'}</span></div>
      <InThisGame followed={followed} sides={sides} row={selected} />
      <div className="ch" style={{ borderTop: '2px solid var(--black)' }}>
        <b>PLAY-BY-PLAY</b><span className="src">{followed ? `${followed.plays.length} plays` : ''}</span>
      </div>
      <Plays game={followed} />
    </div>
  </section>;
}

function followKey(g: LiveNflGame) { return `${g.away.abbreviation}-${g.home.abbreviation}`; }
function clockWords(g: LiveNflGame) {
  if (g.status === 'post') return 'FINAL';
  if (g.status === 'pre') return kickoffText(g.kickoff);
  return `${g.period > 4 ? 'OT' : `Q${g.period}`} ${g.clock}`;
}

export function NflList({ games, followed, onFollow, words }: { games: LiveNflGame[];
  followed?: LiveNflGame; onFollow: (id: string) => void; words: string }) {
  if (!games.length) return <div className="scroll"><PulseIdle words={words} /></div>;
  const order = { in: 0, pre: 1, post: 2 } as const;
  return <div className="scroll">
    {[...games].sort((a, b) => order[a.status] - order[b.status]).map(g => {
      const has = (t: LiveNflGame['home']) => g.status === 'in' && g.state?.possession === t.id;
      return <PulseAct key={g.id} className={`nfl${g.id === followed?.id ? ' sel' : ''}`
        + `${g.status === 'post' ? ' dim' : ''}`} label={`Follow ${g.away.abbreviation} at `
        + g.home.abbreviation} onAction={() => onFollow(followKey(g))}>
        {[g.away, g.home].map(t => <div className="r" key={t.id}>
          <span className={`pos${has(t) ? '' : ' no'}`} />{t.abbreviation}
          <b>{g.status === 'pre' ? '' : t.score}</b></div>)}
        <div className="meta">
          <span className={`st${g.status === 'in' ? ' live' : g.status === 'post' ? ' fin' : ''}`}>
            {g.status === 'in' ? 'LIVE' : g.status === 'post' ? 'FINAL' : 'TO PLAY'}</span>
          <span>{gameTime(g)}</span>
        </div>
      </PulseAct>;
    })}
  </div>;
}

export function FollowedGame({ game }: { game: LiveNflGame }) {
  const s = game.state;
  const known = game.status === 'in' && s !== null && s.toEndzone >= 0
    && [game.home.id, game.away.id].includes(s.possession);
  const home = known && s.possession === game.home.id;
  const at = (yards: number) => `${7 + Math.max(0, Math.min(100, yards)) * 0.86}%`;
  const spot = known ? fieldPositions(s, Boolean(home)) : undefined;
  const team = known ? (home ? game.home : game.away).abbreviation : '';
  return <>
    <div className="gh">
      <div className="t"><b>{game.away.score}</b>{game.away.abbreviation}<span>away</span></div>
      <div className="clk">{clockWords(game)}<small>{gameTime(game)}</small></div>
      <div className="t r"><span>home</span>{game.home.abbreviation}<b>{game.home.score}</b></div>
    </div>
    <div>
      {game.status === 'pre' ? <PulseIdle words={`Kickoff ${kickoffText(game.kickoff)}.`} />
        : <>
          <div className="field" role="img" aria-label={known
            ? `${team} ball, ${s.toEndzone} yards to the end zone` : 'Field position not supplied'}>
            <div className="ez l"><span>{game.away.abbreviation}</span></div>
            <div className="ez r"><span>{game.home.abbreviation}</span></div>
            {[10, 20, 30, 40, 50, 60, 70, 80, 90].map(y => <span key={y} className="yl"
              style={{ left: at(y) }}>{Math.min(y, 100 - y)}</span>)}
            {spot && <>
              <span className="los" style={{ left: at(spot.line) }} />
              {s && s.down > 0 && <span className="ftd" style={{ left: at(spot.first) }} />}
              <span className="ball" style={{ left: at(spot.line) }} />
            </>}
          </div>
          <div className="dd">
            {known ? <><b>{team} ball</b>
              <span>{s.down > 0 ? `${ordinal(s.down)} & ${s.distance}` : 'between downs'}</span>
              <span>{s.toEndzone} to go for six</span></>
              : <span>{game.status === 'post' ? 'Final' : 'Field position not supplied'}</span>}
            <span className="lp">{game.last?.text ?? ''}</span>
          </div>
        </>}
    </div>
  </>;
}
function ordinal(n: number) { return `${n}${['th', 'st', 'nd', 'rd'][n] ?? 'th'}`; }

/** League team codes and ESPN's differ for a few clubs; both name the same NFL team. */
const ESPN_TEAM: Record<string, string> = { JAC: 'JAX', WAS: 'WSH', GBP: 'GB', KCC: 'KC', NEP: 'NE',
  NOS: 'NO', SFO: 'SF', TBB: 'TB', LVR: 'LV', OAK: 'LV', SDC: 'LAC', STL: 'LAR' };
export function espnTeam(team?: string) { return team ? ESPN_TEAM[team] ?? team : ''; }

export function InThisGame({ followed, sides, row }: { followed?: LiveNflGame; sides: StarterSides;
  row?: PulseRow }) {
  if (!followed) return <PulseIdle words="Follow an NFL game to see who in this matchup is playing in it." />;
  const teams = [followed.away.abbreviation, followed.home.abbreviation];
  const inGame = (rows: PulseStarterRow[], mine: boolean) => rows
    .filter(r => teams.includes(espnTeam(r.team))).map(r => ({ ...r, mine }));
  const players = [...inGame(sides.left, true), ...inGame(sides.right, false)];
  if (!players.length) return <PulseIdle words={`No starter in ${row ? `${row.game.awayName} v `
    + row.game.homeName : 'this matchup'} plays in ${teams.join(' at ')}.`} />;
  return <div className="scroll">{players.map(p =>
    <div key={p.player} className={`onf ${p.mine ? 'me' : 'th'}`} title={p.words}>
      <span className="rib" />
      <span className="nm">{p.name}<small>{p.slot} · {p.team} · {p.mine ? row?.game.awayName
        : row?.game.homeName}</small></span>
      <b>{pointsLabel(p.points)}</b>
    </div>)}</div>;
}

export function Plays({ game }: { game?: LiveNflGame }) {
  if (!game) return <PulseIdle words="Plays show for the game you follow." />;
  if (!game.plays.length) return <PulseIdle words={game.status === 'pre'
    ? 'The game has not kicked off.' : 'No plays are held for this game yet.'} />;
  const abbr = (id?: string) => id === game.home.id ? game.home.abbreviation
    : id === game.away.id ? game.away.abbreviation : '';
  return <div className="scroll">{[...game.plays].sort((a, b) => b.sequence - a.sequence)
    .slice(0, 60).map(p => {
      const team = abbr(p.start?.possession);
      return <div key={p.id} className={`sp-row ${p.start?.possession === game.home.id ? 'th' : 'me'}`}>
        <span className="tm">Q{p.period}<br />{p.clock}</span>
        <span className="rib" />
        <span className="hs">{team || '·'}</span>
        <span className="what">{p.text}</span>
        <span className="d">{p.awayScore}-{p.homeScore}<small>{game.away.abbreviation}-{game.home.abbreviation}</small></span>
      </div>;
    })}</div>;
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

type StarterSides = { left: PulseStarterRow[]; right: PulseStarterRow[]; private: boolean };

/** useStarterSides reads the chosen matchup's two lineups once, for the starters and for who is in
 *  the followed NFL game. */
function useStarterSides(reading: NativePulse, week: number | undefined, results: PulseResults | undefined,
  row: PulseRow | undefined, league: string, port: HostedPort): StarterSides {
  const away = row?.game.away;
  const home = row?.game.home;
  const starters = (franchise?: string) => scoreOf(results, franchise ?? '')?.starters
    .map(s => s.player) ?? [];
  const matchup = usePulseRead<PulseMatchup>(
    JSON.stringify(['pulse-matchup', league, week, away, home]),
    () => week && away && home
      ? readPulseMatchup(league, reading.season, week, [away, home],
        { [away]: starters(away), [home]: starters(home) }, port)
      : Promise.reject(new Error('Choose a matchup to read its lineups.')));
  const projections = matchup.reading?.projections ?? {};
  return {
    left: starterRows(scoreOf(results, away ?? ''), matchup.reading?.lineups[away ?? ''], projections),
    right: starterRows(scoreOf(results, home ?? ''), matchup.reading?.lineups[home ?? ''], projections),
    private: Boolean(matchup.reading?.errors.length || matchup.error),
  };
}

function Starters({ results, row, sides }: { results?: PulseResults; row?: PulseRow;
  sides: StarterSides }) {
  const [side, setSide] = useState<'off' | 'def'>('off');
  if (!row) return <PulseIdle words="No game is read for this week yet." />;
  const awayScore = scoreOf(results, row.game.away);
  const homeScore = scoreOf(results, row.game.home);
  if (!awayScore?.starters.length && !homeScore?.starters.length)
    return <PulseIdle words={`${row.game.words} Starters appear once both franchises have a lineup `
      + 'recorded for this week.'} />;
  // Each side keeps its own offence and defence in lineup order, so a pair is always like for like.
  const split = (rows: PulseStarterRow[]) => ({ off: rows.filter(r => OFFENCE.includes(r.slot)),
    def: rows.filter(r => !OFFENCE.includes(r.slot)) });
  const left = split(sides.left);
  const right = split(sides.right);
  const total = (rows: PulseStarterRow[]) => rows.reduce((n, r) => n + (r.points ?? 0), 0);
  const count = (k: 'off' | 'def') => Math.max(left[k].length, right[k].length);
  const shown = count(side) ? side : side === 'off' ? 'def' : 'off';
  const pairs = Array.from({ length: count(shown) }, (_unused, i) =>
    [left[shown][i], right[shown][i]] as const);
  return <div className="mym v2">
    <div className="mh">
      <span className="a">{row.game.awayName.toUpperCase()}</span>
      <span className="t">{pointsLabel(awayScore?.points ?? awayScore?.provisional)}</span>
      <span className="t" />
      <span className="t">{pointsLabel(homeScore?.points ?? homeScore?.provisional)}</span>
      <span className="b">{row.game.homeName.toUpperCase()}</span>
    </div>
    <div className="mside">
      {(['off', 'def'] as const).map(k => <PulseAct key={k} selected={shown === k}
        disabled={!count(k)} onAction={() => setSide(k)}>
        <span className="sv">{total(left[k]).toFixed(1)}</span>
        <b>{k === 'off' ? 'OFFENCE' : 'DEFENCE'} · {count(k)}</b>
        <span className="sv">{total(right[k]).toFixed(1)}</span>
      </PulseAct>)}
    </div>
    <div className="mrows">
      {pairs.map(([a, b], i) => <StarterRow key={`${shown}-${a?.player ?? i}-${b?.player ?? i}`}
        away={a} home={b} />)}
    </div>
    {sides.private && <p role="status" className="src">
      A starter&rsquo;s NFL team and lock come from that franchise&rsquo;s own lineup, which is
      private to its team.</p>}
  </div>;
}

function StarterRow({ away, home }: { away?: PulseStarterRow; home?: PulseStarterRow }) {
  // A lock is known only for the franchise whose lineup this viewer may read; silence, not "OPEN".
  const facts = (p: PulseStarterRow) => [p.team, p.locked === undefined ? undefined
    : p.locked ? 'LOCKED' : 'OPEN', p.projection === undefined ? undefined
    : `proj ${p.projection.toFixed(1)}`].filter(Boolean).join(' · ');
  const cell = (p?: PulseStarterRow, right = false) => <div className={right ? 'p r' : 'p'}
    title={p?.words}>
    <div className="nm">{p?.name ?? '—'}</div>
    {p && <div className="ln">{facts(p)}</div>}
  </div>;
  const score = (p?: PulseStarterRow, other?: PulseStarterRow, right = false) => {
    const lead = p?.points !== undefined && (other?.points === undefined || p.points >= other.points);
    return <span className={`s${right ? ' r' : ''} ${p?.points === undefined ? 'tbd'
      : lead ? 'up' : 'dn'}`}>{pointsLabel(p?.points)}</span>;
  };
  return <div className="mr">
    {cell(away)}
    {score(away, home)}
    <span className="ps">{away?.slot ?? home?.slot ?? '—'}</span>
    {score(home, away, true)}
    {cell(home, true)}
  </div>;
}
