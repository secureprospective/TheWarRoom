import type { Snapshot, RosterPlayer } from '../data/contract';
import { playerCardModel, provenanceSlot, inheritedProvenance } from '../cards/playerModel';

export function franchiseRoster(snapshot: Snapshot, franchiseId: string) {
  const franchise = snapshot.franchises.value.find((f) => f.id === franchiseId);
  if (!franchise) throw new Error(`Roster: unknown franchise ${franchiseId}`);
  const roster =
    snapshot.rosters.value.find((r) => r.franchiseId === franchiseId)?.players ?? [];
  const names = new Map(snapshot.players.value.map((p) => [p.id, p.name ?? p.id]));
  const groups = (
    [
      ['ROSTER', 'Active'],
      ['IR', 'IR'],
      ['TAXI_SQUAD', 'Taxi'],
    ] as const
  ).map(([status, label]) => ({
    label,
    players: roster
      .filter((p) => p.rosterStatus === status)
      .sort(
        (a: RosterPlayer, b: RosterPlayer) =>
          b.salary - a.salary ||
          (names.get(a.id) ?? a.id).localeCompare(names.get(b.id) ?? b.id),
      ),
  }));
  const asOf = new Date();
  const provenance = provenanceSlot([
    ['Players', snapshot.players.provenance], ['Contracts', snapshot.rosters.provenance],
  ]);
  return {
    provenance,
    asOf,
    groups: groups.map((group) => ({
      ...group,
      cards: group.players.map((player) => {
        const model = playerCardModel(snapshot, franchiseId, player.id, asOf);
        return { model, provenance: inheritedProvenance(model.provenance, provenance) };
      }),
    })),
    capUsed: franchise.capUsed,
    capRoom: franchise.capRoom,
    salaryCap: snapshot.league.value.salaryCap,
  };
}
