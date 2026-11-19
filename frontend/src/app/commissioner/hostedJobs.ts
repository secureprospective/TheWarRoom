import type { Job, Jobs } from './jobs';

export const jobLanes = [
  ['now', 'NOW · DUE WITHIN 2 HOURS'],
  ['holding', 'HOLDING THINGS UP'],
  ['today', 'TODAY'],
  ['week', 'THIS WEEK'],
  ['later', 'LATER'],
  ['no-clock', 'NO CLOCK'],
] as const;
export type JobLane = (typeof jobLanes)[number][0];
export type JobKind = 'Ruling' | 'Deadline' | 'Roster' | 'People' | 'System' | 'Other';
export type LaneJobs = Record<JobLane, Job[]>;
export type JobSnoozeChoice = { label: string; until: string };

export function jobSnoozeChoices(job: Job, now: number): JobSnoozeChoice[] {
  const tomorrow = new Date(now);
  tomorrow.setDate(tomorrow.getDate() + 1);
  tomorrow.setHours(9, 0, 0, 0);
  const choices = [
    { label: '3 hours', until: new Date(now + 3 * 60 * 60 * 1000).toISOString() },
    { label: 'Tomorrow at 9 AM (local)', until: tomorrow.toISOString() },
  ];
  const deadline = Date.parse(job.closes);
  if (job.closes && Number.isFinite(deadline) && deadline > now)
    choices.push({ label: 'Until next deadline', until: job.closes });
  return choices;
}

const kindByControl: Record<string, JobKind> = {
  'CM-001': 'Ruling', 'CM-002': 'Ruling', 'CM-003': 'Roster', 'CM-004': 'Roster',
  'CM-005': 'Ruling', 'CM-006': 'Ruling', 'CM-007': 'Ruling', 'CM-008': 'Ruling',
  'CM-009': 'Ruling', 'CM-010': 'Roster', 'CM-011': 'Roster', 'CM-012': 'Deadline',
  'CM-013': 'Deadline', 'CM-014': 'Ruling', 'CM-015': 'Deadline', 'CM-016': 'Ruling',
  'CM-017': 'Ruling', 'CM-018': 'People', 'CM-019': 'People', 'CM-020': 'People',
  'CM-021': 'People', 'CM-022': 'People', 'CM-023': 'Deadline', 'CM-024': 'Deadline',
  'CM-025': 'Deadline', 'CM-026': 'Deadline', 'CM-027': 'Deadline', 'CM-028': 'Roster',
  'CM-029': 'Roster', 'CM-030': 'Roster', 'CM-031': 'Ruling', 'CM-032': 'Ruling',
  'CM-033': 'Ruling', 'CM-034': 'Roster', 'CM-035': 'Deadline', 'CM-036': 'Deadline',
  'CM-037': 'Ruling', 'CM-038': 'Roster', 'CM-039': 'Ruling', 'CM-040': 'Deadline',
  'CM-041': 'Deadline', 'CM-042': 'Deadline', 'CM-043': 'Deadline', 'CM-044': 'Ruling',
  'CM-045': 'Deadline', 'CM-046': 'Ruling', 'CM-047': 'Ruling', 'CM-048': 'Ruling',
  'CM-051': 'System', 'CM-054': 'System',
};

export function jobKind(job: Job): JobKind {
  for (const control of job.controls) {
    const kind = kindByControl[control];
    if (kind) return kind;
  }
  return 'Other';
}

function dueAt(job: Job): number | undefined {
  const value = job.closes || job.opens;
  if (!value) return undefined;
  const instant = Date.parse(value);
  return Number.isFinite(instant) ? instant : undefined;
}

function laneFor(job: Job, now: number): JobLane {
  const due = dueAt(job);
  if (due !== undefined && due - now <= 2 * 60 * 60 * 1000) return 'now';
  if (job.state === 'blocked') return 'holding';
  if (due === undefined) return 'no-clock';
  const hours = (due - now) / (60 * 60 * 1000);
  if (hours <= 24) return 'today';
  if (hours <= 168) return 'week';
  return 'later';
}

function compareJobs(left: Job, right: Job, now: number): number {
  const leftDue = dueAt(left);
  const rightDue = dueAt(right);
  const leftDueNow = leftDue !== undefined && leftDue <= now;
  const rightDueNow = rightDue !== undefined && rightDue <= now;
  if (leftDueNow !== rightDueNow) return leftDueNow ? -1 : 1;
  const leftBlocks = left.state === 'blocked';
  const rightBlocks = right.state === 'blocked';
  if (leftBlocks !== rightBlocks) return leftBlocks ? -1 : 1;
  return (leftDue ?? Infinity) - (rightDue ?? Infinity);
}

/** Groups the complete server reading; no team/job count cap or client-side job production. */
export function orderJobLanes(reading: Jobs, now: number): LaneJobs {
  const lanes: LaneJobs = {
    now: [], holding: [], today: [], week: [], later: [], 'no-clock': [],
  };
  for (const job of reading.jobs) lanes[laneFor(job, now)].push(job);
  for (const [lane] of jobLanes) lanes[lane].sort((a, b) => compareJobs(a, b, now));
  return lanes;
}

export function filterJobLanes(lanes: LaneJobs, kind: JobKind | 'All kinds'): LaneJobs {
  return Object.fromEntries(jobLanes.map(([lane]) => [lane,
    lanes[lane].filter(job => kind === 'All kinds' || jobKind(job) === kind),
  ])) as LaneJobs;
}

export const jobKinds: readonly JobKind[] = ['Ruling', 'Deadline', 'Roster', 'People', 'System', 'Other'];
