import type { HostedReading } from './hostedRoom';

// Preview targets are ledger identifiers; owners read the names from their held league view.
export function hostedWords(reading: HostedReading | undefined, words: string): string {
  if (!reading) return words;
  const names = new Map<string, string>([
    ...reading.roster.franchises.map(f => [f.value, f.label] as [string, string]),
    ...reading.roster.players.map(p => [p.id, p.name] as [string, string]),
    ...reading.roster.pool.map(p => [p.value, p.label] as [string, string]),
    ...reading.markets.trades.offers.map((t, i) => [t.id, `Trade ${i + 1}`] as [string, string]),
    ...reading.markets.waivers.players.map(p => [p.window, `Waiver window for ${p.name}`] as [string, string]),
    ...reading.markets.bids.players.flatMap(p => p.own ? [[p.own.id, `Your bid for ${p.name}`] as [string, string]] : []),
  ]);
  const pattern = [...names.keys()].sort((a, b) => b.length - a.length)
    .map(id => id.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|');
  return pattern ? words.replace(new RegExp(`\\b(?:${pattern})\\b`, 'g'), id => names.get(id) || id) : words;
}
