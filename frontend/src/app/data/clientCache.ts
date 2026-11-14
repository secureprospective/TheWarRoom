import type { PublicStorage } from './session';

type RecordValue = { key: string; value: string; bytes: number; used: number; schema: number };
function validRow(value: unknown): value is RecordValue {
  if (!value || typeof value !== 'object') return false;
  const row = value as RecordValue;
  return row.schema === 1 && typeof row.key === 'string' && typeof row.value === 'string'
    && Number.isFinite(row.used) && row.bytes === new TextEncoder().encode(row.key + row.value).byteLength;
}

// This adapter stores only serialized public entries selected by the read session, never requests.
// Each operation closes its connection; an unavailable/blocked database is an optional cache miss.
export function createPublicReadStorage(options: {
  indexedDB?: IDBFactory; name?: string; maxEntries?: number; maxBytes?: number;
} = {}): PublicStorage {
  const factory = options.indexedDB ?? globalThis.indexedDB;
  const name = options.name ?? 'twr-public-reads';
  let lastUsed = 0;
  const touch = (previous = 0) => {
    lastUsed = Math.max(Date.now(), lastUsed + 1, previous + 1);
    return lastUsed;
  };
  const maxEntries = options.maxEntries ?? 32;
  const maxBytes = options.maxBytes ?? 4 * 1024 * 1024;
  async function transaction<T>(fallback: T,
    work: (store: IDBObjectStore, finish: (result: T) => void) => void): Promise<T> {
    if (!factory) return fallback;
    return new Promise(resolve => {
      let db: IDBDatabase | undefined;
      let result = fallback;
      let finished = false;
      const finish = () => {
        if (finished) return;
        finished = true;
        db?.close();
        resolve(result);
      };
      const timer = setTimeout(finish, 1000);
      const done = () => { clearTimeout(timer); finish(); };
      try {
        const request = factory.open(name, 1);
        request.onupgradeneeded = () => { request.result.createObjectStore('reads', { keyPath: 'key' }); };
        request.onerror = done;
        request.onblocked = done;
        request.onsuccess = () => {
          db = request.result;
          if (finished) { db.close(); return; }
          try {
            const tx = db.transaction('reads', 'readwrite');
            tx.oncomplete = done;
            tx.onerror = () => { result = fallback; done(); };
            tx.onabort = () => { result = fallback; done(); };
            work(tx.objectStore('reads'), value => { result = value; });
          } catch { done(); }
        };
      } catch { done(); }
    });
  }
  return {
    get: key => transaction<string | undefined>(undefined, (store, finish) => {
      const request = store.get(key);
      request.onsuccess = () => {
        try {
          const row: unknown = request.result;
          if (!validRow(row) || row.key !== key) { store.delete(key); return; }
          store.put({ ...row, used: touch(row.used) });
          finish(row.value);
        } catch { /* Quota/security errors are an optional cache miss. */ }
      };
    }),
    put: (key, value) => transaction(undefined, store => {
      const bytes = new TextEncoder().encode(key + value).byteLength;
      if (bytes > maxBytes || maxEntries < 1) return;
      const request = store.getAll();
      request.onsuccess = () => {
        try {
          const rows = (request.result as { key: string }[]).filter(row => row.key !== key);
          let total = bytes;
          const valid: RecordValue[] = [];
          for (const row of rows) {
            if (validRow(row)) valid.push(row);
            else store.delete(row.key);
          }
          valid.sort((a, b) => b.used - a.used);
          let count = 1;
          for (const row of valid) {
            if (count >= maxEntries || total + row.bytes > maxBytes) store.delete(row.key);
            else { count++; total += row.bytes; }
          }
          const used = touch(Math.max(0, ...valid.map(row => row.used)));
          store.put({ key, value, bytes, used, schema: 1 });
        } catch { /* Full storage does not fail the online reading. */ }
      };
    }),
    remove: key => transaction(undefined, store => { store.delete(key); }),
  };
}
