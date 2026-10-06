import type { Snapshot } from '../data/contract';
import { PlayerCard } from '../cards/PlayerCard';
import { playerCardModel } from '../cards/playerModel';
import type { PlayerSubject } from './state';

export function PlayerInspector({
  snapshot,
  subject,
}: {
  snapshot: Snapshot;
  subject: PlayerSubject;
}) {
  const model = playerCardModel(snapshot, subject.franchiseId, subject.id, new Date());
  return (
    <>
      <PlayerCard
        snapshot={snapshot}
        franchiseId={subject.franchiseId}
        playerId={subject.id}
        asOf={new Date()}
        density="tactical"
      />
      <section className="zone interrogate">
        <h5>Interrogate</h5>
        <div className="provenance-detail">{model.provenance.detail}</div>
        <dl className="kv">
          <dt>Status</dt>
          <dd>{model.contractStatus}</dd>
          <dt>Years</dt>
          <dd>{model.years ?? 'Unavailable'}</dd>
          <dt>Salary</dt>
          <dd>{model.salary}</dd>
        </dl>
        {['On-field-now', 'Dynasty value', 'Dead cap', 'Form', 'Market'].map((label) => (
          <p key={label} className="not-wired">
            {label} · not wired
          </p>
        ))}
      </section>
    </>
  );
}
