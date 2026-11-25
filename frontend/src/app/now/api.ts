const unavailable = async (): Promise<never> => {
  throw new Error('League Pulse is unavailable in this view; no sample data is substituted.');
};

export const targetNow = (_week: number, _reportWeek: number, _mine: string) => unavailable();
export const targetNowRefresh = (_week: number, _force: boolean) => unavailable();
export const targetLiveGame = (_open: boolean, _follow: string, _mine: string) => unavailable();
