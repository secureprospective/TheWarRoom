import { createReadQuery } from '../data/query';
import { initialScope, readOffers, type OfferReading, type Scope } from './provider';
export type OfferFilter = {
  field: 'direction' | 'window' | 'range' | 'team' | 'retry';
  value: string;
};
export function createOfferState(read = readOffers) {
  const query = createReadQuery('offers.TargetOffers', {
    initial: (): { scope: Scope; reading?: OfferReading } => ({ scope: initialScope }),
    read, accept: reading => ({ reading }),
  });
  const store = query.store;
  let opening: Promise<void> | undefined;
  function refresh(scope: Scope) {
    store.setState({ scope });
    return query.load([scope]);
  }
  return {
    use: query.use,
    peek: store.getState,
    refresh: () => refresh(store.getState().scope),
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
