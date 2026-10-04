import { useState } from 'react';
import { useAppStore } from '../store/app';

// AdminPanel lists every calibration parameter, league-wide first and then each position's
// scouting settings and fitted model values, with the value in effect. The filter matches the
// setting, the position or the source. Apply writes an override; the store checks the range,
// and the next Score League run uses it. An override pins the value against later shipped
// defaults, even when it equals today's, so overridden rows show amber and offer Reset.
export function AdminPanel() {
  const params = useAppStore((s) => s.params);
  const setParam = useAppStore((s) => s.setParam);
  const resetParam = useAppStore((s) => s.resetParam);
  const error = useAppStore((s) => s.error);
  const [edits, setEdits] = useState<Record<string, string>>({});
  const [filter, setFilter] = useState('');

  if (!params) return <p style={{ color: 'var(--text-secondary)' }}>Loading params…</p>;
  if (!params.ok) return <div className="twr-banner twr-banner--warn">Error: {params.error}</div>;

  const words = filter.toLowerCase().split(/\s+/).filter(Boolean);
  const rows = params.params
    .filter((p) => {
      const text = `${p.key} ${p.position || 'league'} ${source(p.calibrated)}`.toLowerCase();
      return words.every((w) => text.includes(w));
    })
    .sort((a, b) => a.position.localeCompare(b.position) || a.key.localeCompare(b.key));
  const cell = { padding: '3px 8px', borderBottom: '1px solid var(--hairline)' };
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {error && <div className="twr-banner twr-banner--warn">{error}</div>}
      <input
        className="twr-input"
        style={{ width: 320 }}
        placeholder="Filter: e.g. WR model.arc, fitted, l4.film"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />
      <span style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>
        {rows.length} of {params.params.length} settings
      </span>
      <table style={{ borderCollapse: 'collapse', fontFamily: 'var(--mono)', fontSize: 12 }}>
        <thead>
          <tr style={{ color: 'var(--text-secondary)', textAlign: 'left' }}>
            {['Position', 'Setting', 'Source', 'Value', 'Default', 'Range', ''].map((h) => (
              <th key={h} style={cell}>
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((p) => {
            const id = `${p.key}@${p.position}`;
            return (
              <tr key={id} title={p.description}>
                <td style={cell}>{p.position || 'league'}</td>
                <td style={cell}>{p.key}</td>
                <td style={{ ...cell, color: 'var(--text-tertiary)' }}>{source(p.calibrated)}</td>
                <td
                  style={{ ...cell, color: p.overridden ? 'var(--amber-loud)' : 'var(--text-primary)' }}
                  title={p.overridden ? 'Set in Admin: it stays until Reset, whatever the shipped default' : undefined}
                >
                  {show(p.value)}
                </td>
                <td style={{ ...cell, color: 'var(--text-tertiary)' }}>{show(p.default)}</td>
                <td style={{ ...cell, color: 'var(--text-tertiary)' }}>
                  [{p.min}, {p.max}]
                </td>
                <td style={cell}>
                  <input
                    type="number"
                    step="any"
                    className="twr-input"
                    style={{ width: 80 }}
                    placeholder={show(p.value)}
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
                  {p.overridden && (
                    <>
                      {' '}
                      <button
                        type="button"
                        className="twr-btn"
                        title="Clear the Admin value and follow the shipped default"
                        onClick={() => void resetParam(p.key, p.position)}
                      >
                        Reset
                      </button>
                    </>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function source(calibrated: boolean): string {
  return calibrated ? 'fitted' : 'hand-set';
}

// show keeps four significant figures: fitted values carry float noise past that.
function show(v: number): string {
  return String(Number(v.toPrecision(4)));
}
