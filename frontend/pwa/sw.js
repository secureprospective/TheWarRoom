const CACHE = 'twr-static-shell-v1';
const APP_SHELL = __APP_SHELL__;

self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(APP_SHELL))
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
