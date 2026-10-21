import type { Board } from '../reports/contract';
export { parseBoard } from '../reports/contract';
export type { Board } from '../reports/contract';
export type ReportProvider = {
  read(): Promise<Board>; refresh(force: boolean): Promise<void>; openMFL(): Promise<void>;
};
export { provider } from './api';
