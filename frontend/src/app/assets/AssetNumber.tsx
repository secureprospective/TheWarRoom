import type { AssetCell } from './contract';
import { assetState, type AssetKey } from './state';

// AssetNumber is one C-01 cell for a player: Go's headline and qualifier, the detail on hover.
// Inline (table cells) puts the qualifier on the same line.
export function AssetNumber({ playerId, which, inline = false }:
  { playerId: string; which: AssetKey; inline?: boolean }) {
  const cell = assetState.useCell(playerId, which);
  return <AssetCellView cell={cell} inline={inline} />;
}

export function AssetCellView({ cell, inline = false }: { cell: AssetCell; inline?: boolean }) {
  const title = [cell.text, cell.sub, cell.detail].filter(Boolean).join(' · ');
  return <span className={cell.held ? 'asset-value' : 'asset-value asset-missing'} title={title}>
    {cell.text}{cell.sub && (inline ? ` · ${cell.sub}` : <span className="asset-sub">{cell.sub}</span>)}
  </span>;
}
