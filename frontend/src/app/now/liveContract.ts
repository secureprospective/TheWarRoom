import { array, choice, integer, object, requiredText, text, timestamp, id } from '../data/parseValues';
import { report, type Report } from '../reports/contract';

export type FieldState = { possession: string; down: number;
  distance: number; yardLine: number; toEndzone: number };
export type Play = { id: string; sequence: number; text: string; clock: string; period: number; at: string;
  homeScore: number; awayScore: number; start: FieldState | null; end: FieldState | null;
  participants: string[] };
export type Team = { id: string; abbreviation: string; score: number };
export type Box = { team: string; category: string; labels: string[];
  rows: { name: string; stats: string[] }[] };
export type Game = { id: string; kickoff: string; home: Team; away: Team; status: 'pre' | 'in' | 'post';
  clock: string; period: number; state: FieldState | null; last: Play | null; box: Box[]; plays: Play[] };
export type Spine = { player: { id: string; name: string; team: string; mine: boolean;
  photo: { src: string; alt: string } }; play: Play; game: string };
export type LiveView = { games: Game[]; followed: Game | null; source: string; asOf: string; failure: string;
  detailAsOf: string; detailFailure: string; scoring: Play[]; spine: Spine[]; unmatched: number;
  weekSpine: Spine[]; weekHeld: number; weekGames: number; box: Report; active: boolean };
function count(v: unknown, p: string, max: number, min = 0): number {
  const n = integer(v, p, min); if (n > max) throw new Error(`${p}: out of range`); return n;
}
function bool(v: unknown, p: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${p}: expected boolean`); return v;
}
function field(v: unknown, p: string): FieldState | null {
  if (v === null) return null;
  const r = object(v, p, ['possession', 'down', 'distance', 'yardLine', 'toEndzone']);
  return { possession: text(r.possession, p), down: count(r.down, p, 4, -1),
    distance: count(r.distance, p, 100),
    yardLine: count(r.yardLine, p, 100), toEndzone: count(r.toEndzone, p, 100, -1) };
}
function play(v: unknown, p: string): Play {
  const r = object(v, p, ['id', 'sequence', 'text', 'clock', 'period', 'at', 'homeScore', 'awayScore',
    'start', 'end', 'participants']);
  return { id: requiredText(r.id, p), sequence: integer(r.sequence, p), text: requiredText(r.text, p),
    clock: text(r.clock, p),
    period: count(r.period, p, 10), at: timestamp(r.at, p),
    homeScore: count(r.homeScore, p, 200), awayScore: count(r.awayScore, p, 200),
    start: field(r.start, p), end: field(r.end, p),
    participants: array(r.participants, p, (v, p) => id(v, p)) };
}
function team(v: unknown, p: string): Team {
  const r = object(v, p, ['id', 'abbreviation', 'score']);
  const abbreviation = text(r.abbreviation, p);
  if (!/^[A-Z]{2,3}$/.test(abbreviation)) throw new Error(`${p}: invalid team abbreviation`);
  return { id: text(r.id, p), abbreviation, score: count(r.score, p, 200) };
}
function box(v: unknown, p: string): Box {
  const r = object(v, p, ['team', 'category', 'labels', 'rows']);
  const labels = array(r.labels, p, text);
  const rows = array(r.rows, p, (v, p) => {
    const a = object(v, p, ['name', 'stats']); const stats = array(a.stats, p, text);
    if (stats.length !== labels.length) throw new Error(`${p}: stats/labels mismatch`);
    return { name: requiredText(a.name, p), stats };
  });
  return { team: text(r.team, p), category: text(r.category, p), labels, rows };
}
export function game(v: unknown, p: string): Game {
  const r = object(v, p, ['id', 'kickoff', 'home', 'away', 'status', 'clock', 'period', 'state', 'last',
    'box', 'plays']);
  return { id: requiredText(r.id, p), kickoff: timestamp(r.kickoff, p), home: team(r.home, p),
    away: team(r.away, p),
    status: choice(r.status, p, ['pre', 'in', 'post']), clock: text(r.clock, p),
    period: count(r.period, p, 10),
    state: field(r.state, p), last: r.last === null ? null : play(r.last, p), box: array(r.box, p, box),
    plays: array(r.plays, p, play) };
}
function spine(v: unknown, p: string): Spine {
  const r = object(v, p, ['player', 'play', 'game']);
  const a = object(r.player, p, ['id', 'name', 'team', 'mine', 'photo']);
  const photo = object(a.photo, p, ['src', 'alt']);
  const src = text(photo.src, p);
  if (src !== '' && !/^\/headshot\/\d+$/.test(src)) throw new Error(`${p}: invalid cached photo path`);
  return { player: { id: id(a.id, p, true), name: requiredText(a.name, p), team: text(a.team, p),
    mine: bool(a.mine, p), photo: { src, alt: text(photo.alt, p) } }, play: play(r.play, p),
    game: text(r.game, p) };
}
export function parseLive(v: unknown): LiveView {
  const p = 'Live game';
  const r = object(v, p, ['games', 'followed', 'source', 'asOf', 'failure',
    'detailAsOf', 'detailFailure', 'scoring', 'spine', 'unmatched', 'weekSpine', 'weekHeld', 'weekGames',
    'box', 'active']);
  return { games: array(r.games, p, game), followed: r.followed === null ? null : game(r.followed, p),
    source: text(r.source, p), asOf: text(r.asOf, p), failure: text(r.failure, p),
    detailAsOf: text(r.detailAsOf, p),
    detailFailure: text(r.detailFailure, p), scoring: array(r.scoring, p, play),
    spine: array(r.spine, p, spine), unmatched: integer(r.unmatched, p),
    weekSpine: array(r.weekSpine, p, spine), weekHeld: integer(r.weekHeld, p),
    weekGames: integer(r.weekGames, p),
    box: report(r.box, p), active: bool(r.active, p) };
}
