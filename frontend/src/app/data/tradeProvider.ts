import type { Receipt, TradeReading } from './contract';
import { parseTrades } from './parseTrades';
import { parseReceipt } from './parseEnvelope';
import { targetDraftTradeAccept, targetTrades } from '../trades/api';

export async function trades(franchiseId: string): Promise<TradeReading> {
  return parseTrades(await targetTrades(franchiseId));
}
export async function draftTradeAccept(franchiseId: string, tradeId: string): Promise<Receipt> {
  return parseReceipt(await targetDraftTradeAccept(franchiseId, tradeId));
}
