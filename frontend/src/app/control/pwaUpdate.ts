// Keeps the installed web app on the newest release without a reinstall.
// A release installs a new service worker (frontend/pwa/sw.js, versioned at build time). This
// module asks the browser to check for it, and reloads the page once when the new worker takes
// over. It never reloads while an input has focus: it waits for the next visibility change.

export const PWA_SCRIPT = '/app/sw.js';
export const PWA_SCOPE = '/app/';
export const CHECK_EVERY_MS = 30 * 60 * 1000;
export const RELOAD_KEY = 'twr-pwa-reloaded-at';
// Loop breaker: at most one automatic reload per window, even if the worker keeps changing.
export const RELOAD_GUARD_MS = 60 * 1000;

type Store = Pick<Storage, 'getItem' | 'setItem'>;

export interface PwaUpdateDeps {
  container: ServiceWorkerContainer | undefined;
  doc: Document;
  reload: () => void;
  store: Store | undefined;
  now: () => number;
  win: Pick<Window, 'addEventListener' | 'removeEventListener'>;
}

let running = false;

function safeSessionStorage(): Store | undefined {
  try {
    return window.sessionStorage;
  } catch {
    return undefined;
  }
}

function typingNow(doc: Document): boolean {
  const el = doc.activeElement as HTMLElement | null;
  if (!el) return false;
  return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT'
    || el.isContentEditable === true;
}

// Starts the update checks. Safe to call from several places: only the first live call does work.
// Returns a function that stops the checks (tests use it; the app never needs to).
export function startPwaUpdates(overrides: Partial<PwaUpdateDeps> = {}): () => void {
  const container = overrides.container
    ?? (typeof navigator === 'undefined' ? undefined : navigator.serviceWorker);
  if (running || !container) return () => {};
  running = true;
  // Defaults are read only when not overridden, so tests can run without a DOM.
  const deps = {
    container,
    doc: overrides.doc ?? document,
    reload: overrides.reload ?? (() => window.location.reload()),
    store: overrides.store ?? safeSessionStorage(),
    now: overrides.now ?? (() => Date.now()),
  };

  // A first install also fires controllerchange. Only a replaced worker needs a reload.
  const hadController = container.controller !== null;
  let stopped = false;
  let pending = false;
  let reloading = false;
  let registration: ServiceWorkerRegistration | undefined;
  let timer: ReturnType<typeof setInterval> | undefined;

  const reloadNow = () => {
    if (reloading) return;
    if (typingNow(deps.doc)) {
      pending = true;
      return;
    }
    pending = false;
    const last = Number(deps.store?.getItem(RELOAD_KEY) ?? 0);
    if (deps.now() - last < RELOAD_GUARD_MS) return;
    reloading = true;
    deps.store?.setItem(RELOAD_KEY, String(deps.now()));
    deps.reload();
  };

  const onControllerChange = () => {
    if (hadController) reloadNow();
  };

  const onVisibility = () => {
    if (deps.doc.visibilityState === 'visible') void registration?.update().catch(() => {});
    if (pending) reloadNow();
  };

  // A page from an older release asks for code files the server no longer has (a tab opened
  // before the new worker took over). Vite reports that as vite:preloadError; load the new release
  // once instead of showing a broken screen. The reload guard stops a loop if the file is truly gone.
  const onStaleChunk = (event: Event) => {
    const last = Number(deps.store?.getItem(RELOAD_KEY) ?? 0);
    if (reloading || deps.now() - last < RELOAD_GUARD_MS) return;
    event.preventDefault();
    reloading = true;
    deps.store?.setItem(RELOAD_KEY, String(deps.now()));
    deps.reload();
  };
  const win = overrides.win ?? (typeof window === 'undefined' ? undefined : window);
  win?.addEventListener('vite:preloadError', onStaleChunk);

  container.register(PWA_SCRIPT, { scope: PWA_SCOPE, updateViaCache: 'none' }).then(reg => {
    if (stopped) return;
    registration = reg;
    container.addEventListener('controllerchange', onControllerChange);
    deps.doc.addEventListener('visibilitychange', onVisibility);
    timer = setInterval(() => { void reg.update().catch(() => {}); }, CHECK_EVERY_MS);
  }).catch(() => {
    // Registration failed: let the next online or startup call try again.
    running = false;
  });

  return () => {
    stopped = true;
    running = false;
    win?.removeEventListener('vite:preloadError', onStaleChunk);
    container.removeEventListener('controllerchange', onControllerChange);
    deps.doc.removeEventListener('visibilitychange', onVisibility);
    clearInterval(timer);
  };
}
