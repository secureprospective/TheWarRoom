import type { Snapshot } from '../data/contract';
import { Act } from '../commands/Act';
import { RailLeaguePicker } from './leaguePicker';

export function FranchisePicker({ snapshot }: { snapshot: Snapshot }) {
  return (
    <section className="franchise-picker">
      <RailLeaguePicker />
      <h4>View a franchise</h4>
      {snapshot.franchises.value.map((f) => (
        <Act variant="button" key={f.id} verb="franchise.set" args={{ franchiseId: f.id }}>
          {f.name ?? f.id}
        </Act>
      ))}
    </section>
  );
}
