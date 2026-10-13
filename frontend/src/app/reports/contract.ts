import { array, choice, object, text } from '../data/parseValues';
import type { MarketRange } from '../market/contract';
import { endpointById } from '../registry';
import type { EndpointId } from '../registry';

export type ReportRow = {
  cells: string[]; range: MarketRange | null; sort?: number[]; titles?: string[];
  mine?: boolean; bars?: number[]; group?: string;
};
export type ReportEntity = { text: string; kind: 'player' | 'franchise'; id: string };
export type Report = {
  entities?: ReportEntity[];
  id: string; title: string; note: string; status: 'ok' | 'empty' | 'failed' | 'stale';
  source: string; asOf: string; how: string; columns: string[];
  sortAscending?: string; sortDescending?: string;
  rows: ReportRow[]; summary: ReportRow[]; link: { label: string; endpoint: EndpointId } | null;
  barMax?: number; barLabels?: string[];
};
export type Board = {
  title: string; loading: string; refresh: string; howLabel: string; mflLabel: string;
  mflUrl: string; reports: Report[];
};
export function range(v: unknown, p: string): MarketRange | null {
  if (v == null) return null;
  const r = object(v, p, ['value', 'lo', 'hi', 'text', 'max']);
  const finite = (n: unknown): number => {
    if (typeof n !== 'number' || !Number.isFinite(n) || n < 0) throw new Error(`${p}: invalid range`);
    return n;
  };
  const result = { value: finite(r.value), lo: finite(r.lo), hi: finite(r.hi), text: text(r.text, p),
    ...(r.max === undefined ? {} : { max: finite(r.max) }) };
  if (result.max !== undefined && (result.max <= 0 || result.hi > result.max))
    throw new Error(`${p}: invalid range axis`);
  if (result.lo > result.value || result.value > result.hi) throw new Error(`${p}: reversed range`);
  return result;
}
export function report(v: unknown, p: string): Report {
  const r = object(v, p, ['id', 'title', 'note', 'status', 'source', 'asOf', 'how',
    'columns', 'rows', 'summary', 'link', 'sortAscending', 'sortDescending',
    'barMax', 'barLabels', 'entities']);
  const columns = array(r.columns, p, text);
  const barLabels = r.barLabels === undefined ? undefined : array(r.barLabels, p, text);
  const barMax = r.barMax === undefined ? undefined : r.barMax;
  if (barMax !== undefined && (typeof barMax !== 'number' || !Number.isFinite(barMax) || barMax <= 0))
    throw new Error(`${p}: invalid chart scale`);
  return {
    entities: r.entities === undefined ? undefined : array(r.entities, p, (v, p) => {
      const e = object(v, p, ['text', 'kind', 'id']);
      const id = text(e.id, p), name = text(e.text, p);
      if (!/^\d{4,5}$/.test(id) || !name) throw new Error(`${p}: invalid entity identity`);
      return { text: name, kind: choice(e.kind, p, ['player', 'franchise']), id };
    }),
    id: text(r.id, p), title: text(r.title, p), note: text(r.note, p),
    status: choice(r.status, p, ['ok', 'empty', 'failed', 'stale']), source: text(r.source, p),
    asOf: text(r.asOf, p), how: text(r.how, p), columns,
    sortAscending: r.sortAscending === undefined ? undefined : text(r.sortAscending, p),
    sortDescending: r.sortDescending === undefined ? undefined : text(r.sortDescending, p),
    link: r.link === null ? null : (() => {
      const l = object(r.link, `${p}.link`, ['label', 'endpoint']);
      const endpoint = text(l.endpoint, `${p}.link`);
      endpointById(endpoint as EndpointId);
      return { label: text(l.label, `${p}.link`), endpoint: endpoint as EndpointId };
    })(),
    ...(barLabels && barMax !== undefined ? { barLabels, barMax } : {}),
    summary: r.summary === null ? [] : array(r.summary, p, (v, p) => {
      const row = object(v, p, ['cells', 'range', 'sort', 'titles']);
      const cells = array(row.cells, p, text);
      if (cells.length !== 2 || row.range !== null) throw new Error(`${p}: invalid status line`);
      return { cells, range: null };
    }),
    rows: array(r.rows, p, (v, p) => {
      const row = object(v, p, ['cells', 'range', 'sort', 'titles', 'mine', 'bars', 'group']);
      const cells = array(row.cells, p, text);
      if (cells.length !== columns.length) throw new Error(`${p}: column count mismatch`);
      const sort = row.sort === undefined ? undefined : array(row.sort, p, (v, p) => {
        if (typeof v !== 'number' || !Number.isFinite(v)) throw new Error(`${p}: invalid sort key`);
        return v;
      });
      const titles = row.titles === undefined ? undefined : array(row.titles, p, text);
      if ((sort && sort.length !== columns.length) || (titles && titles.length !== columns.length))
        throw new Error(`${p}: column metadata mismatch`);
      const bars = row.bars === undefined ? undefined : array(row.bars, p, (v, p) => {
        if (typeof v !== 'number' || !Number.isFinite(v) || v < 0 || barMax === undefined || v > barMax)
          throw new Error(`${p}: bar outside the chart scale`);
        return v;
      });
      if (bars && bars.length !== barLabels?.length) throw new Error(`${p}: bar count mismatch`);
      if (row.mine !== undefined && typeof row.mine !== 'boolean') throw new Error(`${p}: invalid mine flag`);
      if (row.group !== undefined && (typeof row.group !== 'string' || !row.group))
        throw new Error(`${p}: invalid group`);
      return { cells, range: range(row.range, p), sort, titles,
        ...(row.mine ? { mine: true } : {}), ...(bars ? { bars } : {}),
        ...(row.group ? { group: row.group } : {}) };
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
