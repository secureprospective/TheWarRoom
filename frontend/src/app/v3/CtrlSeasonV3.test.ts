import { describe, expect, it } from 'vitest';
import { filterSeasonWindows, groupSeasonActions, scheduleSummaryRows,
  type SeasonWindow } from './CtrlSeasonV3';

const windows: SeasonWindow[] = [
  { id: 'lineup', opens: ['lineup'], start: '2026-09-01T00:00:00Z', live: true },
  { id: 'waiver', opens: ['waiver', 'trade'], start: '2026-09-08T00:00:00Z', live: false },
];

describe('filterSeasonWindows', () => {
  it('returns all windows or only live ones', () => {
    expect(filterSeasonWindows(windows, 'all')).toHaveLength(2);
    expect(filterSeasonWindows(windows, 'live').map(w => w.id)).toEqual(['lineup']);
    expect(filterSeasonWindows(undefined, 'all')).toEqual([]);
  });
});

describe('scheduleSummaryRows', () => {
  it('formats published weeks, games and draft weeks', () => {
    expect(scheduleSummaryRows({ published: 14, games: 196, draft: 2 })).toEqual([
      { label: 'Published weeks', value: 14 },
      { label: 'Games', value: 196 },
      { label: 'Draft weeks', value: 2 },
    ]);
    expect(scheduleSummaryRows(undefined)).toEqual([]);
  });
});

describe('groupSeasonActions', () => {
  const available: [string, string][] = [
    ['schedule.generate', 'Generate schedule'],
    ['schedule.edit', 'Edit a game'],
    ['bracket.build', 'Build playoff bracket'],
    ['title.set', 'Crown champion'],
  ];
  it('separates schedule actions from playoff and title actions', () => {
    expect(groupSeasonActions(available, 'schedule').map(([cmd]) => cmd))
      .toEqual(['schedule.generate', 'schedule.edit']);
    expect(groupSeasonActions(available, 'playoffs').map(([cmd]) => cmd))
      .toEqual(['bracket.build', 'title.set']);
  });
});
