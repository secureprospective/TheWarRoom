import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import type { ReportRow } from './contract';

type Sort = { column: number; desc: boolean };
const store = createStore<Record<string, Sort>>(() => ({}));
export function sortedRows(rows: ReportRow[], sort?: Sort): ReportRow[] {
  if (!sort) return rows;
  return rows.map((row, index) => ({ row, index })).sort((a, b) => {
    const order = (a.row.sort?.[sort.column] ?? 0) - (b.row.sort?.[sort.column] ?? 0);
    return (sort.desc ? -order : order) || a.index - b.index;
  }).map(({ row }) => row);
}
export const reportSort = {
  use: () => useStore(store),
  set: ({ id, column }: { id: string; column: number }) => {
    if (!['M-046', 'M-047', 'M-067', 'M-069'].includes(id) ||
      !Number.isInteger(column) || column < 0 || column >= (id === 'M-047' ? 4 : 2))
      throw new Error('Report sort: invalid column');
    const before = store.getState()[id];
    store.setState({ [id]: { column, desc: before?.column === column && !before.desc } });
  },
};
