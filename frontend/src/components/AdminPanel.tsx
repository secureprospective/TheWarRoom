import { useState } from 'react';
import { useHarnessStore } from '../store/harness';

// AdminPanel is the live tuning surface: change a calibration param, apply, and the rankings
// re-score so the operator sees the board move (the sandbox's functional gate). Values are
// the B4 globals (cap-tier %, decay rate); the store enforces range.
//
// B-2 restyle: on the Session-C token contract + control classes (.twr-panel/.twr-input/
// .twr-btn), off the old slate/emerald Tailwind.
export function AdminPanel() {
  const params = useHarnessStore((s) => s.params);
  const setParam = useHarnessStore((s) => s.setParam);
  const error = useHarnessStore((s) => s.error);
  const [edits, setEdits] = useState<Record<string, string>>({});

  if (!params) return <p style={{ color: 'var(--text-secondary)' }}>Loading params…</p>;
  if (!params.ok) return <div className="twr-banner twr-banner--warn">Error: {params.error}</div>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <h3
        style={{
          margin: 0,
          fontFamily: 'var(--mono)',
          fontSize: 11,
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
          color: 'var(--text-secondary)',
        }}
      >
        Live calibration
      </h3>
      {error && <div className="twr-banner twr-banner--warn">{error}</div>}
      {params.params.map((p) => (
        <div key={`${p.Key}:${p.Position}`} className="twr-panel">
          <div style={{ fontFamily: 'var(--mono)', color: 'var(--text-primary)' }}>{p.Key}</div>
          <div style={{ color: 'var(--text-tertiary)' }}>
            default {p.Default} · range [{p.Min}, {p.Max}]
            {p.Position && p.Position !== 'global' ? ` · ${p.Position}` : ''}
          </div>
          <div style={{ marginTop: 6, display: 'flex', gap: 8 }}>
            <input
              type="number"
              step="0.01"
              className="twr-input"
              style={{ width: 96 }}
              placeholder={String(p.Default)}
              value={edits[p.Key] ?? ''}
              onChange={(e) => setEdits({ ...edits, [p.Key]: e.target.value })}
            />
            <button
              type="button"
              className="twr-btn"
              onClick={() => {
                const v = parseFloat(edits[p.Key]);
                if (Number.isFinite(v)) void setParam(p.Key, v);
              }}
            >
              Apply
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
