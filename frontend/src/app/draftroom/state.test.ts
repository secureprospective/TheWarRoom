import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { createDraftRoom } from './state';
import wire from './draft.wire.testdata.json';

const receipt = { entry: 'act.1', league: 'native.a', revision: 9, recordedAt: '2027-07-14T18:00:00Z',
  message: 'Your pick is recorded.' };
const reading = (league: string) => ({ ...wire.running, league });

describe('The draft room’s reading', () => {
  it('never shows one league’s draft while another league’s is being read', async () => {
    let release: (value: unknown) => void = () => undefined;
    const hold = () => new Promise(resolve => { release = resolve; });
    const port = { pick: async () => receipt,
      read: async (scope: { league: string }) =>
        (scope.league === 'native.b' ? hold() : reading(scope.league)) };
    const room = createDraftRoom(port);
    await room.read('native.a', 'franchise.01');
    expect(room.peek().draft?.league).toBe('native.a');
    const second = room.read('native.b', 'franchise.01');
    expect(room.peek().draft).toBeUndefined();
    release(reading('native.b'));
    await second;
    expect(room.peek().draft?.league).toBe('native.b');
  });
  it('keeps the last draft it read, and says why, when a look at it fails', async () => {
    let fail = false;
    const room = createDraftRoom({ pick: async () => receipt, read: async () => {
      if (fail) throw new Error('The ledger is busy.');
      return reading('native.a');
    } });
    await room.read('native.a', 'franchise.01');
    fail = true;
    await room.read('native.a', 'franchise.01');
    expect(room.peek()).toMatchObject({ error: 'The ledger is busy.', pending: false });
    expect(room.peek().draft?.season).toBe(2027);
  });
  it('lets only the newest look stand, whatever order the answers arrive in', async () => {
    const waiting: ((value: unknown) => void)[] = [];
    const room = createDraftRoom({ pick: async () => receipt,
      read: () => new Promise(resolve => { waiting.push(resolve); }) });
    const old = room.read('native.a', 'franchise.01');
    const newest = room.read('native.a', 'franchise.02');
    waiting[1]({ ...reading('native.a'), revision: 20 });
    await newest;
    waiting[0]({ ...reading('native.a'), revision: 10 });
    await old;
    expect(room.peek().draft?.revision).toBe(20);
  });
  it('asks for the draft that is next, reading at Go’s own time', async () => {
    const asked: unknown[] = [];
    const room = createDraftRoom({ pick: async () => receipt, read: async scope => {
      asked.push(scope);
      return reading('native.a');
    } });
    await room.read('native.a', 'franchise.01');
    expect(asked).toEqual([{ league: 'native.a', season: 0, franchise: 'franchise.01',
      at: '0001-01-01T00:00:00Z' }]);
  });
});

describe('The Draft room is lazy', () => {
  it('is loaded on demand from the Draft room, and so is its tray', () => {
    const market = readFileSync('src/app/market/Market.tsx', 'utf8');
    expect(market).toContain("lazy(() => import('../draftroom/DraftRoom'))");
    const room = readFileSync('src/app/draftroom/DraftRoom.tsx', 'utf8');
    expect(room).toContain("lazy(() => import('./DraftTray'))");
    expect(readFileSync('src/app/shell/TargetApp.tsx', 'utf8')).not.toContain('draftroom');
  });
});

it('retains refusal wording during retry and fences a read when the room is reset', async () => {
  let fail = true;
  let finish!: (value: unknown) => void;
  const room = createDraftRoom({ pick: async () => receipt, read: async () => {
    if (fail) throw new Error('The ledger is busy.');
    return new Promise(resolve => { finish = resolve; });
  } });
  await room.read('native.a', 'franchise.01');
  fail = false;
  const pending = room.read('native.a', 'franchise.01');
  expect(room.peek()).toMatchObject({ error: 'The ledger is busy.', pending: true });
  room.reset();
  finish(reading('native.a'));
  await pending;
  expect(room.peek()).toEqual({ error: '', pending: false, pick: { status: 'idle' } });
});
