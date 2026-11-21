import { useEffect, useRef, useState } from 'react';
import type { Game, Play } from '../now/liveContract';
import { clockWords } from '../shell/hostedGames';
import { createFieldView, type FieldGame, type FieldView } from './fieldView';
import { club, clubColours } from './liveClubs';
import { useScoreRoll } from './liveMotion';
import { bannerText, downWords, driveStart, fieldPlay, fieldSpot, freshPlays, latestSequence, PlayQueue,
  presentation, spotWords, type FieldPlay, type Side, type Spot } from './livePlays';
import './liveField.css';

/** useReducedMotion follows the device's prefers-reduced-motion setting. */
export function useReducedMotion(): boolean {
  const query = '(prefers-reduced-motion: reduce)';
  const [reduced, setReduced] = useState(() => typeof window.matchMedia === 'function'
    && window.matchMedia(query).matches);
  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return;
    const mq = window.matchMedia(query);
    const on = (e: MediaQueryListEvent) => setReduced(e.matches);
    mq.addEventListener?.('change', on);
    return () => mq.removeEventListener?.('change', on);
  }, []);
  return reduced;
}

type Moment = { play: Play; fp?: FieldPlay; after?: Spot; drive: number };
type Card = { play: Play; fp?: FieldPlay };

const fieldGame = (s: Spot, drive: number, over = false): FieldGame =>
  ({ poss: s.poss, x: s.x, lat: 0, down: Math.max(1, s.down), dist: s.dist, driveStart: drive, over });
/* scorer from the play's own words: the receiver on a touchdown pass, else the first named player */
const scorerOf = (p: FieldPlay) => (p.how === 'pass' && p.type === 'td'
  ? p.text.match(/\bto ([A-Z][\w'-]*\.\s?[\w'-]+)/)?.[1]
  : p.text.match(/^([A-Z][\w'-]*\.\s?[\w'-]+)/)?.[1]) ?? '';

const initialCard = (game: Game): Card | undefined => {
  const last = game.last ?? [...game.plays].sort((a, b) => b.sequence - a.sequence)[0];
  return last ? { play: last, fp: fieldPlay(last, game) } : undefined;
};

/** LiveField is mock v1's live block for the followed game: score bar, drive bar, the 3D field and the
 *  latest-play card. New held plays play out on the field one at a time; unknown ones show only the card. */
export function LiveField({ game, fresh, reduced: forced }: {
  game: Game; fresh?: string; reduced?: boolean;
}) {
  const prefers = useReducedMotion();
  const reduced = forced ?? prefers;
  const reducedRef = useRef(reduced); reducedRef.current = reduced;
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<FieldView>();
  const queue = useRef<PlayQueue<Moment>>();
  const seen = useRef<number>();
  const gameRef = useRef(game); gameRef.current = game;
  const [card, setCard] = useState<Card | undefined>(() => initialCard(game));
  const [banner, setBanner] = useState<{ big: string; small: string }>();
  const spot = fieldSpot(game.state, game);
  const hasSpot = Boolean(spot);
  const teamsKey = `${game.id}:${game.away.abbreviation}:${game.home.abbreviation}`;
  const mountedTeams = useRef(teamsKey);

  useEffect(() => {
    const g = gameRef.current;
    if (!host.current) return;
    if (mountedTeams.current !== teamsKey) {
      mountedTeams.current = teamsKey;
      setCard(initialCard(g));
      setBanner(undefined);
    }
    const v = createFieldView(host.current, { L: club(g.away.abbreviation), R: club(g.home.abbreviation) },
      () => reducedRef.current);
    view.current = v;
    const s = fieldSpot(g.state, g);
    if (s) void v.sync(fieldGame(s, driveStart(g, s), g.status === 'post'), { instant: true });
    seen.current = latestSequence(g);
    let timer = 0, toldTimer = 0;
    const q = new PlayQueue<Moment>(async m => {
      if (presentation(reducedRef.current) === 'banner') {
        setCard({ play: m.play, fp: m.fp });
        if (m.fp) {
          setBanner(bannerText(m.fp, gameRef.current, m.after));
          clearTimeout(timer);
          timer = window.setTimeout(() => setBanner(undefined), m.fp.scoring ? 5000 : 2600);
        }
        if (m.after) await v.sync(fieldGame(m.after, m.drive), { instant: true });
        return;
      }
      if (!m.fp) {
        setCard({ play: m.play });
        if (m.after) await v.sync(fieldGame(m.after, m.drive));
        return;
      }
      const fp = m.fp;
      const told = new Promise<void>(r => {
        clearTimeout(toldTimer);
        toldTimer = window.setTimeout(() => { toldTimer = 0; setCard({ play: m.play, fp }); r(); },
          fp.scoring ? 1500 : 320);
      });
      await v.play(fp);
      if (fp.scoring) await v.banner(fp, scorerOf(fp));
      if (m.after) await v.sync(fieldGame(m.after, m.drive), { first: fp.first, flip: fp.flip });
      await told;
    });
    queue.current = q;
    const onResize = () => v.resize();
    window.addEventListener('resize', onResize);
    return () => {
      q.close(); v.destroy(); clearTimeout(timer); clearTimeout(toldTimer);
      window.removeEventListener('resize', onResize);
      view.current = undefined;
    };
  }, [teamsKey]);

  useEffect(() => {
    if (hasSpot) view.current?.resize();
  }, [hasSpot]);

  useEffect(() => {
    const plays = freshPlays(seen.current, game);
    if (!plays.length) return;
    seen.current = latestSequence(game);
    const newest = plays[plays.length - 1];
    for (const play of plays) {
      const after = (play === newest ? fieldSpot(game.state, game) : undefined) ?? fieldSpot(play.end, game);
      queue.current?.push({ play, fp: fieldPlay(play, game), after,
        drive: after ? driveStart(game, after) : 0 });
    }
  }, [game]);

  const L = clubColours(club(game.away.abbreviation)), R = clubColours(club(game.home.abbreviation));
  const poss: Side | undefined = spot?.poss;
  const dd = spot ? downWords(spot) : '';
  return <div className={`lf hw hw-b hw-x hw-c${reduced ? ' rm' : ''}`} data-testid="live-field"
    style={{ ['--tcL' as string]: L.color, ['--tfL' as string]: L.fillc, ['--tcR' as string]: R.color,
      ['--tfR' as string]: R.fillc }}>
    <section className="score" aria-label="Score" data-poss={poss}
      style={{ ['--pc' as string]: poss ? (poss === 'L' ? L : R).color : undefined }}>
      <ScoreSide k="L" t={L} team={game.away} gameId={game.id} reduced={reduced} />
      <div className="mid"><div className="mid-clock"><b>{clockWords(game)}</b>
        {game.status === 'in' && <span className="live-dot" />}</div>
        {dd && <div className="mid-dd">{dd}</div>}
        {spot && <div className="mid-spot">{spotWords(spot.x, game)}</div>}</div>
      <ScoreSide k="R" t={R} team={game.home} gameId={game.id} reduced={reduced} />
    </section>
    {spot && <DriveBar spot={spot} drive={driveStart(game, spot)} game={game}
      colour={(poss === 'L' ? L : R).color} />}
    <section className="fieldwrap" aria-label="Field">
      <div className="fhead"><b className="lbl">NFL LIVE</b>
        <span className="fhead-r">{fresh || `${L.name} at ${R.name}`}</span></div>
      <div className="field" ref={host} hidden={!spot} />
      {!spot && <p className="m">{game.status === 'pre' ? 'The game has not kicked off.'
        : game.status === 'post' ? 'Final' : 'Field position not supplied'}</p>}
      <div className={`rmb${banner ? ' on' : ''}`} role="status" aria-live="polite">
        {banner && <><b>{banner.big}</b><span>{banner.small}</span></>}</div>
    </section>
    <section className="card" aria-label="Latest play"><PlayCard card={card} game={game} /></section>
  </div>;
}

function ScoreSide({ k, t, team, gameId, reduced }: {
  k: Side; t: ReturnType<typeof clubColours>; team: Game['home']; gameId: string; reduced: boolean;
}) {
  const { display, scoring, rollSeq } = useScoreRoll(team.score, gameId, reduced);
  return <div className={`tm tm-${k}`} style={{ ['--tc' as string]: t.color }}>
    <div className="tm-id"><b className="tm-ab" style={{ background: t.fillc, color: t.onc,
      borderColor: t.trimc }}>{team.abbreviation}</b>
      {team.record && <span className="tm-rec">
        <span className="tm-city">{t.name} · </span>{team.record}</span>}
    </div>
    <div className={`tm-sc${scoring ? ' scoring' : ''}`}>
      <span key={rollSeq} className={rollSeq > 0 && !reduced ? 'roll' : undefined}>{display}</span>
    </div>
    <i className="tm-poss" title="has the ball"><svg viewBox="0 0 26 14" aria-hidden="true">
      <path className="lth" d="M1 7C6 -.5 20 -.5 25 7C20 14.5 6 14.5 1 7Z" />
      <path className="lc" d="M8 7h10M11 5v4M13 5v4M15 5v4" /></svg></i>
  </div>;
}

/** DriveBar is the mock's drive bar: away goal left, home goal right, drive start, ball and line to gain. */
function DriveBar({ spot, drive, game, colour }: { spot: Spot; drive: number; game: Game; colour: string }) {
  const d = spot.poss === 'L' ? 1 : -1, goal = (spot.poss === 'L' ? 100 - spot.x : spot.x) <= spot.dist;
  const pc = (yd: number) => `${(yd + 10) / 120 * 100}%`;
  const at = (yd: number) => ({ left: pc(yd) });
  return <div className="dbar" data-dir={d > 0 ? 'r' : 'l'} data-testid="live-drive-bar" role="img"
    aria-label={`Ball on ${spotWords(spot.x, game)}`} style={{ ['--pc' as string]: colour }}>
    <div className="db-track"><i className="db-ez l" /><i className="db-ez r" />
      <i className="db-drive" style={{ left: pc(Math.min(drive, spot.x)),
        width: `${Math.abs(spot.x - drive) / 1.2}%`, transform: 'none' }} />
      <i className="db-start" style={{ ...at(drive), transform: 'none' }} />
      {spot.down > 0 && <i className="db-first" style={{ ...at(spot.x + d * spot.dist), transform: 'none',
        opacity: goal ? 0 : 1 }} />}
      <i className="db-ball" style={{ ...at(spot.x), transform: 'none' }} /></div>
    <div className="db-nums"><span>{game.away.abbreviation}</span><span>20</span><span>50</span>
      <span>20</span><span>{game.home.abbreviation}</span></div>
  </div>;
}

/** PlayCard is the mock's latest-play card, fed only by the held play. */
function PlayCard({ card, game }: { card?: Card; game: Game }) {
  if (!card) return <div className="card-in"><div className="c-top"><span className="tag o">LAST PLAY</span>
    <b className="c-title">{game.status === 'pre' ? 'Not kicked off' : 'No play held yet'}</b></div></div>;
  const { play, fp } = card;
  const start = fieldSpot(play.start, game);
  const tag = fp?.tag || 'LAST PLAY';
  const ab = fp ? (fp.poss === 'L' ? game.away : game.home) : undefined;
  const tc = ab ? clubColours(club(ab.abbreviation)).color : undefined;
  return <div key={play.id} className={`card-in${fp?.scoring ? ' is-score' : ''}${fp?.flip && !fp.scoring
    ? ' is-turn' : ''}`} style={{ ['--tc' as string]: tc }}>
    <div className="c-top">
      <span className={`tag ${fp?.scoring ? 'am' : tag === 'FLAG' ? 'yl' : 'o'}`}>{tag}</span>
      <b className="c-title">{fp?.title ?? 'Last play'}</b></div>
    <p className="c-text">{play.text}</p>
    <div className="c-ctx">
      {start && start.down > 0 ? `${downWords(start)} at ${spotWords(start.x, game)} · ` : ''}
      Q{play.period} {play.clock}</div>
  </div>;
}
