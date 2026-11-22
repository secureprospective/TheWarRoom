import { useCallback, useMemo, useRef } from 'react';
import type { HostedPort } from '../commissioner/hostedRoom';
import { onHostedTopics } from '../data/hostedReads';
import type { Game } from '../now/liveContract';
import { LiveField, useReducedMotion } from '../phone/LiveField';
import { presentation } from '../phone/livePlays';
import '../phone/LiveTab.css';
import { clockWords, useNow } from '../shell/hostedGames';
import { chooseWallLayout, TICKER_H, useViewport, type RailMode } from './layout';
import { nextKickoff, wallCount } from './pickGames';
import { useWallSpotlight, type SpotlightItem } from './spotlight';
import { useWallGames } from './useWallGames';
import {
  buildTicker, selectBigPlays, selectFirstDowns,
  type BigPlay, type FirstDownPlay, type TickerItem,
} from './wallPlays';
import './wall.css';

type Subscribe = (load: () => void) => () => void;
const topicsFor = (league: string): Subscribe => load =>
  onHostedTopics(topics => { if (topics.some(topic => topic.league === league)) load(); });

/** LiveWall is the full-screen NFL broadcast wall: fluid viewport grid of real phone Live fields,
 *  full-board scrolling ticker, queued play spotlights, and a Big Plays rail in leftover space. */
export function LiveWall({ league, port, subscribe, search = window.location.search }: {
  league: string; port: HostedPort; subscribe?: Subscribe; search?: string;
}) {
  const cap = wallCount(search);
  const changes = useRef(subscribe ?? topicsFor(league)); changes.current = subscribe ?? topicsFor(league);
  const onChange = useCallback<Subscribe>(load => changes.current(load), []);
  const wall = useWallGames(league, port, cap, onChange);
  const now = useNow(30_000);
  const reduced = useReducedMotion();
  const mode = presentation(reduced);
  const wallRef = useRef<HTMLDivElement>(null);
  const viewport = useViewport(wallRef);
  const layout = useMemo(
    () => chooseWallLayout(wall.picked.length, viewport.w, viewport.h),
    [wall.picked.length, viewport.w, viewport.h],
  );
  const ticker = useMemo(() => buildTicker(wall.allGames), [wall.allGames]);
  const bigPlays = useMemo(() => selectBigPlays(wall.allGames, 8), [wall.allGames]);
  const firstDowns = useMemo(() => selectFirstDowns(wall.allGames, 16), [wall.allGames]);
  const spotlight = useWallSpotlight(wall.picked);

  return <div className="wall" ref={wallRef} data-testid="live-wall" data-motion={mode}
    data-rail={layout.rail} style={{
      ['--k' as string]: layout.scale,
      ['--ticker-h' as string]: `${TICKER_H}px`,
      ['--wall-pad' as string]: `${layout.pad}px`,
      ['--wall-gap' as string]: `${layout.gap}px`,
    }}>
    <TickerBar items={ticker} />
    {spotlight && <SpotlightBanner item={spotlight} />}
    <main className={`wall__body wall__body--${layout.rail}`}>
      {wall.picked.length === 0
        ? <div className="wall__empty" role="status">
          <p>{wall.board || wall.error ? 'No NFL games live right now' : 'Reading the NFL board…'}</p>
          {wall.board && <p className="wall__next">{nextKickoff(wall.board, now)}</p>}
          {wall.error && !wall.board && <p className="wall__next">{wall.error}</p>}
        </div>
        : <>
          <div className="wall__grid-wrap">
            <div className="wall__grid" style={{
              gridTemplateColumns: `repeat(${layout.cols}, ${layout.tileW}px)`,
              gap: `${layout.gap}px`,
            }}>
              {wall.picked.map(g => <Tile key={g.id} game={g} scale={layout.scale}
                bezel={layout.bezel} headH={layout.headH} tileGap={layout.tileGap}
                phoneW={layout.phoneW} phoneH={layout.phoneH} reduced={reduced}
                spotlight={spotlight?.gameId === g.id ? spotlight : undefined} />)}
            </div>
          </div>
          {layout.rail !== 'none' && <BigPlaysRail plays={bigPlays} firstDowns={firstDowns}
            mode={layout.rail} width={layout.railW} height={layout.railH} />}
        </>}
    </main>
  </div>;
}

function TickerBar({ items }: { items: TickerItem[] }) {
  const duration = `${Math.max(20, items.length * 4.5)}s`;
  return <header className="wall__ticker" aria-label="NFL scoreboard ticker">
    <div className="wall__brand">
      <b className="wall__mark">THE WAR ROOM</b>
    </div>
    <div className="wall__ticker-view">
      {items.length > 0
        ? <div className="wall__ticker-track" style={{ ['--ticker-s' as string]: duration }}>
          <TickerList items={items} duplicate={false} />
          <TickerList items={items} duplicate />
        </div>
        : <span className="wall__ticker-idle">NFL LIVE BOARD</span>}
    </div>
  </header>;
}

function TickerList({ items, duplicate }: { items: TickerItem[]; duplicate: boolean }) {
  return <ul className="wall__ticker-list" aria-hidden={duplicate || undefined}
    data-testid={duplicate ? undefined : 'wall-ticker-list'}>
    {items.map(item => <li key={`${duplicate ? 'dup-' : ''}${item.id}`}
      className={`wall__tk wall__tk--${item.status}`}>
      <span className="wall__tk-side">
        <i className="wall__tk-bar" style={{ background: item.awayColor }} />
        <b className="wall__tk-ab">{item.awayAb}</b>
        {item.poss === 'L' && <i className="wall__tk-poss" aria-label="possession"
          style={{ background: item.awayColor }} />}
        <em className="wall__tk-sc">{item.status === 'pre' ? '–' : item.awayScore}</em>
      </span>
      <span className="wall__tk-sep">@</span>
      <span className="wall__tk-side">
        <i className="wall__tk-bar" style={{ background: item.homeColor }} />
        <b className="wall__tk-ab">{item.homeAb}</b>
        {item.poss === 'R' && <i className="wall__tk-poss" aria-label="possession"
          style={{ background: item.homeColor }} />}
        <em className="wall__tk-sc">{item.status === 'pre' ? '–' : item.homeScore}</em>
      </span>
      <span className="wall__tk-clk">
        {item.status === 'in' && <i className="wall__tk-live" />}
        {item.clock}
      </span>
    </li>)}
  </ul>;
}

function SpotlightBanner({ item }: { item: SpotlightItem }) {
  return <aside className="wall__banner" role="status" aria-live="assertive"
    data-testid="wall-spotlight" data-kind={item.kind} style={{
      ['--spot-c' as string]: item.color,
      ['--spot-f' as string]: item.fill,
      ['--spot-ink' as string]: item.ink,
    }}>
    <div className="wall__banner-top">
      <span className="wall__banner-tag">{item.tag}</span>
      <span className="wall__banner-sub">{item.subtitle}</span>
    </div>
    <h2 className="wall__banner-title">{item.title}</h2>
    <p className="wall__banner-text">{item.text}</p>
  </aside>;
}

function BigPlaysRail({ plays, firstDowns, mode, width, height }: {
  plays: BigPlay[]; firstDowns: FirstDownPlay[];
  mode: Exclude<RailMode, 'none'>; width: number; height: number;
}) {
  const cols = mode === 'bottom' ? (width >= 1100 ? 4 : width >= 680 ? 2 : 1) : 1;
  const compact = mode === 'bottom' && height < 340;
  return <aside className={`wall__rail wall__rail--${mode}${compact ? ' wall__rail--tight' : ''}`}
    aria-label="Big plays and first downs" data-testid="wall-big-plays" style={{
      width: mode === 'right' ? width : '100%',
      height: mode === 'bottom' ? height : '100%',
      ['--rail-cols' as string]: cols,
    }}>
    <RailHalf title="BIG PLAYS" subtitle="SCORING · TURNOVERS · 20+ YDS"
      empty="Waiting for scoring plays, turnovers and 20+ yard gains."
      testId="wall-rail-big" itemClass="wall__bp" items={plays} />
    <RailHalf title="FIRST DOWNS" subtitle="CHAINS MOVED · NEWEST FIRST"
      empty="Waiting for first downs."
      testId="wall-rail-first" itemClass="wall__fd" items={firstDowns} />
  </aside>;
}

function RailHalf({ title, subtitle, empty, testId, itemClass, items }: {
  title: string; subtitle: string; empty: string; testId: string; itemClass: string;
  items: (BigPlay | FirstDownPlay)[];
}) {
  return <section className="wall__rail-half" aria-label={title} data-testid={testId}>
    <header className="wall__rail-head">
      <b>{title}</b>
      <span>{subtitle}</span>
    </header>
    <div className="wall__rail-scroll">
      {items.length === 0
        ? <p className="wall__rail-empty">{empty}</p>
        : <ol className="wall__rail-list">
          {items.map(p => <li key={p.id} className={`${itemClass} wall__bp--${p.kind}`}
            style={{ ['--bp-c' as string]: p.color, ['--bp-f' as string]: p.fill,
              ['--bp-ink' as string]: p.ink }}>
            <div className="wall__bp-top">
              <span className="wall__bp-tag">{p.tag}</span>
              <span className="wall__bp-game">{p.matchup}</span>
              <span className="wall__bp-clk">{p.clock}</span>
            </div>
            <b className="wall__bp-title">{p.headline}</b>
            <p className="wall__bp-text">{p.text}</p>
            <span className="wall__bp-score">{p.score}</span>
          </li>)}
        </ol>}
    </div>
  </section>;
}

function Tile({ game, scale, bezel, headH, tileGap, phoneW, phoneH, reduced, spotlight }: {
  game: Game; scale: number; bezel: number; headH: number; tileGap: number;
  phoneW: number; phoneH: number; reduced: boolean; spotlight?: SpotlightItem;
}) {
  const name = `${game.away.abbreviation} at ${game.home.abbreviation}`;
  return <section className={`wall__tile${spotlight ? ' wall__tile--spot' : ''}`}
    aria-label={name} data-game-id={game.id} data-spotlight={spotlight ? 'on' : undefined}
    style={{
      width: phoneW,
      gap: tileGap,
      ['--k' as string]: scale,
      ...(spotlight ? { ['--spot-c' as string]: spotlight.color,
        ['--spot-f' as string]: spotlight.fill } : {}),
    }}>
    <header className="wall__head" style={{ height: headH }}>
      <b>{name}</b>
      <span>{clockWords(game)}</span>
    </header>
    <div className="wall__phone" style={{ width: phoneW, height: phoneH }}>
      <div className="wall__bezel" style={{ transform: `scale(${scale})`, padding: bezel }}>
        <div className="wall__screen live6">
          <LiveField game={game} reduced={reduced} />
          <section className="live6__card" aria-label="Play by play">
            <div className="live6__head">
              <b>Play by play</b><span className="live6__m">newest first</span>
            </div>
            <WallPlays game={game} />
          </section>
        </div>
      </div>
    </div>
  </section>;
}

function WallPlays({ game }: { game: Game }) {
  if (!game.plays.length) return <p className="live6__note">{game.status === 'pre'
    ? 'The game has not kicked off.' : 'No plays are held for this game yet.'}</p>;
  const abbr = (id?: string) => id === game.home.id ? game.home.abbreviation
    : id === game.away.id ? game.away.abbreviation : '·';
  return <ol className="live6__plays">{[...game.plays].sort((a, b) => b.sequence - a.sequence).slice(0, 6)
    .map(p => <li key={p.id}><span className="live6__m">Q{p.period} {p.clock}</span>
      <b>{abbr(p.start?.possession)}</b><span>{p.text}</span>
      <em>{p.awayScore}-{p.homeScore}</em></li>)}</ol>;
}

export default LiveWall;
