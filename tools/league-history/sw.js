const PREFIX = `legacy-nfl-history:${self.registration.scope}:`;
const SHELL = `${PREFIX}shell-v2`;
const ARCHIVE = `${PREFIX}archive-v2`;
const ASSETS = ['./','./index.html','./style.css','./app.js','./model.mjs','./behavior.mjs','./behavior_views.mjs','./history_views.mjs','./manifest.webmanifest','./icon.svg','./icon-192.png','./icon-512.png'];
const urls = new Set([...ASSETS,'./data/archive.json','./data/revision.json'].map(path => new URL(path,self.registration.scope).href));

self.addEventListener('install',event => {
  event.waitUntil(caches.open(SHELL).then(cache => cache.addAll(ASSETS)).then(() => self.skipWaiting()));
});

self.addEventListener('activate',event => {
  event.waitUntil((async () => {
    for (const name of await caches.keys()) {
      if (name.startsWith(PREFIX) && ![SHELL,ARCHIVE].includes(name)) await caches.delete(name);
    }
    await self.clients.claim();
  })());
});

self.addEventListener('fetch',event => {
  if (event.request.method !== 'GET' || !urls.has(event.request.url.split('#')[0])) return;
  event.respondWith((async () => {
    const cache = await caches.open(event.request.url.includes('/data/') ? ARCHIVE : SHELL);
    try {
      const response = await fetch(event.request);
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      if (event.request.url.endsWith('/data/archive.json')) {
        const data = await response.clone().json();
        if (data.schema !== 2 || data.behavior?.version !== 1 || !Array.isArray(data.events)) throw new Error('Invalid archive response; keeping the previous offline copy');
      }
      await cache.put(event.request,response.clone());
      return response;
    } catch (error) {
      const saved = await cache.match(event.request);
      if (saved) return saved;
      return new Response(`No offline copy is available: ${error.message}`,{status:503,headers:{'Content-Type':'text/plain'}});
    }
  })());
});

self.addEventListener('message',event => {
  if (event.data?.type !== 'CACHE_ARCHIVE') return;
  event.waitUntil((async () => {
    try {
      const cache = await caches.open(ARCHIVE);
      const revisionURL = new URL('./data/revision.json',self.registration.scope);
      const archiveURL = new URL('./data/archive.json',self.registration.scope);
      const previous = await cache.match(revisionURL);
      let revision;
      try {
        revision = await fetch(revisionURL,{cache:'no-store'});
        if (!revision.ok) throw new Error(`HTTP ${revision.status}`);
      } catch (error) {
        if (previous && await cache.match(archiveURL)) {event.source?.postMessage({type:'CACHE_READY'}); return;}
        throw error;
      }
      const current = await revision.clone().json();
      const old = previous ? await previous.json() : null;
      if (current.revision !== old?.revision || !await cache.match(archiveURL)) {
        const response = await fetch(archiveURL,{cache:'no-store'});
        if (!response.ok) throw new Error(`Archive returned HTTP ${response.status}`);
        const data = await response.clone().json();
        if (data.schema !== 2 || data.behavior?.version !== 1 || !Array.isArray(data.events)) throw new Error('Archive format is invalid');
        await cache.put(archiveURL,response);
        await cache.put(revisionURL,revision);
      }
      event.source?.postMessage({type:'CACHE_READY'});
    } catch (error) {event.source?.postMessage({type:'CACHE_ERROR',message:error.message});}
  })());
});
