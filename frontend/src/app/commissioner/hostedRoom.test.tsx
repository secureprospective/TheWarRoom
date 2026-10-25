import { expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import HostedLeague from './HostedLeague';
import { previewChange } from './preview';
import { createHostedRoom, type HostedPort } from './hostedRoom';
import type { HostedResponse } from './api';
import type { NativeRequest } from '../commands/native';
import wire from './hosted.wire.testdata.json';
import preview from './preview.wire.testdata.json';
import rollover from './rollover.wire.testdata.json';

const readings = wire as Record<string, HostedResponse>;
function fixture() {
  let offline = false;
  const port = vi.fn<HostedPort>(async (request) => {
    if (offline) throw new Error('disconnected');
    if (request.mode === 'preview') return { ...readings['lineup.read'], data: preview.confirm };
    if (request.mode === 'execute')
      return {
        ...readings['lineup.read'],
        data: {
          entry: 'act.saved',
          league: 'native.hosted',
          revision: 10,
          recordedAt: '2026-10-25T10:00:00Z',
          message: 'Recorded',
        },
      };
    return readings[request.command];
  });
  return {
    room: createHostedRoom('native.hosted', 'franchise.01', port),
    port,
    disconnect: () => {
      offline = true;
    },
    reconnect: () => {
      offline = false;
    },
  };
}
const change = (): NativeRequest => ({
  command: 'commish.lineup.set',
  mode: 'preview',
  arguments: {
    league: 'native.hosted',
    command: 'commish.lineup.set',
    args: { lineup: { season: 2026, franchise: 'franchise.01', week: 1, starters: ['0099'] } },
    reason: '',
    expected: 1,
    request: 'once',
    effective: 'now',
    visibility: 'private',
  },
});

it('parses the real hosted Go readings before permitting a change', async () => {
  const f = fixture();
  await f.room.refresh();
  expect(f.room.peek().error).toBe('');
  expect(f.room.canChange()).toBe(true);
  expect(f.room.peek().reading?.roster.players[1].name).toBe('Injured prospect');
  expect(f.room.peek().reading?.draft.available[0].name).toBe('Practice rookie');
  expect(f.room.peek().reading?.label).toBe('This league is managed here');
  await f.room.review(change(), 'Set lineup');
  expect(f.port.mock.calls.filter(([r]) => r.mode === 'execute')).toHaveLength(0);
  await f.room.confirm();
  expect(f.port.mock.calls.filter(([r]) => r.mode === 'execute')).toHaveLength(1);
  expect(f.port.mock.calls.find(([r]) => r.mode === 'execute')?.[1]).toBe(
    readings['lineup.read'].football,
  );
});

it('keeps disconnected views read-only without queueing changes', async () => {
  const f = fixture();
  await f.room.refresh();
  const saved = f.room.peek().reading;
  f.disconnect();
  await f.room.refresh();
  expect(f.room.peek().reading).toBe(saved);
  expect(f.room.peek().stale).toBe(true);
  const calls = f.port.mock.calls.length;
  await f.room.review(change(), 'Set lineup');
  await f.room.confirm();
  expect(f.port).toHaveBeenCalledTimes(calls);
  f.reconnect();
  await f.room.refresh();
  expect(f.room.canChange()).toBe(true);
  expect(f.port.mock.calls.every(([r]) => r.mode === 'read')).toBe(true);
});

it('requires a fresh read after ambiguous delivery, without an automatic retry', async () => {
  const f = fixture();
  await f.room.refresh();
  await f.room.review(change(), 'Set lineup');
  f.disconnect();
  await f.room.confirm();
  expect(f.room.peek().error).toContain('may have reached the server');
  expect(f.room.canChange()).toBe(false);
  f.reconnect();
  await f.room.refresh();
  expect(f.port.mock.calls.filter(([r]) => r.mode === 'execute')).toHaveLength(1);
  await f.room.confirm();
  expect(f.port.mock.calls.filter(([r]) => r.mode === 'execute')).toHaveLength(1);
});

it('rejects mixed evidence and fenced authority', async () => {
  const port = vi.fn<HostedPort>(async (r) => ({ ...readings[r.command], writable: false }));
  const room = createHostedRoom('native.hosted', 'franchise.01', port);
  await room.refresh();
  expect(room.canChange()).toBe(false);
  const mixed = createHostedRoom('native.hosted', 'franchise.01', async (r) => ({
    ...readings[r.command],
    football: r.command === 'draft.read' ? 'different' : readings[r.command].football,
  }));
  await mixed.refresh();
  expect(mixed.peek().stale).toBe(true);
  expect(mixed.peek().reading).toBeUndefined();
});

it('starts read-only with discoverable tasks and no raw player identifiers', () => {
  const html = renderToStaticMarkup(
    <HostedLeague
      league="native.hosted"
      franchise="franchise.01"
      name="Practice team 1"
      role="Commissioner"
    />,
  );
  expect(html).toContain('Set lineup');
  expect(html).toContain('Roster and reserves');
  expect(html).toContain('Draft room');
  expect(html).toContain('No changes can be made');
  expect(html).not.toContain('franchise.01');
});

it('does not restore private readings after the account screen has been closed', async () => {
  let release!: () => void;
  const wait = new Promise<void>((resolve) => {
    release = resolve;
  });
  const room = createHostedRoom('native.hosted', 'franchise.01', async (r) => {
    await wait;
    return readings[r.command];
  });
  const pending = room.refresh();
  room.dispose();
  release();
  await pending;
  expect(room.peek().reading).toBeUndefined();
  expect(room.canChange()).toBe(false);
});

it.each(['bid.place', 'waivers.claim', 'trade.offer', 'commish.score.franchise', 'season.rollover'] as const)(
  'checks and confirms %s once, with no replay after refresh', async command => {
    const f = fixture();
    await f.room.refresh();
    const request = { ...change(), command, arguments: { ...change().arguments, command } } as NativeRequest;
    await f.room.review(request, 'Review hosted workflow');
    expect(f.port.mock.calls.filter(([r]) => r.mode === 'execute')).toHaveLength(0);
    await Promise.all([f.room.confirm(), f.room.confirm()]);
    const calls = f.port.mock.calls.filter(([r]) => r.mode === 'execute');
    expect(calls).toHaveLength(1);
    expect(calls[0][0].arguments).toEqual(request.arguments);
    await f.room.refresh();
    await f.room.confirm();
    expect(f.port.mock.calls.filter(([r]) => r.mode === 'execute')).toHaveLength(1);
  },
);

it('confirms a real rollover receipt and refreshes the new season for the commissioner', async () => {
  const after = rollover.readings as Record<string, HostedResponse>;
  let rolled = false;
  const room = createHostedRoom('native.hosted', 'franchise.01', async request => {
    if (request.mode === 'preview') return { ...readings['lineup.read'], data: preview.confirm };
    if (request.mode === 'execute') {
      rolled = true;
      return rollover.receipt as HostedResponse;
    }
    if (rolled && request.command === 'lineup.read') throw new Error('New season schedule is not selected');
    return (rolled ? after : readings)[request.command];
  });
  await room.refresh();
  const request = { ...change(), command: 'season.rollover', arguments: { ...change().arguments, command: 'season.rollover' } } as NativeRequest;
  await room.review(request, 'Move to the next season');
  await room.confirm();
  expect(room.peek().error).toBe('');
  expect(room.peek().stale).toBe(false);
  expect(room.peek().reading?.roster.season).toBe(2027);
  expect(room.peek().reading?.lineup.available).toBe(false);
});

it('keeps other workflows readable when new-season lineup evidence is unavailable', async () => {
  const room = createHostedRoom('native.hosted', 'franchise.01', async request => {
    if (request.command === 'lineup.read') throw new Error('New season schedule is not selected');
    return readings[request.command];
  });
  await room.refresh();
  expect(room.peek().stale).toBe(false);
  expect(room.peek().reading?.lineup.available).toBe(false);
  expect(room.peek().reading?.lineup.words).toContain('football evidence');
  expect(room.peek().reading?.markets.standings).toBeDefined();
});

it('says each previewed change once, in plain words, without a stray timing word', () => {
  const now = (before: string, after: string, when = 'now') => previewChange({ before, after, when });
  expect(now('No lineup recorded', 'Starting players: Practice quarterback')).toBe(
    'No lineup recorded → Starting players: Practice quarterback',
  );
  const ledger = 'Set the week 1 lineup of Practice team 1 to 1 starter';
  expect(now('This change is not recorded', ledger, 'in force now')).toBe(
    `This change is not recorded → ${ledger}`,
  );
  const unset = 'Practice team 1, week 1: No lineup was set.';
  const set = "Practice team 1's week 1 lineup is set. Valid lineup.";
  expect(now(unset, set)).toBe(`${unset} → ${set}`);
  expect(now('Practice team 1, Active roster', 'Practice team 1, Injured reserve')).toBe(
    'Practice team 1, Active roster → Practice team 1, Injured reserve',
  );
  expect(now('20 to 70', '18 to 80', 'from the next rollover')).toBe(
    '20 to 70 → 18 to 80 (from the next rollover)',
  );
});

it('clears the recorded-change notice when the follow-up refresh fails', async () => {
  const f = fixture();
  await f.room.refresh();
  await f.room.review(change(), 'Set lineup');
  await f.room.confirm();
  expect(f.room.peek().message).toBe('The server recorded your change. Refreshing the league.');
  f.disconnect();
  await f.room.refresh();
  expect(f.room.peek().error).toContain('Could not refresh your league');
  expect(f.room.peek().message).toBe('');
});
