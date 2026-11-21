import { describe, expect, it } from 'vitest';
import { DIVISIONS_UNAVAILABLE, filterSeasonWindows, groupSeasonActions, scheduleSummaryRows,
  seasonWindowName, seasonWindowOpens, type SeasonWindow } from './CtrlSeasonV3';

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

describe('seasonWindowName and seasonWindowOpens', () => {
  it('names calendar windows even when opens is empty (such as the contracts window)', () => {
    expect(seasonWindowName({ id: 'contracts', opens: [] })).toBe('Contracts');
    expect(seasonWindowOpens({ opens: [] })).toBe('');
  });

  it('formats known and custom calendar window names and actions', () => {
    expect(seasonWindowName({ id: 'rfa_tenders', opens: ['rfa_tender'] })).toBe('RFA tenders');
    expect(seasonWindowName({ id: 'open_fa', opens: ['ufa_bid'] })).toBe('Open FA');
    expect(seasonWindowName({ id: 'cut_buyout', opens: ['cut', 'buyout'] })).toBe('Cut / buyout');
    expect(seasonWindowName({ id: 'spring_cleanup', opens: ['cut'] })).toBe('Spring cleanup');
    expect(seasonWindowName({ id: '', opens: ['lineup'] })).toBe('Lineup');
    expect(seasonWindowOpens({ opens: ['cut', 'waiver_claim'] })).toBe('Cut, Waiver claim');
  });
});

describe('divisions availability notice', () => {
  it('uses plain language without developer placeholder text', () => {
    expect(DIVISIONS_UNAVAILABLE.toLowerCase()).not.toContain('not on the server app yet');
    expect(DIVISIONS_UNAVAILABLE).toContain('Division');
  });
});
