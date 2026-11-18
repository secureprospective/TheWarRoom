import { useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { countdown, type Job, type Jobs } from './jobs';
import { filterJobLanes, jobKinds, jobKind, jobLanes, orderJobLanes, type JobKind } from './hostedJobs';
import { firstHostedJobTool } from './hostedControlRoom';
import { when } from '../control/room/room';
import type { ActDefinition } from './actTable';

export function HostedControlRoomJobs({ jobs, now = Date.now(), onOpen, onUnavailable }: {
  jobs?: Jobs; now?: number; onOpen(definition: ActDefinition, job: Job): void;
  onUnavailable(words: string): void;
}) {
  const [kind, setKind] = useState<JobKind | 'All kinds'>('All kinds');
  const ordered = jobs ? filterJobLanes(orderJobLanes(jobs, now), kind) : undefined;
  const dueWords = (job: Job) => {
    const due = job.closes || job.opens;
    if (!due) return '—';
    return job.closes ? countdown(due, now) : `Opens ${when(due)}`;
  };
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
      <div className="cr-pb hosted-job-lanes" aria-live="polite">
        {!jobs && <p role="status">The league jobs have not been read yet.</p>}
        {jobs && ordered && jobLanes.map(([lane, title]) => {
          const rows = ordered[lane];
          return <section key={lane} data-lane={lane} aria-label={title}>
            <h3 className={lane === 'now' ? 'hosted-job-urgent' : undefined}>{title} · {rows.length}</h3>
            {rows.length === 0 ? <p className="hosted-job-empty">No jobs in this lane.</p> : rows.map(job => {
              const jobKindName = jobKind(job);
              return <HQAct key={job.id} className="cr-job" label={job.name}
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
              </HQAct>;
            })}
          </section>;
        })}
      </div>
    </div>
  </div>;
}
