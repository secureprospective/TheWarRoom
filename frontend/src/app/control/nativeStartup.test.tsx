import { afterEach, expect, it, vi } from 'vitest';
import { mount, render, settle, unmount } from '../commissioner/commissionerDom';
import { act } from 'react';
import { nativeAction } from './nativeApi';
import { NativeStartup } from './NativeStartup';

const local = vi.hoisted(() => ({ calls: 0 }));
vi.mock('../../TargetMount', () => ({
  TargetMount: () => { local.calls++; return <p>Local reference app</p>; },
}));
afterEach(async () => { await unmount(); vi.unstubAllGlobals(); local.calls = 0; });

it('mounts hosted account access without mounting any local shell or MFL setup', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true,
    headers: new Headers({ 'Content-Type': 'application/json' }), json: async () => ({
    server: 'https://league.example', league: 'native.test', message: '', name: '', role: '',
    franchise: '', franchiseName: '', seed: '', connection: '', helperId: '', confirmation: '', seats: [],
  }) }));
  const host = await mount(render(NativeStartup));
  await vi.waitFor(async () => {
    await settle();
    expect(host.textContent).toContain('No MFL account or key needed');
  }, { timeout: 5000 });
  expect(host.textContent).toContain('Join with a code');
  expect(local.calls).toBe(0);
});

it('fails closed on malformed native status instead of starting the local shell', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true, headers: new Headers({ 'Content-Type': 'application/json' }),
    json: async () => ({ bearer: 'forbidden' }),
  }));
  const host = await mount(render(NativeStartup));
  await settle();
  expect(host.textContent).toContain('Close and reopen');
  expect(local.calls).toBe(0);
});

it('replaces account access with the update screen after a live server refusal', async () => {
  const initial = { server: 'https://league.example', league: 'native.test', message: '', name: '',
    role: '', franchise: '', franchiseName: '', seed: '', connection: '', helperId: '',
    confirmation: '', seats: [] };
  const fetcher = vi.fn().mockResolvedValue({ ok: true,
    headers: new Headers({ 'Content-Type': 'application/json' }), json: async () => initial });
  vi.stubGlobal('fetch', fetcher);
  const host = await mount(render(NativeStartup));
  await vi.waitFor(async () => { await settle(); expect(host.textContent).toContain('Join with a code'); });
  fetcher.mockResolvedValue({ ok: false, status: 426,
    headers: new Headers({ 'Content-Type': 'application/json' }),
    json: async () => ({ ...initial, update: 'app' }) });
  await act(async () => { await nativeAction('enter'); });
  await vi.waitFor(async () => {
    await settle(); expect(host.textContent).toContain('Update and reopen');
  });
  expect(host.textContent).not.toContain('Join with a code');
  expect(local.calls).toBe(0);
});
