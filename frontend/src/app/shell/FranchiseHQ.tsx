import type { Snapshot } from '../data/contract';
import { PlayerCard } from '../cards/PlayerCard';
import { formatMoney } from '../cards/format';
import { provenanceSlot } from '../cards/playerModel';
import { SignalChip } from '../look/Slots';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { franchiseRoster } from './roster';

export function FranchiseHQ({ snapshot }: { snapshot: Snapshot }) {
  const s = commands.use();
  if (!s.franchiseId)
    return (
      <section className="franchise-picker">
        <h4>Choose my franchise</h4>
        {snapshot.franchises.value.map((f) => (
          <Act key={f.id} verb="franchise.set" args={{ franchiseId: f.id }}>
            {f.name ?? f.id}
          </Act>
        ))}
      </section>
    );
  const roster = franchiseRoster(snapshot, s.franchiseId);
  return (
    <section className="hq-roster" data-density={s.density}>
      <h4>Roster</h4>
      <p className="not-wired">lineup · ring 1</p>
      <div className="roster-cap">
        {(['capUsed', 'capRoom', 'salaryCap'] as const).map((key, index) => (
          <span key={key}>
            {['Cap used', 'Cap room', 'Salary cap'][index]}{' '}
            <b>{roster[key] === undefined ? 'Unavailable' : formatMoney(roster[key]!)}</b>
          </span>
        ))}
        <SignalChip
          {...provenanceSlot([['Franchises', snapshot.franchises.provenance]])}
        />
      </div>
      {roster.groups.map((group) => (
        <section key={group.label}>
          <h5>
            {group.label} · {group.players.length}
          </h5>
          <div className="roster-cards" aria-label={`${group.label} roster`}>
            {group.players.map((player) => (
              <Act
                key={player.id}
                verb="inspector.open"
                args={{
                  subject: { kind: 'player', id: player.id, franchiseId: s.franchiseId! },
                }}
                active={
                  s.subject?.id === player.id && s.subject.franchiseId === s.franchiseId
                }
                label={`Inspect ${snapshot.players.value.find((p) => p.id === player.id)?.name ?? player.id}`}
              >
                <PlayerCard
                  snapshot={snapshot}
                  franchiseId={s.franchiseId!}
                  playerId={player.id}
                  asOf={new Date()}
                  density={s.density}
                />
              </Act>
            ))}
          </div>
          {!group.players.length && <p className="not-wired">No players</p>}
        </section>
      ))}
    </section>
  );
}
