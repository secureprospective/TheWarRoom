import { expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import HostedLeague from './HostedLeague';
import { createHostedRoom, type HostedPort } from './hostedRoom';
import type { HostedResponse } from './api';
import type { NativeRequest } from '../commands/native';
import wire from './hosted.wire.testdata.json';
import preview from './preview.wire.testdata.json';

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
