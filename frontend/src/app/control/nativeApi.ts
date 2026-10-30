// Native-only desktop transport. Go owns HTTPS and all bearer/recent-auth secrets.
import type { NativeRequest } from '../commands/native';
import { object, text } from '../data/parseValues';
export type NativeAction = 'update' | 'retry' | 'status' | 'recover' | 'enroll' | 'login' | 'enter' | 'forget'
  | 'revoke-device' | 'revoke-all' | 'totp-begin' | 'totp-confirm' | 'totp-step-up' | 'recovery-codes';
export type HelperAction = 'helper-issue' | 'helper-revoke' | 'helper-check' | 'helper-confirm';
export type NativeSeat = { franchise: string; name: string };
export type NativeView = {
  helperPreview?: string;
  update?: 'app' | 'server';
  server: string; league: string; message: string; name: string; role: string;
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
const textKeys = ['server', 'league', 'message', 'name', 'role', 'franchise', 'franchiseName', 'seed',
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
  return parseNativeView(await response.json());
}
async function post(body: Record<string, string>): Promise<NativeView> {
  const response = await fetch('/desktop/native', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, cache: 'no-store',
    body: JSON.stringify(body),
  });
  if (response.status === 426) {
    const view = parseNativeView(await response.json());
    if (!view.update) throw new Error('The update request could not be read. Close and reopen the app.');
    updateListeners.forEach(listener => listener(view));
    return view;
  }
  if (!response.ok) throw new Error(await response.text());
  return parseNativeView(await response.json());
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
