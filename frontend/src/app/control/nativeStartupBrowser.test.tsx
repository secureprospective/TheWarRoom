import { afterEach, expect, it, vi } from 'vitest';
import { nativeAction, nativeStartup } from './nativeApi';

afterEach(() => { vi.unstubAllGlobals(); });

it('gives an account with no league seat the sign-in view, not "server could not be reached"', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' } });
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 403,
    headers: new Headers({ 'Content-Type': 'text/plain' }), text: async () => 'league access unavailable' }));
  const view = await nativeStartup();
  expect(view?.message).toContain('has no team in a league on this server');
  expect(view?.league ?? '').toBe('');
});

it('opening a league with no seat gives the plain no-team message, not the raw refusal', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' } });
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 403,
    headers: new Headers({ 'Content-Type': 'text/plain' }), text: async () => 'league access unavailable' }));
  const view = await nativeAction('enter');
  expect(view.message).toContain('has no team in a league on this server');
  expect(view.league).toBe('');
});
