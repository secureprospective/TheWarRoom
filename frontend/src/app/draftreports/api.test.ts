import { afterEach, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { provider, readDraftLab, readNFLDraft, subscribeDraft } from './api';
import wire from './wire.testdata.json';
import { parseBoard } from './provider';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const scope = { from: 2017, to: 2026, seasons: 3, group: 'DL', segments: [], nfl: [] };
const flush = () => new Promise<void>(resolve => setTimeout(resolve, 0));

it('routes all five calls through draft.API and parses their wire output', async () => {
  const lab = JSON.parse(readFileSync('../testdata/wire/draft/lab.json', 'utf8'));
  const nfl = JSON.parse(readFileSync('../testdata/wire/draft/nfl-held.json', 'utf8'));
  const API = {
    TargetDraftReports: vi.fn(async () => wire),
    TargetDraftReportsRefresh: vi.fn(async () => {}),
    TargetOpenMFLDraftRoom: vi.fn(async () => {}),
    TargetDraftLab: vi.fn(async () => lab),
    TargetNFLDraft: vi.fn(async () => nfl),
  };
  vi.stubGlobal('window', { go: { draft: { API } } });
  expect(await provider.read()).toEqual(parseBoard(wire));
  await provider.refresh(true);
  expect(API.TargetDraftReportsRefresh).toHaveBeenCalledWith(true);
  await provider.openMFL();
  expect(API.TargetOpenMFLDraftRoom).toHaveBeenCalledOnce();
  expect((await readDraftLab(scope)).count).toBe(lab.count);
  expect(API.TargetDraftLab).toHaveBeenCalledWith(scope);
  expect((await readNFLDraft()).id).toBe('M-087');
  expect(API.TargetNFLDraft).toHaveBeenCalledOnce();
  API.TargetDraftLab.mockRejectedValueOnce(new Error('offline'));
  expect((await readDraftLab(scope)).absenceReason).toContain('Draft read failed: Error: offline. Retry the held archive reading.');
});

it('preserves the plain-browser refusal without substituting fixtures', async () => {
  for (const window of [undefined, {}]) {
    vi.stubGlobal('window', window);
    await expect(provider.read()).rejects.toThrow('Draft reports require the desktop app');
    await expect(provider.refresh(false)).rejects.toThrow('Draft reports require the desktop app');
    await expect(provider.openMFL()).rejects.toThrow('Draft reports require the desktop app');
    expect(await readDraftLab(scope)).toMatchObject({ status: 'not_wired', threshold: 40 });
  }
});

it('owns season event delivery, cleanup and disposal before import resolution', async () => {
  const stopRuntime = vi.fn();
  let callback: (() => void) | undefined;
  const EventsOnMultiple = vi.fn((_name: string, changed: () => void) => {
    callback = changed;
    return stopRuntime;
  });
  vi.stubGlobal('window', { runtime: { EventsOnMultiple } });
  const changed = vi.fn();
  const stop = subscribeDraft(changed);
  await flush();
  for (let i = 0; i < 30 && EventsOnMultiple.mock.calls.length === 0; i++) await flush();
  expect(EventsOnMultiple).toHaveBeenCalledWith('target:season', changed, -1);
  callback?.();
  expect(changed).toHaveBeenCalledOnce();
  stop();
  expect(stopRuntime).toHaveBeenCalledOnce();
  EventsOnMultiple.mockClear();
  const disposed = subscribeDraft(changed);
  disposed();
  await flush();
  expect(EventsOnMultiple).not.toHaveBeenCalled();
});
