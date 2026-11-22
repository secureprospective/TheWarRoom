import { createReadQuery } from '../data/query';
import { provider, type Board, type ReportProvider } from './provider';

export function createReportState(reader: ReportProvider,
  command: 'draft.TargetDraftReports' | 'market.TargetMarketReports' = 'draft.TargetDraftReports') {
  const query = createReadQuery(command, {
    initial: (): { board?: Board } => ({}), read: () => reader.read(), accept: board => ({ board }),
  });
  const refresh = (force = false) => query.load([], { before: () => reader.refresh(force) });
  async function action(action: 'refresh' | 'mfl') {
    if (action === 'refresh') return refresh(true);
    query.set({ error: '' });
    try { await reader.openMFL(); }
    catch (cause) { query.set({ error: String(cause) }); }
  }
  return { use: query.use, peek: query.peek, refresh, action };
}
export const reportState = createReportState(provider);
