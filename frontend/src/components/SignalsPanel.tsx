import { useEffect } from 'react';
import { useSignalsStore } from '../store/signals';
import { main } from '../../wailsjs/go/models';

// SignalsPanel shows every data source's health, each signal's freshness and reach by season,
// and how many rostered players at each position it has data for. An error shows while it is
// newer than the source's last good load.
const cell = { padding: '4px 10px', fontFamily: 'var(--mono)', fontSize: 12 } as const;
const head = { margin: 0, fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-secondary)' } as const;
const positions = ['QB', 'RB', 'WR', 'TE', 'K', 'DT', 'DE', 'LB', 'CB', 'S'];

function when(iso: string) {
  return iso ? new Date(iso).toLocaleString() : 'never';
}

function share(s: main.PositionShare | undefined) {
  if (!s || s.rostered === 0) return '—';
  return `${Math.round((100 * s.withData) / s.rostered)}%`;
}

function coverageLabel(f: main.FeedView) {
  if (f.coverageSeason === -1) return 'all seasons';
  if (f.coverageSeason === 0) return 'player facts';
  return String(f.coverageSeason);
}

export function SignalsPanel() {
  const { view, loading, error, read, load } = useSignalsStore();
  useEffect(() => {
    void read();
  }, [read]);
  const seasons = Array.from(new Set((view?.feeds ?? []).flatMap((f) => f.seasons.map((s) => s.season))))
    .filter((s) => s > 0)
    .sort();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <button type="button" className="twr-btn" disabled={loading} onClick={() => void load()}>
          {loading ? 'Loading…' : 'Load signals'}
        </button>
        <button type="button" className="twr-btn" disabled={loading} onClick={() => void read()}>
          Refresh view
        </button>
        {view?.lastLoad?.finishedAt && (
          <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-tertiary)' }}>
            last load {when(view.lastLoad.finishedAt)} · {view.lastLoad.loads.filter((l) => l.status === 'loaded').length}{' '}
            files loaded · {view.lastLoad.loads.filter((l) => l.status === 'failed').length} failed
          </span>
        )}
      </div>
      {error && <div className="twr-banner twr-banner--warn">{error}</div>}
      {view?.ok && (
        <>
          <h3 style={head}>SOURCES</h3>
          <table style={{ borderCollapse: 'collapse', color: 'var(--text-primary)', width: 'fit-content' }}>
            <tbody>
              {view.sources.map((s) => (
                <tr key={s.source} style={{ borderTop: '1px solid var(--hairline)' }}>
                  <td style={cell}>{s.source}</td>
                  <td style={cell}>{s.name}</td>
                  <td style={{ ...cell, color: s.state === 'lost' ? 'var(--amber-base)' : undefined }}>{s.state}</td>
                  <td style={cell}>last good load {when(s.lastSuccess)}</td>
                  <td style={{ ...cell, color: 'var(--amber-base)' }}>{s.lastError}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <h3 style={head}>SIGNALS · PLAYERS WITH DATA BY SEASON (WAITING FOR A MATCH)</h3>
          <table style={{ borderCollapse: 'collapse', color: 'var(--text-primary)', width: 'fit-content' }}>
            <thead>
              <tr style={{ color: 'var(--text-secondary)', textAlign: 'left' }}>
                <th style={cell}>Signal</th>
                <th style={cell}>Current file</th>
                {seasons.map((s) => (
                  <th key={s} style={cell}>
                    {s}
                  </th>
                ))}
                <th style={cell}>Player facts</th>
              </tr>
            </thead>
            <tbody>
              {view.feeds.map((f) => (
                <tr key={f.feed} style={{ borderTop: '1px solid var(--hairline)' }}>
                  <td style={cell}>{f.feed}</td>
                  <td style={{ ...cell, color: f.fresh ? undefined : 'var(--amber-base)' }}>
                    {f.fresh ? 'fresh' : 'stale'} · {when(f.lastLoaded)}
                  </td>
                  {seasons.map((s) => {
                    const c = f.seasons.find((x) => x.season === s);
                    return (
                      <td key={s} style={cell}>
                        {c ? `${c.players} (${c.waiting})` : ''}
                      </td>
                    );
                  })}
                  <td style={cell}>
                    {f.seasons
                      .filter((x) => x.season === 0)
                      .map((x) => `${x.players} (${x.waiting})`)
                      .join('')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <h3 style={head}>ROSTERED PLAYERS WITH DATA, BY POSITION</h3>
          <table style={{ borderCollapse: 'collapse', color: 'var(--text-primary)', width: 'fit-content' }}>
            <thead>
              <tr style={{ color: 'var(--text-secondary)', textAlign: 'left' }}>
                <th style={cell}>Signal</th>
                <th style={cell}>Measured in</th>
                {positions.map((p) => (
                  <th key={p} style={cell}>
                    {p}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {view.feeds.map((f) => (
                <tr key={f.feed} style={{ borderTop: '1px solid var(--hairline)' }}>
                  <td style={cell}>{f.feed}</td>
                  <td style={cell}>{coverageLabel(f)}</td>
                  {positions.map((p) => (
                    <td key={p} style={cell}>
                      {share(f.coverage.find((c) => c.position === p))}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}
