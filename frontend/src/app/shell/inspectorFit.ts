import { nativeBindings } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { WarRoomContext } from './warroomReads';
import { parseWarRoomDelta, type WarRoomDeltaSide } from './warroomTradeReads';

const directFit: HostedPort = async request => {
  if (request.command !== 'lineupdelta.read') throw new Error('Choose a lineup comparison');
  return { label: 'Local reference', writable: false, football: '',
    data: await nativeBindings().TargetNativeLineupDelta(request.arguments) };
};
export type InspectorFit = { side: WarRoomDeltaSide; removing: boolean; notes: string[] };
export async function readInspectorFit(c: WarRoomContext, player: string): Promise<InspectorFit> {
  if (!c.mine || !c.week) throw new Error('Choose your franchise and a scoring week to read lineup fit');
  const owners = c.snapshot.rosters.value.filter(r => r.players.some(p => p.id === player));
  const owner = owners.find(r => r.franchiseId === c.mine) ?? (owners.length === 1 ? owners[0] : undefined);
  if (!owner) throw new Error('Choose a rostered player copy to read its lineup fit');
  const removing = owner.franchiseId === c.mine;
  const partner = removing ? c.snapshot.franchises.value.find(f => f.id !== c.mine)?.id : owner.franchiseId;
  if (!partner) throw new Error('No other franchise is held for this lineup comparison');
  const raw = { league: c.native?.league ?? '', franchise: c.mine, partner, give: removing ? [player] : [],
    get: removing ? [] : [player], season: c.snapshot.league.value.season, week: c.week };
  const port = c.native?.port ?? directFit;
  const { data } = await port({ command: 'lineupdelta.read', mode: 'read', arguments: raw });
  const r = parseWarRoomDelta(data);
  if (r.league !== raw.league || r.season !== raw.season || r.week !== raw.week ||
    r.you.franchise !== c.mine || r.them.franchise !== partner)
    throw new Error('Lineup fit belongs to another player context');
  return { side: r.you, removing, notes: r.notes };
}
