// Native-only desktop transport. Go owns HTTPS and all bearer/recent-auth secrets.
import { observeView, prepareHostedAction } from '../data/hostedReads';
import type { NativeRequest } from '../commands/native';
import { object, text } from '../data/parseValues';
export type NativeAction = 'update' | 'retry' | 'status' | 'recover' | 'enroll' | 'login' | 'enter' | 'forget'
  | 'revoke-device' | 'revoke-all' | 'totp-begin' | 'totp-confirm' | 'totp-step-up' | 'recovery-codes';
export type HelperAction = 'helper-issue' | 'helper-revoke' | 'helper-check' | 'helper-confirm';
export type NativeSeat = { franchise: string; name: string };
export type NativeView = {
  helperPreview?: string;
  update?: 'app' | 'server';
  authGeneration: string; server: string; league: string; message: string; name: string; role: string;
  franchise: string; franchiseName: string; seed: string;
  // A helper's connection line is shown once; a confirmation code covers one change.
  connection: string; helperId: string; confirmation: string; seats: NativeSeat[];
  // New recovery codes are shown once, right after they are printed.
  recoveryCodes?: string[];
};
const updateListeners = new Set<(view: NativeView) => void>();
export function subscribeNativeUpdate(listener: (view: NativeView) => void): () => void {
  updateListeners.add(listener);
  return () => { updateListeners.delete(listener); };
}
const textKeys = ['authGeneration', 'server', 'league', 'message', 'name', 'role',
  'franchise', 'franchiseName', 'seed',
  'connection', 'helperId', 'confirmation'];
const isSeat = (seat: unknown) => Boolean(seat) && typeof seat === 'object'
  && Object.keys(seat as object).length === 2
  && typeof (seat as NativeSeat).franchise === 'string' && typeof (seat as NativeSeat).name === 'string';
export function parseNativeView(raw: unknown): NativeView {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Unreadable desktop response.');
  const value = raw as Record<string, unknown>;
  if (Object.keys(value).length !== textKeys.length + 1 + ('helperPreview' in value ? 1 : 0)
      + ('update' in value ? 1 : 0) + ('recoveryCodes' in value ? 1 : 0)
    || ('update' in value && value.update !== 'app' && value.update !== 'server')
    || ('helperPreview' in value && typeof value.helperPreview !== 'string')
    || ('recoveryCodes' in value && (!Array.isArray(value.recoveryCodes) || value.recoveryCodes.length !== 8
      || !value.recoveryCodes.every(c => typeof c === 'string' && c.length === 43)))
    || textKeys.some(key => typeof value[key] !== 'string')
    || !/^\d+$/.test(String(value.authGeneration))
    || !Array.isArray(value.seats) || !value.seats.every(isSeat)
    || !String(value.server).startsWith('https://') || !String(value.league).startsWith('native.'))
    throw new Error('Unreadable desktop response.');
  return value as NativeView;
}
export async function nativeStartup(): Promise<NativeView | null> {
  const response = await fetch('/desktop/native', { cache: 'no-store' });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error('Desktop startup unavailable.');
  // Vite's local development fallback serves index.html for unknown routes.
  if (response.headers.get('Content-Type')?.startsWith('text/html')) return null;
  const view = parseNativeView(await response.json());
  observeView(view);
  return view;
}
async function post(body: Record<string, string>): Promise<NativeView> {
  prepareHostedAction(body.action);
  const response = await fetch('/desktop/native', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, cache: 'no-store',
    body: JSON.stringify(body),
  });
  if (response.status === 426) {
    const view = parseNativeView(await response.json());
    observeView(view);
    if (!view.update) throw new Error('The update request could not be read. Close and reopen the app.');
    updateListeners.forEach(listener => listener(view));
    return view;
  }
  if (!response.ok) throw new Error(await response.text());
  const view = parseNativeView(await response.json());
  observeView(view);
  return view;
}
export function nativeAction(
  action: NativeAction, username = '', password = '', code = '',
): Promise<NativeView> {
  return post({ action, username, password, code });
}
export function helperAction(
  action: HelperAction, fields: { helper?: string; franchise?: string; code?: string; change?: string },
): Promise<NativeView> {
  return post({ action, ...fields });
}
// Only recognize the envelope here; Go validates the change and the league describes it.
export function readConfirmRequest(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed.startsWith('twrc1.')) return false;
  try {
    const json = atob(trimmed.slice(6).replace(/-/g, '+').replace(/_/g, '/'));
    const request = JSON.parse(json) as { tokenId?: unknown; change?: { command?: unknown } };
    return typeof request.tokenId === 'string' && typeof request.change?.command === 'string';
  } catch { return false; }
}
export type HostedResponse = { label: string; writable: boolean; football: string; data: unknown };
export async function hostedWorkflow(request: NativeRequest, football = ''): Promise<HostedResponse> {
  const { nativeEnvelope } = await import('../commands/native');
  const response = await fetch('/desktop/workflow', {
    method: 'POST', cache: 'no-store', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...nativeEnvelope(request), football }),
  });
  if (!response.ok) {
    throw new Error('The league could not accept this action. '
      + 'Refresh before retrying; check your connection, seat and authenticator.');
  }
  const r = object(await response.json(), 'Hosted league', ['label', 'writable', 'football', 'data']);
  if (typeof r.writable !== 'boolean') throw new Error('Unreadable league authority. Refresh to try again.');
  return {
    label: text(r.label, 'League location'), writable: r.writable,
    football: text(r.football, 'Football evidence'), data: r.data,
  };
}

export type ChangeTopic = {
  kind: 'league' | 'team' | 'scores' | 'values' | 'clock'; version: string;
  franchise?: string; season?: number; week?: number; run?: string; nextBoundary?: string;
};
export type NativeChanges = {
  authGeneration: string; seq: number; topics: ChangeTopic[];
  state: 'connected' | 'reconnecting' | 'signed-out' | 'refused';
  serverEpoch?: string; league?: string;
};
function parseChanges(raw: unknown): NativeChanges {
  const r = object(raw, 'Desktop changes',
    ['authGeneration', 'seq', 'topics', 'state', 'serverEpoch', 'league']);
  if (typeof r.authGeneration !== 'string' || !/^\d+$/.test(r.authGeneration)
    || !Number.isSafeInteger(r.seq) || Number(r.seq) < 0 || !Array.isArray(r.topics)
    || !['connected', 'reconnecting', 'signed-out', 'refused'].includes(String(r.state)))
    throw new Error('Unreadable desktop changes.');
  for (const topic of r.topics) {
    const t = object(topic, 'Change topic',
      ['kind', 'version', 'franchise', 'season', 'week', 'run', 'nextBoundary']);
    if (!['league', 'team', 'scores', 'values', 'clock'].includes(String(t.kind))
      || typeof t.version !== 'string' || !/^\d+$/.test(t.version))
      throw new Error('Unreadable change topic.');
    for (const key of ['franchise', 'run', 'nextBoundary']) {
      if (key in t && typeof t[key] !== 'string') throw new Error('Unreadable topic scope.');
    }
    for (const key of ['season', 'week']) {
      if (key in t && (!Number.isSafeInteger(t[key]) || Number(t[key]) < 1))
        throw new Error('Unreadable topic scope.');
    }
    if ((t.kind === 'team' && !t.franchise)
      || (t.kind === 'scores' && (!t.season || !t.week))
      || (t.kind === 'values' && (!t.season || !t.run))) throw new Error('Missing topic scope.');
  }
  for (const key of ['serverEpoch', 'league']) {
    if (key in r && typeof r[key] !== 'string') throw new Error('Unreadable change identity.');
  }
  return r as NativeChanges;
}

// The read owner subscribes once, invalidates first, then notifies its screen listeners.
export function subscribeChanged(listener: (change: NativeChanges) => void): () => void {
  const controller = new AbortController();
  let timer: ReturnType<typeof setTimeout> | undefined;
  let after = 0;
  let backoff = 250;
  async function poll(): Promise<void> {
    try {
      const response = await fetch('/desktop/changes', {
        method: 'POST', cache: 'no-store', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ after }), signal: controller.signal,
      });
      if (!response.ok) throw new Error('Desktop changes unavailable.');
      const change = parseChanges(await response.json());
      if (controller.signal.aborted) return;
      if (change.seq > after) { after = change.seq; listener(change); }
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
}
