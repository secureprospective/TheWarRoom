import { FranchisePicker } from './FranchisePicker';
import { useMemo } from 'react';
import type { Snapshot } from '../data/contract';
import { formatMoney } from '../cards/format';
import { provenanceSlot } from '../cards/playerModel';
import { SignalChip } from '../look/Slots';
import { commands } from '../commands/registry';
import { franchiseRoster } from './roster';
import { assertShortList } from './shortList';
import { LineupRoster } from './LineupRoster';

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
  if (!roster) return <FranchisePicker snapshot={snapshot} />;
  return (
    <section className="hq-roster">
      <LineupRoster
        snapshot={snapshot}
        roster={roster}
        franchiseId={s.franchiseId!}
        selected={s.subject}
        cap={
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
        }
      />
    </section>
  );
}
