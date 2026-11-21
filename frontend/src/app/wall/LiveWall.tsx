import { useCallback, useEffect, useRef, useState } from 'react';
import type { HostedPort } from '../commissioner/hostedRoom';
import { onHostedTopics } from '../data/hostedReads';
import type { Game } from '../now/liveContract';
import { clockWords, useNow } from '../shell/hostedGames';
import { LiveField } from '../phone/LiveField';
import { nextKickoff, wallCount, wallGrid } from './pickGames';
import { useWallGames } from './useWallGames';
import './wall.css';

type Subscribe = (load: () => void) => () => void;
const topicsFor = (league: string): Subscribe => load =>
  onHostedTopics(topics => { if (topics.some(topic => topic.league === league)) load(); });

/** LiveWall is the marketing wall: one phone per live NFL game on a fixed 1920x1080 stage (OBS size),
 *  each phone the real phone Live field fed by that game's detail read. */
export function LiveWall({ league, port, subscribe, search = window.location.search }: {
  league: string; port: HostedPort; subscribe?: Subscribe; search?: string;
}) {
  const cap = wallCount(search);
  const changes = useRef(subscribe ?? topicsFor(league)); changes.current = subscribe ?? topicsFor(league);
  const onChange = useCallback<Subscribe>(load => changes.current(load), []);
  const read = useWallGames(league, port, cap, onChange);
  const now = useNow(30_000);
  const fit = useFit();
  const { cols, scale } = wallGrid(read.picked.length);
  return <div className="wall" data-testid="live-wall">
    <div className="wall__stage" style={{ transform: `scale(${fit})` }}>
      <b className="wall__mark">THE WAR ROOM</b>
      {read.picked.length === 0
        ? <div className="wall__empty" role="status">
          <p>{read.board || read.error ? 'No NFL games live right now' : 'Reading the NFL board…'}</p>
          {read.board && <p className="wall__next">{nextKickoff(read.board, now)}</p>}
          {read.error && !read.board && <p className="wall__next">{read.error}</p>}
        </div>
        : <div className="wall__grid" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
          {read.picked.map(g => <Tile key={g.id} game={g} scale={scale} />)}
        </div>}
    </div>
  </div>;
}

function Tile({ game, scale }: { game: Game; scale: number }) {
  const bezel = 12;
  const name = `${game.away.abbreviation} at ${game.home.abbreviation}`;
  return <section className="wall__tile" aria-label={name}>
    <header className="wall__head"><b>{name}</b>
      <span>{clockWords(game)}</span></header>
    <div className="wall__phone"
      style={{ width: (390 + 2 * bezel) * scale, height: (844 + 2 * bezel) * scale }}>
      <div className="wall__bezel" style={{ transform: `scale(${scale})`, padding: bezel }}>
        <div className="wall__screen"><LiveField game={game} reduced={false} /></div>
      </div>
    </div>
  </section>;
}

/** useFit scales the 1920x1080 stage down to a smaller window; at 1920x1080 it is exactly 1. */
function useFit() {
  const calc = () => Math.min(window.innerWidth / 1920, window.innerHeight / 1080);
  const [fit, setFit] = useState(calc);
  useEffect(() => {
    const on = () => setFit(calc());
    window.addEventListener('resize', on);
    return () => window.removeEventListener('resize', on);
  }, []);
  return fit;
}

export default LiveWall;
