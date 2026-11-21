import { useState } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct } from '../shell/HQAct';
import type { HostedPort } from './hostedRoom';
import { activeJobSnoozes, countdown, type Job, type JobSnooze, type JobSnoozes, type Jobs } from './jobs';
import { filterJobLanes, jobKinds, jobKind, jobLanes, jobSnoozeChoices,
  orderJobLanes, type JobKind } from './hostedJobs';
import { firstHostedJobTool } from './hostedControlRoomData';
import { when } from '../control/room/room';
import type { ActDefinition } from './actTable';

export function HostedControlRoomJobs({ league, jobs, snoozes, now = Date.now(), snoozeError = '',
  snoozeErrorCode, port, pendingJob = '', onSnooze, onWake, onOpen, onUnavailable }: {
  league: string; jobs?: Jobs; snoozes?: JobSnoozes; now?: number; snoozeError?: string;
  snoozeErrorCode?: string; port?: HostedPort; pendingJob?: string;
  onSnooze(job: Job, until: string): void; onWake(snooze: JobSnooze): void;
  onOpen(definition: ActDefinition, job: Job): void; onUnavailable(words: string): void;
}) {
  const [kind, setKind] = useState<JobKind | 'All kinds'>('All kinds');
  const active = activeJobSnoozes(snoozes, league, now);
  const snoozedIDs = new Set(active.map(snooze => snooze.jobId));
  const openJobs = jobs ? { ...jobs, jobs: jobs.jobs.filter(job => !snoozedIDs.has(job.id)) } : undefined;
  const ordered = openJobs ? filterJobLanes(orderJobLanes(openJobs, now), kind) : undefined;
  const dueWords = (job: Job) => {
    const due = job.closes || job.opens;
    if (!due) return '—';
    return job.closes ? countdown(due, now) : `Opens ${when(due)}`;
  };
  const jobByID = new Map((jobs?.jobs ?? []).map(job => [job.id, job]));
  return <div className="cr-col hosted-jobs" data-testid="hosted-jobs">
    <div className="cr-panel" style={{ flex: 1 }}>
      <div className="cr-ph">
        JOBS <span className="sp" />
        <span className="ct">{jobs ? `${jobs.jobs.length} from the league job read`
          : 'Reading league jobs…'}</span>
      </div>
      <div className="cr-pb hosted-job-controls">
        <label>Filter by kind
          <select aria-label="Filter jobs by kind" value={kind} onChange={event =>
            setKind(event.target.value as JobKind | 'All kinds')}>
            <option>All kinds</option>
            {jobKinds.map(value => <option key={value}>{value}</option>)}
          </select>
        </label>
      </div>
      {snoozeError && <FailureAlert
        message={snoozeError}
        code={snoozeErrorCode}
        className="cr-note"
        league={league}
        screen="Control Room · Jobs"
        route="control/jobs"
        port={port}
      />}
      <div className="cr-pb hosted-job-lanes" aria-live="polite">
        {!jobs && <p role="status">The league jobs have not been read yet.</p>}
        <section data-lane="snoozed" aria-label="Snoozed jobs">
          <h3>SNOOZED · {snoozes ? active.length : '—'}</h3>
          {!snoozes && !snoozeError && <p role="status">Reading your private snoozes…</p>}
          {snoozes && active.length === 0 &&
            <p className="hosted-job-empty">No snoozed jobs.</p>}
          {active.map(snooze => {
            const job = jobByID.get(snooze.jobId);
            return <div className="hosted-job-snoozed-row" key={`${snooze.league}:${snooze.jobId}`}>
              <span className="hosted-job-snoozed-name"><b>{job?.name ?? snooze.jobId}</b>
                {job?.words && <span>{job.words}</span>}</span>
              <time dateTime={snooze.until}>Until {new Date(snooze.until).toLocaleString()}</time>
              <HQAct disabled={pendingJob !== '' || !snoozes}
                label={`Wake ${job?.name ?? snooze.jobId}`} onAction={() => onWake(snooze)}>Wake</HQAct>
            </div>;
          })}
        </section>
        {jobs && ordered && jobLanes.map(([lane, title]) => {
          const rows = ordered[lane];
          return <section key={lane} data-lane={lane} aria-label={title}>
            <h3 className={lane === 'now' ? 'hosted-job-urgent' : undefined}>{title} · {rows.length}</h3>
            {rows.length === 0 ? <p className="hosted-job-empty">No jobs in this lane.</p> : rows.map(job => {
              const jobKindName = jobKind(job);
              return <div className="hosted-job-row" key={job.id}>
                <HQAct className="cr-job" label={job.name}
                  attributes={{ 'data-job': job.id, 'data-kind': jobKindName }}
                  onAction={() => {
                    const definition = firstHostedJobTool(job);
                    if (definition) onOpen(definition, job);
                    else onUnavailable(`No server action is registered for ${job.name} yet.`);
                  }}>
                  <i className={`dot ${lane === 'now' ? 'st-now' : 'st-neutral'}`} />
                  <span className="nm"><b>{job.name}</b><span>{job.words}</span></span>
                  <span className="cnt">{jobKindName}</span>
                  <span className="when">{dueWords(job)}</span>
                </HQAct>
                <details className="hosted-job-snooze">
                  <summary aria-label={`Snooze ${job.name}`}>Snooze</summary>
                  <div className="hosted-job-snooze-options">
                    {jobSnoozeChoices(job, now).map(choice => <HQAct key={choice.label}
                      disabled={pendingJob !== '' || !snoozes}
                      label={`Snooze ${job.name} for ${choice.label}`}
                      onAction={() => onSnooze(job, choice.until)}>{choice.label}</HQAct>)}
                  </div>
                </details>
              </div>;
            })}
          </section>;
        })}
      </div>
    </div>
  </div>;
}
