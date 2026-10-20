import { parseBoard } from '../reports/contract';
import type { Board } from '../reports/contract';
export { parseBoard } from '../reports/contract';
export type { Board } from '../reports/contract';
export type ReportProvider = {
  read(): Promise<Board>; refresh(force: boolean): Promise<void>; openMFL(): Promise<void>;
};
const bindings = async () => {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Draft reports require the desktop app; no browser fixture is substituted.');
  return import('../../../wailsjs/go/main/App');
};
export const provider: ReportProvider = {
  read: async () => parseBoard(await (await bindings()).TargetDraftReports()),
  refresh: async (force) => (await bindings()).TargetDraftReportsRefresh(force),
  openMFL: async () => (await bindings()).TargetOpenMFLDraftRoom(),
};
