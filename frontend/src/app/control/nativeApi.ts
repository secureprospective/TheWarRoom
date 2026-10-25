// Native-only desktop transport. Go owns HTTPS and all bearer/recent-auth secrets.
export type NativeAction = 'status' | 'recover' | 'enroll' | 'login' | 'enter' | 'forget'
  | 'revoke-device' | 'revoke-all' | 'totp-begin' | 'totp-confirm' | 'totp-step-up';
export type HelperAction = 'helper-issue' | 'helper-revoke' | 'helper-confirm';
export type NativeSeat = { franchise: string; name: string };
export type NativeView = {
  server: string; league: string; message: string; name: string; role: string;
  franchise: string; franchiseName: string; seed: string;
  // A helper's connection line is shown once; a confirmation code covers one change.
  connection: string; helperId: string; confirmation: string; seats: NativeSeat[];
};
const textKeys = ['server', 'league', 'message', 'name', 'role', 'franchise', 'franchiseName', 'seed',
  'connection', 'helperId', 'confirmation'];
const isSeat = (seat: unknown) => Boolean(seat) && typeof seat === 'object'
  && Object.keys(seat as object).length === 2
  && typeof (seat as NativeSeat).franchise === 'string' && typeof (seat as NativeSeat).name === 'string';
export function parseNativeView(raw: unknown): NativeView {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Unreadable desktop response.');
  const value = raw as Record<string, unknown>;
  if (Object.keys(value).length !== textKeys.length + 1
    || textKeys.some(key => typeof value[key] !== 'string')
    || !Array.isArray(value.seats) || !value.seats.every(isSeat)
    || !String(value.server).startsWith('https://') || !String(value.league).startsWith('native.'))
    throw new Error('Unreadable desktop response.');
  return value as NativeView;
}
export async function nativeStartup(): Promise<NativeView | null> {
  const response = await fetch('/desktop/native', { cache: 'no-store' });
  if (!response.ok) return null;
  // Vite's local development fallback serves index.html for unknown routes.
  if (response.headers.get('Content-Type')?.startsWith('text/html')) return null;
  return parseNativeView(await response.json());
}
async function post(body: Record<string, string>): Promise<NativeView> {
  const response = await fetch('/desktop/native', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, cache: 'no-store',
    body: JSON.stringify(body),
  });
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
// readConfirmRequest shows the person what they are about to confirm. Go checks it again.
export function readConfirmRequest(text: string): { command: string; reason: string; args: string } | null {
  const trimmed = text.trim();
  if (!trimmed.startsWith('twrc1.')) return null;
  try {
    const json = atob(trimmed.slice(6).replace(/-/g, '+').replace(/_/g, '/'));
    type Change = { command?: unknown; reason?: unknown; args?: unknown };
    const change = (JSON.parse(json) as { change?: Change }).change;
    if (!change || typeof change.command !== 'string') return null;
    return { command: change.command, reason: typeof change.reason === 'string' ? change.reason : '',
      args: JSON.stringify(change.args ?? {}) };
  } catch { return null; }
}
