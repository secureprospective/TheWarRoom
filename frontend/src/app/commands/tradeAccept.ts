import type { Provider } from '../data/provider';
import { orderReceipts, type createMovesState } from './moves';
import { loadSessionMoves } from './sessionMoves';

export async function planTradeAccept(
  moves: ReturnType<typeof createMovesState>, provider: Provider, franchiseId: string, tradeId: string,
): Promise<void> {
  const key = `trade:${franchiseId}:${tradeId}`;
  if (moves.read().drafting[key]) return;
  moves.write({
    drafting: { ...moves.read().drafting, [key]: true },
    draftErrors: { ...moves.read().draftErrors, [key]: '' },
  });
  try {
    const receipt = await provider.draftTradeAccept(franchiseId, tradeId);
    moves.write({ moves: orderReceipts([
      receipt, ...moves.read().moves.filter((r) => r.correlationId !== receipt.correlationId),
    ]) });
    await loadSessionMoves(moves, provider, franchiseId);
  } catch (cause) {
    moves.write({ draftErrors: { ...moves.read().draftErrors, [key]: String(cause) } });
  } finally {
    moves.write({ drafting: { ...moves.read().drafting, [key]: false } });
  }
}
