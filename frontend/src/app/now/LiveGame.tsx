import { useEffect, useState } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportView } from '../reports/ReportView';
import { liveState, startLivePolling } from './liveState';
import type { FieldState, Game, LiveView, Spine } from './liveContract';
import './livegame.css';

// Home attacks right in this normalized diagram, not the physical stadium direction.
export function fieldPositions(state: FieldState, home: boolean) {
  const line = state.yardLine;
  const first = Math.max(0, Math.min(100, line + (home ? state.distance : -state.distance)));
  return { line, first, direction: home ? 1 : -1 };
}
export function Field({ game }: { game: Game }) {
  const state = game.state;
  if (!state || state.toEndzone < 0 || ![game.home.id, game.away.id].includes(state.possession))
    return <p>Field position not supplied.</p>;
  const home = state.possession === game.home.id;
  const { line, first, direction } = fieldPositions(state, home);
  const x = (yards: number) => 30 + yards * 5;
  const team = home ? game.home.abbreviation : game.away.abbreviation;
  const down = state.down > 0 ? `${state.down} & ${state.distance}` : 'between downs';
  const label = `${team} possession; ${down}; ${state.toEndzone} yards to the end zone`;
  return <figure className="live-field">
    <svg viewBox="0 0 560 170" role="img" aria-label={label}>
      <rect x="30" y="10" width="500" height="120" fill="none" stroke="currentColor" />
      {Array.from({ length: 11 }, (_, i) => <g key={i}>
        <line x1={x(i * 10)} x2={x(i * 10)} y1="10" y2="130" className="field-yard" />
        {i > 0 && i < 10 && <text x={x(i * 10)} y="148" textAnchor="middle">{Math.min(i, 10 - i) * 10}</text>}
      </g>)}
      <line data-line="scrimmage" x1={x(line)} x2={x(line)} y1="10" y2="130" className="field-scrimmage" />
      {state.down > 0 && <line data-line="first-down" x1={x(first)} x2={x(first)} y1="10" y2="130"
        className="field-first" />}
      <path d={`M ${x(line)} 70 h ${direction * 20} ` +
        `m ${-direction * 7} -7 l ${direction * 7} 7 l ${-direction * 7} 7`}
        fill="none" stroke="currentColor" strokeWidth="2" />
      <text x="30" y="165">{game.home.abbreviation} goal</text>
      <text x="530" y="165" textAnchor="end">{game.away.abbreviation} goal</text>
    </svg>
    <figcaption>{label} · solid: scrimmage · dashed: first down.
      Home attacks right; diagram normalized.</figcaption>
  </figure>;
}
export function gameTime(g: Game) {
  if (g.status === 'post') return 'Final';
  if (g.status === 'pre') return `Kickoff ${new Date(g.kickoff).toLocaleString()}`;
  return `${g.period > 4 ? 'OT' : `Q${g.period}`} · ${g.clock || 'Clock not supplied'}`;
}
function Photo({ player }: { player: Spine['player'] }) {
  const [failed, fail] = useState(false);
  return player.photo.src && !failed ? <img className="spine-photo"
    src={player.photo.src} alt={player.photo.alt}
    onError={() => fail(true)} /> : <svg className="spine-photo" viewBox="0 0 40 40"
      role="img" aria-label={`${player.name}: photo unavailable`}>
      <circle cx="20" cy="13" r="7" fill="currentColor" />
      <path d="M7 38V31C7 19 33 19 33 31V38Z" fill="currentColor" />
    </svg>;
}
export function orderedSpine(entries: Spine[]) {
  return [...entries].sort((a, b) => b.play.sequence - a.play.sequence);
}
export function PlaySpine({ entries, unmatched }: { entries: Spine[]; unmatched: number }) {
  return <section data-endpoint="X-02" aria-label="Fantasy play spine" className="play-spine">
    <h4>Our players in this game</h4>
    <p>{unmatched} participant appearances unmatched by ESPN crosswalk id; never matched by name.</p>
    {entries.length === 0 && <p>No held plays matched a starter in your matchup.
      Choose your franchise and follow a game.</p>}
    <ol>{orderedSpine(entries).map(e => <li key={`${e.play.id}:${e.player.id}`}
      data-side={e.player.mine ? 'mine' : 'theirs'}>
      <Photo player={e.player} />
      <div><strong>{e.player.name}</strong> · {e.player.team} · {e.player.mine ? 'Mine' : 'Opponent'}
        <p>{e.play.text}</p>
        <small>Q{e.play.period} · {e.play.clock}
          {e.play.at !== '0001-01-01T00:00:00Z' && ` · ${new Date(e.play.at).toLocaleTimeString()}`}</small>
      </div>
    </li>)}</ol>
  </section>;
}
function score(g: Game) {
  const away = `${g.away.abbreviation}${g.status === 'pre' ? '' : ` ${g.away.score}`}`;
  const home = `${g.home.abbreviation}${g.status === 'pre' ? '' : ` ${g.home.score}`}`;
  return `${away} at ${home}`;
}
export function LiveGameView({ view }: { view: LiveView }) {
  const g = view.followed;
  return <>
    {g && <div className="followed-bar market-controls" aria-label="Followed game">
        <strong>{score(g)} · {gameTime(g)}</strong>
        <Act verb="game.follow"
        args={{ id: '' }} label="Stop following game">Stop following</Act>
      </div>}
    <section className="live-game" aria-label="Current NFL week live games">
    <h4>Current NFL week</h4>
    <div className="market-controls game-strip" aria-label="Follow an NFL game">
      {view.games.map(game => <Act key={game.id} verb="game.follow"
        args={{ id: `${game.away.abbreviation}-${game.home.abbreviation}` }}
        active={g?.id === game.id} label={`Follow ${game.away.abbreviation} at ${game.home.abbreviation}`}>
        {score(game)} · {gameTime(game)}
      </Act>)}
    </div>
    <p>{view.source} · As of {view.asOf}</p>
    {view.failure && <p role="status">Refresh failed: {view.failure}. The held view stays available.</p>}
    {!view.active && <p>No NFL game window is live. Feeds are not being fetched.</p>}
    {g && <>
      {g.status !== 'pre' && <Field game={g} />}
      <p>{g.status === 'pre' ? 'Not started.' : g.last?.text || 'Last play not supplied.'}</p>
      <p>ESPN game details as of {view.detailAsOf || 'not held'}</p>
      {view.detailFailure && <p role="status">{view.detailFailure}</p>}
      <ReportView report={view.box} howLabel="How this is counted" heading="h4" />
      <PlaySpine entries={view.spine} unmatched={view.unmatched} />
    </>}
    {!g && <p>Pick a game to follow. Its bar stays here while you browse Now reports.</p>}
  </section></>;
}
export default function LiveGame() {
  const state = liveState.use(); const mine = commands.use().franchiseId;
  useEffect(() => startLivePolling(mine ?? ''), [mine, state.follow]);
  return <>
    {state.error && <p role="alert">{state.error}</p>}
    {state.pending && !state.view && <p role="status">Reading held games…</p>}
    {state.view && <LiveGameView view={state.view} />}
  </>;
}
