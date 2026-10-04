import { useState } from 'react';
import { useHarnessStore } from '../store/harness';

// AdminPanel lists every calibration parameter, league-wide first and then each position's
// scouting settings, with the value in effect. Apply writes an override; the store checks the
// range, and the next Score League run uses it.
export function AdminPanel() {
  const params = useHarnessStore((s) => s.params);
  const setParam = useHarnessStore((s) => s.setParam);
  const error = useHarnessStore((s) => s.error);
  const [edits, setEdits] = useState<Record<string, string>>({});

  if (!params) return <p style={{ color: 'var(--text-secondary)' }}>Loading params…</p>;
  if (!params.ok) return <div className="twr-banner twr-banner--warn">Error: {params.error}</div>;

  const rows = [...params.params].sort(
    (a, b) => a.position.localeCompare(b.position) || a.key.localeCompare(b.key),
  );
  const cell = { padding: '3px 8px', borderBottom: '1px solid var(--hairline)' };
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {error && <div className="twr-banner twr-banner--warn">{error}</div>}
      <table style={{ borderCollapse: 'collapse', fontFamily: 'var(--mono)', fontSize: 12 }}>
        <thead>
          <tr style={{ color: 'var(--text-secondary)', textAlign: 'left' }}>
            {['Position', 'Setting', 'Value', 'Default', 'Range', ''].map((h) => (
              <th key={h} style={cell}>
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((p) => {
            const id = `${p.key}@${p.position}`;
            const edited = p.value !== p.default;
            return (
              <tr key={id} title={p.description}>
                <td style={cell}>{p.position || 'league'}</td>
                <td style={cell}>{p.key}</td>
                <td style={{ ...cell, color: edited ? 'var(--amber-loud)' : 'var(--text-primary)' }}>{p.value}</td>
                <td style={{ ...cell, color: 'var(--text-tertiary)' }}>{p.default}</td>
                <td style={{ ...cell, color: 'var(--text-tertiary)' }}>
                  [{p.min}, {p.max}]
                </td>
                <td style={cell}>
                  <input
                    type="number"
                    step="0.01"
                    className="twr-input"
                    style={{ width: 80 }}
                    placeholder={String(p.value)}
                    value={edits[id] ?? ''}
                    onChange={(e) => setEdits({ ...edits, [id]: e.target.value })}
                  />{' '}
                  <button
                    type="button"
                    className="twr-btn"
                    onClick={() => {
                      const v = parseFloat(edits[id]);
                      if (Number.isFinite(v)) void setParam(p.key, p.position, v);
                    }}
                  >
                    Apply
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
