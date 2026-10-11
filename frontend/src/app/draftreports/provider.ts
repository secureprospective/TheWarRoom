import { array, choice, object, text } from '../data/parseValues';
import type { MarketRange } from '../market/contract';

export type ReportRow = { cells: string[]; range: MarketRange | null };
export type Report = {
  id: string; title: string; note: string; status: 'ok' | 'empty' | 'failed' | 'stale';
  source: string; asOf: string; how: string; columns: string[];
  rows: ReportRow[]; summary: ReportRow[]; seeEveryPick: string;
};
export type Board = {
  title: string; loading: string; refresh: string; howLabel: string; mflLabel: string;
  mflUrl: string; reports: Report[];
};
function range(v: unknown, p: string): MarketRange | null {
  if (v == null) return null;
  const r = object(v, p, ['value', 'lo', 'hi', 'text']);
  const finite = (n: unknown): number => {
    if (typeof n !== 'number' || !Number.isFinite(n) || n < 0) throw new Error(`${p}: invalid range`);
    return n;
  };
  const result = { value: finite(r.value), lo: finite(r.lo), hi: finite(r.hi), text: text(r.text, p) };
  if (result.lo > result.value || result.value > result.hi) throw new Error(`${p}: reversed range`);
  return result;
}
function report(v: unknown, p: string): Report {
  const r = object(v, p, ['id', 'title', 'note', 'status', 'source', 'asOf', 'how', 
    'columns', 'rows', 'summary', 'seeEveryPick']);
  const columns = array(r.columns, p, text);
  return {
    id: text(r.id, p), title: text(r.title, p), note: text(r.note, p),
    status: choice(r.status, p, ['ok', 'empty', 'failed', 'stale']), source: text(r.source, p),
    asOf: text(r.asOf, p), how: text(r.how, p), columns,
    seeEveryPick: text(r.seeEveryPick, p),
    summary: r.summary === null ? [] : array(r.summary, p, (v, p) => {
      const row = object(v, p, ['cells', 'range']);
      const cells = array(row.cells, p, text);
      if (cells.length !== 2 || row.range !== null) throw new Error(`${p}: invalid status line`);
      return { cells, range: null };
    }),
    rows: array(r.rows, p, (v, p) => {
      const row = object(v, p, ['cells', 'range']);
      const cells = array(row.cells, p, text);
      if (cells.length !== columns.length) throw new Error(`${p}: column count mismatch`);
      return { cells, range: range(row.range, p) };
    }),
  };
}
export function parseBoard(v: unknown): Board {
  const p = 'Draft reports';
  const r = object(v, p, ['title', 'loading', 'refresh', 'howLabel', 'mflLabel', 'mflUrl', 'reports']);
  const mflUrl = text(r.mflUrl, p);
  if (mflUrl && !/^https:\/\/[^/]+\/\d{4}\/options\?L=\d+&O=17$/.test(mflUrl))
    throw new Error(`${p}: unexpected MFL read link`);
  return {
    title: text(r.title, p), loading: text(r.loading, p), refresh: text(r.refresh, p),
    howLabel: text(r.howLabel, p), mflLabel: text(r.mflLabel, p), mflUrl,
    reports: array(r.reports, p, report),
  };
}
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
