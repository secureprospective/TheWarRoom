import type * as Bindings from '../../../wailsjs/go/contracts/API';
import type { contracts, snapshot } from '../../../wailsjs/go/models';

// The only file that touches the contracts bindings; callers parse the unknown wire values.
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Contracts and cap need the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { contracts: { API: typeof Bindings } } }).go.contracts.API;
}

export async function targetContracts(scope: unknown): Promise<unknown> {
  return bindings().TargetContracts(scope as snapshot.RawContractScope);
}
export async function targetPlayerContract(id: string): Promise<unknown> {
  return bindings().TargetPlayerContract(id);
}
export async function targetContractActs(raw: unknown): Promise<unknown> {
  return bindings().TargetContractActs(raw as contracts.RawContractAct);
}
export async function targetReleasePlans(raw: unknown): Promise<unknown> {
  return bindings().TargetReleasePlans(raw as contracts.RawReleaseRequest);
}
export async function targetWhatIf(raw: unknown): Promise<unknown> {
  return bindings().TargetWhatIf(raw as contracts.RawWhatIfRequest);
}
export async function targetLineup(franchiseId: string): Promise<unknown> {
  return bindings().TargetLineup(franchiseId);
}
export async function targetCheckLineup(franchiseId: string, starters: string[]): Promise<unknown> {
  return bindings().TargetCheckLineup(franchiseId, starters);
}
export async function targetDraftLineup(franchiseId: string, starters: string[]): Promise<unknown> {
  return bindings().TargetDraftLineup(franchiseId, starters);
}
export async function targetRosterReports(mine: string): Promise<unknown> {
  return bindings().TargetRosterReports(mine);
}
