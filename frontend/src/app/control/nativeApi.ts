// Native-only desktop transport. Go owns HTTPS and all bearer/recent-auth secrets.
export type NativeAction = 'status' | 'recover' | 'enroll' | 'login' | 'enter' | 'forget'
  | 'revoke-device' | 'revoke-all' | 'totp-begin' | 'totp-confirm' | 'totp-step-up';
export type NativeView = {
  server: string; league: string; message: string; name: string; role: string;
  franchise: string; franchiseName: string; seed: string;
};
export function parseNativeView(raw: unknown): NativeView {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Unreadable desktop response.');
  const value = raw as Record<string, unknown>;
  const keys = ['server', 'league', 'message', 'name', 'role', 'franchise', 'franchiseName', 'seed'];
  if (Object.keys(value).length !== keys.length || keys.some(key => typeof value[key] !== 'string')
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
export async function nativeAction(
  action: NativeAction, username = '', password = '', code = '',
): Promise<NativeView> {
  const response = await fetch('/desktop/native', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, cache: 'no-store',
    body: JSON.stringify({ action, username, password, code }),
  });
  if (!response.ok) throw new Error(await response.text());
  return parseNativeView(await response.json());
}
