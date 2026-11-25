import type { ReportProvider } from './provider';

const unavailable = async (): Promise<never> => {
  throw new Error('Draft reports require the desktop app; no browser fixture is substituted.');
};

export const provider: ReportProvider = {
  read: unavailable,
  refresh: unavailable,
  openMFL: unavailable,
};

export const readNFLDraft = unavailable;
export const subscribeDraft = (_changed: () => void) => () => {};
