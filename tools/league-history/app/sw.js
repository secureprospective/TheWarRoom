// Offline copy of the Lab: the app files and the league data, refreshed whenever online.
const VERSION = 'lab-v10';
const SHELL = ['./', 'index.html', 'style.css', 'manifest.webmanifest', 'icon.svg', 'icon-192.png', 'icon-512.png',
  'js/main.js', 'js/model.js', 'js/engine.js', 'js/draftlab.js', 'js/teams.js', 'js/ui.js',
  'js/views/market.js', 'js/views/draft.js', 'js/views/tradelog.js', 'js/views/method.js',
  'data/lab.json'];

self.addEventListener('install', event => {
  event.waitUntil(caches.open(VERSION).then(cache => cache.addAll(SHELL)).then(() => self.skipWaiting()));
});

self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    for (const name of await caches.keys()) if (name !== VERSION) await caches.delete(name);
    await self.clients.claim();
  })());
});

// Network first, so a rebuilt data file shows up at once; the saved copy covers offline use.
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);
  if (event.request.method !== 'GET' || url.origin !== self.location.origin) return;
  event.respondWith((async () => {
    const cache = await caches.open(VERSION);
    try {
      const response = await fetch(event.request);
      if (response.ok) await cache.put(event.request, response.clone());
      return response;
    } catch {
      const saved = await cache.match(event.request, {ignoreSearch: true});
      return saved || new Response('Not available offline', {status: 503, headers: {'Content-Type': 'text/plain'}});
    }
  })());
});
