import { describe, expect, it } from 'vitest';
import fixture from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { CONTRACT_STATUSES, FRESHNESS_STATES, ROSTER_STATUSES } from '../data/contract';
import { playerCardModel, provenanceSlot, freshnessSignals } from './playerModel';
import { specimenPlayers } from '../specimen/selection';

function input() {
  const snapshot = parseSnapshot(structuredClone(fixture));
  const roster = snapshot.rosters.value[0];
  const contract = roster.players[0];
  const player = snapshot.players.value.find(
    (candidate) => candidate.id === contract.id,
  )!;
  return { snapshot, roster, contract, player };
}
const asOf = new Date('2026-10-06T00:00:00Z');

describe('player-card view model', () => {
  it.each(ROSTER_STATUSES)('selects only held status chips for %s', (rosterStatus) => {
    const { snapshot, roster, contract, player } = input();
    contract.rosterStatus = rosterStatus;
    player.isRookie = false;
    const model = playerCardModel(snapshot, roster.franchiseId, player.id, asOf);
    expect(model.status).toEqual(
      rosterStatus === 'IR'
        ? [{ signal: 'red', label: 'IR' }]
        : rosterStatus === 'TAXI_SQUAD'
          ? [{ signal: 'blue', label: 'Taxi' }]
          : [],
    );
    expect(model.numbers).toEqual([
      { label: 'On-field-now', state: 'not wired' },
      { label: 'Dynasty value', state: 'not wired' },
    ]);
    expect(model.unwiredRows).toBe('Form and market · not wired');
    expect(model).not.toHaveProperty('injury');
    expect(model).not.toHaveProperty('bye');
    expect(model).not.toHaveProperty('deadCap');
    expect(model).not.toHaveProperty('verdict');
  });
  it.each(CONTRACT_STATUSES)('keeps %s as a neutral contract label', (contractStatus) => {
    const { snapshot, roster, contract, player } = input();
    contract.contractStatus = contractStatus;
    player.isRookie = true;
    const model = playerCardModel(snapshot, roster.franchiseId, player.id, asOf);
    expect(model.contractStatus).toBe(contractStatus);
    expect(model.status).toContainEqual({ signal: 'blue', label: 'Rookie' });
    expect(model.status.map((chip) => chip.label)).not.toContain(contractStatus);
  });
  it('omits missing age and years, retains zero years and formats cents', () => {
    const { snapshot, roster, contract, player } = input();
    delete player.birthdate;
    delete contract.yearsRemaining;
    contract.salary = 1_200_000_000;
    const model = playerCardModel(snapshot, roster.franchiseId, player.id, asOf);
    expect(model.age).toBeUndefined();
    expect(model.years).toBeUndefined();
    expect(model.salary).toBe('$12.0M');
    contract.yearsRemaining = 0;
    expect(playerCardModel(snapshot, roster.franchiseId, player.id, asOf).years).toBe(
      '0 years',
    );
    contract.yearsRemaining = 1;
    expect(playerCardModel(snapshot, roster.franchiseId, player.id, asOf).years).toBe(
      '1 year',
    );
  });
  it('shows the weakest section on the one provenance dot, with every section in the detail', () => {
    const { snapshot, roster, player } = input();
    snapshot.players.provenance.freshness.state = 'fail';
    snapshot.rosters.provenance.freshness.state = 'stale';
    const model = playerCardModel(snapshot, roster.franchiseId, player.id, asOf);
    expect(model.provenance.signal).toBe('red');
    expect(model.provenance.detail).toContain('Players · fixture · fail');
    expect(model.provenance.detail).toContain('Contracts · fixture · stale');
  });
  it.each(FRESHNESS_STATES)(
    'labels a lone %s section by origin and date, never by its note',
    (state) => {
      const { snapshot } = input();
      const provenance = snapshot.rosters.provenance;
      provenance.freshness.state = state;
      const slot = provenanceSlot([['Contracts', provenance]]);
      expect(slot.signal).toBe(freshnessSignals[state]);
      expect(slot.label).toBe(`fixture · ${provenance.freshness.fetchedAt.slice(5, 10)}`);
      provenance.kind = 'live';
      expect(provenanceSlot([['Contracts', provenance]]).label).toBe(
        [
          { live: 'fresh', stale: 'stale', fail: 'failing' }[state],
          provenance.freshness.fetchedAt.slice(5, 10),
        ].join(' · '),
      );
    },
  );
  it('rejects a player not on the requested roster', () => {
    const { snapshot, roster } = input();
    expect(() => playerCardModel(snapshot, roster.franchiseId, 'absent', asOf)).toThrow(
      'missing player or roster contract',
    );
  });
  it('chooses the real fixture spread without coercing leading-zero IDs', () => {
    const snapshot = parseSnapshot(fixture);
    const choices = specimenPlayers(snapshot);
    expect(choices.map((choice) => choice.reason)).toEqual([
      'Quarterback',
      'IDP on IR',
      'Taxi rookie',
      'Leading-zero ID',
      'Birthdate absent',
    ]);
    for (const choice of choices) {
      const model = playerCardModel(snapshot, choice.franchiseId, choice.playerId, asOf);
      expect(model.id).toBe(choice.playerId);
      if (choice.reason === 'Leading-zero ID') expect(model.id).toMatch(/^0/);
      if (choice.reason === 'Birthdate absent') expect(model.age).toBeUndefined();
      if (choice.reason === 'Quarterback') expect(model.position).toBe('QB');
      if (choice.reason === 'Taxi rookie')
        expect(model.status.map((chip) => chip.label)).toEqual(['Taxi', 'Rookie']);
      if (choice.reason === 'IDP on IR')
        expect(model.status).toContainEqual({ signal: 'red', label: 'IR' });
    }
  });
});
