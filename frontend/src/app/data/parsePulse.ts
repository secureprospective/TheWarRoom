import type { PulseMatchup, PulsePlayer, PulseReading, PulseSide } from './contract';
import { array, id, integer, object, text } from './parseValues';
import { provenance } from './parse';

function score(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(`${path}: expected finite number`);
  }
  return value;
}
function player(value: unknown, path: string): PulsePlayer {
  const r = object(value, path, ['id', 'name', 'position', 'team', 'score', 'secondsRemaining']);
  return {
    id: id(r.id, `${path}.id`, true),
    name: text(r.name, `${path}.name`),
    position: text(r.position, `${path}.position`),
    team: text(r.team, `${path}.team`),
    score: score(r.score, `${path}.score`),
    secondsRemaining: integer(r.secondsRemaining, `${path}.secondsRemaining`),
  };
}
function side(value: unknown, path: string): PulseSide {
  const r = object(value, path, [
    'franchiseId', 'name', 'score', 'secondsRemaining', 'playing', 'yetToPlay', 'starters',
  ]);
  return {
    franchiseId: id(r.franchiseId, `${path}.franchiseId`),
    name: text(r.name, `${path}.name`),
    score: score(r.score, `${path}.score`),
    secondsRemaining: integer(r.secondsRemaining, `${path}.secondsRemaining`),
    playing: integer(r.playing, `${path}.playing`),
    yetToPlay: integer(r.yetToPlay, `${path}.yetToPlay`),
    starters: array(r.starters, `${path}.starters`, player),
  };
}
function matchup(value: unknown, path: string): PulseMatchup {
  const r = object(value, path, ['home', 'away']);
  return { home: side(r.home, `${path}.home`), away: side(r.away, `${path}.away`) };
}
export function parsePulse(value: unknown): PulseReading {
  const r = object(value, 'pulse', ['week', 'matchups', 'provenance']);
  return {
    week: integer(r.week, 'pulse.week'),
    matchups: array(r.matchups, 'pulse.matchups', matchup),
    provenance: provenance(r.provenance, 'pulse.provenance'),
  };
}
