import { useState } from 'react';
import { useHarnessStore } from '../store/harness';

// AdminPanel is the live tuning surface: change a calibration param, apply, and the rankings
// re-score so the operator sees the board move (the sandbox's functional gate). Values are
// the B4 globals (cap-tier %, decay rate); the store enforces range.
export function AdminPanel() {
  const params = useHarnessStore((s) => s.params);
  const setParam = useHarnessStore((s) => s.setParam);
  const error = useHarnessStore((s) => s.error);
  const [edits, setEdits] = useState<Record<string, string>>({});

  if (!params) return <p className="text-slate-400">Loading params…</p>;
  if (!params.ok) return <p className="text-rose-400">Error: {params.error}</p>;

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-slate-300">Live calibration</h3>
      {error && <p className="text-xs text-rose-400">{error}</p>}
      {params.params.map((p) => (
        <div key={`${p.Key}:${p.Position}`} className="rounded border border-slate-700 p-2 text-xs">
          <div className="font-mono text-slate-200">{p.Key}</div>
          <div className="text-slate-500">
            default {p.Default} · range [{p.Min}, {p.Max}]
            {p.Position && p.Position !== 'global' ? ` · ${p.Position}` : ''}
          </div>
          <div className="mt-1 flex gap-2">
            <input
              type="number"
              step="0.01"
              placeholder={String(p.Default)}
              value={edits[p.Key] ?? ''}
              onChange={(e) => setEdits({ ...edits, [p.Key]: e.target.value })}
              className="w-24 rounded bg-slate-800 px-2 py-1 text-slate-100"
            />
            <button
              type="button"
              onClick={() => {
                const v = parseFloat(edits[p.Key]);
                if (Number.isFinite(v)) void setParam(p.Key, v);
              }}
              className="rounded bg-emerald-700 px-3 py-1 text-white hover:bg-emerald-600"
            >
              Apply
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
