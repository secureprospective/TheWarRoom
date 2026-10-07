import type { DraftingProvider, Provider } from '../data/provider';
import type { PlayerSubject } from '../shell/state';
import { orderReceipts, type createMovesState } from './moves';

export async function loadSessionMoves(
  state: ReturnType<typeof createMovesState>,
  provider: Provider,
  franchiseId: string,
) {
  if (state.read().movesLoading[franchiseId]) return;
  state.write({
    movesLoading: { ...state.read().movesLoading, [franchiseId]: true },
    movesErrors: { ...state.read().movesErrors, [franchiseId]: '' },
  });
  try {
    if (provider.kind === 'fixture') state.write({ demo: await provider.demo() });
    else {
      const loaded = await provider.moves(franchiseId);
      const merged = new Map(loaded.map((receipt) => [receipt.correlationId, receipt]));
      for (const receipt of state.read().moves) merged.set(receipt.correlationId, receipt);
      state.write({ moves: orderReceipts([...merged.values()]) });
    }
  } catch (cause) {
    state.write({ movesErrors: { ...state.read().movesErrors, [franchiseId]: String(cause) } });
  } finally {
    state.write({ movesLoading: { ...state.read().movesLoading, [franchiseId]: false } });
  }
}

export async function finishDraft(
  state: ReturnType<typeof createMovesState>,
  provider: DraftingProvider,
  subject: PlayerSubject,
  key: string,
) {
  try {
    const receipt = await provider.draftIR(subject.franchiseId, subject.id);
    state.write({ moves: orderReceipts([receipt, ...state.read().moves]) });
  } catch (cause) {
    state.write({ draftErrors: { ...state.read().draftErrors, [key]: String(cause) } });
  } finally {
    state.write({ drafting: { ...state.read().drafting, [key]: false } });
  }
}
