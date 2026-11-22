import { Fragment } from 'react';
import type { Game } from '../now/liveContract';
import { club, clubColours } from './liveClubs';

export type PregameGuy = {
  id: string;
  name: string;
  position: string;
  team: string;
  side: string;
  mine: boolean;
  projection?: number;
  average?: number;
};

const NICKNAMES: Record<string, string> = {
  ARI: 'CARDINALS', ATL: 'FALCONS', BAL: 'RAVENS', BUF: 'BILLS',
  CAR: 'PANTHERS', CHI: 'BEARS', CIN: 'BENGALS', CLE: 'BROWNS',
  DAL: 'COWBOYS', DEN: 'BRONCOS', DET: 'LIONS', GB: 'PACKERS',
  HOU: 'TEXANS', IND: 'COLTS', JAX: 'JAGUARS', KC: 'CHIEFS',
  LV: 'RAIDERS', LAC: 'CHARGERS', LAR: 'RAMS', MIA: 'DOLPHINS',
  MIN: 'VIKINGS', NE: 'PATRIOTS', NO: 'SAINTS', NYG: 'GIANTS',
  NYJ: 'JETS', PHI: 'EAGLES', PIT: 'STEELERS', SF: '49ERS',
  SEA: 'SEAHAWKS', TB: 'BUCCANEERS', TEN: 'TITANS', WSH: 'COMMANDERS',
};

type ClockParts = { day: string; time: string; period: string; countdown: string };

function kickoffClock(kickoff: string, now: number): ClockParts {
  const ms = Date.parse(kickoff);
  if (!kickoff || Number.isNaN(ms)) {
    return { day: 'PRE-GAME', time: 'TBD', period: 'CT', countdown: '' };
  }
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'America/Chicago',
    weekday: 'short',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  }).formatToParts(new Date(ms));
  const pick = (t: string) => parts.find(p => p.type === t)?.value ?? '';
  const day = pick('weekday').toUpperCase();
  const time = `${pick('hour')}:${pick('minute')}`;
  const period = `${pick('dayPeriod').toUpperCase()} CT`;
  const diff = ms - now;
  let countdown = 'KICKOFF SOON';
  if (diff >= 86_400_000) {
    const days = Math.floor(diff / 86_400_000);
    const hrs = Math.floor((diff % 86_400_000) / 3_600_000);
    countdown = `IN ${days}D ${hrs}H`;
  } else if (diff >= 3_600_000) {
    const hrs = Math.floor(diff / 3_600_000);
    const mins = Math.floor((diff % 3_600_000) / 60_000);
    countdown = `IN ${hrs}H ${mins}M`;
  } else if (diff >= 60_000) {
    countdown = `IN ${Math.floor(diff / 60_000)}M`;
  } else if (diff > 0) {
    countdown = 'IN <1M';
  }
  return { day, time, period, countdown };
}

function stadiumHeading(game: Game): string {
  const v = game.venue;
  if (!v) return '';
  const city = v.city ? (v.state ? `${v.city}, ${v.state}` : v.city) : '';
  return [v.name, city].filter(Boolean).join(' · ').toUpperCase();
}

function weatherLine(game: Game): string {
  if (game.venue?.indoor || game.weather?.text === 'Indoors') return 'Indoors';
  const w = game.weather;
  if (!w) return '';
  const parts: string[] = [];
  if (w.tempF !== undefined) parts.push(`${w.tempF}°F`);
  if (w.text) parts.push(w.text);
  if (w.wind) parts.push(`Wind ${w.wind}`);
  if (w.precipPct !== undefined) parts.push(`${w.precipPct}% rain`);
  return parts.join(' · ');
}

/** LivePregame renders the Broadcast pre-game preview (mock section B) before kickoff. */
export function LivePregame({ game, guys, now, reduced }: {
  game: Game;
  guys: PregameGuy[];
  now: number;
  reduced: boolean;
}) {
  const awayClub = club(game.away.abbreviation);
  const homeClub = club(game.home.abbreviation);
  const awayCol = clubColours(awayClub);
  const homeCol = clubColours(homeClub);
  const awayNick = NICKNAMES[awayClub.ab] ?? awayClub.name;
  const homeNick = NICKNAMES[homeClub.ab] ?? homeClub.name;
  const clock = kickoffClock(game.kickoff, now);

  const pred = game.predictor;
  const predTotal = pred ? pred.away + pred.home : 0;
  const hasPred = Boolean(pred && predTotal > 0);
  const awayPct = pred ? Math.round(pred.away) : 0;
  const homePct = pred ? Math.round(pred.home) : 0;
  const awayWidth = pred && predTotal > 0 ? (pred.away / predTotal) * 100 : 0;

  const heading = stadiumHeading(game);
  const conditions = weatherLine(game);
  const showWeatherCredit = Boolean(conditions && conditions !== 'Indoors');
  const hasThird = Boolean(heading || conditions);

  return <div className="lf bc-wrap" data-testid="live-pregame" style={{
    ['--bc-lp' as string]: awayClub.pri,
    ['--bc-ls' as string]: awayClub.sec,
    ['--bc-lon' as string]: awayCol.onc,
    ['--bc-rp' as string]: homeClub.pri,
    ['--bc-rs' as string]: homeClub.sec,
    ['--bc-ron' as string]: homeCol.onc,
  }}>
    <div className="bc-stage" aria-label={`${game.away.abbreviation} at ${game.home.abbreviation}`}>
      <div className="bc-l">
        <span className="bc-ab">{game.away.abbreviation}</span>
        <span className="bc-nm">{awayNick}</span>
        {game.away.record && <span className="bc-rec">{game.away.record}</span>}
      </div>
      <div className="bc-r">
        <span className="bc-ab">{game.home.abbreviation}</span>
        <span className="bc-nm">{homeNick}</span>
        {game.home.record && <span className="bc-rec">{game.home.record}</span>}
      </div>
      <div className="bc-vs" data-testid="pregame-clock">
        <b>{clock.day}</b>
        <b className="t">{clock.time}</b>
        <b>{clock.period}</b>
        {clock.countdown && <span className="bc-cd">{clock.countdown}</span>}
      </div>
    </div>

    {hasPred && <div className="bc-wp" data-testid="pregame-winbar" role="img"
      aria-label={`${game.away.abbreviation} ${awayPct}% · `
        + `${game.home.abbreviation} ${homePct}% win chance`}>
      <span style={{ width: `${awayWidth}%`, background: awayClub.pri }} />
      <span style={{ width: `${100 - awayWidth}%`, background: homeClub.pri }} />
      <em className="l">{game.away.abbreviation} {awayPct}%</em>
      <em className="c">WIN CHANCE</em>
      <em className="r">{game.home.abbreviation} {homePct}%</em>
    </div>}

    {hasThird && <div className="bc-third" data-testid="pregame-stadium">
      <div className="bc-tag">FROM<br />THE<br />STADIUM</div>
      <div className="bc-tx">
        {heading && <b>{heading}</b>}
        {conditions && <span data-testid="pregame-weather">{conditions}</span>}
        {showWeatherCredit && <small className="bc-wx-credit"
          data-testid="pregame-weather-credit">Weather: Open-Meteo</small>}
      </div>
    </div>}

    {guys.length > 0 && <div className={`bc-crawl${reduced ? ' is-static' : ''}`}
      data-testid="pregame-crawl" aria-label="Your guys in this game">
      <span className="bc-live">YOUR GUYS</span>
      <div className="bc-ct">
        <div className="bc-cr">
          <CrawlItems guys={guys} dup={false} />
          {!reduced && <CrawlItems guys={guys} dup={true} />}
        </div>
      </div>
    </div>}
  </div>;
}

function guyStat(guy: PregameGuy): string {
  if (guy.projection !== undefined && Number.isFinite(guy.projection)) {
    return `proj ${guy.projection.toFixed(1)}`;
  }
  if (guy.average !== undefined && Number.isFinite(guy.average)) {
    return `avg ${guy.average.toFixed(1)}`;
  }
  return '';
}

function CrawlItems({ guys, dup }: { guys: PregameGuy[]; dup: boolean }) {
  return <>
    {guys.map(g => {
      const stat = guyStat(g);
      return <Fragment key={`${dup ? 'd' : 'p'}-${g.side}-${g.id}`}>
        <span className={`cr${dup ? ' bc-dup' : ''}`} aria-hidden={dup || undefined}>
          <b className={g.mine ? 'me' : 'op'}>{g.name}</b>{' '}
          {g.position} · {g.team} · {g.side}
          {stat && <> · <span className="cr-stat">{stat}</span></>}
        </span>
        <i className={`dot${dup ? ' bc-dup' : ''}`} aria-hidden="true" />
      </Fragment>;
    })}
  </>;
}
