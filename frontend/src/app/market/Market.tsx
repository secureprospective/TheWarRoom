import { lazy, Suspense, useEffect } from 'react';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { assets, assetLabels, phases } from './contract';
import { marketState, prefetchMarket } from './state';
import { ReadingView } from './ReadingView';
import './market.css';

import type { Snapshot } from '../data/contract';
const Counterparties = lazy(() => import('../counterparties/Counterparties'));
const OffersHistory = lazy(() => import('../offers/Offers'));
const Draft = lazy(() => import('./Draft'));
const Pool = lazy(() => import('../pool/Pool'));
const AssetBoardView = lazy(() =>
  import('../assets/AssetBoardView').then((m) => ({ default: m.AssetBoardView })),
);
const PoolInspector = lazy(() =>
  import('../pool/Pool').then((m) => ({ default: m.PoolInspector })),
);
export default function Market({ kind, snapshot }: { kind?: string; snapshot?: Snapshot }) {
  const s = commands.use();
  if (kind?.startsWith('team:') || (!kind && s.workspace[s.node] === 'counterparties'))
    return (
      <Suspense fallback={<p role="status">Opening Counterparties…</p>}>
        <Counterparties fid={kind?.slice(5)} snapshot={snapshot} />
      </Suspense>
    );
  if (!kind && s.workspace[s.node] === 'draft-room')
    return <Suspense fallback={null}><Draft snapshot={snapshot} /></Suspense>;
  if (!kind && s.workspace[s.node] === 'offers-history')
    return <Suspense fallback={null}><OffersHistory /></Suspense>;
  if (!kind && s.workspace[s.node] === 'valuations-and-pool')
    return (
      <Suspense fallback={<p role="status">Opening the free-agent pool…</p>}>
        {snapshot && <AssetBoardView snapshot={snapshot} />}
        <Pool snapshot={snapshot} />
      </Suspense>
    );
  if (kind?.startsWith('pool:'))
    return <Suspense fallback={null}><PoolInspector id={kind.slice(5)} /></Suspense>;
  if (kind) return <MarketInspector kind={kind} />;
  return <MarketWorkspace />;
}
export function MarketWorkspace() {
  const s = commands.use();
  const draft = s.endpointIds.some((id) => /^L-(1[5-9]|2[0-2])$/.test(id));
  return (
    <>
      <div className="market-controls" aria-label="Market views">
        <Act verb="endpoint.open" args={{ id: 'L-01' }} label="Trade Market" active={!draft}>
          Trade Market
        </Act>
        <Act verb="endpoint.open" args={{ id: 'L-15' }}
          label="Draft lab" active={draft}>
          Draft lab
        </Act>
      </div>
      {draft ? (
        <Suspense fallback={<p role="status">Opening Draft lab…</p>}>
          <Draft />
        </Suspense>
      ) : (
        <MarketScreen />
      )}
    </>
  );
}
function MarketScreen() {
  const s = marketState.use();
  useEffect(prefetchMarket, []);
  if (!s.lists) return null;
  const windowLabels: Record<string, string> = {
    spring: 'Offseason',
    summer: 'Rookie draft through kickoff',
    season: 'In season',
  };
  return (
    <div className="market">
      <h4>What the league pays for wins</h4>
      <p>
        Historical trades, not live quotes. 100% is the baseline; each bar is the likely range, with the
        baseline marked.
      </p>
      <section className="market-filters" aria-label="Market filters">
        <h5>Time of year</h5>
        <div className="market-controls" data-endpoint="L-06">
          <Act verb="market" args={{ field: 'reset', value: '' }} label="Show all year and assets">
            Everything
          </Act>
          {(['spring', 'summer', 'season'] as const).map((id) => (
            <Act
              key={id}
              verb="market"
              args={{ field: 'window', value: id }}
              label={`Choose ${windowLabels[id]}`}
            >
              {windowLabels[id]}
            </Act>
          ))}
        </div>
        <div className="market-controls">
          {phases.map(([id, label]) => (
            <Act
              key={id}
              verb="market"
              args={{ field: 'phase', value: id }}
              active={s.scope.phases?.includes(id) ?? false}
              label={`Filter ${label}`}
            >
              {label}
            </Act>
          ))}
        </div>
        <h5>Assets</h5>
        <div className="market-controls" data-endpoint="L-07">
          {assets.map((kind) => (
            <span key={kind}>
              <Act
                verb="market"
                args={{ field: 'asset', value: kind }}
                active={s.scope.assets?.includes(kind) ?? false}
                label={`Filter ${assetLabels[kind] ?? kind}`}
              >
                {assetLabels[kind] ?? kind}
              </Act>
              <Act
                verb="market"
                args={{ field: 'inspect', value: kind }}
                label={`Inspect ${assetLabels[kind] ?? kind}`}
                variant="icon"
              >
                Inspect
              </Act>
            </span>
          ))}
        </div>
        <div className="market-controls" data-endpoint="L-08">
          {['together', 'split'].map((value) => (
            <Act
              key={value}
              verb="market"
              args={{ field: 'age', value }}
              active={s.scope.byAge === (value === 'split')}
              label={value === 'split' ? 'Split players by age' : 'Players together'}
            >
              {value === 'split' ? 'Split by age' : 'Together'}
            </Act>
          ))}
        </div>
        {canRetry(s.lists, s.clock, s.stretches) && (
          <Act verb="market" args={{ field: 'retry', value: '' }} label="Retry Market archive reading">
            Retry reading
          </Act>
        )}
        {s.pending && s.lists && (
          <p role="status">
            Updating filters; the figures below still show the previous scope until Go replies.
          </p>
        )}
      </section>
      <ReadingView reading={s.lists} title="Going rate and buy / sell lists" inspect />
      <ReadingView reading={s.clock} title="Pick clock" />
      <ReadingView reading={s.stretches} title="Trades per season by stretch" />
    </div>
  );
}
export function MarketInspector({ kind }: { kind: string }) {
  const s = marketState.use();
  useEffect(() => {
    void marketState.profile(kind);
  }, [kind, s.applied]);
  if (!s.profiles[kind]) return null;
  return (
    <div className="market">
      <ReadingView reading={s.profiles[kind]} title={`Asset profile · ${kind}`} inspect />
      {kind.startsWith('pick') && (
        <Suspense fallback={<p role="status">Opening draft pick history…</p>}>
          <Draft kind={kind} />
        </Suspense>
      )}
    </div>
  );
}

export { prefetchMarket as warm } from './state';
export const set = (change: import('./contract').MarketFilter) => {
  if (change.field.startsWith('counterpart-'))
    return import('../counterparties/state').then((m) => m.counterState.change(change));
  if (change.field.startsWith('draft-')) return import('./Draft').then((m) => m.set(change));
  if (change.field.startsWith('pool-')) return import('../pool/Pool').then((m) => m.set(change));
  return marketState.change(change);
};

export function canRetry(...readings: (import('./contract').SurfaceReading | undefined)[]) {
  return readings.some((r) => r?.status === 'failed');
}
