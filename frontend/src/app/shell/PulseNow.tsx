import { useEffect, useState } from 'react';
import { HQAct as PulseAct } from './HQAct';
import { percent, pointsLabel, type PulseOdds, type PulseResults, type PulseScore }
  from '../data/nativePulse';
import { readPulseMatchup, usePulseRead, type NativePulse, type PulseMatchup } from './pulseReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import { fieldPositions, gameTime, kickoffText } from '../now/LiveGame';
import type { Game as LiveNflGame, LiveView as LiveNflView } from '../now/liveContract';
import type { usePulseLive } from './hqLive';
import { boxLine, compactLine } from './statLine';
import { PulseDrawer } from './PulseDrawer';
import { applyFilter, bySlot, changePoints, changeVerdict, filterCounts, groupRows, pairStarters, pulseRows,
  scoreOf, starterRows, statChanges, PULSE_FILTERS, type PulseFilter, type PulseRow, type PulseStarterRow,
  type StarterPair, type StatChange }
  from './pulseModel';

/** PulseIdle is the calm empty state: what is missing, and what will fill it. */
export function PulseIdle({ words }: { words: string }) {
  return <div className="idle" role="status">{words}</div>;
}

type Reading<T> = { reading?: T; pending?: boolean; error?: string };
export type NowDrawer = 'box' | 'corr';

export function PulseNow({ reading, mine, week, results, league, port, live, drawer, onClose, onChanges }: {
  reading: NativePulse; mine: string; week?: number; results: Reading<PulseResults>;
  league: string; port: HostedPort; live: ReturnType<typeof usePulseLive>;
  drawer?: NowDrawer; onClose: () => void; onChanges: (count: number) => void;
}) {
  const [side, setSide] = useState<'mu' | 'nfl'>('mu');
  const [filter, setFilter] = useState<PulseFilter>('all');
  const [picked, pick] = useState<string>();
  const current = week === reading.week;
  // Only the week being played can be live, and only while an NFL game is actually under way.
  const playing = Boolean(live.view?.games.some(g => g.status === 'in'));
  const rows = results.reading ? pulseRows(results.reading, mine, reading.seats, current, playing) : [];
  const selected = rows.find(r => r.key === picked) ?? rows.find(r => r.mine) ?? rows[0];
  const sides = useStarterSides(reading, week, results.reading, selected, league, port, mine);
  const changes = statChanges(sides.left, sides.right);
  useEffect(() => onChanges(changes.length), [onChanges, changes.length]);
  const games = current ? live.view?.games ?? [] : [];
  const followed = current ? live.view?.followed ?? undefined : undefined;
  return <section className="ws cols on" id="ws-live">
    <div className="pcol">
      <div className="ch">
        <div className="seg">
          <PulseAct active={side === 'mu'} onAction={() => setSide('mu')}>MATCHUPS</PulseAct>
          <PulseAct active={side === 'nfl'} onAction={() => setSide('nfl')}>NFL GAMES</PulseAct>
        </div>
        <span>{week ? `week ${week}` : 'no week'}</span>
      </div>
      {side === 'nfl'
        ? <NflList games={games} followed={followed} onFollow={live.follow}
            words={!current ? 'NFL games are shown for the week being played.'
              : live.error ? `The NFL scoreboard is unavailable · ${live.error}`
              : !live.view ? 'Reading the NFL scoreboard…' : 'No NFL games are held for this week.'} />
        : <MatchupList rows={rows} filter={filter} onFilter={setFilter} results={results}
            selected={selected} onPick={key => pick(key)}
            odds={week === reading.odds?.week ? reading.odds : undefined} />}
    </div>
    <div className="pcol fieldwrap">
      <div className="follow">
        <span className="h">FOLLOWING</span>
        {games.filter(g => g.status === 'in' || g.id === followed?.id).slice(0, 3).map(g =>
          <PulseAct key={g.id} className={`fchip${g.id === followed?.id ? ' on' : ''}`}
            label={`Follow ${g.away.abbreviation} at ${g.home.abbreviation}`}
            onAction={() => live.follow(followKey(g))}>
            {g.away.abbreviation} <b>{g.away.score}–{g.home.score}</b> {g.home.abbreviation}
            <span className="q">{clockWords(g)}</span>
          </PulseAct>)}
        {!games.some(g => g.status === 'in') && !followed && <span className="src">
          {!current ? 'live games show for the week being played'
            : live.error ? `NFL scoreboard unavailable · ${live.error}`
            : !live.view ? 'Reading the NFL scoreboard…' : 'no NFL game is being played right now'}
        </span>}
      </div>
      {followed ? <FollowedGame game={followed} /> : <>
        <div className="gh"><span className="src">Follow an NFL game to see its field.</span></div>
        <div><PulseIdle words={playing ? 'Pick a live game above or under NFL GAMES.'
          : 'The field, the drive and the down show while an NFL game is being played.'} /></div>
      </>}
      <div className="spine">
        <div className="ch">
          <b>{selected?.mine ? 'MY MATCHUP · STARTERS' : <>MATCHUP · STARTERS
            {rows.some(r => r.mine) && <PulseAct className="mbtn back" onAction={() => pick(undefined)}>
              MY MATCHUP ‹</PulseAct>}</>}</b>
          <span>{selected ? `${selected.game.awayName} · ${selected.game.homeName}` : ''}</span>
        </div>
        <Starters results={results.reading} row={selected} sides={sides} games={games} />
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
    <BoxDrawer open={drawer === 'box'} onClose={onClose} sides={sides} week={week} />
    <StatChangesDrawer open={drawer === 'corr'} onClose={onClose} sides={sides} week={week}
      changes={changes} left={scoreOf(results.reading, sides.leftId ?? '')}
      right={scoreOf(results.reading, sides.rightId ?? '')} />
  </section>;
}

/** BoxDrawer is the chosen matchup's fantasy box score, one side at a time. */
function BoxDrawer({ open, onClose, sides, week }: { open: boolean; onClose: () => void;
  sides: StarterSides; week?: number }) {
  const [right, setRight] = useState(false);
  const rows = [...(right ? sides.right : sides.left)].sort((a, b) => bySlot(a.slot, b.slot));
  return <PulseDrawer open={open} onClose={onClose} title="Fantasy box score"
    sub={`${sides.leftName} vs ${sides.rightName} · week ${week ?? '—'}`} tabs={<>
      <PulseAct active={!right} onAction={() => setRight(false)}>
        {sides.mine ? 'YOU' : sides.leftName.toUpperCase()}</PulseAct>
      <PulseAct active={right} onAction={() => setRight(true)}>
        {sides.mine ? 'THEM' : sides.rightName.toUpperCase()}</PulseAct>
    </>}>
      {!rows.length ? <PulseIdle words="No lineup is recorded for this side this week." />
        : <div className="scroll"><table className="ct"><thead><tr>
          <th>SLOT</th><th>PLAYER</th><th>LINE 1</th><th>LINE 2</th><th>TD / TO</th><th>STATE</th><th>PTS</th>
        </tr></thead><tbody>
          {rows.map(r => <tr key={r.player} title={r.words}>
            <td>{r.slot}</td><td>{r.name}</td>
              {boxLine(r.line, r.slot).map((words, i) => <td key={i}>{words}</td>)}
            <td>{r.final ? 'done' : r.points !== undefined ? 'playing' : 'to play'}</td>
            <td>{pointsLabel(r.points)}</td>
          </tr>)}
        </tbody></table></div>}
  </PulseDrawer>;
}

/** StatChangesDrawer says which starters in the chosen matchup the final statistics re-scored, and
 *  whether any of it moved the result. */
function StatChangesDrawer({ open, onClose, sides, week, changes, left, right }: { open: boolean;
  onClose: () => void; sides: StarterSides; week?: number; changes: StatChange[];
  left?: PulseScore; right?: PulseScore }) {
  const margin = left?.points !== undefined && right?.points !== undefined
    ? left.points - right.points : undefined;
  const names = { left: sides.leftName, right: sides.rightName, mine: sides.mine };
  const who = (c: StatChange) => sides.mine ? (c.left ? 'YOU' : 'THEM') : c.left ? names.left : names.right;
  return <PulseDrawer open={open} onClose={onClose} title="Stat changes"
    sub={`last live line vs final stats · week ${week ?? '—'}`}>
    {!changes.length
      ? <PulseIdle
        words="The final stats scored every starter in this matchup the same as the last live line." />
      : <div className="scroll"><table className="ct"><thead><tr>
        <th>WK</th><th>PLAYER</th><th>STAT</th><th>WAS</th><th>NOW</th><th>PTS Δ</th><th>RESULT</th>
      </tr></thead><tbody>
        {changes.map(c => <tr key={c.player}>
          <td>{week}</td><td>{c.name}</td><td>{c.stats}</td>
          <td>{changePoints(c.was)}</td><td>{changePoints(c.now)}</td>
          <td>{changePoints(c.delta, true)} {who(c)}</td>
          <td>{changeVerdict(c, margin, Boolean(left?.recorded || right?.recorded), names)}</td>
        </tr>)}
      </tbody></table></div>}
    <div className="drawer-note">Compares the last live line this app held with the final stats, for the
      starters in this matchup. It is not the complete list of NFL stat corrections.</div>
  </PulseDrawer>;
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
  if (!players.length && sides.reading)
    return <PulseIdle words="Reading this matchup’s lineups…" />;
  if (!players.length) return <PulseIdle words={`No starter in ${row ? `${row.game.awayName} v `
    + row.game.homeName : 'this matchup'} plays in ${teams.join(' at ')}.`} />;
  return <div className="scroll">{players.map(p =>
    <div key={p.player} className={`onf ${p.mine ? 'me' : 'th'}`} title={p.words}>
      <span className="rib" />
      <span className="nm">{p.name}
        <small>{p.slot} · {p.team} · {p.mine ? sides.leftName : sides.rightName}</small></span>
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
        <span
          className="d">{p.awayScore}-{p.homeScore}<small>{game.away.abbreviation}-{
            game.home.abbreviation}</small></span>
      </div>;
    })}</div>;
}

function MatchupList({ rows, filter, onFilter, results, selected, onPick, odds }: {
  rows: PulseRow[]; filter: PulseFilter; onFilter: (f: PulseFilter) => void;
  results: Reading<PulseResults>; selected?: PulseRow; onPick: (key: string) => void;
  odds?: PulseOdds;
}) {
  const counts = filterCounts(rows);
  const shown = applyFilter(rows, filter);
  return <>
    <div className="fbar">
      {PULSE_FILTERS.map(f => <PulseAct key={f.key} className="mbtn" active={filter === f.key}
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
            selected={row.key === selected?.key} onPick={onPick}
            homeWin={odds?.games.find(g => g.home === row.game.home
              && g.away === row.game.away)?.homeWin} />)}
        </div>)}
    </div>
  </>;
}

function GameRow({ row, selected, onPick, homeWin }: { row: PulseRow; selected: boolean;
  onPick: (key: string) => void; homeWin?: number }) {
  // Win chance is the league's own, shown for the first-named side until the game has a result.
  const chance = row.decided || homeWin === undefined ? undefined : 1 - homeWin;
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
      <span className="src">{chance !== undefined ? percent(chance)
        : row.margin === undefined ? '' : `by ${row.margin.toFixed(1)}`}</span>
    </div>
    <div className="wpl">{chance !== undefined && <i style={{ width: percent(chance) }} />}</div>
  </PulseAct>;
}

type StarterSides = { left: PulseStarterRow[]; right: PulseStarterRow[]; private: boolean;
  /** The viewer's own franchise sits on the left whenever it plays in the chosen matchup. */
  leftId?: string; rightId?: string; leftName: string; rightName: string; mine: boolean;
  /** The lineups are still being read, so who plays where is not known yet. */
  reading?: boolean };

/** useStarterSides reads the chosen matchup's two lineups once, for the starters and for who is in
 *  the followed NFL game. */
function useStarterSides(reading: NativePulse, week: number | undefined, results: PulseResults | undefined,
  row: PulseRow | undefined, league: string, port: HostedPort, mine: string): StarterSides {
  const away = row?.game.away;
  const home = row?.game.home;
  const starters = (franchise?: string) => scoreOf(results, franchise ?? '')?.starters
    .map(s => s.player) ?? [];
  const matchup = usePulseRead<PulseMatchup>(
    JSON.stringify(['pulse-matchup', league, week, away, home]),
    publish => week && away && home
      ? readPulseMatchup(league, reading.season, week, [away, home],
        { [away]: starters(away), [home]: starters(home) }, port, publish)
      : Promise.reject(new Error('Choose a matchup to read its lineups.')));
  const projections = matchup.reading?.projections ?? {};
  const flip = home !== undefined && home === mine;
  const [leftId, rightId] = flip ? [home, away] : [away, home];
  const side = (id?: string) =>
    starterRows(scoreOf(results, id ?? ''), matchup.reading?.lineups[id ?? ''], projections);
  return {
    left: side(leftId), right: side(rightId), leftId, rightId,
    leftName: (flip ? row?.game.homeName : row?.game.awayName) ?? '',
    rightName: (flip ? row?.game.awayName : row?.game.homeName) ?? '',
    mine: leftId !== undefined && leftId === mine,
    private: Boolean(matchup.reading?.errors.length || matchup.error),
    reading: !matchup.reading && !matchup.error,
  };
}

function Starters({ results, row, sides, games }: { results?: PulseResults; row?: PulseRow;
  sides: StarterSides; games: LiveNflGame[] }) {
  if (!row) return <PulseIdle words="No game is read for this week yet." />;
  const leftScore = scoreOf(results, sides.leftId ?? '');
  const rightScore = scoreOf(results, sides.rightId ?? '');
  if (!leftScore?.starters.length && !rightScore?.starters.length)
    return <PulseIdle words={`${row.game.words} Starters appear once both franchises have a lineup `
      + 'recorded for this week.'} />;
  const groups = pairStarters(sides.left, sides.right);
  const count = groups.reduce((n, g) => n + g.pairs.length, 0);
  // The mock's one list: a heading bar per unit, then one equal row per pair. A league that starts
  // more than the mock's sixteen keeps every row on screen by setting the detail beside the name.
  const rows = ['auto', ...groups.map(g => `auto repeat(${g.pairs.length},minmax(0,1fr))`)].join(' ');
  return <div className={`mym${count > 16 ? ' tight' : ''}`} style={{ gridTemplateRows: rows }}>
    <div className="mh">
      <span className="a">{sides.mine ? 'YOU · ' : ''}{sides.leftName.toUpperCase()}</span>
      <span className="t" style={{ textAlign: 'right', paddingRight: 10 }}>
        {pointsLabel(leftScore?.points ?? leftScore?.provisional)}</span>
      <span className="t" />
      <span className="t" style={{ textAlign: 'left', paddingLeft: 10 }}>
        {pointsLabel(rightScore?.points ?? rightScore?.provisional)}</span>
      <span className="b">{sides.rightName.toUpperCase()}{sides.mine ? ' · THEM' : ''}</span>
    </div>
    {groups.flatMap(g => [<div className="mgrp" key={g.heading}>{g.heading}</div>,
      ...g.pairs.map((pair, i) => <StarterRow key={`${g.heading}-${pair.slot}-${i}`} pair={pair}
        games={games} defense={g.heading === 'DEFENSE'} />)])}
  </div>;
}

/** gameOf is the live NFL game a starter's club is in, when the scoreboard and the club are known. */
/** Line is what a starter has done: the stat line once anything is recorded, the projection before. */
function Line({ p }: { p: PulseStarterRow }) {
  const parts = compactLine(p.line, p.slot);
  if (!parts.length) return p.final || p.projection === undefined ? null
    : <div className="ln wait">proj {p.projection.toFixed(1)}</div>;
  return <div className="ln">{parts.map((part, i) => <span key={part}>
    {i > 0 && <span className="sep">|</span>}
    {part.endsWith(' TD') ? <span className="td">{part}</span>
      : part.split(/(\d+(?:\.\d+)?)/).map((piece, j) => j % 2 ? <b key={j}>{piece}</b> : piece)}
  </span>)}</div>;
}

function gameOf(p: PulseStarterRow | undefined, games: LiveNflGame[]) {
  const club = espnTeam(p?.team);
  return club ? games.find(g => g.away.abbreviation === club || g.home.abbreviation === club) : undefined;
}

function StarterRow({ pair, games, defense }: { pair: StarterPair; games: LiveNflGame[];
  defense: boolean }) {
  const { left, right } = pair;
  const chip = (p?: PulseStarterRow, r = false) => {
    const g = gameOf(p, games);
    const state = g ? (g.status === 'in' ? 'L' : g.status === 'post' ? 'F' : 'P') : p?.final ? 'F' : 'P';
    return <span className={`chipc ${state}${r ? ' r' : ''}`}>
      {g ? clockWords(g) : p?.final ? 'FINAL' : ''}</span>;
  };
  // On the field: the starter's club has the ball (offense) or is defending it (defense).
  const ball = (p?: PulseStarterRow) => {
    const g = gameOf(p, games);
    const club = g && (g.home.abbreviation === espnTeam(p?.team) ? g.home : g.away);
    const has = g?.status === 'in' && g.state?.possession ? g.state.possession === club?.id : undefined;
    return <span className={`ball2${has !== undefined && has !== defense ? '' : ' off'}`}
      title="on the field" />;
  };
  const cell = (p?: PulseStarterRow, r = false) => <div className={r ? 'p r' : 'p'} title={p?.words}>
    <div className="nm">{r && p?.team && <small>{p.team}</small>}<span>{p?.name ?? ''}</span>
      {!r && p?.team && <small>{p.team}</small>}</div>
    {p && <Line p={p} />}
  </div>;
  const score = (p?: PulseStarterRow, other?: PulseStarterRow, r = false) => {
    const lead = p?.points !== undefined && (other?.points === undefined || p.points >= other.points);
    return <span className={`s${r ? ' r' : ''} ${p?.points === undefined ? 'tbd'
      : lead ? 'up' : 'dn'}`}>{pointsLabel(p?.points)}</span>;
  };
  return <div className="mr">
    {cell(left)}{chip(left)}{ball(left)}
    {score(left, right)}<span className="ps">{pair.slot}</span>{score(right, left, true)}
    {ball(right)}{chip(right, true)}{cell(right, true)}
  </div>;
}
