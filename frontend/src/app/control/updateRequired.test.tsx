import { afterEach, expect, it, vi } from 'vitest';
import { all, button, click, mount, render, settle, unmount } from '../commissioner/commissionerDom';
import UpdateRequired from './UpdateRequired';
import { nativeAction, parseNativeView, subscribeNativeUpdate, type NativeView } from './nativeApi';

const initial: NativeView = { server: 'https://league.example', league: 'native.test', message: '',
  update: 'app', name: '', role: '', franchise: '', franchiseName: '', seed: '', connection: '',
  helperId: '', confirmation: '', authGeneration: '0', seats: [] };
afterEach(async () => { await unmount(); vi.unstubAllGlobals(); });

it('keeps the typed update signal from a hosted refusal and rejects unknown sides', async () => {
  const listener = vi.fn();
  const unsubscribe = subscribeNativeUpdate(listener);
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 426, json: async () => initial }));
  expect(await nativeAction('enter')).toEqual(initial);
  expect(listener).toHaveBeenCalledWith(initial);
  unsubscribe();
  expect(() => parseNativeView({ ...initial, update: 'anything' })).toThrow();
});
