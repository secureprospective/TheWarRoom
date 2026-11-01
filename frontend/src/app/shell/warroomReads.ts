import type { Node } from './nodes';
import { useContractRead } from '../contracts/Contracts';
import { contractsProvider } from '../contracts/provider';
import { controlAPI } from '../control/api';
import { raceAPI } from '../race/api';
import { report } from '../reports/contract';
import type { Snapshot } from '../data/contract';
import { readNativeHQ } from '../commissioner/hqReading';
import type { HostedPort } from '../commissioner/hostedRoom';
import { hqEvents } from './hqReads';
import { settingsFromReport, type HQCap } from './hqModel';
import { standingFromReport } from './hqPlayerData';
export type WarRoomNative = { league: string; mine: string; port?: HostedPort;
  leave: (node?: Node) => void; week?: number };
export type WarRoomContext = { snapshot: Snapshot; mine: string; week: number; native?: WarRoomNative };
export type WarRoomHeaderReading = { name?: string; record?: string; standing?: string;
  starterN?: number; best?: string; rank?: string; runway?: HQCap[]; errors: string[] };
export function useWarRoomRead<T>(key: string, read: () => Promise<T>) {
  return useContractRead(key, read, hqEvents);
}
export async function readWarRoomHeader({ snapshot, mine, week, native }: WarRoomContext):
  Promise<WarRoomHeaderReading> {
  if (native) {
    const r = await readNativeHQ(native.league, mine, native.port);
    return { name: r.leagueName, record: r.record, standing: r.standing,
      starterN: r.settings.starters, runway: r.runway, errors: r.errors };
  }
  const result: WarRoomHeaderReading = { errors: [] };
  const attempt = async (read: () => Promise<void>) => { try { await read(); } catch (e) {
    result.errors.push(String(e)); } };
  await Promise.all([
    attempt(async () => { const settings = report(await controlAPI.settings(), 'Settings');
      result.starterN = settingsFromReport(settings).starters;
      result.name = settings.rows.find(r => r.cells[0] === 'League name')?.cells[1]; }),
    attempt(async () => { const race = await raceAPI.read(week, mine, mine, 'topn');
      Object.assign(result, standingFromReport(race.standings));
      result.starterN = race.starterN;
      const power = race.board.reports.find(r => r.id === 'M-008');
      const row = power?.rows.find(r => r.mine);
      if (row && power) {
        const score = power.columns.findIndex(c => c === 'Roster value');
        if (score >= 0) result.best = row.cells[score];
        result.rank = row.cells[0];
      }
    }),
    attempt(async () => { const r = await contractsProvider().contracts({ mine, team: mine, position: '',
      sort: 'salary', desc: true });
      const cap = r.cap.value;
      if (cap) result.runway = [{ season: cap.season, max: cap.cap?.cents, room: cap.room?.cents },
        ...cap.future.map(y => ({ season: y.season, committed: y.cost.cents }))];
    }),
  ]);
  return result;
}
