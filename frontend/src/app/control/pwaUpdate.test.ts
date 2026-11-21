import { afterEach, expect, it, vi } from 'vitest';
import { CHECK_EVERY_MS, PWA_SCOPE, PWA_SCRIPT, RELOAD_GUARD_MS, startPwaUpdates } from './pwaUpdate';

// A fake service worker container, page document, and storage. The real browser objects are not
// needed: the module only uses register, controller, events, visibilityState and activeElement.
type Env = { values: Map<string, string>; clock: { t: number } };
const stops: Array<() => void> = [];

afterEach(() => {
  vi.useRealTimers();
  stops.splice(0).forEach(stop => stop());
});

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve();
};

function env(): Env {
  return { values: new Map(), clock: { t: 1_000_000 } };
}

function fixture(controller: unknown = {}, shared: Env = env()) {
  const reg = { update: vi.fn(async () => undefined) };
  const container = Object.assign(new EventTarget(), {
    controller,
    register: vi.fn(async () => reg),
  });
  const doc = Object.assign(new EventTarget(), {
    visibilityState: 'visible',
    activeElement: null as unknown,
  });
  const reload = vi.fn();
  const win = new EventTarget();
  const store = {
    getItem: (key: string) => shared.values.get(key) ?? null,
    setItem: (key: string, value: string) => { shared.values.set(key, value); },
  };
  const stop = startPwaUpdates({
    container: container as unknown as ServiceWorkerContainer,
    doc: doc as unknown as Document,
    reload,
    store,
    now: () => shared.clock.t,
    win: win as unknown as Window,
  });
  stops.push(stop);
  return { reg, container, doc, reload, stop, shared, win };
}

const controllerChange = (c: EventTarget) => c.dispatchEvent(new Event('controllerchange'));
const visibility = (d: EventTarget) => d.dispatchEvent(new Event('visibilitychange'));

it('registers the worker with updateViaCache none so the browser never serves a stale sw.js', async () => {
  const f = fixture();
  await flush();
  expect(f.container.register).toHaveBeenCalledWith(PWA_SCRIPT, { scope: PWA_SCOPE, updateViaCache: 'none' });
});

it('asks for an update when the page becomes visible, not when it hides', async () => {
  const f = fixture();
  await flush();
  f.doc.visibilityState = 'hidden';
  visibility(f.doc);
  expect(f.reg.update).not.toHaveBeenCalled();
  f.doc.visibilityState = 'visible';
  visibility(f.doc);
  expect(f.reg.update).toHaveBeenCalledTimes(1);
});

it('checks for an update every 30 minutes', async () => {
  vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
  const f = fixture();
  await flush();
  expect(CHECK_EVERY_MS).toBe(30 * 60 * 1000);
  vi.advanceTimersByTime(CHECK_EVERY_MS - 1);
  expect(f.reg.update).not.toHaveBeenCalled();
  vi.advanceTimersByTime(1);
  expect(f.reg.update).toHaveBeenCalledTimes(1);
});

it('reloads exactly once when a replacement worker takes over', async () => {
  const f = fixture({});
  await flush();
  controllerChange(f.container);
  controllerChange(f.container);
  expect(f.reload).toHaveBeenCalledTimes(1);
});

it('does not reload on a first install, when there was no worker before', async () => {
  const f = fixture(null);
  await flush();
  controllerChange(f.container);
  expect(f.reload).not.toHaveBeenCalled();
});

it('does not loop: a second automatic reload inside the guard window is refused', async () => {
  const shared = env();
  const first = fixture({}, shared);
  await flush();
  controllerChange(first.container);
  expect(first.reload).toHaveBeenCalledTimes(1);
  first.stop();

  // The reloaded page starts fresh with the same session storage, a few seconds later.
  shared.clock.t += 5_000;
  const second = fixture({}, shared);
  await flush();
  controllerChange(second.container);
  expect(second.reload).not.toHaveBeenCalled();

  // Once the guard window has passed, a real later release reloads again.
  second.stop();
  shared.clock.t += RELOAD_GUARD_MS;
  const third = fixture({}, shared);
  await flush();
  controllerChange(third.container);
  expect(third.reload).toHaveBeenCalledTimes(1);
});

it('does not reload while an input has focus, and reloads at the next visibility change', async () => {
  const f = fixture({});
  await flush();
  f.doc.activeElement = { tagName: 'INPUT', isContentEditable: false };
  controllerChange(f.container);
  expect(f.reload).not.toHaveBeenCalled();

  // Still typing when the page is visibility-checked: keep waiting.
  visibility(f.doc);
  expect(f.reload).not.toHaveBeenCalled();

  f.doc.activeElement = null;
  visibility(f.doc);
  expect(f.reload).toHaveBeenCalledTimes(1);
});

it('treats a focused textarea as typing too', async () => {
  const f = fixture({});
  await flush();
  f.doc.activeElement = { tagName: 'TEXTAREA', isContentEditable: false };
  controllerChange(f.container);
  expect(f.reload).not.toHaveBeenCalled();
});

it('registers once even when the app calls the starter from several places', async () => {
  const f = fixture();
  await flush();
  const again = startPwaUpdates({
    container: f.container as unknown as ServiceWorkerContainer,
    doc: f.doc as unknown as Document,
    reload: f.reload,
  });
  stops.push(again);
  await flush();
  expect(f.container.register).toHaveBeenCalledTimes(1);
});

it('lets a later call retry after a failed registration', async () => {
  const reg = { update: vi.fn(async () => undefined) };
  const container = Object.assign(new EventTarget(), {
    controller: null,
    register: vi.fn().mockRejectedValueOnce(new Error('blocked')).mockResolvedValue(reg),
  });
  const deps = {
    container: container as unknown as ServiceWorkerContainer,
    doc: Object.assign(new EventTarget(), { visibilityState: 'visible' }) as unknown as Document,
    reload: vi.fn(),
    store: undefined,
  };
  stops.push(startPwaUpdates(deps));
  await flush();
  stops.push(startPwaUpdates(deps));
  await flush();
  expect(container.register).toHaveBeenCalledTimes(2);
});

it('reloads once onto the new release when an old page cannot load a code file', async () => {
  const f = fixture();
  await flush();
  const stale = new Event('vite:preloadError', { cancelable: true });
  f.win.dispatchEvent(stale);
  expect(f.reload).toHaveBeenCalledTimes(1);
  expect(stale.defaultPrevented).toBe(true);
  f.win.dispatchEvent(new Event('vite:preloadError', { cancelable: true }));
  expect(f.reload).toHaveBeenCalledTimes(1);
});
