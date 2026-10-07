import './lineup.css';
import { useMemo, type ReactNode } from 'react';
import type { Snapshot } from '../data/contract';
import { PlayerCard } from '../cards/PlayerCard';
import { SignalChip } from '../look/Slots';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { MatrixRoster } from './MatrixRoster';
import type { PlayerSubject } from './state';
import { useLineup } from './lineups';
import { lineupGroups, lineupText } from './roster';
import { provenanceSlot } from '../cards/playerModel';
import type { franchiseRoster } from './roster';

export function LineupRoster({ snapshot, roster, franchiseId, selected, cap }: {
  snapshot: Snapshot;
  roster: ReturnType<typeof franchiseRoster>;
  franchiseId: string;
  selected: PlayerSubject | null;
  cap: ReactNode;
}) {
  const { reading: cached, error } = useLineup(commands.lineups, franchiseId, snapshot);
  const reading = error === undefined ? cached : undefined;
  const groups = useMemo(() => reading ? lineupGroups(snapshot, roster, reading) : roster.groups,
    [snapshot, roster, reading]);
  return (
    <>
      <h4>Roster</h4>
      <p className="lineup-reading" role="status">
        {lineupText(reading, error)}{' '}
        {reading && <>
          <SignalChip {...provenanceSlot([['Lineup', reading.provenance]])} />{' '}
          <SignalChip {...provenanceSlot([['Rules', reading.rulesSource]])} />
        </>}
      </p>
      {cap}
      {groups.map((group) => (
        <section key={group.label}>
          <h5>{group.label} · {group.players.length} <SignalChip {...roster.provenance} /></h5>
          <MatrixRoster cards={group.cards} franchiseId={franchiseId} selected={selected} />
          <div className="roster-cards" aria-label={`${group.label} roster`}>
            {group.cards.map(({ model, provenance }) => (
              <Act
                key={model.id}
                verb="inspector.open"
                args={{ subject: { kind: 'player', id: model.id, franchiseId } }}
                active={selected?.id === model.id && selected.franchiseId === franchiseId}
                label={`Inspect ${model.name}`}
              >
                <PlayerCard
                  snapshot={snapshot}
                  franchiseId={franchiseId}
                  playerId={model.id}
                  asOf={roster.asOf}
                  provenance={provenance}
                  model={model}
                />
              </Act>
            ))}
          </div>
          {!group.players.length && <p className="not-wired">No players</p>}
        </section>
      ))}
    </>
  );
}
