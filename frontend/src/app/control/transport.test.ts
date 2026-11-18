import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import { afterEach, expect, it, vi } from 'vitest';
import { nativeTransport, selectNativeTransport, webTransport, type NativeTransport } from './transport';
import type { NativeRequest } from '../commands/native';

const admission = { league: 'native.test', name: 'The League', franchise: 'franchise.01',
  franchiseName: 'Home', role: 'Owner', seats: [{ franchise: 'franchise.01', name: 'Home' }] };
const jsonResponse = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
  status, headers: { 'Content-Type': 'application/json' },
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

it('selects the Wails port only when window.go exists', async () => {
  expect(selectNativeTransport(true).kind).toBe('wails');
  expect(selectNativeTransport(false)).toBe(webTransport);
  expect(nativeTransport.kind).toBe('wails');
});

it('restores, signs in and signs out through the cookie session without a bearer', async () => {
  const fetcher = vi.fn().mockResolvedValue(jsonResponse({ protocol: 1, admission }));
  vi.stubGlobal('fetch', fetcher);
  await webTransport.startup();
  expect(fetcher.mock.calls[0][0]).toBe('/app/session');
  expect(fetcher.mock.calls[0][1]).toMatchObject({
    method: 'GET', credentials: 'same-origin', cache: 'no-store',
  });

  await webTransport.action({ action: 'login', username: 'owner', password: 'private-password',
    code: '123456' });
  const [loginPath, loginInit] = fetcher.mock.calls[1];
  expect(loginPath).toBe('/app/session');
  expect(loginInit).toMatchObject({ method: 'POST', credentials: 'same-origin', cache: 'no-store' });
  expect(JSON.parse(loginInit.body)).toEqual({ protocol: 1, username: 'owner', password: 'private-password',
    code: '123456' });
  expect(loginInit.headers).toEqual({ 'Content-Type': 'application/json' });

  await webTransport.action({ action: 'forget', username: '', password: '', code: '' });
  expect(fetcher.mock.calls[2][0]).toBe('/app/session');
  expect(fetcher.mock.calls[2][1]).toMatchObject({ method: 'DELETE', credentials: 'same-origin' });
  for (const [, init] of fetcher.mock.calls) {
    expect(init.headers ?? {}).not.toHaveProperty('Authorization');
    expect(String(init.body ?? '')).not.toMatch(/bearer/i);
  }
});

it('runs browser startup, sign-in and sign-out through the public session API', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' },
    addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() });
  vi.resetModules();
  const fetcher = vi.fn()
    .mockResolvedValueOnce(jsonResponse({ protocol: 1, admission }))
    .mockResolvedValueOnce(jsonResponse({ protocol: 1, admission }))
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetcher);
  const api = await import('./nativeApi');
  const restored = await api.nativeStartup();
  expect(restored?.league).toBe('native.test');
  expect((await api.nativeAction('login', 'owner', 'private-password', '123456')).franchise)
    .toBe('franchise.01');
  expect((await api.nativeAction('forget')).name).toBe('');
  expect(fetcher.mock.calls.map(([path]) => path)).toEqual([
    '/app/session', '/app/session', '/app/session',
  ]);
  expect(fetcher.mock.calls.map(([, init]) => init.method)).toEqual(['GET', 'POST', 'DELETE']);
  for (const [, init] of fetcher.mock.calls) expect(init.credentials).toBe('same-origin');
});

it('rejects a browser session response that includes bearer material', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' } });
  vi.resetModules();
  const fetcher = vi.fn()
    .mockResolvedValueOnce(jsonResponse({ protocol: 1, admission }))
    .mockResolvedValueOnce(jsonResponse({ protocol: 1, admission, bearer: 'must-not-enter-js-state' }));
  vi.stubGlobal('fetch', fetcher);
  const api = await import('./nativeApi');
  await api.nativeStartup();
  await expect(api.nativeAction('login', 'owner', 'private-password'))
    .rejects.toThrow('Unreadable account response.');
});

it('sends the generated workflow on the Go client route and wire envelope', async () => {
  const fetcher = vi.fn().mockResolvedValue(jsonResponse({ protocol: 1 }));
  vi.stubGlobal('fetch', fetcher);
  const request = { command: 'player.note.read', mode: 'read',
    arguments: { league: 'native.test', player: '001' } } as NativeRequest;
  await webTransport.workflow(request, 'football-digest');
  const [path, init] = fetcher.mock.calls[0];
  expect(path).toBe('/v1/native/workflow');
  expect(init).toMatchObject({ method: 'POST', credentials: 'same-origin', cache: 'no-store' });
  expect(JSON.parse(init.body)).toEqual({ protocol: 1, workflow: {
    command: 'player.note.read', version: '1', mode: 'read',
    arguments: { league: 'native.test', player: '001' }, football: 'football-digest',
  } });
  expect(init.headers).toEqual({ 'Content-Type': 'application/json' });
});

it('does not open the desktop bearer-only invalidation stream for browser sessions', () => {
  const eventSource = vi.fn();
  vi.stubGlobal('EventSource', eventSource);
  webTransport.setIdentity('native.test', '4');
  const listener = vi.fn();
  const close = webTransport.subscribeChanged(listener);
  expect(eventSource).not.toHaveBeenCalled();
  expect(listener).not.toHaveBeenCalled();
  close();
});

it('refuses account operations that the browser-cookie contract does not admit', async () => {
  const fetcher = vi.fn();
  vi.stubGlobal('fetch', fetcher);
  await expect(webTransport.action({ action: 'enroll' })).rejects.toThrow('not available');
  await expect(webTransport.action({ action: 'revoke-all' })).rejects.toThrow('not available');
  expect(fetcher).not.toHaveBeenCalled();
});

it('service worker handles only the precached shell and leaves JSON API routes on the network', () => {
  const template = readFileSync(new URL('../../../pwa/sw.js', import.meta.url), 'utf8');
  const shell = ['/app/', '/app/assets/main.a1b2c3d4.js'];
  const source = template.replace('__APP_SHELL__', JSON.stringify(shell));
  const handlers = new Map<string, (event: Record<string, unknown>) => void>();
  const self = { location: { origin: 'https://league.example' },
    addEventListener: (type: string, listener: (event: Record<string, unknown>) => void) =>
      handlers.set(type, listener),
    clients: { claim: vi.fn() }, skipWaiting: vi.fn() };
  const cache = { match: vi.fn(async () => undefined) };
  runInNewContext(source, { self, URL, fetch: vi.fn(async () => new Response()),
    caches: { open: vi.fn(async () => cache) } });
  const fetchHandler = handlers.get('fetch');
  expect(fetchHandler).toBeDefined();
  for (const request of [
    { url: 'https://league.example/v1/native/workflow', method: 'POST', mode: 'cors' },
    { url: 'https://league.example/api/league.json', method: 'GET', mode: 'cors' },
  ]) {
    const respondWith = vi.fn();
    fetchHandler!({ request, respondWith });
    expect(respondWith).not.toHaveBeenCalled();
  }
  const asset = { url: 'https://league.example/app/assets/main.a1b2c3d4.js', method: 'GET', mode: 'cors' };
  const respondWith = vi.fn();
  fetchHandler!({ request: asset, respondWith });
  expect(respondWith).toHaveBeenCalledOnce();
});

it('keeps the browser interface aligned with the transport contract', () => {
  const required: (keyof NativeTransport)[] = [
    'kind', 'startup', 'action', 'helper', 'workflow', 'subscribeChanged', 'setIdentity',
  ];
  expect(Object.keys(webTransport).sort()).toEqual(required.sort());
});
