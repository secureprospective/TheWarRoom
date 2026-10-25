import * as Bindings from '../../../wailsjs/go/commissioner/API';

export * from '../../../wailsjs/go/commissioner/API';
export type CommissionerBindings = typeof Bindings;
export type { commissioner } from '../../../wailsjs/go/models';

// The generated module owns invocation; the adapter preserves each screen's no-desktop wording.
function bindings(message: string): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error(message);
  return Bindings;
}
export const ledgerBindings = () => bindings('The league ledger requires the desktop app.');
export const nativeBindings = () => bindings('League changes require the desktop app.');
export const draftBindings = () => bindings('The draft room requires the desktop app.');

import type { NativeRequest } from '../commands/native';
import { object, text } from '../data/parseValues';

export type HostedResponse = { label: string; writable: boolean; football: string; data: unknown };
export async function hostedWorkflow(request: NativeRequest, football = ''): Promise<HostedResponse> {
  const { nativeEnvelope } = await import('../commands/native');
  const response = await fetch('/desktop/workflow', { method: 'POST', cache: 'no-store',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...nativeEnvelope(request), football }) });
  if (!response.ok) throw new Error('The league could not accept this action. Refresh before retrying; check your connection, seat and authenticator.');
  const r = object(await response.json(), 'Hosted league', ['label', 'writable', 'football', 'data']);
  if (typeof r.writable !== 'boolean') throw new Error('Unreadable league authority. Refresh to try again.');
  return { label: text(r.label, 'League location'), writable: r.writable,
    football: text(r.football, 'Football evidence'), data: r.data };
}
