// The build step swaps the version placeholder below for a hash of the shell and this file, so every
// release installs a new worker and the activate step drops the previous release's cache.
const BUILD = '__TWR_BUILD__';
const CACHE = `twr-static-shell-${BUILD}`;
const APP_SHELL = __APP_SHELL__;

self.addEventListener('install', event => {
  // cache: 'reload' skips the browser HTTP cache so the new shell comes from the server.
  event.waitUntil(caches.open(CACHE)
    .then(cache => cache.addAll(APP_SHELL.map(path => new Request(path, { cache: 'reload' }))))
    .then(() => self.skipWaiting()));
});

self.addEventListener('activate', event => {
  event.waitUntil(caches.keys().then(keys => Promise.all(keys
    .filter(key => key.startsWith('twr-static-shell-') && key !== CACHE)
    .map(key => caches.delete(key))))
    .then(() => self.clients.claim()));
});

self.addEventListener('fetch', event => {
  const request = event.request;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (request.mode === 'navigate' && url.pathname.startsWith('/app/')) {
    event.respondWith(fetch(request).catch(() => caches.open(CACHE)
      .then(cache => cache.match('/app/'))));
    return;
  }
  if (!APP_SHELL.includes(url.pathname)) return;
  event.respondWith(caches.open(CACHE).then(cache => cache.match(request)
    .then(cached => cached || fetch(request))));
});
