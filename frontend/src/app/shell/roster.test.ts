import { expect, it } from 'vitest';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { franchiseRoster, lineupGroups } from './roster';
import { playerCardModel } from '../cards/playerModel';

it('reuses Active card models and leaves the exact IR and Taxi groups intact', () => {
  const snapshot = heldSnapshot();
  const roster = franchiseRoster(snapshot, '0025');
  const reading = heldLineup();
  const split = lineupGroups(snapshot, roster, reading);
  expect(split.slice(2)).toEqual(roster.groups.slice(1));
  expect(split[2]).toBe(roster.groups[1]);
  expect(split[3]).toBe(roster.groups[2]);
  for (const group of split.slice(0, 2)) {
    for (const card of group.cards) {
      expect(card.model).toBe(roster.groups[0].cards.find((c) => c.model.id === card.model.id)!.model);
    }
  }
  expect(split[0].cards.map((c) => c.model.id)).toEqual(reading.starters.map((p) => p.id));
  expect(split[1].cards.map((c) => c.model.id)).toEqual(reading.bench.map((p) => p.id));
});
it('uses unavailable contract facts for a dropped starter and its real directory identity', () => {
  const snapshot = heldSnapshot();
  const reading = heldLineup();
  const id = reading.starters[0].id;
  snapshot.rosters.value[0].players = snapshot.rosters.value[0].players.filter((p) => p.id !== id);
  const roster = franchiseRoster(snapshot, '0025');
  const model = lineupGroups(snapshot, roster, reading)[0].cards[0].model;
  expect(model.name).toBe(snapshot.players.value.find((p) => p.id === id)!.name);
  expect(model.salary).toBe('Unavailable');
  expect(model.contractStatus).toBe('Unavailable');
  expect(model.years).toBeUndefined();
  expect(model.rosterNote).toBe('not on the active roster');
});
it('uses the ID when the directory is missing, without fabricating identity or salary', () => {
  const snapshot = heldSnapshot();
  const model = playerCardModel(snapshot, '0025', '99999', new Date(), true);
  expect(model.name).toBe('99999');
  expect(model.position).toBeUndefined();
  expect(model.team).toBeUndefined();
  expect(model.salary).toBe('Unavailable');
  expect(model.contractStatus).toBe('Unavailable');
  const id = heldLineup().starters[0].id;
  snapshot.players.value = snapshot.players.value.filter((p) => p.id !== id);
  const roster = franchiseRoster(snapshot, '0025');
  expect(roster.groups[0].cards.find((c) => c.model.id === id)!.model.name).toBe(id);
});
