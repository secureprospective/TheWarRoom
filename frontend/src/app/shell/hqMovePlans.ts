import { poolProvider } from '../pool/provider';
import { releaseAction, onReleaseChange } from '../contracts/releases';
import { contractActAction, onContractActsChange } from '../contracts/contractActState';
import { useContractRead } from '../contracts/Contracts';
import { hqEvents } from './hqReads';
import type { HQMove } from './HQMoves';
import type { RailReceipt } from './EnvelopeRail';

const changes = { onChange: (load: () => void) => {
  const stop = [hqEvents.onChange(load), onReleaseChange(load), onContractActsChange(load)];
  return () => stop.forEach(fn => fn());
} };
export async function readHQMovePlans(mine: string): Promise<{ moves: HQMove[]; errors: string[] }> {
  const result: { moves: HQMove[]; errors: string[] } = { moves: [], errors: [] };
  const move = (id: string, receipt: RailReceipt, players: string[]): HQMove => ({
    id, state: receipt.state, intent: receipt.spec.intent, players,
    at: receipt.audit.at(-1)?.at ?? '', note: receipt.audit.at(-1)?.note ?? '',
  });
  const read = async (load: () => Promise<HQMove[]>) => {
    try { result.moves.push(...await load()); } catch (e) { result.errors.push(String(e)); }
  };
  await Promise.all([
    read(async () => {
      const value = await poolProvider.plans(mine);
      if (value.status !== 'ok') throw new Error(value.absenceReason ?? 'Bids unavailable');
      return value.value.plans.filter(p => p.franchiseId === mine)
        .map(p => move(p.id, p.receipt, [p.playerId]));
    }),
    read(async () => (await releaseAction({ mine, action: 'read' })).value?.map(p =>
      move(p.id, p.receipt, [p.player])) ?? []),
    read(async () => (await contractActAction({ mine, action: 'read' })).value?.flatMap(p =>
      p.receipt ? [move(p.id, p.receipt, [p.player])] : []) ?? []),
  ]);
  return result;
}
export function useHQMovePlans(mine: string, enabled: boolean) {
  return useContractRead(`${mine}:${enabled}`, () => enabled ? readHQMovePlans(mine)
    : Promise.resolve({ moves: [], errors: [] }), changes);
}
export function mergeHQMoves(moves: HQMove[], extra: HQMove[]): HQMove[] {
  const byId = new Map(moves.concat(extra).filter(m => m.state !== 'withdrawn').map(m => [m.id, m]));
  return [...byId.values()].sort((a, b) => Date.parse(b.at) - Date.parse(a.at));
}
