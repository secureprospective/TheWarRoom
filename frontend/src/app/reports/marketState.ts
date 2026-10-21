import { createReportState } from '../draftreports/state';
import { parseBoard } from './contract';
import { marketAPI } from '../market/api';

export const marketReportState = createReportState({
  read: async () => parseBoard(await marketAPI.reports()),
  refresh: async force => marketAPI.refreshReports(force),
  openMFL: async () => { throw new Error('Player market reports do not open an owner page.'); },
});
