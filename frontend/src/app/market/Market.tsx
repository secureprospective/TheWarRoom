import { lazy, Suspense } from 'react';
import type { Snapshot } from '../data/contract';
import type { MarketFilter } from './contract';
import './market.css';
const Pool = lazy(() => import('../pool/Pool'));
const AssetBoardView = lazy(() =>
  import('../assets/AssetBoardView').then(m => ({ default: m.AssetBoardView })));
const PoolInspector = lazy(() => import('../pool/Pool').then(m => ({ default: m.PoolInspector })));
export default function Market({ kind, snapshot }: { kind?: string; snapshot?: Snapshot }) {
  if (kind?.startsWith('pool:'))
    return <Suspense fallback={null}><PoolInspector id={kind.slice(5)} /></Suspense>;
  if (kind) return null;
  return <Suspense fallback={<p role="status">Opening the free-agent pool…</p>}>
    {snapshot && <AssetBoardView snapshot={snapshot} />}<Pool snapshot={snapshot} />
  </Suspense>;
}
export const set = async (change: MarketFilter) => {
  if (change.field.startsWith('pool-')) return import('../pool/Pool').then(m => m.set(change));
  throw new Error('Historical market is unavailable');
};
