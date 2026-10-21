import { counterparties } from '../../../wailsjs/go/models';
import type * as Bindings from '../../../wailsjs/go/counterparties/API';
import { parseCounterparties, parseTeamPanel } from './provider';
import type { CounterpartiesReading, CounterScope, TeamPanelReading } from './contract';

function bindings(): typeof Bindings {
  return (window as unknown as { go: { counterparties: { API: typeof Bindings } } })
    .go.counterparties.API;
}

function unavailable(reason: string, status: 'failed' | 'not_wired'): CounterpartiesReading {
  return {
    status,
    absenceReason: reason,
    count: 0,
    threshold: 40,
    baseline: '',
    cutoff: '',
    method: '',
    inputSHA: '',
    buildStamp: '',
    howCounted: '',
  };
}
function teamUnavailable(reason: string, status: 'failed' | 'not_wired'): TeamPanelReading {
  return {
    status,
    absenceReason: reason,
    count: 0,
    threshold: 1,
    baseline: '',
    cutoff: '',
    method: '',
    inputSHA: '',
    buildStamp: '',
    howCounted: '',
  };
}
async function read<T>(
  call: () => Promise<T>,
  fail: (reason: string, status: 'failed' | 'not_wired') => T,
): Promise<T> {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go) {
    return fail(
      'Counterparties needs the desktop app; no browser fixtures are substituted',
      'not_wired',
    );
  }
  try {
    return await call();
  } catch (cause) {
    return fail(
      `Counterparties read failed: ${String(cause)}. Retry the held archive reading.`,
      'failed',
    );
  }
}
export const counterpartProvider = {
  surface(scope: CounterScope) {
    return read(
      async () => parseCounterparties(await bindings().TargetCounterparties(scope)),
      unavailable,
    );
  },
  team(scope: CounterScope, fid: string) {
    return read(
      async () => parseTeamPanel(await bindings().TargetCounterpartTeam(
        new counterparties.RawCounterpartTeam({ scope, fid }))),
      teamUnavailable,
    );
  },
};
