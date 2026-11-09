import { marketAPI } from '../market/api';
import { parseBoard } from '../reports/contract';
import { readPlayerUsage, researchAPI } from '../research/api';
import { targetInjuries } from '../data/api';
import { parseInjuries, parseUsage, type Usage } from './hqPlayerData';
import type { PlayerReading } from '../research/contract';

export const readHQMarket = async () => parseBoard(await marketAPI.reports());
export const readHQInjuries = async () => parseInjuries(await targetInjuries());
export type RosterFacts = Map<string, { usage?: Usage; research?: PlayerReading; error: string }>;
export async function readRosterFacts(ids: string[]): Promise<RosterFacts> {
  const entries = await Promise.all(ids.map(async id => {
    const [usage, research] = await Promise.allSettled([
      readPlayerUsage(id).then(parseUsage), researchAPI.player!(id),
    ]);
    return [id, { usage: usage.status === 'fulfilled' ? usage.value : undefined,
      research: research.status === 'fulfilled' ? research.value : undefined,
      error: [usage, research].flatMap(r => r.status === 'rejected' ? [String(r.reason)] : []).join(' · ')
        }] as const;
  }));
  return new Map(entries);
}
