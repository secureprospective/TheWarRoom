import type { Game, Play } from '../now/liveContract';
import { club, clubColours } from '../phone/liveClubs';
import { fieldPlay, type FieldPlay, type Side } from '../phone/livePlays';
import { clockWords } from '../shell/hostedGames';

export type TickerItem = {
  id: string;
  awayAb: string; awayScore: number; awayColor: string;
  homeAb: string; homeScore: number; homeColor: string;
  clock: string; status: Game['status']; poss?: Side;
};

export type BigPlayKind = 'td' | 'fg' | 'turnover' | 'gain';

export type BigPlay = {
  id: string; gameId: string; playId: string;
  kind: BigPlayKind; tag: string;
  teamAb: string; color: string; fill: string; ink: string;
  headline: string; text: string;
  matchup: string; score: string; clock: string;
  atMs: number; sequence: number;
};

const statusRank: Record<Game['status'], number> = { in: 0, pre: 1, post: 2 };

/** buildTicker turns every game on the board into a ticker item, live games first. */
export function buildTicker(games: Game[]): TickerItem[] {
  const ordered = [...games].sort((a, b) =>
    statusRank[a.status] - statusRank[b.status] ||
    (Date.parse(a.kickoff) || 0) - (Date.parse(b.kickoff) || 0) ||
    a.id.localeCompare(b.id));
  return ordered.map(g => {
    const awayC = clubColours(club(g.away.abbreviation));
    const homeC = clubColours(club(g.home.abbreviation));
    const poss: Side | undefined = g.status !== 'in' ? undefined
      : g.state?.possession === g.away.id ? 'L'
        : g.state?.possession === g.home.id ? 'R' : undefined;
    return {
      id: g.id,
      awayAb: g.away.abbreviation, awayScore: g.away.score, awayColor: awayC.color,
      homeAb: g.home.abbreviation, homeScore: g.home.score, homeColor: homeC.color,
      clock: clockWords(g), status: g.status, poss,
    };
  });
}

/** isTurnover identifies plays that flip possession as turnovers (interceptions, lost fumbles, downs). */
export function isTurnover(fp: FieldPlay): boolean {
  return fp.flip && !fp.scoring && fp.type !== 'punt' && fp.type !== 'fgmiss';
}

/** isSpotlightPlay is true for scoring plays (TD, FG) and turnovers. */
export function isSpotlightPlay(fp: FieldPlay | undefined): fp is FieldPlay {
  return Boolean(fp && (fp.scoring || isTurnover(fp)));
}

const sideAb = (side: Side, game: Game) => side === 'L' ? game.away.abbreviation : game.home.abbreviation;

/** spotlightClub names the celebrating club: the offense on a score, or the defense on a turnover. */
export function spotlightClub(fp: FieldPlay, game: Game): string {
  if (fp.scoring) return sideAb(fp.poss, game);
  const winner: Side = fp.end ? fp.end.poss : fp.poss === 'L' ? 'R' : 'L';
  return sideAb(winner, game);
}

function surname(name: string): string {
  const trimmed = name.trim();
  const dot = trimmed.indexOf('.');
  return dot >= 0 ? trimmed.slice(dot + 1).trim() || trimmed : trimmed;
}

const NAME_RE = '([A-Z][\\w\'-]*\\.\\s?(?:St\\.\\s?)?[A-Z][\\w\'-]*)';
const TO_RE = new RegExp(`\\bto ${NAME_RE}`);
const INT_RE = new RegExp(`INTERCEPTED by ${NAME_RE}`, 'i');
const FUM_RE = new RegExp(`RECOVERED by [A-Z]{2,3}-${NAME_RE}`, 'i');
const LEAD_RE = new RegExp(`(?:^|\\s)${NAME_RE}`);

function playActor(fp: FieldPlay): string {
  if (fp.type === 'int') {
    const by = fp.text.match(INT_RE);
    return by ? surname(by[1]) : '';
  }
  if (fp.type === 'fumble') {
    const rec = fp.text.match(FUM_RE);
    return rec ? surname(rec[1]) : '';
  }
  if (fp.how === 'pass' || fp.type === 'pass') {
    const to = fp.text.match(TO_RE);
    return to ? surname(to[1]) : '';
  }
  const lead = fp.text.match(LEAD_RE);
  return lead ? surname(lead[1]) : '';
}

/** spotlightTitle formats a broadcast headline from the held play (e.g. "TOUCHDOWN KC — Kelce 8 yd"). */
export function spotlightTitle(fp: FieldPlay, game: Game): string {
  const team = spotlightClub(fp, game);
  const actor = playActor(fp);
  if (fp.scoring === 'td') {
    const detail = [actor, fp.gain > 0 ? `${fp.gain} yd` : ''].filter(Boolean).join(' ');
    return detail ? `TOUCHDOWN ${team} — ${detail}` : `TOUCHDOWN ${team}`;
  }
  if (fp.scoring === 'fg') {
    return fp.kick > 0 ? `FIELD GOAL ${team} — ${fp.kick} yd` : `FIELD GOAL ${team}`;
  }
  if (fp.type === 'int') {
    return actor ? `INTERCEPTION ${team} — ${actor}` : `INTERCEPTION ${team}`;
  }
  if (fp.type === 'fumble') {
    return actor ? `FUMBLE ${team} — ${actor}` : `FUMBLE RECOVERY ${team}`;
  }
  return `TURNOVER ${team}`;
}

function gamePlays(game: Game): Play[] {
  const list = [...game.plays];
  if (game.last && !list.some(p => p.id === game.last!.id)) list.push(game.last);
  return list;
}

/** selectBigPlays picks the newest `limit` scoring, turnover, or 20+ yard plays across all games. */
export function selectBigPlays(games: Game[], limit = 8): BigPlay[] {
  const found: BigPlay[] = [];
  for (const game of games) {
    for (const play of gamePlays(game)) {
      const fp = fieldPlay(play, game);
      if (!fp) continue;
      const turn = isTurnover(fp);
      const bigGain = !fp.scoring && !turn && fp.type !== 'penalty' && fp.gain >= 20;
      if (!fp.scoring && !turn && !bigGain) continue;
      const kind: BigPlayKind = fp.scoring === 'td' ? 'td' : fp.scoring === 'fg' ? 'fg'
        : turn ? 'turnover' : 'gain';
      const teamAb = fp.scoring || bigGain ? sideAb(fp.poss, game) : spotlightClub(fp, game);
      const c = clubColours(club(teamAb));
      const actor = playActor(fp);
      const tag = fp.scoring === 'td' ? 'TOUCHDOWN' : fp.scoring === 'fg' ? 'FIELD GOAL'
        : fp.type === 'int' ? 'INTERCEPTION' : turn ? 'TURNOVER' : `${fp.gain} YDS`;
      const headline = fp.scoring || turn ? spotlightTitle(fp, game)
        : [teamAb, actor, `${fp.gain} yd ${fp.type === 'pass' ? 'catch' : 'run'}`]
          .filter(Boolean).join(' · ');
      found.push({
        id: `${game.id}:${play.id}`, gameId: game.id, playId: play.id,
        kind, tag, teamAb, color: c.color, fill: c.fillc, ink: c.onc,
        headline, text: play.text,
        matchup: `${game.away.abbreviation} @ ${game.home.abbreviation}`,
        score: `${game.away.abbreviation} ${play.awayScore} · ${game.home.abbreviation} ${play.homeScore}`,
        clock: `${play.period > 4 ? 'OT' : `Q${play.period}`} ${play.clock}`,
        atMs: Date.parse(play.at) || 0, sequence: play.sequence,
      });
    }
  }
  return found
    .sort((a, b) => b.atMs - a.atMs || b.sequence - a.sequence || b.gameId.localeCompare(a.gameId))
    .slice(0, limit);
}
