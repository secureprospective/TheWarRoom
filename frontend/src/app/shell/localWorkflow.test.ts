// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { localWorkflow } from './localWorkflow';

type Call = { method: string; arguments: unknown[] };
// desktop installs a fake commissioner binding and returns it, so a test can replace one method.
function desktop(calls: Call[]): Record<string, unknown> {
  const record = (method: string, data: unknown) => (...args: unknown[]) => {
    calls.push({ method, arguments: args });
    return Promise.resolve(data);
  };
  const authority = { label: 'This computer', writable: true, football: 'local league',
    data: { league: 'native.seed', revision: 2 } };
  const api: Record<string, unknown> = {
    TargetWorkflow: record('TargetWorkflow', authority),
    TargetLeagueCalendar: record('TargetLeagueCalendar', { league: 'native.seed', revision: 2,
      decision: {}, clock: { windows: [], exceptions: [], tradeDeadline: null,
        secondsToTradeDeadline: null } }),
  };
  Object.defineProperty(window, 'go', { configurable: true, value: { commissioner: { API: api } } });
  return api;
}
afterEach(() => { delete (window as unknown as { go?: unknown }).go; });

// The workflow envelope carries one general read scope: a league, a franchise, a week, a season and
// a time. A calendar needs the viewer's zone and the action it is asked about, which that scope
// cannot hold, so sending it through the envelope is refused before it reaches the ledger and the
// War Room header has no window to show. The calendar goes to its own binding, as the week and the
// cap check already do.
it('reads an app league calendar through its own binding, not the workflow envelope', async () => {
  const calls: Call[] = [];
  desktop(calls);
  const response = await localWorkflow({ command: 'calendar.read', mode: 'read', arguments: {
    league: 'native.seed', season: 2026, franchise: 'franchise.01', action: 'trade',
    localZone: 'America/Chicago' } });
  expect(calls.map(c => c.method)).toEqual(['TargetWorkflow', 'TargetLeagueCalendar']);
  expect(JSON.parse(calls[0].arguments[0] as string).command).toBe('native.assets.read');
  expect(calls[1].arguments[0]).toMatchObject({ league: 'native.seed', season: 2026,
    franchise: 'franchise.01', action: 'trade', localZone: 'America/Chicago' });
  expect((response.data as { league: string }).league).toBe('native.seed');
});

it('still sends a plain league reading through the workflow envelope', async () => {
  const calls: Call[] = [];
  desktop(calls);
  await localWorkflow({ command: 'native.assets.read', mode: 'read',
    arguments: { league: 'native.seed' } });
  expect(calls.map(c => c.method)).toEqual(['TargetWorkflow']);
});

it('refuses a league reading outside the desktop app', async () => {
  await expect(localWorkflow({ command: 'calendar.read', mode: 'read', arguments: {
    league: 'native.seed', season: 2026, franchise: 'franchise.01', action: 'trade',
    localZone: 'America/Chicago' } })).rejects.toThrow('desktop app');
});

// TargetWorkflow refuses commands outside its bounded set with "this league action is not available
// here"; Home's TRANSACTIONS and ALERTS read the ledger and the jobs board, so each goes to its own binding.
it.each([
  ['commish.ledger.read', 'TargetCommissionerLedger'],
  ['commish.jobs.read', 'TargetCommissionerJobs'],
] as const)(
  'reads %s through its own binding, not the workflow envelope', async (command, method) => {
    const calls: Call[] = [];
    const api = desktop(calls);
    api[method] = (...args: unknown[]) => { calls.push({ method, arguments: args });
      return Promise.resolve({ league: 'native.seed' }); };
    const response = await localWorkflow({ command, mode: 'read', arguments: { league: 'native.seed' } });
    expect(calls.map(c => c.method)).toEqual(['TargetWorkflow', method]);
    expect(JSON.parse(calls[0].arguments[0] as string).command).toBe('native.assets.read');
    expect(calls[1].arguments).toEqual(['native.seed']);
    expect((response.data as { league: string }).league).toBe('native.seed');
  });

it('coalesces repeated reads with only one trailing refresh', async () => {
  const api = desktop([]);
  let release!: () => void;
  const pending = new Promise<void>(resolve => { release = resolve; });
  const call = vi.fn(async () => { await pending; return { label: 'Local', writable: false,
    football: '', data: { league: 'native.seed' } }; });
  api.TargetWorkflow = call;
  const reads = Array.from({ length: 20 }, () => localWorkflow({ command: 'lineup.read', mode: 'read',
    arguments: { league: 'native.seed', franchise: 'franchise.01', week: 8, at: '' } }));
  expect(call).toHaveBeenCalledTimes(1);
  release();
  await Promise.all(reads);
  expect(call).toHaveBeenCalledTimes(2);
});
