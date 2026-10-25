import { afterEach, expect, it, vi } from 'vitest';
import { all, button, click, mount, render, settle, unmount } from '../commissioner/commissionerDom';
import UpdateRequired from './UpdateRequired';
import { nativeAction, parseNativeView, subscribeNativeUpdate, type NativeView } from './nativeApi';

const initial: NativeView = { server: 'https://league.example', league: 'native.test', message: '',
  update: 'app', name: '', role: '', franchise: '', franchiseName: '', seed: '', connection: '',
  helperId: '', confirmation: '', seats: [] };
afterEach(async () => { await unmount(); vi.unstubAllGlobals(); });

it('offers one action, explains league safety, and permits retry after failure', async () => {
  const fetcher = vi.fn().mockResolvedValue({ ok: false,
    text: async () => 'The update could not be checked. Try again. Your league is unchanged.' });
  vi.stubGlobal('fetch', fetcher);
  const host = await mount(render(UpdateRequired, { initial, onRetry: vi.fn() }));
  expect(all(host, 'button')).toHaveLength(1);
  expect(host.textContent).toContain('Your league stays on the server');
  expect(host.textContent).not.toContain('native.test');
  await click(button(host, 'Update and reopen')); await settle();
  expect(host.textContent).toContain('Try again. Your league is unchanged.');
  await click(button(host, 'Update and reopen')); await settle();
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(JSON.parse(fetcher.mock.calls[0][1].body).action).toBe('update');
});

it('never offers desktop installation for an older server', async () => {
  const next = { ...initial, update: undefined }; delete next.update;
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => next }));
  const retry = vi.fn();
  const host = await mount(render(UpdateRequired, {
    initial: { ...initial, update: 'server' }, onRetry: retry }));
  expect(host.textContent).toContain('Ask your commissioner');
  await click(button(host, 'Try again')); await settle();
  expect(retry).toHaveBeenCalledWith(next);
});

it('keeps the typed update signal across the webview boundary and rejects unknown sides', async () => {
  const listener = vi.fn();
  const unsubscribe = subscribeNativeUpdate(listener);
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 426, json: async () => initial }));
  expect(await nativeAction('enter')).toEqual(initial);
  expect(listener).toHaveBeenCalledWith(initial);
  unsubscribe();
  expect(() => parseNativeView({ ...initial, update: 'anything' })).toThrow();
});
