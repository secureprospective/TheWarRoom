import { describe, expect, it } from 'vitest';
import { parseJobs } from './jobs';

describe('commissioner jobs', () => {
  it('keeps the matter identity and settling commands', () => {
    const result = parseJobs({
      league: 'native.test', name: 'League', revision: 4,
      jobs: [{
        id: 'lineup.franchise.01', name: 'Lineup with a hole', state: 'live', words: 'Fill the short slot.',
        opens: '', closes: '2026-10-11T17:00:00Z', controls: ['CM-028'],
        identity: { league: 'native.test', season: 2026, week: 6, matter: 'lineup.franchise.01' },
        settles: ['commish.lineup.set', 'commish.lineup.correct'],
      }],
    });
    expect(result.jobs[0].identity).toEqual({
      league: 'native.test', season: 2026, week: 6, matter: 'lineup.franchise.01',
    });
    expect(result.jobs[0].settles).toEqual(['commish.lineup.set', 'commish.lineup.correct']);
  });

  it('keeps older server rows readable without fabricating a matter identity', () => {
    const result = parseJobs({
      league: 'native.test', name: 'League', revision: 4,
      jobs: [{ id: 'setup', name: 'Setup', state: 'live', words: '', opens: '', closes: '', controls: [] }],
    });
    expect(result.jobs[0]?.identity).toBeUndefined();
    expect(result.jobs[0]?.settles).toBeUndefined();
  });
});
