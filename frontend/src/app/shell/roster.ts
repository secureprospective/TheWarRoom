import type { Snapshot, RosterPlayer, LineupReading } from '../data/contract';
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
        // A directory gap shows the ID rather than taking down the whole roster.
        const model = playerCardModel(snapshot, franchiseId, player.id, asOf, true);
        return { model, provenance: inheritedProvenance(model.provenance, provenance) };
      }),
    })),
    capUsed: franchise.capUsed,
    capRoom: franchise.capRoom,
    salaryCap: snapshot.league.value.salaryCap,
  };
}

export function lineupGroups(
  snapshot: Snapshot,
  roster: ReturnType<typeof franchiseRoster>,
  reading: LineupReading,
) {
  const existing = new Map(roster.groups.flatMap((g) => g.cards.map((c) => [c.model.id, c] as const)));
  const active = new Set(roster.groups[0].players.map((p) => p.id));
  const split = ([['Starters', reading.starters], ['Bench', reading.bench]] as const).map(
    ([label, players]) => ({
      label,
      players,
      cards: players.map((player) => {
        const card = existing.get(player.id);
        const model = card?.model ?? playerCardModel(
          snapshot, reading.franchise, player.id, roster.asOf, true,
        );
        const missing = label === 'Starters' && !active.has(player.id);
        return {
          model: missing ? {
            ...model,
            status: [...model.status, { signal: 'amber' as const, label: 'not on the active roster' }],
            rosterNote: 'not on the active roster',
          } : model,
          provenance: card?.provenance ?? inheritedProvenance(model.provenance, roster.provenance),
        };
      }),
    }),
  );
  return [...split, ...roster.groups.slice(1)];
}

export function lineupText(reading?: LineupReading, error?: string): string {
  if (!reading) return error ? `Lineup unavailable · ${error}` : 'Reading lineup…';
  const { check, starters, week } = reading;
  const count = `${starters.length} of ${reading.starterCount} starters`;
  if (reading.rulesSource.freshness.state === 'fail') {
    return check.problems.map((p) => p.message).join(' · ');
  }
  if (!starters.length) return `No saved lineup · ${reading.provenance.freshness.note}`;
  const prefix = `Week ${week} lineup`;
  if (!check.legal) return `${prefix} · not legal · ${check.problems.map((p) => p.message).join(' · ')}`;
  if (check.full) return `${prefix} · legal · ${count}`;
  return [`${prefix} · legal, partial · ${count}`,
    ...check.problems.filter((p) => p.kind === 'short').map((p) => p.message)].join(' · ');
}
