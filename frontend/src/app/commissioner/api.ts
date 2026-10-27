import * as Bindings from '../../../wailsjs/go/commissioner/API';

export * from '../../../wailsjs/go/commissioner/API';
export type CommissionerBindings = typeof Bindings;
export type { commissioner, market } from '../../../wailsjs/go/models';

// The generated module owns invocation; the adapter preserves each screen's no-desktop wording.
function bindings(message: string): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error(message);
  return Bindings;
}
export const ledgerBindings = () => bindings('The league ledger requires the desktop app.');
export const nativeBindings = () => bindings('League changes require the desktop app.');
export const draftBindings = () => bindings('The draft room requires the desktop app.');

export { hostedWorkflow, type HostedResponse } from '../control/nativeApi';
