import { array, choice, integer, object, text, timestamp } from '../data/parseValues';

// The Go jobs reading (TargetCommissionerJobs): what the commissioner is working on, worked out in Go
// from the league's rules. Go also orders it: live jobs first. The strip shows it as it arrives.
const jobStates = ['live', 'upcoming', 'blocked', 'unbuilt'] as const;
export type JobState = (typeof jobStates)[number];
export type Job = {
  id: string; name: string; state: JobState; words: string;
  // RFC 3339 instants; empty when the state has none.
  opens: string; closes: string;
  // The census rows the job opens. They find the job's group; they are never shown.
  controls: string[];
};
export type Jobs = { league: string; name: string; revision: number; jobs: Job[] };

export const startingJob = (jobs?: Jobs) => jobs?.jobs.find(job => job.id === 'seed');

export function parseJobs(v: unknown): Jobs {
  const r = object(v, 'League jobs', ['league', 'name', 'revision', 'jobs']);
  return {
    league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'League version'),
    jobs: array(r.jobs, 'Jobs', (j, p) => {
      const o = object(j, p, ['id', 'name', 'state', 'words', 'opens', 'closes', 'controls']);
      return {
        id: text(o.id, p), name: text(o.name, p), state: choice(o.state, `${p}.state`, jobStates),
        words: text(o.words, p), opens: timestamp(o.opens, `${p}.opens`),
        closes: timestamp(o.closes, `${p}.closes`), controls: array(o.controls, `${p}.controls`, text),
      };
    }),
  };
}

export { localTime } from '../clock/localTime';

const unit = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;
// How long until an instant, in the two largest units that matter.
export function countdown(iso: string, nowMs: number): string {
  if (Date.parse(iso) <= nowMs) return 'Deadline passed';
  const minutes = Math.floor((Date.parse(iso) - nowMs) / 60_000);
  const days = Math.floor(minutes / 1440), hours = Math.floor((minutes % 1440) / 60);
  if (days > 0) return `${unit(days, 'day')} ${unit(hours, 'hour')}`;
  if (hours > 0) return `${unit(hours, 'hour')} ${unit(minutes % 60, 'minute')}`;
  return minutes > 0 ? unit(minutes, 'minute') : 'under a minute';
}
