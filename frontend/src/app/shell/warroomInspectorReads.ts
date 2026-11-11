import { parseNativeValues, type LeaguePlayerValue } from '../data/nativeValues';
import { readInspectorFit, type InspectorFit } from './inspectorFit';
import type { PlayerReading } from '../research/contract';
import type { Reading, PlayerContract } from '../contracts/contract';
import { contractsProvider } from '../contracts/provider';
import { researchAPI } from '../research/api';
import { readAgeArc } from '../market/api';
import { readPersonalNotes, personalWire } from '../offers/api';
import { readNativeCut, type NativeCut } from '../commissioner/hqReading';
import { hostedWorkflow } from '../commissioner/api';
import { parseArc, parseNotes, type Arc } from './hqPlayerData';
import { parseWarRoomProjections } from './warroomMarketReads';
import type { WarRoomContext } from './warroomReads';
export type WarRoomInspectorReading = { research?: PlayerReading; arc?: Arc;
  contract?: Reading<PlayerContract>; cut?: NativeCut; notes?: ReturnType<typeof parseNotes>;
  valuesPreparing?: boolean; value?: LeaguePlayerValue; fit?: InspectorFit;
  projection?: ReturnType<typeof parseWarRoomProjections>[number]; errors: string[] };
// Each fact lands on the card as it arrives: the lineup fit is slow and must not hold the values back.
export async function readWarRoomInspector(c: WarRoomContext, id: string,
  publish?: (reading: WarRoomInspectorReading) => void): Promise<WarRoomInspectorReading> {
  const result: WarRoomInspectorReading = { errors: [] };
  const apply = async (fn: () => Promise<void>) => { try { await fn(); } catch (e) {
    result.errors.push(String(e)); } finally { publish?.({ ...result, errors: [...result.errors] }); } };
  if (c.native) {
    const native = c.native;
    await Promise.all([
      apply(async () => { const season = c.snapshot.league.value.season;
        const { data } = await (native.port ?? hostedWorkflow)({ command: 'value.read', mode: 'read',
          arguments: { league: native.league, franchise: '', season } });
        const values = parseNativeValues(data, native.league, '', season);
        result.valuesPreparing = values.preparing;
        result.value = values.players.find(p => p.player === id); }),
      apply(async () => { result.fit = await readInspectorFit(c, id); }),
      apply(async () => { result.arc = parseArc(await readAgeArc(id)); }),
      apply(async () => {
        const owners = c.snapshot.rosters.value.filter(r => r.players.some(p => p.id === id));
        const owner = owners.find(r => r.franchiseId === c.mine) ??
          (owners.length === 1 ? owners[0] : undefined);
        if (!owner) throw new Error('Choose the rostered copy to quote its contract');
        result.cut = await readNativeCut(native.league, owner.franchiseId, id, native.port);
      }),
      apply(async () => {
        if (!c.week) return;
        const season = c.snapshot.league.value.season;
        const { data } = await (native.port ?? hostedWorkflow)({ command: 'projections.read', mode: 'read',
          arguments: { league: native.league, players: id, season, week: c.week } });
        result.projection = parseWarRoomProjections(data, native.league, season, c.week).find(p =>
          p.player === id);
      }),
    ]);
    return result;
  }
  await Promise.all([
    apply(async () => { result.fit = await readInspectorFit(c, id); }),
    apply(async () => { const research = await researchAPI.player!(id);
      if (research.id !== id) throw new Error('Research belongs to another player');
      result.research = research; }),
    apply(async () => { result.arc = parseArc(await readAgeArc(id)); }),
    apply(async () => { const contract = await contractsProvider().player(id);
      if (contract.value && contract.value.player.id !== id) throw new Error('Wrong player contract');
      result.contract = contract; }),
    apply(async () => { const notes = parseNotes(await readPersonalNotes(c.mine, id));
      if (notes.player !== id) throw new Error('Private notes belong to another player');
      result.notes = notes; }),
  ]);
  return result;
}
export async function changeWarRoomNote(c: WarRoomContext, id: string, text: string, at?: string) {
  if (c.native) throw new Error('Hosted private notes are not available yet');
  await personalWire.change(c.mine, 'note', { id, entity: 'player', action: at ? 'remove' : 'add', text,
    ...(at ? { at } : {}) });
}
