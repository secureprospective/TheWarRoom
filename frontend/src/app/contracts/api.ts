const unavailable = async (): Promise<never> => {
  throw new Error('Contracts and cap are unavailable in this view; no sample data is substituted.');
};

export const targetContracts = (_scope: unknown): Promise<unknown> => unavailable();
export const targetPlayerContract = (_id: string): Promise<unknown> => unavailable();
export const targetContractActs = (_raw: unknown): Promise<unknown> => unavailable();
export const targetReleasePlans = (_raw: unknown): Promise<unknown> => unavailable();
export const targetWhatIf = (_raw: unknown): Promise<unknown> => unavailable();
export const targetLineup = (_franchiseId: string): Promise<unknown> => unavailable();
export const targetCheckLineup = (_franchiseId: string, _starters: string[]):
  Promise<unknown> => unavailable();
export const targetDraftLineup = (_franchiseId: string, _starters: string[]):
  Promise<unknown> => unavailable();
export const targetRosterReports = (_mine: string): Promise<unknown> => unavailable();
export const targetAcquisition = (_raw: unknown): Promise<unknown> => unavailable();
