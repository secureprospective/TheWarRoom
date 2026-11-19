// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { HostedControlRoomJobs } from './HostedControlRoomJobs';
import type { Job, Jobs, JobSnoozes } from './jobs';

const now = Date.parse('2026-11-18T12:00:00Z');
const job: Job = { id: 'review', name: 'Review the waiver ruling', state: 'upcoming',
  words: 'Open the commissioner tool.', opens: '', closes: '2026-11-21T12:00:00Z', controls: ['CM-027'] };
const reading: Jobs = { league: 'native.test', name: 'Test league', revision: 1, jobs: [job] };
const snooze = { league: 'native.test', jobId: job.id, until: '2026-11-18T15:00:00Z' };
let host: HTMLDivElement;
let root: Root;
const onSnooze = vi.fn();
const onWake = vi.fn();

beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
  onSnooze.mockReset();
  onWake.mockReset();
});

afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

function render(snoozes: JobSnoozes, jobs: Jobs | null = reading) {
  return act(async () => root.render(<HostedControlRoomJobs league={reading.league} jobs={jobs ?? undefined}
    snoozes={snoozes} now={now} onSnooze={onSnooze} onWake={onWake} onOpen={vi.fn()}
    onUnavailable={vi.fn()} />));
}

it('shows private snoozes apart from the open job lanes and wakes one', async () => {
  await render({ snoozes: [snooze] });
  expect(host.querySelector('[data-lane="snoozed"]')?.textContent).toContain('Review the waiver ruling');
  expect(host.querySelector('.hosted-job-row')).toBeNull();
  const wake = host.querySelector<HTMLButtonElement>('[aria-label="Wake Review the waiver ruling"]');
  expect(wake).not.toBeNull();
  await act(async () => wake!.click());
  expect(onWake).toHaveBeenCalledWith(snooze);

  await render({ snoozes: [] });
  expect(host.querySelector('.hosted-job-row')?.textContent).toContain('Review the waiver ruling');
});

it('keeps the account snoozed list readable when the league job read is unavailable', async () => {
  await render({ snoozes: [snooze] }, null);
  expect(host.textContent).toContain('The league jobs have not been read yet.');
  expect(host.querySelector('[data-lane="snoozed"]')?.textContent).toContain(snooze.jobId);
});

it('offers the three-hour, tomorrow-morning and deadline choices for an open job', async () => {
  await render({ snoozes: [] });
  const details = host.querySelector('details.hosted-job-snooze');
  expect(details).not.toBeNull();
  await act(async () => details!.querySelector('summary')!.click());
  const choices = [...host.querySelectorAll<HTMLButtonElement>('.hosted-job-snooze-options button')];
  expect(choices.map(button => button.textContent)).toEqual([
    '3 hours', 'Tomorrow at 9 AM (local)', 'Until next deadline',
  ]);
  await act(async () => choices[0].click());
  expect(onSnooze).toHaveBeenCalledWith(job, '2026-11-18T15:00:00.000Z');
});
