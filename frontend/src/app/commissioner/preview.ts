import { array, choice, integer, object, text } from '../data/parseValues';
import type { Safeguard } from './rules';

export type Preview = {
  question: string;
  target: { league: string; name: string; version: number };
  fields: { label: string; before: string; beforeSource: string; after: string;
    afterSource: string; when: string }[];
  affects: { readings: string[]; count: number | null; words: string };
  blockers: string[];
  // safeguard is null only for a refused change: it has blockers and nothing to confirm.
  safeguard: Safeguard | null;
};

export function parsePreview(value: unknown): Preview {
  const p = object(value, 'Change preview',
    ['question', 'target', 'fields', 'affects', 'blockers', 'safeguard']);
  const t = object(p.target, 'Target', ['league', 'name', 'version']);
  const a = object(p.affects, 'Affects', ['readings', 'count', 'words']);
  const blockers = array(p.blockers, 'Blockers', text);
  return {
    question: text(p.question, 'Question'),
    target: { league: text(t.league, 'League'), name: text(t.name, 'League name'),
      version: integer(t.version, 'League version') },
    fields: array(p.fields, 'Fields', (value, path) => {
      const f = object(value, path, ['label', 'before', 'beforeSource', 'after', 'afterSource', 'when']);
      return { label: text(f.label, path), before: text(f.before, path), after: text(f.after, path),
        beforeSource: text(f.beforeSource, path), afterSource: text(f.afterSource, path),
        when: text(f.when, path) };
    }),
    affects: { readings: array(a.readings, 'Affected readings', text),
      count: a.count === null ? null : integer(a.count, 'Affected reading count'),
      words: text(a.words, 'Affected readings explanation') },
    blockers,
    safeguard: p.safeguard === '' && blockers.length > 0 ? null
      : choice(p.safeguard, 'Safeguard', ['confirm', 'hold', 'reason'] as const),
  };
}
