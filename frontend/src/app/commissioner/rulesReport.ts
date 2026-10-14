import type { Report } from '../reports/contract';
import { scaled } from './ruleDraft';
import type { RuleRow, Rules } from './rules';

const scoringField = 'scoring.catalogue';
type Stat = { Label: string; Group: string; Used: boolean; Points: number;
  Bonuses: { At: number; Points: number }[]; Positions: string[] };
const base = { note: '', status: 'ok' as const, asOf: '', link: null, summary: [] as Report['summary'] };

// rulesReport lays every rule out as one table grouped by section, with what is waiting beside each.
// Scoring has its own tables, so it is left out here.
export function rulesReport(rules: Rules): Report {
  const rows = rules.sections.flatMap(section => section.rows.filter(r => r.field !== scoringField)
    .map(r => ({
      group: section.label, range: null,
      cells: [r.label, r.value, r.state, r.citation,
        r.pending.map(p => `${p.value}: ${p.when}`).join('\n') || 'Nothing waiting'],
    })));
  return { ...base, id: 'league-rules', title: 'League rules',
    source: `${rules.name} · League version ${rules.revision}`,
    how: 'Each setting is read from this league’s ledger. A change set to take effect from the next '
      + 'rollover waits in the last column and is not in force yet.',
    columns: ['Rule', 'Setting', 'State', 'Where it comes from', 'Waiting'], rows };
}

function stats(row: RuleRow): Stat[] {
  const held = row.current;
  if (typeof held !== 'object' || held === null) return [];
  return Object.values(held as Record<string, Stat>).filter(s => typeof s === 'object' && s !== null);
}
const statRows = (list: Stat[]) => list.slice().sort((a, b) =>
  a.Group.localeCompare(b.Group) || a.Label.localeCompare(b.Label)).map(s => ({
  group: s.Group, range: null,
  cells: [s.Label, scaled(s.Points, 4),
    s.Bonuses.map(b => `At ${scaled(b.At, 4)}: ${scaled(b.Points, 4)}`).join('\n') || 'None',
    s.Positions.join(', ') || 'None'],
}));

// scoringReports splits the scoring schema into the stats in use and the ones left unused.
export function scoringReports(row: RuleRow): { used: Report; unused: Report | null } {
  const all = stats(row);
  const make = (id: string, title: string, list: Stat[], source: string): Report => ({
    ...base, id, title, source, columns: ['Stat', 'Points', 'Bonuses', 'Positions'], rows: statRows(list),
    how: 'Points are what one of the stat earns. A bonus adds its points once the stat reaches '
      + 'the level shown.',
  });
  const unused = all.filter(s => !s.Used);
  return {
    used: make('league-scoring', 'Scoring: stats in use', all.filter(s => s.Used), row.citation),
    unused: unused.length === 0 ? null : make('league-scoring-unused',
      `Stats not used (${unused.length})`, unused, 'These stats exist but earn nothing in this league.'),
  };
}
