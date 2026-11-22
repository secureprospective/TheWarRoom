import type { NativeRequest } from '../commands/native';

export type RawChangeListener = (value: unknown) => void;
export interface NativeTransport {
  readonly kind: 'wails' | 'web';
  startup(): Promise<Response>;
  action(body: Record<string, string>): Promise<Response>;
  helper(body: Record<string, string>): Promise<Response>;
  workflow(request: NativeRequest, football: string): Promise<Response>;
  subscribeChanged(listener: RawChangeListener): () => void;
  setIdentity(league: string, authGeneration: string): void;
}

const json = (body: unknown): RequestInit => ({
  method: 'POST', cache: 'no-store', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
});

export const wailsTransport: NativeTransport = {
  kind: 'wails',
  startup: () => fetch('/desktop/native', { cache: 'no-store' }),
  action: body => fetch('/desktop/native', json(body)),
  helper: body => fetch('/desktop/native', json(body)),
  workflow: async (request, football) => {
    const { nativeEnvelope } = await import('../commands/native');
    return fetch('/desktop/workflow', json({ ...nativeEnvelope(request), football }));
  },
  setIdentity: () => {},
  subscribeChanged(listener) {
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    let after = 0;
    let backoff = 250;
    async function poll(): Promise<void> {
      try {
        const response = await fetch('/desktop/changes', {
          ...json({ after }), signal: controller.signal,
        });
        if (!response.ok) throw new Error('Desktop changes unavailable.');
        const change: unknown = await response.json();
        if (controller.signal.aborted) return;
        const row = change as { seq?: unknown };
        if (typeof row.seq === 'number' && row.seq > after) {
          after = row.seq;
          listener(change);
        }
        backoff = 250;
        timer = setTimeout(() => { void poll(); }, 0);
      } catch {
        if (controller.signal.aborted) return;
        timer = setTimeout(() => { void poll(); }, backoff);
        backoff = Math.min(backoff * 2, 30_000);
      }
    }
    void poll();
    return () => { controller.abort(); clearTimeout(timer); };
  },
};

function reportOffline(): void {
  if (typeof window !== 'undefined') window.dispatchEvent(new Event('twr:offline'));
}
async function request(path: string, init: RequestInit = {}): Promise<Response> {
  try {
    return await fetch(path, { ...init, cache: 'no-store', credentials: 'same-origin' });
  } catch (error) {
    reportOffline();
    throw error;
  }
}
function post(path: string, body: unknown): Promise<Response> {
  return request(path, { ...json(body), credentials: 'same-origin' });
}
function sessionRequest(): Promise<Response> {
  return request('/app/session', { method: 'GET' });
}

export const webTransport: NativeTransport = {
  kind: 'web',
  startup: sessionRequest,
  async action(body) {
    switch (body.action) {
      case 'login':
        return post('/app/session', {
          protocol: 1, username: body.username, password: body.password,
          ...(body.code ? { code: body.code } : {}),
        });
      case 'status':
      case 'enter':
        return sessionRequest();
      case 'forget':
        return request('/app/session', { method: 'DELETE' });
      default:
        throw new Error('This account feature is not available in the browser app.');
    }
  },
  async helper() {
    throw new Error('AI helper management is not available in the browser app.');
  },
  workflow: async (requestBody, football) => {
    const { isAccountPreferenceWorkflow, nativeEnvelope } = await import('../commands/native');
    const path = isAccountPreferenceWorkflow(requestBody.command)
      ? '/v1/account/preferences' : '/v1/native/workflow';
    return post(path, { protocol: 1, workflow: { ...nativeEnvelope(requestBody), football } });
  },
  setIdentity: () => {},
  subscribeChanged() {
    // The server's invalidation stream is not admitted for browser-cookie sessions.
    return () => {};
  },
};

export function selectNativeTransport(wails: boolean): NativeTransport {
  return wails ? wailsTransport : webTransport;
}

export const nativeTransport = selectNativeTransport(
  typeof window === 'undefined' || (window as { go?: unknown }).go !== undefined,
);

export function isWebTransport(): boolean {
  return nativeTransport.kind === 'web';
}
