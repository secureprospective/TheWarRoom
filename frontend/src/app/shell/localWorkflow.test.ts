// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { localWorkflow } from './localWorkflow';

type Call = { method: string; arguments: unknown[] };
function desktop(calls: Call[]) {
  const record = (method: string, data: unknown) => (...args: unknown[]) => {
    calls.push({ method, arguments: args });
    return Promise.resolve(data);
  };
  const authority = { label: 'This computer', writable: true, football: 'local league',
    data: { league: 'native.seed', revision: 2 } };
  Object.defineProperty(window, 'go', { configurable: true, value: { commissioner: { API: {
    TargetWorkflow: record('TargetWorkflow', authority),
    TargetLeagueCalendar: record('TargetLeagueCalendar', { league: 'native.seed', revision: 2,
      decision: {}, clock: { windows: [], exceptions: [], tradeDeadline: null,
        secondsToTradeDeadline: null } }),
  } } } });
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
