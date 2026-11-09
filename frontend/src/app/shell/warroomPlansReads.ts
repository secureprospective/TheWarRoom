import { hostedWorkflow } from '../commissioner/api';
import { parseHostedTrades } from '../commissioner/hostedMarkets';
import { parseBuilder } from './tradeBuilderState';
import { targetTradeBuilderRead } from '../trades/api';
import type { WarRoomContext } from './warroomReads';

type Offer = ReturnType<typeof parseHostedTrades>['offers'][number];
export type LookingBack = { offers: { id: string; partner: string; date: string; status: string }[];
  partners: { name: string; talks: number; deals: number }[]; saved?: number; offersNote?: string;
  savedNote?: string };
const ENDED = ['approved', 'declined', 'expired', 'failed', 'vetoed', 'withdrawn'];
const DAY = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' });
// Ended offers newest first, and every franchise you have traded talk with, busiest first.
export function lookingBack(mine: string, all: Offer[]): Pick<LookingBack, 'offers' | 'partners'> {
  const names = new Map(all.flatMap(o => o.teams));
  const ended = all.filter(o => o.parties.includes(mine) && ENDED.includes(o.status))
    .sort((a, b) => b.offeredAt.localeCompare(a.offeredAt));
  const by = new Map<string, { name: string; talks: number; deals: number }>();
  const offers = ended.map(o => {
    const others = o.parties.filter(p => p !== mine).map(p => names.get(p) ?? p);
    for (const name of others) {
      const t = by.get(name) ?? { name, talks: 0, deals: 0 };
      t.talks++; if (o.status === 'approved') t.deals++;
      by.set(name, t);
    }
    return { id: o.id, partner: others.join(' + '), date: DAY.format(new Date(o.offeredAt)),
      status: o.status === 'approved' ? 'accepted' : o.status };
  });
  return { offers, partners: [...by.values()].sort((a, b) => b.talks - a.talks || b.deals - a.deals) };
}
export async function readLookingBack(c: WarRoomContext): Promise<LookingBack> {
  if (!c.native) {
    const r = parseBuilder(await targetTradeBuilderRead(c.mine));
    if (!r.value || r.status !== 'ok') throw new Error(r.absenceReason ?? 'No saved proposals');
    return { offers: [], partners: [], saved: r.value.plans.length,
      offersNote: 'MFL keeps only live offers; finished ones are not read here' };
  }
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'trades.read', mode: 'read',
    arguments: { league: c.native.league, viewer: c.mine } });
  const reading = parseHostedTrades(data);
  if (reading.league !== c.native.league || reading.franchise !== c.mine)
    throw new Error('Trade offers belong to another viewer');
  return { ...lookingBack(c.mine, reading.offers),
    savedNote: 'Saved proposals have no read in this league yet' };
}
