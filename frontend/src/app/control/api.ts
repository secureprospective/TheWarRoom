const unavailable = async (): Promise<never> => {
  throw new Error('Control Room requires the desktop app; no browser fixture is substituted.');
};

export const controlAPI = {
  settings: unavailable, bylaws: unavailable, accounting: unavailable, news: unavailable,
  openAccounting: unavailable,
};
export const readAlerts = (_id: string) => unavailable();
export const getParams: () => Promise<ParamsResult> = unavailable;
export const setParam = (_key: string, _position: string, _value: number): 
  Promise<ParamsResult> => unavailable();
export const resetParam = (_key: string, _position: string): 
  Promise<ParamsResult> => unavailable();

export type ParamView = {
  key: string; position: string; value: number; default: number; min: number; max: number;
  label: string; group: string; description: string; calibrated: boolean; overridden: boolean;
};
export type ParamsResult = { ok: boolean; error: string; params: ParamView[] };
