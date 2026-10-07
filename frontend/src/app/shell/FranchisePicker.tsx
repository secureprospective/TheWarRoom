import type { Snapshot } from '../data/contract';
import { Act } from '../commands/Act';

export function FranchisePicker({ snapshot }: { snapshot: Snapshot }) {
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
}
