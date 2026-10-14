import { useEffect, useRef } from 'react';
import { connectClock } from '../clock/ticker';
import type { Deadline } from '../data/contract';
import { Act } from '../commands/Act';
import { countdown, localTime, type Job, type Jobs } from './jobs';

// What a job says about its time, in the viewer's own time zone.
function timing(job: Job) {
  if (job.state === 'live' && job.closes)
    return { text: 'Closes', at: job.closes };
  if (job.state === 'upcoming' && job.opens)
    return { text: 'Opens', at: job.opens };
  return null;
}
const stateWords = { live: 'Live', upcoming: 'Coming up', blocked: 'Blocked', unbuilt: 'Not built yet' };

// JobsStrip lists the league's jobs in the order Go gives: live first, the rest dimmed. Each is one click
// from the first control it opens.
export function JobsStrip({ jobs, clock = Date.now }: { jobs: Jobs; clock?: () => number }) {
  const root = useRef<HTMLElement>(null);
  useEffect(() => {
    const deadlines: Deadline[] = jobs.jobs.map(job => ({
      id: job.id, label: job.name, at: job.closes || job.opens || undefined,
      urgency: 'U0', pinned: false, promoted: false,
    }));
    if (root.current) return connectClock(root.current, deadlines, clock,
      (at, now) => at ? countdown(at, now) : '');
  }, [jobs, clock]);
  const now = clock();
  return (
    <section ref={root} className="jobs-strip" aria-label="Jobs">
      <h4>Jobs</h4>
      <ul className="jobs">
        {jobs.jobs.map(job => {
          const when = timing(job);
          const due = when ? `. ${when.text}: ${localTime(when.at)}` : '';
          const lines = [stateWords[job.state], ...(job.state === 'unbuilt' ? [] : [job.words]),
            ...(job.state === 'live' && !job.closes ? ['No deadline'] : [])];
          return (
            <li key={job.id} className={`job job-${job.state}`} data-state={job.state} data-deadline={job.id}>
              <Act verb="commissioner.surface" variant="card"
                args={{ screen: 'landing', action: 'open', value: job.controls[0] ?? '' }}
                label={`${job.name}. ${lines.join('. ')}${due}`}>
                <strong>{job.name}</strong>
                {lines.map(line => <span key={line} className="job-line">{line}</span>)}
                {when && <span className="job-line">{when.text}{' · '}
                  <span data-countdown>{when.at ? countdown(when.at, now) : 'No deadline'}</span>
                  {when.at && <> · <time dateTime={when.at}>{localTime(when.at)}</time></>}</span>}
              </Act>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
