import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import type { MFLKeyStatus } from '../data/contract';

// activity names the request in flight; error is a binding that rejected (the app did not
// start), shown as-is while the last known state keeps driving the controls.
export type MFLKeyState = MFLKeyStatus & { pending: boolean; activity?: string; error?: string };

export function createMFLKeyState() {
  const store = createStore<MFLKeyState>(() => ({
    state: 'absent', league: '', season: 0, pending: false,
  }));
  return {
    read: store.getState,
    use: () => useStore(store),
    // A replacement drops optional details from the previous outcome.
    write: (status: MFLKeyState) => store.setState(status, true),
  };
}
