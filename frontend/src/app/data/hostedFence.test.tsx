import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import type { NativeChanges, NativeView } from '../control/nativeApi';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(yes => { resolve = yes; });
  return { promise, resolve };
}
const view: NativeView = { authGeneration: '1', server: 'https://league.example', league: 'native.test',
  name: 'League', message: '', role: 'Owner', franchise: 'one', franchiseName: 'One', seed: '',
  connection: '', helperId: '', confirmation: '', seats: [] };
const response = ({ league, name, role, franchise, franchiseName, seats }: NativeView) => ({
  ok: true, status: 200, headers: new Headers({ 'Content-Type': 'application/json' }),
  json: async () => ({ protocol: 1,
    admission: { league, name, role, franchise, franchiseName, seats },
    leagues: [{ id: league, name, role, franchise, franchiseName }],
  }),
});
let stream: ReadableStreamDefaultController<Uint8Array>;
function streamResponse() {
  return new Response(new ReadableStream<Uint8Array>({ start: controller => { stream = controller; } }));
}
function send(change: unknown) {
  stream.enqueue(new TextEncoder().encode(`event: invalidate\ndata: ${JSON.stringify(change)}\n\n`));
}
let status = deferred<ReturnType<typeof response>>();
let statuses = 0;
let seq = 0;
let owner: typeof import('./hostedReads');
let dom: typeof import('../commissioner/commissionerDom');
let act: typeof import('react').act;
let host: Awaited<ReturnType<typeof dom.mount>>;
let endSession = vi.fn<() => void>();
beforeEach(async () => {
  vi.resetModules();
  vi.stubGlobal('window', { location: { origin: 'https://league.example' } });
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
  status = deferred(); statuses = 0; seq = 0;
  vi.stubGlobal('fetch', vi.fn((url: string) => {
    if (url === '/v1/native/invalidations') return streamResponse();
    statuses++;
    return status.promise;
  }));
  const react = await import('react');
  act = react.act;
  dom = await import('../commissioner/commissionerDom');
  owner = await import('./hostedReads');
  const HostedLeague = (await import('../commissioner/HostedLeague')).default;
  // A joined private query observes actual session ends, including setIdentity's internal reset.
  endSession = vi.fn();
  vi.mocked(fetch).mockResolvedValueOnce(response(view) as unknown as Response);
  await (await import('../control/nativeApi')).nativeStartup();
  owner.hostedReads.join({ invalidate: () => undefined, reset: endSession }, 'private');
  const port = async () => { throw new Error('Offline workflow'); };
  function LeagueFence() {
    const fence = react.useSyncExternalStore(owner.onHostedChange, owner.hostedSnapshot);
    return fence.blocked ? null : react.createElement(HostedLeague, {
      key: fence.key, league: view.league, franchise: fence.view!.franchise, name: 'One',
      role: fence.view!.role, port,
    });
  }
  host = await dom.mount(react.createElement(LeagueFence));
  await dom.flush();
  await dom.click(dom.button(host, 'Roster and reserves'));
  expect(dom.button(host, 'Roster and reserves').getAttribute('aria-current')).toBe('true');
});
afterEach(async () => {
  await dom.unmount();
  vi.clearAllTimers(); vi.useRealTimers(); vi.unstubAllGlobals();
});
async function change(state: NativeChanges['state'], league = false, serverEpoch = '') {
  await act(async () => {
    send({ seq: ++seq, authGeneration: '1', state, serverEpoch, league: view.league,
      topics: league ? [{ kind: 'league', version: String(seq) }] : [] });
    await vi.advanceTimersByTimeAsync(0);
  });
}
async function answer(value: ReturnType<typeof response>) {
  await act(async () => { status.resolve(value); await vi.advanceTimersByTimeAsync(0); });
}
it('keeps HostedLeague local state through five league hints and reconnect resync hints', async () => {
  await change('connected', false, 'first-epoch');
  for (let index = 0; index < 5; index++) await change('connected', true, 'first-epoch');
  await change('reconnecting');
  await change('connected', false, 'first-epoch');
  expect(statuses).toBe(1);
  const first = status;
  status = deferred();
  await act(async () => { first.resolve(response(view)); await vi.advanceTimersByTimeAsync(0); });
  expect(statuses).toBe(2);
  await answer(response(view));
  expect(statuses).toBe(2);
  expect(endSession).not.toHaveBeenCalled();
  expect(dom.button(host, 'Roster and reserves').getAttribute('aria-current')).toBe('true');
});
it('keeps HostedLeague local state when an ordinary league status refresh fails', async () => {
  await change('connected', true);
  await answer({ ...response(view), ok: false, status: 503 });
  expect(owner.hostedSnapshot().blocked).toBe(false);
  expect(endSession).not.toHaveBeenCalled();
  expect(dom.button(host, 'Roster and reserves').getAttribute('aria-current')).toBe('true');
});
it('remounts HostedLeague only when the status reply changes its role', async () => {
  await change('connected', true);
  expect(endSession).not.toHaveBeenCalled();
  expect(dom.button(host, 'Roster and reserves').getAttribute('aria-current')).toBe('true');
  await answer(response({ ...view, role: 'Commissioner' }));
  expect(endSession).toHaveBeenCalledOnce();
  expect(dom.button(host, 'Set lineup').getAttribute('aria-current')).toBe('true');
  expect(dom.button(host, 'Roster and reserves').getAttribute('aria-current')).toBeNull();
});
it('keeps a known epoch change hidden when its admission refresh fails', async () => {
  await change('connected', false, 'first');
  expect(endSession).not.toHaveBeenCalled();
  await change('connected', false, 'second');
  expect(owner.hostedSnapshot().blocked).toBe(true);
  expect(endSession).toHaveBeenCalledOnce();
  await answer({ ...response(view), ok: false, status: 503 });
  expect(owner.hostedSnapshot().blocked).toBe(true);
  expect(host.textContent).toBe('');
});
it('returns the owner to the same screen after a server restart remounts HostedLeague', async () => {
  await change('connected', false, 'first');
  await change('connected', false, 'second');
  expect(host.textContent).toBe('');
  await answer(response(view));
  expect(owner.hostedSnapshot().blocked).toBe(false);
  expect(endSession).toHaveBeenCalled();
  expect(dom.button(host, 'Roster and reserves').getAttribute('aria-current')).toBe('true');
});
it('forgets the remembered screen when someone signs in or out at the window', () => {
  owner.hostedScreens.remember('native.test:one:Owner:pulse', true);
  expect(owner.hostedScreens.recall('native.test:one:Owner:pulse', false)).toBe(true);
  owner.prepareHostedAction('forget');
  expect(owner.hostedScreens.recall('native.test:one:Owner:pulse', false)).toBe(false);
});
