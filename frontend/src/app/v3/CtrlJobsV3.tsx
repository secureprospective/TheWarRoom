// Desktop v3 Control Room, Jobs tab, after TheWarRoom-mock-v3-FINAL.html (screens.js control.jobs): the
// commissioner's lanes with kind filter, snooze/wake and open-tool, a season clock and league health.
// Everything comes from the hosted room's own reads and actions (useHostedControlRoom via `c`).
import { useState, type ReactNode } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct } from '../shell/HQAct';
import { HostedCommissionerActTray } from '../commissioner/CommissionerActTray';
import { firstHostedJobTool } from '../commissioner/hostedControlRoomData';
import { filterJobLanes, jobKind, jobKinds, jobLanes, jobSnoozeChoices, orderJobLanes,
  type JobKind } from '../commissioner/hostedJobs';
import { activeJobSnoozes, countdown, type Job, type Jobs } from '../commissioner/jobs';
import { when } from '../control/room/room';
import type { ControlBodyProps } from './ControlRoomV3';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** The jobs that have a deadline, soonest first (passed deadlines lead). */
export function seasonClock(jobs: Jobs | undefined): Job[] {
  return (jobs?.jobs ?? []).filter(job => Number.isFinite(Date.parse(job.closes)))
    .sort((a, b) => Date.parse(a.closes) - Date.parse(b.closes));
}

/** Jobs counted by kind, in the filter's order; kinds with no job are left out. */
export function kindCounts(jobs: Jobs | undefined): { kind: JobKind; count: number }[] {
  const all = jobs?.jobs ?? [];
  return jobKinds.map(kind => ({ kind, count: all.filter(job => jobKind(job) === kind).length }))
    .filter(row => row.count > 0);
}

export function CtrlJobsV3({ c, league, port, badge }: ControlBodyProps) {
  const [kind, setKind] = useState<JobKind | 'All kinds'>('All kinds');
  const jobs = c.jobsState.jobs;
  const now = c.jobNow;
  const snoozes = c.jobSnoozes;
  const active = activeJobSnoozes(snoozes, league, now);
  const snoozedIDs = new Set(active.map(s => s.jobId));
  const jobByID = new Map((jobs?.jobs ?? []).map(job => [job.id, job]));
  const ordered = jobs ? filterJobLanes(orderJobLanes(
    { ...jobs, jobs: jobs.jobs.filter(job => !snoozedIDs.has(job.id)) }, now), kind) : undefined;
  const busy = c.snoozePending !== '' || !snoozes;
  const dueWords = (job: Job) => job.closes ? countdown(job.closes, now)
    : job.opens ? `Opens ${when(job.opens)}` : '—';
  const clock = seasonClock(jobs);
  const counts = kindCounts(jobs);
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.6fr) minmax(0,1fr)',
    gridTemplateRows: 'minmax(0,1fr) minmax(0,1fr)' }}>
    <Panel title="Jobs" badge={badge} style={{ gridRow: '1/3' }}
      tag={jobs ? <span className="tag o">{jobs.jobs.length}</span> : null}
      foot={jobs ? `${jobs.jobs.length} from the league job read` : 'Reading league jobs…'}>
      <div className="fbar">{(['All kinds', ...jobKinds] as const).map(k => <HQAct key={k}
        className="chip" selected={kind === k} onAction={() => setKind(k)}>{k}</HQAct>)}</div>
      {c.jobsState.error && <St><FailureAlert message={c.jobsState.error}
        code={c.jobsState.errorCode} league={league} screen="Control Room · Jobs"
        route="control/jobs" port={port} /></St>}
      {c.snoozeError && <St><FailureAlert message={c.snoozeError}
        code={c.snoozeErrorCode} league={league} screen="Control Room · Jobs"
        route="control/jobs" port={port} /></St>}
      {c.routeWords && <St><span role="status">{c.routeWords}</span></St>}
      {!jobs && !c.jobsState.error && <St>The league jobs have not been read yet.</St>}
      <div className="sec" data-lane="snoozed">SNOOZED · {snoozes ? active.length : '—'}</div>
      {!snoozes && !c.snoozeError && <St>Reading your private snoozes…</St>}
      {snoozes && active.length === 0 && <St>No snoozed jobs.</St>}
      {active.map(s => {
        const job = jobByID.get(s.jobId);
        return <div key={`${s.league}:${s.jobId}`} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
          <span className="ell">{job?.name ?? s.jobId}</span>
          <span className="m">Until {new Date(s.until).toLocaleString()}</span>
          <span className="acts"><HQAct className="btn sm" disabled={busy}
            label={`Wake ${job?.name ?? s.jobId}`} onAction={() => c.wakeJob(s)}>Wake</HQAct></span></div>;
      })}
      {ordered && jobLanes.map(([lane, title]) => <div key={lane} data-lane={lane}>
        <div className="sec">{title} · {ordered[lane].length}</div>
        {ordered[lane].length === 0 && <St>No jobs in this lane.</St>}
        {ordered[lane].map(job => <div key={job.id} className="row" data-job={job.id}
          style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto auto auto' }}>
          <span className="tag">{jobKind(job)}</span>
          <span className="ell"><b>{job.name}</b> <span className="d">{job.words}</span></span>
          <span className="m">{dueWords(job)}</span>
          <details className="hosted-job-snooze">
            <summary aria-label={`Snooze ${job.name}`}>Snooze</summary>
            {jobSnoozeChoices(job, now).map(ch => <HQAct key={ch.label} className="btn sm" disabled={busy}
              label={`Snooze ${job.name} for ${ch.label}`}
              onAction={() => c.snoozeJob(job, ch.until)}>{ch.label}</HQAct>)}
          </details>
          <span className="acts"><HQAct className="btn sm" label={`Open tool for ${job.name}`}
            onAction={() => {
              const definition = firstHostedJobTool(job);
              if (definition) c.openTool(definition, job);
              else c.showRoute(`No server action is registered for ${job.name} yet.`);
            }}>Open tool ▸</HQAct></span></div>)}
      </div>)}
      {c.definition && c.openedTab === 'jobs' && <HostedCommissionerActTray definition={c.definition}
        league={league} room={c.room} port={port} initial={c.initialValues} />}
      {c.job && <p className="m">Opened from: {c.job.name}</p>}
    </Panel>
    <Panel title="Season clock" badge={badge}
      foot="jobs with a deadline, soonest first">
      {!jobs ? <St>Reading league jobs…</St> : !clock.length ? <St>No job has a deadline.</St>
        : clock.map(job => <div key={job.id} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
          <span className="ell">{job.name}</span>
          <span className="m">{countdown(job.closes, now)}</span></div>)}
    </Panel>
    <Panel title="League health" badge={badge} foot="counted from the same league job read">
      {!jobs ? <St>Reading league jobs…</St> : !counts.length ? <St>No jobs to count.</St>
        : counts.map(r => <div key={r.kind} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
          <span className="k">{r.kind.toUpperCase()}</span><span className="v">{r.count}</span></div>)}
    </Panel>
  </div></div>;
}
