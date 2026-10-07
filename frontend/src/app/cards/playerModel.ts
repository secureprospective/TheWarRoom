import type { Freshness, Position, Provenance, Snapshot } from '../data/contract';
import type { LabelledSignal, Signal } from '../look/channels';
import { ageAt, formatMoney } from './format';

export const freshnessSignals: Record<Freshness['state'], Signal> = {
  live: 'green',
  stale: 'amber',
  fail: 'red',
};

const severity: Record<Freshness['state'], number> = { live: 0, stale: 1, fail: 2 };
const restingWords: Record<Freshness['state'], string> = {
  live: 'fresh',
  stale: 'stale',
  fail: 'failing',
};

function describe(section: string, p: Provenance): string {
  const { state, fetchedAt, note } = p.freshness;
  return [section, p.kind, state, p.source, fetchedAt || 'no fetch time', note]
    .filter(Boolean)
    .join(' · ');
}

// A card has one provenance dot. It reports its weakest section, because a card is only as current
// as its oldest input; the per-section detail is one hover away.
export function provenanceSlot(
  sections: readonly (readonly [string, Provenance])[],
): LabelledSignal {
  const worst = sections.reduce((a, b) =>
    severity[b[1].freshness.state] > severity[a[1].freshness.state] ? b : a,
  )[1];
  const oldest = sections
    .map(([, p]) => p.freshness.fetchedAt)
    .filter(Boolean)
    .sort()[0];
  const word = worst.kind === 'fixture' ? 'fixture' : restingWords[worst.freshness.state];
  return {
    signal: freshnessSignals[worst.freshness.state],
    label: oldest ? `${word} · ${oldest.slice(5, 10)}` : word,
    detail: sections.map(([name, p]) => describe(name, p)).join('\n'),
  };
}

export type PlayerCardModel = {
  id: string;
  name: string;
  position?: Position;
  team?: string;
  age?: number;
  numbers: readonly { label: string; state: 'not wired' }[];
  status: LabelledSignal[];
  contractStatus: string;
  rosterNote?: string;
  salary: string;
  years?: string;
  unwiredRows: string;
  provenance: LabelledSignal;
};

export function playerCardModel(
  snapshot: Snapshot,
  franchiseId: string,
  playerId: string,
  asOf: Date,
  allowUnrostered = false,
): PlayerCardModel {
  const player = snapshot.players.value.find((candidate) => candidate.id === playerId);
  const contract = snapshot.rosters.value
    .find((roster) => roster.franchiseId === franchiseId)
    ?.players.find((candidate) => candidate.id === playerId);
  if ((!player || !contract) && !allowUnrostered)
    throw new Error(
      `Player card ${franchiseId}/${playerId}: missing player or roster contract`,
    );
  const status: LabelledSignal[] = [];
  if (contract?.rosterStatus === 'IR') status.push({ signal: 'red', label: 'IR' });
  if (contract?.rosterStatus === 'TAXI_SQUAD')
    status.push({ signal: 'blue', label: 'Taxi' });
  if (player?.isRookie) status.push({ signal: 'blue', label: 'Rookie' });
  return {
    id: playerId,
    name: player?.name ?? playerId,
    position: player?.position,
    team: player?.team,
    age: ageAt(player?.birthdate, asOf),
    numbers: [
      { label: 'On-field-now', state: 'not wired' },
      { label: 'Dynasty value', state: 'not wired' },
    ],
    status,
    contractStatus: contract?.contractStatus ?? 'Unavailable',
    salary: contract ? formatMoney(contract.salary) : 'Unavailable',
    years:
      contract?.yearsRemaining === undefined
        ? undefined
        : `${contract.yearsRemaining} ${contract.yearsRemaining === 1 ? 'year' : 'years'}`,
    unwiredRows: 'Form and market · not wired',
    provenance: provenanceSlot([
      ['Players', snapshot.players.provenance],
      ['Contracts', snapshot.rosters.provenance],
    ]),
  };
}

export function inheritedProvenance(
  own: LabelledSignal,
  container: LabelledSignal,
): 'own' | 'inherited' {
  return own.signal === container.signal && own.label === container.label &&
    own.detail === container.detail ? 'inherited' : 'own';
}
