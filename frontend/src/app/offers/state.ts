import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { initialScope, readOffers, type OfferReading, type Scope } from './provider';
export type OfferFilter = {
  field: 'direction' | 'window' | 'range' | 'team' | 'retry';
  value: string;
};
export function createOfferState(read = readOffers) {
  const store = createStore<{
    scope: Scope; reading?: OfferReading; pending: boolean;
  }>(() => ({ scope: initialScope, pending: false }));
  let revision = 0;
  let opening: Promise<void> | undefined;
  async function refresh(scope: Scope) {
    const token = ++revision;
    store.setState({ scope, pending: true });
    const reading = await read(scope);
    if (token === revision) store.setState({ reading, pending: false });
  }
  return {
    use: () => useStore(store),
    peek: store.getState,
    first: () => {
      if (!opening) opening = refresh(initialScope);
      if (!store.getState().reading) throw opening;
    },
    change: (filter: OfferFilter) => {
      const s = store.getState();
      const scope = { ...s.scope };
      const choices = s.reading?.value;
      if (filter.field === 'direction') {
        if (!choices?.directions.some(p => p.id === filter.value)) throw new Error('Unknown direction');
        scope.direction = filter.value;
      }
      if (filter.field === 'window') {
        if (!choices?.windows.some(p => p.id === filter.value)) throw new Error('Unknown window');
        scope.window = filter.value;
      }
      if (filter.field === 'team') {
        if (!choices?.teams.some(p => p.fid === filter.value)) throw new Error('Unknown counterparty');
        scope.team = filter.value;
      }
      if (filter.field === 'range') {
        const p = choices?.ranges.find(p => p.id === filter.value);
        if (!p) throw new Error('Unknown range');
        scope.from = p.from; scope.to = p.to;
      }
      return refresh(scope);
    },
  };
}
export const offerState = createOfferState();
