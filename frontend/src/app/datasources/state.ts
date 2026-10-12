import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { dataProvider, type DataProvider, type Sources } from './provider';
import type { Report } from '../reports/contract';
import type { main } from '../../../wailsjs/go/models';
export type ParamAction = { kind: 'parameter'; key: string; position: string; value: number | null };
export type DataAction = { kind: 'read' } | { kind: 'folder'; path: string } |
  { kind: 'refresh'; scope: string };
export type Action = DataAction | ParamAction;
export function createDataState(provider: DataProvider) {
  const store = createStore<{ data?: Sources; crosswalk?: Report; params?: main.ParamsResult;
    busy: string; error: string; result: string }>(() => ({ busy: '', error: '', result: '' }));
  async function read() {
    const data = await provider.read();
    store.setState({ data });
    const [crosswalk, params] = await Promise.all([provider.crosswalk(), provider.params()]);
    store.setState({ crosswalk, params });
  }
  async function run(action: Action) {
    if (store.getState().busy) return;
    const busy = action.kind === 'refresh' ? action.scope : action.kind;
    store.setState({ busy, error: '', result: '' });
    try {
      if (action.kind === 'refresh' || action.kind === 'folder') {
        const result = action.kind === 'refresh' ? await provider.refresh(action.scope)
          : await provider.folder(action.path);
        store.setState({ result: result.summary });
        if (!result.ok) store.setState({ error: result.error });
      } else if (action.kind === 'parameter') {
        await provider.parameter(action.key, action.position, action.value);
      }
      await read();
    } catch (cause) {
      store.setState({ error: String(cause) });
    } finally { store.setState({ busy: '' }); }
  }
  return { use: () => useStore(store), read: store.getState, run };
}
export const dataState = createDataState(dataProvider);
