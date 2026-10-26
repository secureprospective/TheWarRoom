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

export type ContractsAction = ContractsFilter | {
  field: 'whatif'; value: import('./whatif').WhatIfAction;
} | { field: 'release'; value: import('./releases').ReleaseAction }
  | { field: 'release-confirm'; value: string }
  | { field: 'contract-close'; value: string }
  | { field: 'contract-open'; value: { mine: string; player: string } }
  | { field: 'contract-keep'; value: import('./contractActState').ContractActRequest }
  | { field: 'contract-act'; value: import('./contractActState').ContractActRequest }
  | { field: 'contract-reveal'; value: string };
export async function runContractsAction(action: ContractsAction) {
  if (action.field === 'contract-close') {
    (await import('./contractActState')).useContractMove.setState({
      player: '', kept: undefined, keptTerms: '', error: '',
    });
  } else if (action.field === 'contract-open') {
    (await import('./contractActState')).openContractMove(action.value.mine, action.value.player);
  } else if (action.field === 'contract-keep') {
    const m = await import('./contractActState');
    await m.keepContractMove(action.value);
    if (m.useContractMove.getState().error) throw new Error(m.useContractMove.getState().error);
  } else if (action.field === 'contract-act') {
    await (await import('./contractActState')).runContractAct(action.value);
  } else if (action.field === 'contract-reveal') {
    (await import('./contractActState')).revealContractPlan(action.value);
  }
  else if (action.field === 'whatif') await (await import('./whatif')).changeWhatIf(action.value);
  else if (action.field === 'release-confirm') (await import('./releases')).confirmRelease(action.value);
  else if (action.field === 'release') {
    const { releaseAction, copyReleaseText } = await import('./releases');
    if (action.value.action === 'handoff') {
      const board = await releaseAction({ mine: action.value.mine, action: 'read' });
      const plan = board.value?.find(plan => plan.id === action.value.id);
      if (plan && !plan.cut) {
        if (!await copyReleaseText(plan.post)) {
          throw new Error('Clipboard unavailable; select the request text.');
        }
      }
    }
    await releaseAction(action.value);
    (await import('./whatif')).notifyWhatIf();
  } else setContractsFilter(action);
}
