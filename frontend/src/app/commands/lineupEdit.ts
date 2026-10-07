import type { Provider } from '../data/provider';
import type { Snapshot, LineupReading, Receipt } from '../data/contract';
import type { createShellState } from '../shell/state';
import type { createLineups } from '../shell/lineups';
import { orderReceipts, type createMovesState } from './moves';
import { loadSessionMoves } from './sessionMoves';

export type LineupAction =
  | { kind: 'edit'; franchiseId: string }
  | { kind: 'toggle'; playerId: string }
  | { kind: 'reset' | 'cancel' | 'draft' }
  | { kind: 'handoff'; correlationId: string };

export function editReason(reading?: LineupReading, error?: string): string | undefined {
  if (error) return error;
  if (!reading) return 'Saved lineup unavailable';
  if (reading.provenance.freshness.state === 'fail') return reading.provenance.freshness.note;
  if (reading.week < 1) return 'Lineup week unknown';
  return undefined;
}
export function changedPlayers(starters: string[], baseline: string[]): string[] {
  return [...starters.filter((id) => !baseline.includes(id)),
    ...baseline.filter((id) => !starters.includes(id))];
}
export async function runLineupAction(
  action: LineupAction,
  state: ReturnType<typeof createShellState>,
  moves: ReturnType<typeof createMovesState>,
  lineups: ReturnType<typeof createLineups>,
  provider: Provider,
  snapshot?: Snapshot,
): Promise<void> {
  const save = (receipt: Receipt) => moves.write({ moves: orderReceipts([
    receipt, ...moves.read().moves.filter((r) => r.correlationId !== receipt.correlationId),
  ]) });
  async function persistPlan(key: string, operation: () => Promise<Receipt>) {
    if (moves.read().drafting[key]) return;
    moves.write({
      drafting: { ...moves.read().drafting, [key]: true },
      draftErrors: { ...moves.read().draftErrors, [key]: '' },
    });
    try {
      const receipt = await operation();
      save(receipt);
      return receipt;
    } catch (cause) {
      moves.write({ draftErrors: { ...moves.read().draftErrors, [key]: String(cause) } });
      throw cause;
    } finally {
      moves.write({ drafting: { ...moves.read().drafting, [key]: false } });
    }
  }
  if (action.kind === 'handoff') {
    const receipt = moves.read().moves.find((r) => r.correlationId === action.correlationId);
    if (receipt?.state !== 'ready') return;
    await persistPlan(`handoff:${action.correlationId}`, () => provider.handOff(action.correlationId));
    return;
  }
  if (action.kind === 'edit') {
    const { reading, error } = lineups.peek(action.franchiseId);
    if (editReason(reading, error) || !reading || reading.franchise !== action.franchiseId ||
      state.read().franchiseId !== action.franchiseId) return;
    state.write({ lineupDraft: {
      franchiseId: action.franchiseId, week: reading.week, starters: reading.starters.map((p) => p.id),
    }, lineupReceiptId: null });
    moves.write({ draftErrors: { ...moves.read().draftErrors, [`lineup:${action.franchiseId}`]: '' } });
    return;
  }
  const draft = state.read().lineupDraft;
  if (!draft) return;
  if (action.kind === 'cancel') {
    const receipt = moves.read().moves.find((r) => r.correlationId === state.read().lineupReceiptId);
    const handed = receipt?.audit.some((entry) => entry.to === 'handed_off');
    state.write({ lineupDraft: null, lineupReceiptId: handed ? state.read().lineupReceiptId : null });
    return;
  }
  if (action.kind === 'toggle') {
    const active = snapshot?.rosters.value.find((r) => r.franchiseId === draft.franchiseId)?.players;
    const starting = draft.starters.includes(action.playerId);
    if (!starting && !active?.some((p) => p.id === action.playerId && p.rosterStatus === 'ROSTER')) return;
    const starters = starting
      ? draft.starters.filter((id) => id !== action.playerId) : [...draft.starters, action.playerId];
    state.write({ lineupDraft: { ...draft, starters } });
    return;
  }
  const { reading, error } = lineups.peek(draft.franchiseId);
  if (editReason(reading, error) || !reading || reading.week !== draft.week) return;
  const baseline = reading.starters.map((p) => p.id);
  if (action.kind === 'reset') {
    state.write({ lineupDraft: { ...draft, starters: baseline } });
    return;
  }
  if (!changedPlayers(draft.starters, baseline).length) return;
  const receipt = await persistPlan(`lineup:${draft.franchiseId}`, () =>
    provider.draftLineup(draft.franchiseId, [...draft.starters]));
  if (!receipt) return;
  if (state.read().lineupDraft === draft) state.write({ lineupReceiptId: receipt.correlationId });
  await loadSessionMoves(moves, provider, draft.franchiseId);
}
