import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { assets, assetLabels, phases } from './contract';
import { marketState, prefetchMarket } from './state';
import { ReadingView } from './ReadingView';
import './market.css';

export default function Market({ kind }: { kind?: string }) {
  if (kind) return <MarketInspector kind={kind} />;
  return <MarketScreen />;
}
function MarketScreen() {
  const s = marketState.use();
  useEffect(prefetchMarket, []);
  if (!s.lists) return null;
  return (
    <div className="market">
      <h4>What the league pays for wins</h4>
      <p>
        Historical trades, not live quotes. 100% is the baseline; each bar is the likely range, with
        the baseline marked.
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
              label={`Choose ${id}`}
            >
              {{ spring: 'Offseason', summer: 'Draft to kickoff', season: 'In season' }[id]}
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
        <Act
          verb="market"
          args={{ field: 'retry', value: '' }}
          label="Retry Market archive reading"
        >
          Retry reading
        </Act>
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
    </div>
  );
}

export { prefetchMarket as warm } from './state';
export const set = marketState.change;
