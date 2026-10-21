import { parseComms, type Comms } from './contract';
import type * as Bindings from '../../../wailsjs/go/comms/API';

export type CommsReader = {
  read(team: string, kind: string): Promise<Comms>;
  refresh(force: boolean): Promise<void>;
  open(destination: string): Promise<void>;
};

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Comms requires the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { comms: { API: typeof Bindings } } }).go.comms.API;
}

export const commsAPI: CommsReader = {
  read: async (team, kind) => parseComms(await bindings().TargetComms({ team, kind })),
  refresh: async force => bindings().TargetCommsRefresh(force),
  open: async destination => bindings().TargetOpenComms(destination),
};

export function subscribeComms(changed: () => void): () => void {
  let disposed = false;
  let stop: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(m => {
    if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
      stop = m.EventsOn('target:season', changed);
  });
  return () => { disposed = true; stop?.(); };
}
