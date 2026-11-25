import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import { afterEach, expect, it, vi } from 'vitest';
import { nativeTransport, type NativeTransport } from './transport';
import type { NativeRequest } from '../commands/native';

const admission = { league: 'native.test', name: 'The League', franchise: 'franchise.01',
  franchiseName: 'Home', role: 'Owner', seats: [{ franchise: 'franchise.01', name: 'Home' }] };
const memberships = [
  { id: 'native.test', name: 'The League', role: 'Owner', franchise: 'franchise.01', franchiseName: 'Home' },
  { id: 'native.second', name: 'Second League', role: 'Commissioner', franchise: '', franchiseName: '' },
];
const sessionResponse = { protocol: 1, admission, leagues: memberships };
const jsonResponse = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
  status, headers: { 'Content-Type': 'application/json' },
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

it('restores, signs in and signs out through the cookie session without a bearer', async () => {
  const fetcher = vi.fn().mockResolvedValue(jsonResponse(sessionResponse));
  vi.stubGlobal('fetch', fetcher);
  await nativeTransport.startup();
  expect(fetcher.mock.calls[0][0]).toBe('/app/session');
  expect(fetcher.mock.calls[0][1]).toMatchObject({
    method: 'GET', credentials: 'same-origin', cache: 'no-store',
  });

  await nativeTransport.action({ action: 'login', username: 'owner', password: 'private-password',
    code: '123456' });
  const [loginPath, loginInit] = fetcher.mock.calls[1];
  expect(loginPath).toBe('/app/session');
  expect(loginInit).toMatchObject({ method: 'POST', credentials: 'same-origin', cache: 'no-store' });
  expect(JSON.parse(loginInit.body)).toEqual({ protocol: 1, username: 'owner', password: 'private-password',
    code: '123456' });
  expect(loginInit.headers).toEqual({ 'Content-Type': 'application/json' });

  await nativeTransport.action({ action: 'forget', username: '', password: '', code: '' });
  expect(fetcher.mock.calls[2][0]).toBe('/app/session');
  expect(fetcher.mock.calls[2][1]).toMatchObject({ method: 'DELETE', credentials: 'same-origin' });
  for (const [, init] of fetcher.mock.calls) {
    expect(init.headers ?? {}).not.toHaveProperty('Authorization');
    expect(String(init.body ?? '')).not.toMatch(/bearer/i);
  }
});

// The change stream also rides fetch; these tests count only the session calls.
function ignoringChangeStream(fetcher: (path: string, init?: RequestInit) => unknown) {
  return (path: string, init?: RequestInit) => (path === '/v1/native/invalidations'
    ? new Promise<Response>(() => {}) : fetcher(path, init));
}

it('runs browser startup, sign-in and sign-out through the public session API', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' },
    addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() });
  vi.resetModules();
  const fetcher = vi.fn()
    .mockResolvedValueOnce(jsonResponse(sessionResponse))
    .mockResolvedValueOnce(jsonResponse(sessionResponse))
    .mockResolvedValueOnce(jsonResponse(sessionResponse))
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', ignoringChangeStream(fetcher));
  const api = await import('./nativeApi');
  const restored = await api.nativeStartup();
  expect(restored?.league).toBe('native.test');
  expect(restored?.leagues).toEqual(memberships);
  expect((await api.nativeAction('login', 'owner', 'private-password', '123456')).franchise)
    .toBe('franchise.01');
  const switched = api.selectBrowserLeague('native.second');
  expect(switched.league).toBe('native.second');
  expect(switched.role).toBe('Commissioner');
  expect(switched.authGeneration).not.toBe(restored!.authGeneration);
  const { hostedSnapshot } = await import('../data/hostedReads');
  expect(hostedSnapshot().view?.league).toBe('native.second');
  expect((await api.nativeAction('status')).league).toBe('native.second');
  expect(() => api.selectBrowserLeague('native.unrelated')).toThrow('not admitted');
  expect((await api.nativeAction('forget')).name).toBe('');
  expect(fetcher.mock.calls.map(([path]) => path)).toEqual([
    '/app/session', '/app/session', '/app/session', '/app/session',
  ]);
  expect(fetcher.mock.calls.map(([, init]) => init.method)).toEqual(['GET', 'POST', 'GET', 'DELETE']);
  for (const [, init] of fetcher.mock.calls) expect(init.credentials).toBe('same-origin');
});

it('rejects a browser session response that includes bearer material', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' } });
  vi.resetModules();
  const fetcher = vi.fn()
    .mockResolvedValueOnce(jsonResponse(sessionResponse))
    .mockResolvedValueOnce(jsonResponse({ ...sessionResponse, bearer: 'must-not-enter-js-state' }));
  vi.stubGlobal('fetch', ignoringChangeStream(fetcher));
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
  await nativeTransport.workflow(request, 'football-digest');
  const [path, init] = fetcher.mock.calls[0];
  expect(path).toBe('/v1/native/workflow');
  expect(init).toMatchObject({ method: 'POST', credentials: 'same-origin', cache: 'no-store' });
  expect(JSON.parse(init.body)).toEqual({ protocol: 1, workflow: {
    command: 'player.note.read', version: '1', mode: 'read',
    arguments: { league: 'native.test', player: '001' }, football: 'football-digest',
  } });
  expect(init.headers).toEqual({ 'Content-Type': 'application/json' });
});

it('routes account layout workflows through the account preference endpoint', async () => {
  const fetcher = vi.fn().mockResolvedValue(jsonResponse({ protocol: 1 }));
  vi.stubGlobal('fetch', fetcher);
  const request = { command: 'home.layout.set', mode: 'execute', arguments: {
    layout: { mode: 'default', tiles: [], over: [], pins: {} }, expectedRevision: 3,
  } } as NativeRequest;
  await nativeTransport.workflow(request, '');
  const [path, init] = fetcher.mock.calls[0];
  expect(path).toBe('/v1/account/preferences');
  expect(JSON.parse(init.body)).toEqual({ protocol: 1, workflow: {
    command: 'home.layout.set', version: '1', mode: 'execute', arguments: request.arguments, football: '',
  } });
});

it('routes private job snooze workflows through the account preference endpoint', async () => {
  const fetcher = vi.fn().mockResolvedValue(jsonResponse({ protocol: 1 }));
  vi.stubGlobal('fetch', fetcher);
  const request = { command: 'job.snoozes.read', mode: 'read', arguments: {} } as NativeRequest;
  await nativeTransport.workflow(request, '');
  const [path, init] = fetcher.mock.calls[0];
  expect(path).toBe('/v1/account/preferences');
  expect(JSON.parse(init.body)).toMatchObject({ protocol: 1, workflow: {
    command: 'job.snoozes.read', mode: 'read', arguments: {},
  } });
});

it('opens the change stream with the session cookie and hands each event to the listener', async () => {
  const frames = 'event: invalidate\ndata: {"protocol":1,"topics":[]}\n\n: keepalive\n\n'
    + 'event: invalidate\ndata: {"protocol":1,"league":"native.test","topics":[]}\n\n';
  const fetcher = vi.fn().mockResolvedValue(new Response(new ReadableStream({
    start(c) { c.enqueue(new TextEncoder().encode(frames)); },
  }), { status: 200, headers: { 'Content-Type': 'text/event-stream' } }));
  vi.stubGlobal('fetch', fetcher);
  nativeTransport.setIdentity('native.test');
  const listener = vi.fn();
  const close = nativeTransport.subscribeChanged(listener);
  await vi.waitFor(() => expect(listener).toHaveBeenCalledTimes(2));
  const [path, init] = fetcher.mock.calls[0];
  expect(path).toBe('/v1/native/invalidations');
  expect(init.credentials).toBe('same-origin');
  expect(init.headers).toEqual({ 'X-TWR-League': 'native.test', 'X-TWR-Invalidation': '2' });
  expect(listener.mock.calls[1][0]).toMatchObject({ league: 'native.test' });
  close();
});

it('refuses account operations that the browser-cookie contract does not admit', async () => {
  const fetcher = vi.fn();
  vi.stubGlobal('fetch', fetcher);
  await expect(nativeTransport.action({ action: 'enroll' })).rejects.toThrow('not available');
  await expect(nativeTransport.action({ action: 'revoke-all' })).rejects.toThrow('not available');
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
    { url: 'https://league.example/v1/account/preferences', method: 'POST', mode: 'cors' },
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
    'startup', 'action', 'workflow', 'subscribeChanged', 'setIdentity',
  ];
  expect(Object.keys(nativeTransport).sort()).toEqual(required.sort());
});

it('captures server X-TWR-Error-Code headers and mints client codes on network failures', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' },
    addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() });
  vi.resetModules();
  const fetcher = vi.fn()
    .mockResolvedValueOnce(new Response('League refused', {
      status: 409,
      headers: {
        'Content-Type': 'text/plain; charset=utf-8',
        'X-TWR-Error-Code': 'TWR-7K9M-2P4X',
      },
    }))
    .mockRejectedValueOnce(new TypeError('Failed to fetch'));
  vi.stubGlobal('fetch', ignoringChangeStream(fetcher));
  const api = await import('./nativeApi');
  const codes = await import('../data/errorCodes');
  const request = {
    command: 'feedback.report.create', mode: 'execute',
    arguments: {
      league: 'native.test', text: 'hi', route: 'home/overview',
      appRelease: 42, viewportWidth: 1200,
    },
  } as NativeRequest;

  let serverErr: unknown;
  try {
    await api.hostedWorkflow(request);
  } catch (err) {
    serverErr = err;
  }
  expect(codes.errorCodeFor(serverErr)).toBe('TWR-7K9M-2P4X');

  let networkErr: unknown;
  try {
    await api.hostedWorkflow(request);
  } catch (err) {
    networkErr = err;
  }
  expect(codes.errorCodeFor(networkErr)).toMatch(/^TWR-C/);
});

it('shares identical reads in flight, then reads fresh', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://league.example' },
    addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() });
  vi.resetModules();
  const body = JSON.stringify({ label: 'L', writable: true, football: 'f', data: { n: 1 } });
  const reply = () => new Response(body, { status: 200, headers: { 'Content-Type': 'application/json' } });
  const fetcher = vi.fn(async () => reply());
  vi.stubGlobal('fetch', ignoringChangeStream(fetcher));
  const api = await import('./nativeApi');
  const read = { command: 'results.read', mode: 'read',
    arguments: { league: 'native.test', season: 2026, week: 3 } } as NativeRequest;
  const [a, b] = await Promise.all([api.hostedWorkflow(read), api.hostedWorkflow(read)]);
  expect(fetcher).toHaveBeenCalledTimes(1);
  expect(a.data).toEqual({ n: 1 });
  expect(a).not.toBe(b);
  await api.hostedWorkflow(read);
  expect(fetcher).toHaveBeenCalledTimes(2);
});
