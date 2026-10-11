import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import type { Scope } from './contract';

type Filters = Omit<Scope, 'mine'>;
export type ContractsFilter =
  | { field: 'team' | 'position'; value: string }
  | { field: 'sort'; value: Scope['sort'] }
  | { field: 'desc'; value: boolean };
export function createContractsFilters() {
  const store = createStore<Filters>(() => ({ team: '', position: '', sort: 'name', desc: false }));
  return {
    use: () => useStore(store),
    set: (change: ContractsFilter) => store.setState({ [change.field]: change.value }),
  };
}
const filters = createContractsFilters();
export const useContractsFilters = filters.use;
export const setContractsFilter = filters.set;
