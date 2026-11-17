import { expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { WarRoomContext, WarRoomNative } from './warroomReads';
import { readWarRoomDraft, saveWarRoomDraftQueue } from './warroomDraftReads';

vi.mock('../draftroom/draft', () => ({ parseDraft: (value: unknown) => value }));

it('reads and replaces a hosted queue without accepting a franchise from the request', async () => {
  const calls: { command: string; mode: string; arguments: unknown }[] = [];
  const port: HostedPort = vi.fn(async (request: Parameters<HostedPort>[0]) => {
    calls.push(request);
    if (request.command === 'draft.read') return { data: { league: 'native.test' },
      label: 'server', writable: true, football: '' };
    return { data: { players: ['0100', '0099'] }, label: 'server', writable: true, football: '' };
  });
  const context = { native: { league: 'native.test', mine: 'franchise.02', leave: () => {}, port },
    mine: 'franchise.02' } as WarRoomContext;
  const reading = await readWarRoomDraft(context);
  expect(reading.queue).toEqual(['0100', '0099']);
  expect(calls.map(call => call.command)).toEqual(['draft.read', 'draft.queue.read']);
  expect(calls[1].arguments).toEqual({ league: 'native.test' });
  expect(calls[1].arguments).not.toHaveProperty('franchise');

  calls.length = 0;
  await saveWarRoomDraftQueue(context.native as WarRoomNative, ['0099', '0100']);
  expect(calls).toEqual([{ command: 'draft.queue.set', mode: 'execute',
    arguments: { league: 'native.test', players: ['0099', '0100'] } }]);
});
