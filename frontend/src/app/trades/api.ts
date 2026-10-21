import type * as Bindings from '../../../wailsjs/go/trades/API';
import type { trades } from '../../../wailsjs/go/models';

// The only file that touches the trades bindings; callers parse the unknown wire values.
export type TradeBuilderRequest = {
  mine: string; other: string; action: string; id: string; give: string[]; get: string[]; rationale: string;
};

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Trades require the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { trades: { API: typeof Bindings } } }).go.trades.API;
}

export async function targetTrades(franchiseId: string): Promise<unknown> {
  return bindings().TargetTrades(franchiseId);
}

export async function targetDraftTradeAccept(franchiseId: string, tradeId: string): Promise<unknown> {
  return bindings().TargetDraftTradeAccept(franchiseId, tradeId);
}

export async function targetTradeBuilder(request: TradeBuilderRequest): Promise<unknown> {
  return bindings().TargetTradeBuilder(request as trades.RawTradeBuilder);
}
