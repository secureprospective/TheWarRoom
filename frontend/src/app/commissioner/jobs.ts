import { array, choice, integer, object, text, timestamp } from '../data/parseValues';

// The Go jobs reading (TargetCommissionerJobs): what the commissioner is working on, worked out in Go
// from the league's rules. Go also orders it: live jobs first. The strip shows it as it arrives.
const jobStates = ['live', 'upcoming', 'blocked', 'unbuilt'] as const;
export type JobState = (typeof jobStates)[number];
export type JobIdentity = { league: string; season: number; week: number; matter: string };
export type Job = {
  id: string; name: string; state: JobState; words: string;
  // RFC 3339 instants; empty when the state has none.
  opens: string; closes: string;
  // The census rows the job opens. They find the job's group; they are never shown.
  controls: string[];
  // Current server reads carry both. Older servers can omit them until their bundle is updated.
  identity?: JobIdentity;
  settles?: string[];
};
export type Jobs = { league: string; name: string; revision: number; jobs: Job[] };
export type JobSnooze = { league: string; jobId: string; until: string };
export type JobSnoozes = { snoozes: JobSnooze[] };

export function parseJobSnoozes(v: unknown): JobSnoozes {
  const r = object(v, 'Job snoozes', ['snoozes']);
  return { snoozes: array(r.snoozes, 'Job snoozes', (s, p) => {
    const o = object(s, p, ['league', 'jobId', 'until']);
    const league = text(o.league, `${p}.league`);
    const jobId = text(o.jobId, `${p}.jobId`);
    const until = timestamp(o.until, `${p}.until`);
    if (!league.trim() || league !== league.trim() || !jobId.trim() || jobId !== jobId.trim() || !until)
      throw new Error(`${p}: invalid snooze`);
    return { league, jobId, until };
  }) };
}

export function activeJobSnoozes(state: JobSnoozes | undefined, league: string, now: number): JobSnooze[] {
  return (state?.snoozes ?? []).filter(snooze => snooze.league === league && Date.parse(snooze.until) > now);
}

export const isJobSnoozed = (state: JobSnoozes | undefined, league: string, jobId: string, now: number) =>
  activeJobSnoozes(state, league, now).some(snooze => snooze.jobId === jobId);

export const startingJob = (jobs?: Jobs) => jobs?.jobs.find(job => job.id === 'seed');
// The startup draft's job opens the draft room on the startup draft rather than on a control.
export const startupDraftJob = 'startup_draft';

export function parseJobs(v: unknown): Jobs {
  const r = object(v, 'League jobs', ['league', 'name', 'revision', 'jobs']);
  return {
    league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'League version'),
    jobs: array(r.jobs, 'Jobs', (j, p) => {
      const o = object(j, p,
        ['id', 'name', 'state', 'words', 'opens', 'closes', 'controls', 'identity', 'settles']);
      const identity = o.identity === undefined ? undefined
        : object(o.identity, `${p}.identity`, ['league', 'season', 'week', 'matter']);
      return {
        id: text(o.id, p), name: text(o.name, p), state: choice(o.state, `${p}.state`, jobStates),
        words: text(o.words, p), opens: timestamp(o.opens, `${p}.opens`),
        closes: timestamp(o.closes, `${p}.closes`), controls: array(o.controls, `${p}.controls`, text),
        ...(identity ? { identity: {
          league: text(identity.league, `${p}.identity.league`),
          season: integer(identity.season, `${p}.identity.season`),
          week: integer(identity.week, `${p}.identity.week`),
          matter: text(identity.matter, `${p}.identity.matter`),
        } } : {}),
        ...(o.settles === undefined ? {} : { settles: array(o.settles, `${p}.settles`, text) }),
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
