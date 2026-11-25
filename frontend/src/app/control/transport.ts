import type { NativeRequest } from '../commands/native';

export type RawChangeListener = (value: unknown) => void;
export interface NativeTransport {
  startup(): Promise<Response>;
  action(body: Record<string, string>): Promise<Response>;
  workflow(request: NativeRequest, football: string): Promise<Response>;
  subscribeChanged(listener: RawChangeListener): () => void;
  setIdentity(league: string): void;
}

const json = (body: unknown): RequestInit => ({
  method: 'POST', cache: 'no-store', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
});

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

export const nativeTransport: NativeTransport = {
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
  workflow: async (requestBody, football) => {
    const { isAccountPreferenceWorkflow, nativeEnvelope } = await import('../commands/native');
    const path = isAccountPreferenceWorkflow(requestBody.command)
      ? '/v1/account/preferences' : '/v1/native/workflow';
    return post(path, { protocol: 1, workflow: { ...nativeEnvelope(requestBody), football } });
  },
  setIdentity(league) {
    if (league === streamLeague) return;
    streamLeague = league;
    restartStream?.();
  },
  subscribeChanged(listener) {
    // The same server stream desktop uses, opened with the session cookie. fetch (not EventSource)
    // because the server reads the league and protocol from headers.
    const controller = new AbortController();
    let attempt: AbortController | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let backoff = 500;
    const watchdog = 45_000; // the server's keepalive is 15 s
    async function open(): Promise<void> {
      if (!streamLeague) { timer = setTimeout(() => { void open(); }, 500); return; }
      const mine = new AbortController();
      attempt = mine;
      const stalled = () => { clearTimeout(idle); idle = setTimeout(() => mine.abort(), watchdog); };
      let idle: ReturnType<typeof setTimeout> | undefined;
      try {
        const response = await fetch('/v1/native/invalidations', {
          cache: 'no-store', credentials: 'same-origin', signal: mine.signal,
          headers: { 'X-TWR-League': streamLeague, 'X-TWR-Invalidation': '2' },
        });
        if (!response.ok || !response.body) throw new Error('Change stream unavailable.');
        backoff = 500;
        stalled();
        await readEvents(response.body, listener, stalled);
      } catch { /* reconnect below unless closed */ }
      clearTimeout(idle);
      if (controller.signal.aborted || attempt !== mine) return;
      timer = setTimeout(() => { void open(); }, backoff);
      backoff = Math.min(backoff * 2, 30_000);
    }
    restartStream = () => { attempt?.abort(); clearTimeout(timer); backoff = 500; void open(); };
    void open();
    return () => {
      controller.abort(); attempt?.abort(); clearTimeout(timer); restartStream = undefined;
    };
  },
};

let streamLeague = '';
let restartStream: (() => void) | undefined;

// readEvents feeds each "invalidate" event's JSON data to the listener until the stream ends.
async function readEvents(
  body: ReadableStream<Uint8Array>, listener: RawChangeListener, alive: () => void,
): Promise<void> {
  const reader = body.pipeThrough(new TextDecoderStream()).getReader();
  let buffer = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) return;
    alive();
    buffer += value;
    if (buffer.length > 16_384) throw new Error('Change stream frame too large.');
    let end = buffer.indexOf('\n\n');
    while (end >= 0) {
      const frame = buffer.slice(0, end);
      buffer = buffer.slice(end + 2);
      const head = 'event: invalidate\ndata: ';
      if (frame.startsWith(head)) {
        try { listener(JSON.parse(frame.slice(head.length))); } catch { /* hint only */ }
      }
      end = buffer.indexOf('\n\n');
    }
  }
}
