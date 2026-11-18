import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import type { HomeBrief } from './homeReads';
import { TileBody } from './homeTiles';

const form = [{ week: 4, result: 'W' as const }, { week: 5, result: 'L' as const }];
const brief = {
  pulse: { league: 'test', name: 'Test League', season: 2026, revision: 1,
    seats: [{ franchise: 'mine', name: 'Mine', abbreviation: 'M', division: '' },
      { franchise: 'other', name: 'Other', abbreviation: 'O', division: '' }],
    week: 6, weekNote: '', lastWeek: 8, fixtures: [], scheduleNote: '', forms: { mine: form,
      other: [{ week: 4, result: 'L' as const }, { week: 5, result: 'W' as const }] },
    formPending: false, formNote: 'Two weeks have a result',
    standings: { rows: [{ franchise: 'mine', record: '4–1' }, { franchise: 'other', record: '3–2' }],
      decided: 1 }, errors: [] },
  results: { league: 'test', season: 2026, week: 6, revision: 1, published: true, words: '',
    games: [{ week: 6, home: 'mine', away: 'other', homeName: 'Mine', awayName: 'Other',
      status: 'No result yet', words: '', why: [] }], scores: [], missed: [] },
  matchup: { lineups: {}, projections: {}, errors: [] },
  at: { pulse: '2026-11-18T00:00:00Z', results: '2026-11-18T00:00:00Z' }, errors: {},
} as unknown as HomeBrief;

it('uses shared W/L form in the Home matchup and recent-form tile', () => {
  const matchup = renderToStaticMarkup(<TileBody id="mu" b={brief} mine="mine" pins={[]} />);
  expect(matchup).toContain('4–1');
  expect(matchup).toContain('Week 4 · Win');
  expect(matchup).toContain('Week 5 · Loss');
  expect(matchup).toContain('Week 4 · Loss');

  const tile = renderToStaticMarkup(<TileBody id="form" b={brief} mine="mine" pins={[]} />);
  expect(tile).toContain('MY RECENT FORM');
  expect(tile).toContain('Week 4 · Win');
  expect(tile).toContain('Week 5 · Loss');

  const unavailableBrief = { ...brief,
    pulse: { ...brief.pulse, formError: 'results source offline', forms: {} } } as unknown as HomeBrief;
  const unavailableTile = renderToStaticMarkup(<TileBody id="form" b={unavailableBrief} mine="mine"
    pins={[]} />);
  expect(unavailableTile).toContain('Unavailable · results source offline');
});
