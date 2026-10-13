import './assetNumber.css';
import { assetState, type AssetKey } from './state';

// AssetNumber is one C-01 cell for a player: Go's headline and qualifier, the detail on hover.
// Table qualifiers wrap independently so a value never clips mid-word.
export function AssetNumber({ playerId, which, inline = false }:
  { playerId: string; which: AssetKey; inline?: boolean }) {
  const cell = assetState.useCell(playerId, which);
  const title = [cell.text, cell.sub, cell.detail].filter(Boolean).join(' · ');
  return <span className={cell.held ? 'asset-value' : 'asset-value asset-missing'} title={title}>
    {cell.text}{cell.sub && <span className={inline ? 'asset-sub asset-table-sub' : 'asset-sub'}>
      {cell.sub}</span>}
  </span>;
}
