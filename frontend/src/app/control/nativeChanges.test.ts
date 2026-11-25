import { afterEach, expect, it, vi } from 'vitest';
import { hostedWorkflow } from './nativeApi';
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });
it('parses viewer-scoped freshness on a hosted private workflow read', async () => {
  const tags = { serverEpoch: 'boot', league: { league: '3', 'team:franchise.01': '2' },
    football: 'pin', live: '', values: '' };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true,
    json: async () => ({ label: 'League', writable: false, football: 'pin', data: {}, tags }) }));
  const response = await hostedWorkflow({ command: 'whatif.stack.read', mode: 'read', arguments: {
    league: 'native.test', viewerZone: 'UTC', moves: [],
  } });
  expect(response.tags).toEqual(tags);
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true,
    json: async () => ({ label: 'League', writable: false, football: 'pin', data: {},
      tags: { ...tags, league: { 'other:private': '1' } } }) }));
  await expect(hostedWorkflow({ command: 'whatif.stack.read', mode: 'read', arguments: {
    league: 'native.test', viewerZone: 'UTC', moves: [],
  } })).rejects.toThrow('Unreadable hosted read tags.');
});

