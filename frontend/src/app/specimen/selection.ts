import type { Snapshot } from '../data/contract';

export function specimenPlayers(snapshot: Snapshot): { franchiseId: string; playerId: string; reason: string }[] {
  const players = new Map(snapshot.players.value.map(player => [player.id, player]));
  const rostered = snapshot.rosters.value.flatMap(roster => roster.players.map(contract => ({
    franchiseId: roster.franchiseId, contract, player: players.get(contract.id),
  })));
  const selections = [
    { reason: 'Quarterback', entry: rostered.find(entry => entry.player?.position === 'QB') },
    { reason: 'IDP on IR', entry: rostered.find(entry => entry.contract.rosterStatus === 'IR'
      && entry.player?.position && ['DT', 'DE', 'LB', 'CB', 'S'].includes(entry.player.position)) },
    { reason: 'Taxi rookie', entry: rostered.find(entry => entry.contract.rosterStatus === 'TAXI_SQUAD' && entry.player?.isRookie) },
    { reason: 'Leading-zero ID', entry: rostered.find(entry => entry.player?.id.startsWith('0')) },
    { reason: 'Birthdate absent', entry: rostered.find(entry => entry.player && entry.player.birthdate === undefined) },
  ];
  return selections.flatMap(({ reason, entry }) => entry?.player
    ? [{ franchiseId: entry.franchiseId, playerId: entry.player.id, reason }] : []);
}
