import type { Snapshot, Provenance } from '../data/contract';
import { provenanceSlot } from '../cards/playerModel';
export function snapshotSummary(snapshot: Snapshot) {
  const sections: [string, Provenance][] = Object.entries(snapshot).map(
    ([name, section]) => [name, section.provenance],
  );
  // The same one-dot provenance as every card: the weakest section, its detail on hover.
  const slot = provenanceSlot(sections);
  return {
    provenance: slot,
    season:
      snapshot.league.provenance.freshness.state === 'fail'
        ? 'Unavailable'
        : snapshot.league.value.season,
    franchises:
      snapshot.franchises.provenance.freshness.state === 'fail'
        ? 'Unavailable'
        : snapshot.franchises.value.length,
    rostered:
      snapshot.rosters.provenance.freshness.state === 'fail'
        ? 'Unavailable'
        : snapshot.rosters.value.reduce((n, roster) => n + roster.players.length, 0),
  };
}
