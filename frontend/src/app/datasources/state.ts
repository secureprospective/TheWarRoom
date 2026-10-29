import type { ParamsResult } from '../control/api';
import { createReadQuery } from '../data/query';
import { dataProvider, type DataProvider, type Sources } from './provider';
import type { Report } from '../reports/contract';
export type ParamAction = { kind: 'parameter'; key: string; position: string; value: number | null };
export type DataAction = { kind: 'read' } | { kind: 'folder'; path: string } |
  { kind: 'refresh'; scope: string } | { kind: 'photos'; on: boolean };
export type Action = DataAction | ParamAction;
export function createDataState(provider: DataProvider) {
  type State = { data?: Sources; crosswalk?: Report; params?: ParamsResult;
    busy: string; result: string; outcomes: Record<string, string> };
  const initial = (): State => ({ busy: '', result: '', outcomes: {} });
  const sources = createReadQuery('sources.TargetDataSources', {
    initial, read: () => provider.read(), accept: data => ({ data }),
  });
  const details = createReadQuery('sources.TargetCrosswalk', {
    initial, store: sources.store, companions: ['control.GetParams'],
    read: async () => {
      const [crosswalk, params] = await Promise.all([provider.crosswalk(), provider.params()]);
      return { crosswalk, params };
    }, accept: value => value,
  });
  const paramsOnly = createReadQuery('control.GetParams', {
    initial, store: sources.store, read: () => provider.params(), accept: params => ({ params }),
  });
  const store = sources.store;
  async function read() {
    const actionError = store.getState().error;
    await sources.load([]);
    if (store.getState().error) return store.getState().error;
    await details.load([]);
    if (store.getState().error) return store.getState().error;
    store.setState({ error: actionError });
    return '';
  }
  async function run(action: Action) {
    if (store.getState().busy) return;
    const busy = action.kind === 'refresh' ? action.scope : action.kind;
    store.setState(s => ({ busy, error: '', result: '', outcomes: { ...s.outcomes, [busy]: '' } }));
    try {
      if (action.kind === 'refresh' || action.kind === 'folder' || action.kind === 'photos') {
        const result = action.kind === 'refresh' ? await provider.refresh(action.scope)
          : action.kind === 'photos' ? await provider.photos(action.on) : await provider.folder(action.path);
        store.setState(s => ({ result: result.summary, outcomes: { ...s.outcomes,
          [busy]: result.ok ? result.summary : result.error } }));
        if (!result.ok) store.setState({ error: result.error });
      } else if (action.kind === 'parameter') {
        await provider.parameter(action.key, action.position, action.value);
        await paramsOnly.load([]);
        if (!store.getState().error) store.setState({
          result: action.value === null ? 'Default restored.' : 'Setting saved.',
        });
        return;
      }
      const error = await read();
      if (error) {
        store.setState(s => ({ error, outcomes: { ...s.outcomes, [busy]: error } }));
        return;
      }
      if (action.kind === 'read') store.setState(s => ({ outcomes: { ...s.outcomes,
        read: s.data!.words.readResult } }));
    } catch (cause) {
      store.setState(s => ({ error: String(cause), outcomes: { ...s.outcomes, [busy]: String(cause) } }));
    } finally { store.setState({ busy: '' }); }
  }
  return { use: sources.use, read: store.getState, run, loadParams: () => paramsOnly.load([]) };
}
export const dataState = createDataState(dataProvider);
