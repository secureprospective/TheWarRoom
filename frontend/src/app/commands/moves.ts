import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import type { EnvelopeDemo, Receipt } from '../data/contract';

export type MovesState = {
  moves: Receipt[];
  drafting: Record<string, boolean>;
  draftErrors: Record<string, string>;
  movesLoading: Record<string, boolean>;
  movesErrors: Record<string, string>;
  demo?: EnvelopeDemo;
};

export function orderReceipts(receipts: Receipt[]): Receipt[] {
  const lastTime = (receipt: Receipt) => {
    const at = receipt.audit.at(-1)?.at;
    return at ? Date.parse(at) : -Infinity;
  };
  return [...receipts].sort((a, b) => lastTime(b) - lastTime(a));
}

export function createMovesState() {
  const store = createStore<MovesState>(() => ({
    moves: [], drafting: {}, draftErrors: {}, movesLoading: {}, movesErrors: {},
  }));
  return {
    read: store.getState,
    use: <T>(selector: (state: MovesState) => T) => useStore(store, selector),
    // Only the registry receives the write capability.
    write: (patch: Partial<MovesState>) => store.setState(patch),
  };
}
