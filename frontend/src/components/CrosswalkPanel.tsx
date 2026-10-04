import { useEffect } from 'react';
import { useCrosswalkStore } from '../store/crosswalk';
import { main } from '../../wailsjs/go/models';

// CrosswalkPanel shows how well DynastyProcess links the league's players to NFL (gsis) ids:
// the match rate per position, rostered and free agents, and every unmatched rostered player
// with the reason. Stage 3's gate reads it.
const cell = { padding: '4px 10px', fontFamily: 'var(--mono)', fontSize: 12 } as const;

function pct(matched: number, total: number) {
  if (total === 0) return '—';
  return `${matched}/${total} · ${((100 * matched) / total).toFixed(1)}%`;
}

function RateRow({ r, bold }: { r: main.CrosswalkRate; bold?: boolean }) {
  const weight = bold ? 700 : 400;
  return (
    <tr style={{ borderTop: '1px solid var(--hairline)', fontWeight: weight }}>
      <td style={cell}>{r.position}</td>
      <td style={cell}>{pct(r.rosteredMatched, r.rostered)}</td>
      <td style={cell}>{pct(r.freeAgentsMatched, r.freeAgents)}</td>
    </tr>
  );
}

export function CrosswalkPanel() {
  const { report, loading, error, read, load } = useCrosswalkStore();
  useEffect(() => {
    void read();
  }, [read]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <button type="button" className="twr-btn" disabled={loading} onClick={() => void load()}>
          {loading ? 'Loading…' : 'Load from DynastyProcess'}
        </button>
        {report?.ok && (
          <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-tertiary)' }}>
            loaded {new Date(report.loadedAt).toLocaleString()} · {report.links} links · {report.promoted}{' '}
            waiting facts resolved
          </span>
        )}
      </div>
      {error && <div className="twr-banner twr-banner--warn">{error}</div>}
      {!report?.ok && !error && (
        <p style={{ color: 'var(--text-secondary)' }}>
          Not loaded since launch. Load it here, or score the league.
        </p>
      )}
      {report?.ok && (
        <>
          <table style={{ borderCollapse: 'collapse', color: 'var(--text-primary)', width: 'fit-content' }}>
            <thead>
              <tr style={{ color: 'var(--text-secondary)', textAlign: 'left' }}>
                <th style={cell}>Pos</th>
                <th style={cell}>Rostered matched</th>
                <th style={cell}>Free agents matched</th>
              </tr>
            </thead>
            <tbody>
              {report.rates.map((r) => (
                <RateRow key={r.position} r={r} />
              ))}
              <RateRow r={report.total} bold />
            </tbody>
          </table>
          <h3 style={{ margin: 0, fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-secondary)' }}>
            UNMATCHED ROSTERED PLAYERS ({report.unmatched?.length ?? 0})
          </h3>
          <table style={{ borderCollapse: 'collapse', color: 'var(--text-primary)' }}>
            <tbody>
              {(report.unmatched ?? []).map((m) => (
                <tr key={m.mflId} style={{ borderTop: '1px solid var(--hairline)' }}>
                  <td style={cell}>{m.mflId}</td>
                  <td style={cell}>{m.name || '—'}</td>
                  <td style={cell}>{m.position}</td>
                  <td style={cell}>{m.franchise}</td>
                  <td style={{ ...cell, color: 'var(--amber-base)' }}>{m.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}
