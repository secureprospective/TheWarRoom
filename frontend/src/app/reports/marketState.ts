import { createReportState } from '../draftreports/state';
import { parseBoard } from './contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Player market reports require the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
export const marketReportState = createReportState({
  read: async () => parseBoard(await bindings().TargetMarketReports()),
  refresh: async force => bindings().TargetMarketReportsRefresh(force),
  openMFL: async () => { throw new Error('Player market reports do not open an owner page.'); },
});
