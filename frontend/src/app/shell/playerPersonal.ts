import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { parseNotes } from './hqPlayerData';

export async function readHostedPlayerNotes(league: string, player: string,
  port: HostedPort = hostedWorkflow) {
  const { data } = await port({ command: 'player.note.read', mode: 'read', arguments: { league, player } });
  const reading = parseNotes(data);
  if (reading.league !== league || reading.player !== player) throw new Error('Private notes scope mismatch');
  return reading;
}

export async function setHostedPlayerNote(league: string, player: string, text: string,
  port: HostedPort = hostedWorkflow, at?: string) {
  await port({ command: 'player.note.set', mode: 'execute', arguments: {
    league, player, text, ...(at ? { at } : {}),
  } });
}

export async function setHostedPlayerWatch(league: string, player: string, watched: boolean,
  port: HostedPort = hostedWorkflow) {
  await port({ command: 'player.watch.set', mode: 'execute', arguments: { league, player, watched } });
}
