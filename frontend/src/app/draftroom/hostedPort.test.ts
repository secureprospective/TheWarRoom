import { expect, it, vi } from 'vitest';
import { hostedDraftPort } from './hostedPort';

it('uses the hosted native transport for draft reads and owner picks', async () => {
  const transport = vi.fn(async request => ({ data: { command: request.command } }));
  const port = hostedDraftPort(transport);
  const scope = { league: 'native.test', season: 2026, franchise: 'franchise.01',
    at: '0001-01-01T00:00:00Z' };
  const choice = { ...scope, player: '1001', request: 'request-1', expected: 4 };

  expect(await port.read(scope)).toEqual({ command: 'draft.read' });
  expect(await port.pick(choice)).toEqual({ command: 'draft.pick' });
  expect(transport.mock.calls.map(([request]) => [request.command, request.mode])).toEqual([
    ['draft.read', 'read'], ['draft.pick', 'execute'],
  ]);
  expect(transport.mock.calls[1][0].arguments).toMatchObject({ franchise: 'franchise.01', player: '1001' });
});
