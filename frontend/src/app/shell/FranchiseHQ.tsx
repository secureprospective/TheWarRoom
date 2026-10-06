import { useMemo } from 'react';
import type { Snapshot } from '../data/contract';
import { PlayerCard } from '../cards/PlayerCard';
import { formatMoney } from '../cards/format';
import { provenanceSlot } from '../cards/playerModel';
import { SignalChip } from '../look/Slots';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { franchiseRoster } from './roster';
import { assertShortList } from './shortList';
import { MatrixRoster } from './MatrixRoster';

export function FranchiseHQ({ snapshot }: { snapshot: Snapshot }) {
  const s = commands.use();
  const roster = useMemo(
    () => s.franchiseId ? franchiseRoster(snapshot, s.franchiseId) : undefined,
    [snapshot, s.franchiseId],
  );
  if (import.meta.env.DEV) {
    assertShortList(snapshot.franchises.value.length);
    if (roster) assertShortList(roster.groups.reduce((n, g) => n + g.players.length, 0));
  }
  if (!roster)
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
  return (
    <section className="hq-roster">
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
            {group.label} · {group.players.length} <SignalChip {...roster.provenance} />
          </h5>
          <MatrixRoster cards={group.cards} franchiseId={s.franchiseId!} selected={s.subject} />
          <div className="roster-cards" aria-label={`${group.label} roster`}>
            {group.cards.map(({ model, provenance }) => (
              <Act
                key={model.id}
                verb="inspector.open"
                args={{
                  subject: { kind: 'player', id: model.id, franchiseId: s.franchiseId! },
                }}
                active={
                  s.subject?.id === model.id && s.subject.franchiseId === s.franchiseId
                }
                label={`Inspect ${model.name}`}
              >
                <PlayerCard
                  snapshot={snapshot}
                  franchiseId={s.franchiseId!}
                  playerId={model.id}
                  asOf={roster.asOf}
                  provenance={provenance}
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
