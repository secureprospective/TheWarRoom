import type { Receipt, Snapshot, TradeAsset } from '../data/contract';

export function tradeAssetLabel(asset: TradeAsset): string {
  const name = asset.name || 'Unnamed asset';
  return asset.position ? `${name} (${asset.position})` : name;
}
export function tradeRailLabels(receipt: Receipt, snapshot: Snapshot) {
  const trade = receipt.spec.expected.trade;
  if (!trade) return undefined;
  const outgoing = receipt.spec.franchiseId === trade.offering;
  const otherId = outgoing ? trade.accepting : trade.offering;
  const other = snapshot.franchises.value.find((f) => f.id === otherId);
  const players = new Map(snapshot.players.value.map((p) => [p.id, p]));
  const assets = new Map(receipt.tradeAssets?.map(asset => [asset.token, asset.name]));
  const names = (tokens: string[]) => tokens.map(token =>
    players.get(token)?.name?.split(',')[0] || assets.get(token) || 'Unnamed asset')
    .join(', ');
  return {
    header: receipt.spec.intent === 'trade.accept' ? `Offer ${trade.tradeId} · ${other?.name || otherId}`
      : `Trade · ${other?.name || 'Franchise name unavailable'}`,
    summary: `+${names(outgoing ? trade.acceptingGives : trade.offeringGives)} ` +
      `−${names(outgoing ? trade.offeringGives : trade.acceptingGives)}`,
  };
}
