import { expect, it } from 'vitest';
import type { Job, Jobs } from './jobs';
import { filterJobLanes, jobKind, orderJobLanes } from './hostedJobs';

const now = Date.parse('2026-11-17T12:00:00Z');
function job(id: string, state: Job['state'], due = '', controls: string[] = []): Job {
  return { id, name: id, state, words: '', opens: '', closes: due, controls };
}
function reading(jobs: Job[]): Jobs {
  return { league: 'native.test', name: 'Test league', revision: 3, jobs };
}

it('orders all six lanes and prioritizes due-now work, then blockers, then the nearest due time', () => {
  const jobs = [
    job('soon', 'live', new Date(now + 60 * 60 * 1000).toISOString()),
    job('blocker-now', 'blocked', new Date(now + 90 * 60 * 1000).toISOString()),
    job('overdue', 'live', new Date(now - 60 * 1000).toISOString()),
    job('blocker-later', 'blocked', new Date(now + 8 * 60 * 60 * 1000).toISOString()),
    job('today', 'upcoming', new Date(now + 10 * 60 * 60 * 1000).toISOString()),
    job('week', 'upcoming', new Date(now + 2 * 24 * 60 * 60 * 1000).toISOString()),
    job('later', 'upcoming', new Date(now + 8 * 24 * 60 * 60 * 1000).toISOString()),
    job('no-clock', 'unbuilt'),
  ];
  const lanes = orderJobLanes(reading(jobs), now);
  expect(lanes.now.map(item => item.id)).toEqual(['overdue', 'blocker-now', 'soon']);
  expect(lanes.holding.map(item => item.id)).toEqual(['blocker-later']);
  expect(lanes.today.map(item => item.id)).toEqual(['today']);
  expect(lanes.week.map(item => item.id)).toEqual(['week']);
  expect(lanes.later.map(item => item.id)).toEqual(['later']);
  expect(lanes['no-clock'].map(item => item.id)).toEqual(['no-clock']);
});

it('orders all jobs at 32-team depth and filters by actTable kind', () => {
  const hours = (value: number) => now + value * 60 * 60 * 1000;
  const all = Array.from({ length: 32 }, (_, index) => {
    const state: Job['state'] = index === 1 || index === 3 ? 'blocked'
      : index === 0 || index === 2 ? 'live' : 'upcoming';
    const due = index === 0 ? hours(-1) : index === 1 ? hours(1) : index === 2 ? hours(0.5)
      : index === 3 ? hours(8) : index === 4 ? hours(10) : index === 5 ? hours(24 * 8)
        : hours(24 * 2);
    return job(`team-${index + 1}`, state, new Date(due).toISOString(),
      [index === 0 ? 'CM-019' : 'CM-036']);
  });
  const lanes = orderJobLanes(reading(all), now);
  expect(Object.values(lanes).flat()).toHaveLength(32);
  expect(lanes.now.slice(0, 3).map(item => item.id)).toEqual(['team-1', 'team-2', 'team-3']);
  expect(lanes.holding.map(item => item.id)).toEqual(['team-4']);
  expect(lanes.today.map(item => item.id)).toEqual(['team-5']);
  expect(lanes.week).toHaveLength(26);
  expect(lanes.later.map(item => item.id)).toEqual(['team-6']);
  expect(lanes.week.filter(item => jobKind(item) === 'People')).toHaveLength(0);
  expect(filterJobLanes(lanes, 'People').now.map(item => item.id)).toEqual(['team-1']);
  expect(Object.values(filterJobLanes(lanes, 'Deadline')).flat()).toHaveLength(31);
});
