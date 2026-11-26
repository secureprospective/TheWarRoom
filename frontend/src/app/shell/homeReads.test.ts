import { describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import { readHomeBrief, readLiveBrief, readThinHomeBrief } from './homeReads';

const rejectingPort = () => vi.fn<HostedPort>(async () => { throw new Error('not configured'); });
const count = (port: ReturnType<typeof rejectingPort>, command: string) =>
  port.mock.calls.filter(([request]) => request.command === command).length;

describe('Home commissioner-only jobs read', () => {
  it('does not ask an owner session for commissioner jobs', async () => {
    const port = rejectingPort();
    await readHomeBrief('league', 'franchise.1', port, undefined, null, true, false);
    expect(count(port, 'commish.jobs.read')).toBe(0);
  });

  it('reads jobs for a commissioner with a franchise and the unassigned commissioner view', async () => {
    const assigned = rejectingPort();
    await readHomeBrief('league', 'franchise.1', assigned, undefined, null, true, true, true);
    expect(count(assigned, 'commish.jobs.read')).toBe(1);
    expect(count(assigned, 'job.snoozes.read')).toBe(1);

    const unassigned = rejectingPort();
    await readThinHomeBrief('league', '', unassigned, undefined, true);
    expect(count(unassigned, 'commish.jobs.read')).toBe(1);
    expect(count(unassigned, 'job.snoozes.read')).toBe(1);

    const visitor = rejectingPort();
    await readThinHomeBrief('league', '', visitor, undefined, false);
    expect(count(visitor, 'commish.jobs.read')).toBe(0);
    expect(count(visitor, 'job.snoozes.read')).toBe(0);
  });

  it('keeps local home layout mode off the hosted account preference route', async () => {
    const local = rejectingPort();
    await readHomeBrief('league', 'franchise.1', local, undefined, null, true, true);
    expect(count(local, 'job.snoozes.read')).toBe(0);
  });
});

it('keeps Thin Home reads when an owner opens Home', async () => {
  const port = rejectingPort();
  await readThinHomeBrief('league', 'franchise.1', port);
  for (const command of [
    'native.assets.read', 'trades.read', 'native.cap.check', 'commish.ledger.filtered.read',
    'bids.read', 'draft.read',
  ]) expect(count(port, command), command).toBe(1);
});

it('preserves the teamless brief and errors', async () => {
  const live = rejectingPort();
  const home = rejectingPort();
  expect(await readLiveBrief('league', '', live)).toEqual(await readThinHomeBrief('league', '', home));
  expect(live.mock.calls).toEqual(home.mock.calls);
});
