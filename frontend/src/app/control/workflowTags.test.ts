import { afterEach, expect, it, vi } from 'vitest';
import { hostedWorkflow, parseReadTags } from './nativeApi';
import type { ReadTags } from '../data/session';

const tags: ReadTags = {
  serverEpoch: 'epoch.one', league: { league: '1', 'scores:2026:1': '2' },
  football: 'pin.one', live: '', values: '',
};
const request = { command: 'native.assets.read', mode: 'read',
  arguments: { league: 'native.hosted' } } as const;
afterEach(() => vi.unstubAllGlobals());

it('preserves exact scoped tags on a full reply and forwards known tags', async () => {
  const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({
    label: 'Server', writable: true, football: 'pin.one', data: { revision: 7 }, tags,
  })));
  vi.stubGlobal('fetch', fetch);
  const result = await hostedWorkflow(request, '', tags);
  expect(result).toMatchObject({ data: { revision: 7 }, tags });
  expect(JSON.parse(fetch.mock.calls[0][1].body).known).toEqual(tags);
});

it('accepts the minimal unchanged reply without inventing a data body', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ unchanged: true, tags }))));
  const result = await hostedWorkflow(request, '', tags);
  expect(result).toEqual({ unchanged: true, tags });
  expect(result).not.toHaveProperty('data');
});

it('refuses unchanged with data, without known, or without tags', async () => {
  for (const reply of [{ unchanged: true, tags, data: {} }, { unchanged: true }]) {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(reply))));
    await expect(hostedWorkflow(request, '', tags)).rejects.toThrow();
  }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ unchanged: true, tags }))));
  await expect(hostedWorkflow(request)).rejects.toThrow();
});

it('retains compatibility with a full protocol-1 reply without tags', async () => {
  const reply = { label: 'Server', writable: false, football: '', data: {} };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(reply))));
  expect(await hostedWorkflow(request)).toEqual(reply);
});

it('validates tag maps and decimal strings before giving them to the read owner', () => {
  for (const league of [null, [], { league: 1 }, { league: '-1' }, { league: '1.1' }])
    expect(() => parseReadTags({ ...tags, league })).toThrow();
  expect(() => parseReadTags({ ...tags, surprise: true })).toThrow();
  expect(parseReadTags(tags)).toEqual(tags);
});
