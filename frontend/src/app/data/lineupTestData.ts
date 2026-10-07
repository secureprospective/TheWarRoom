import { readFileSync } from 'node:fs';
import type { LineupReading, Player, Position, RosterPlayer, Snapshot } from './contract';
import { parseLineup } from './parseLineup';

function directoryPlayers(): Player[] {
  const raw: { id: string; name: string; position: string }[] = JSON.parse(readFileSync(
    '../internal/lineup/testdata/directory-0025.json', 'utf8',
  ));
  return raw.map((p) => ({ ...p, position: (p.position === 'PK' ? 'K' : p.position) as Position }));
}

// IDs and order are TestTargetLineupHeldReading's assertions, not a fixture-provider lineup.
export function heldLineup(): LineupReading {
  const directory = directoryPlayers();
  const byId = new Map(directory.map((p) => [p.id, p.position]));
  const players = (ids: string[]) => ids.map((id) => ({ id, position: byId.get(id) }));
  const provenance = {
    source: 'test:held lineup', kind: 'live',
    freshness: { state: 'live', fetchedAt: '2026-10-07T13:38:00Z', note: '' },
  };
  return parseLineup({
    franchise: '0025', week: 5, starterCount: 21,
    starters: players([
      '16150', '13133', '15754', '16195', '15761', '16617', '15798', '16641', '16846',
      '16694', '16264', '16230', '16303', '16734', '13813', '14892', '15836', '16267',
      '16460', '13322', '15850',
    ]),
    bench: players([
      '15252', '17041', '16387', '17685', '17274', '14315', '16428', '15941',
      '15357', '17150', '16923', '16439', '14168', '16793', '16255', '15904', '16880',
    ]),
    check: { full: true, legal: true, problems: [] },
    provenance,
    rulesSource: {
      ...provenance, source: 'test:real league',
      freshness: { state: 'stale', fetchedAt: '', note: 'active rulebook; fetch time unknown' },
    },
  });
}
export function heldSnapshot(): Snapshot {
  const players = directoryPlayers();
  const roster: { player: { id: string; status: RosterPlayer['rosterStatus'] | 'INJURED_RESERVE' }[] } =
    JSON.parse(readFileSync('../internal/lineup/testdata/roster-0025.json', 'utf8'));
  const provenance = heldLineup().provenance;
  return {
    league: { value: { season: 2026, franchiseCount: 32 }, provenance },
    franchises: { value: [{ id: '0025' }], provenance },
    players: { value: players, provenance },
    rosters: {
      value: [{ franchiseId: '0025', players: roster.player.map((p) => ({
        id: p.id,
        rosterStatus: p.status === 'INJURED_RESERVE' ? 'IR' : p.status,
        salary: 0, contractStatus: 'UFA',
      })) }], provenance,
    },
  };
}
