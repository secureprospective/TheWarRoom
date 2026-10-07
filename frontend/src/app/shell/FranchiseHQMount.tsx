import { lazy, Suspense } from 'react';
import type { Snapshot } from '../data/contract';

let loaded: typeof import('./FranchiseHQ').FranchiseHQ | undefined;
export function prefetchHQ() {
  return import('./FranchiseHQ').then((module) => {
    loaded = module.FranchiseHQ;
    return { default: module.FranchiseHQ };
  });
}
const LazyHQ = lazy(prefetchHQ);
export function FranchiseHQMount({ snapshot }: { snapshot: Snapshot }) {
  const View = loaded ?? LazyHQ;
  return (
    <Suspense fallback={<p role="status">Opening Franchise HQ…</p>}>
      <View snapshot={snapshot} />
    </Suspense>
  );
}
