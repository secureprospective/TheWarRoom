import { POSITIONS } from './contract';
import type { LineupCheck, LineupPlayer, LineupProblem, LineupReading } from './contract';
import { object, id, choice, text, integer, array } from './parseValues';
import { boolean, provenance } from './parse';

function player(value: unknown, path: string): LineupPlayer {
  const r = object(value, path, ['id', 'position']);
  return {
    id: id(r.id, `${path}.id`, true),
    ...(r.position === undefined ? {} : { position: choice(r.position, `${path}.position`, POSITIONS) }),
  };
}
function problem(value: unknown, path: string): LineupProblem {
  const r = object(value, path, ['subject', 'kind', 'message']);
  return {
    subject: text(r.subject, `${path}.subject`),
    kind: choice(r.kind, `${path}.kind`, ['short', 'over', 'unknown'] as const),
    message: text(r.message, `${path}.message`),
  };
}
function check(value: unknown, path: string): LineupCheck {
  const r = object(value, path, ['full', 'legal', 'problems']);
  return {
    full: boolean(r.full, `${path}.full`),
    legal: boolean(r.legal, `${path}.legal`),
    problems: array(r.problems, `${path}.problems`, problem),
  };
}
export function parseLineup(value: unknown): LineupReading {
  const p = 'lineup';
  const r = object(value, p, [
    'franchise', 'week', 'starterCount', 'starters', 'bench', 'check', 'provenance', 'rulesSource',
  ]);
  return {
    franchise: id(r.franchise, `${p}.franchise`),
    week: integer(r.week, `${p}.week`),
    starterCount: integer(r.starterCount, `${p}.starterCount`),
    starters: array(r.starters, `${p}.starters`, player),
    bench: array(r.bench, `${p}.bench`, player),
    check: check(r.check, `${p}.check`),
    provenance: provenance(r.provenance, `${p}.provenance`),
    rulesSource: provenance(r.rulesSource, `${p}.rulesSource`),
  };
}
