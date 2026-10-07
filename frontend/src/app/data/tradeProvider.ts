import type { Receipt, TradeReading } from './contract';
import { parseTrades } from './parseTrades';
import { parseReceipt } from './parseEnvelope';
import { TargetTrades, TargetDraftTradeAccept } from '../../../wailsjs/go/main/App';

export async function trades(franchiseId: string): Promise<TradeReading> {
  return parseTrades(await TargetTrades(franchiseId));
}
export async function draftTradeAccept(franchiseId: string, tradeId: string): Promise<Receipt> {
  return parseReceipt(await TargetDraftTradeAccept(franchiseId, tradeId));
}
