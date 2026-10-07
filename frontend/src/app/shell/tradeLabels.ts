import type { Receipt, Snapshot, TradeAsset } from '../data/contract';

export function tradeAssetLabel(asset: TradeAsset): string {
  const name = asset.name || asset.token;
  return asset.position ? `${name} (${asset.position})` : name;
}
export function tradeRailLabels(receipt: Receipt, snapshot: Snapshot) {
  const trade = receipt.spec.expected.trade;
  if (!trade) return undefined;
  const other = snapshot.franchises.value.find((f) => f.id === trade.offering);
  const players = new Map(snapshot.players.value.map((p) => [p.id, p]));
  const names = (tokens: string[]) => tokens.map((token) => players.get(token)?.name?.split(',')[0] || token)
    .join(', ');
  return {
    header: `Offer ${trade.tradeId} · ${other?.name || trade.offering}`,
    summary: `+${names(trade.offeringGives)} −${names(trade.acceptingGives)}`,
  };
}
