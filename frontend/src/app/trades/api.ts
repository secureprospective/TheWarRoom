export type TradeBuilderRequest = {
  mine: string; other: string; action: string; id: string; give: string[]; get: string[]; rationale: string;
};

const unavailable = async (): Promise<never> => {
  throw new Error('Trades is unavailable in this view; no sample data is substituted.');
};

export const targetTrades = (_franchiseId: string) => unavailable();
export const targetDraftTradeAccept = (_franchiseId: string, _tradeId: string) => unavailable();
export const targetTradeBuilderRead = (_franchiseId: string) => unavailable();
export const targetTradeBuilderPreview = (_request: TradeBuilderRequest) => unavailable();
export const targetTradeBuilder = (_request: TradeBuilderRequest) => unavailable();
