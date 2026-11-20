import { describe, expect, it } from 'vitest';
import { kindCounts, seasonClock } from './CtrlJobsV3';
import type { Job, Jobs } from '../commissioner/jobs';

const job = (id: string, closes: string, controls: string[] = []): Job =>
  ({ id, name: id, state: 'live', words: '', opens: '', closes, controls });
const read = (...jobs: Job[]): Jobs => ({ league: 'L', name: 'L', revision: 1, jobs });

describe('seasonClock', () => {
  it('keeps only jobs with a deadline, soonest first', () => {
    const jobs = read(job('b', '2026-12-02T00:00:00Z'), job('none', ''), job('a', '2026-12-01T00:00:00Z'));
    expect(seasonClock(jobs).map(j => j.id)).toEqual(['a', 'b']);
  });
  it('is empty without a read', () => expect(seasonClock(undefined)).toEqual([]));
});

describe('kindCounts', () => {
  it('counts jobs by kind and leaves out empty kinds', () => {
    const jobs = read(job('1', '', ['CM-001']), job('2', '', ['CM-002']), job('3', '', ['CM-012']));
    expect(kindCounts(jobs)).toEqual([{ kind: 'Ruling', count: 2 }, { kind: 'Deadline', count: 1 }]);
  });
});
