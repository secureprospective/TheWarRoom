import type { Snapshot } from '../data/contract';

export const FRANCHISE_KEY = 'thewarroom.target.my-franchise';
export type SettingsStorage = Pick<Storage, 'getItem' | 'setItem'>;

export function validFranchise(snapshot: Snapshot, id: string | null): string | null {
  return snapshot.franchises.value.some((f) => f.id === id) ? id : null;
}

export function loadFranchise(
  snapshot: Snapshot,
  storage?: SettingsStorage,
): string | null {
  return validFranchise(snapshot, storage?.getItem(FRANCHISE_KEY) ?? null);
}

export function persistFranchise(id: string, storage?: SettingsStorage) {
  storage?.setItem(FRANCHISE_KEY, id);
}

export function browserStorage(): SettingsStorage | undefined {
  return typeof window === 'undefined' ? undefined : window.localStorage;
}
