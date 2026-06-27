import { useMemo, useState } from 'react';
import { useHarnessStore } from '../store/harness';

// RookieTable is Module 1: the scored rookie board with EVERY engine intermediate visible
// (the debuggability bar). A position filter mirrors the spec's position-group tabs. The
// L4 columns are all 1.000 under identity L4 — the header labels that honestly.
const fmt = (n: number) => (Number.isFinite(n) ? n.toFixed(3) : '—');

export function RookieTable() {
  const rookies = useHarnessStore((s) => s.rookies);
  const [pos, setPos] = useState('ALL');

  const positions = useMemo(() => {
    const set = new Set<string>();
    rookies?.rows.forEach((r) => set.add(r.position));
    return ['ALL', ...Array.from(set).sort()];
  }, [rookies]);

  if (!rookies) return <p className="text-slate-400">Loading rankings…</p>;
  if (!rookies.ok) return <p className="text-rose-400">Error: {rookies.error}</p>;

  const rows = rookies.rows.filter((r) => pos === 'ALL' || r.position === pos);

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3 text-sm">
        <span className="rounded bg-amber-900/40 px-2 py-1 text-amber-300">
          Layer 4: {rookies.l4Mode}
        </span>
        <label className="text-slate-400">
          Position:{' '}
          <select
            value={pos}
            onChange={(e) => setPos(e.target.value)}
            className="rounded bg-slate-800 px-2 py-1 text-slate-100"
          >
            {positions.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs">
          <thead className="text-slate-400">
            <tr className="border-b border-slate-700">
              <th className="p-2">#</th>
              <th className="p-2">Player</th>
              <th className="p-2">Pos</th>
              <th className="p-2">Age</th>
              <th className="p-2">Base</th>
              <th className="p-2">AgePull</th>
              <th className="p-2">Film</th>
              <th className="p-2">RAS</th>
              <th className="p-2">Brk</th>
              <th className="p-2">L4</th>
              <th className="p-2">CapMlt</th>
              <th className="p-2">Tier</th>
              <th className="p-2 font-bold">Adjusted</th>
            </tr>
          </thead>
          <tbody className="font-mono">
            {rows.map((r, i) => (
              <tr key={r.mflID} className="border-b border-slate-800 hover:bg-slate-800/40">
                <td className="p-2 text-slate-500">{i + 1}</td>
                <td className="p-2 font-sans text-slate-100">{r.name}</td>
                <td className="p-2">{r.position}</td>
                <td className="p-2">{r.age}</td>
                {r.err ? (
                  <td className="p-2 text-rose-400" colSpan={9}>
                    {r.err}
                  </td>
                ) : (
                  <>
                    <td className="p-2">{fmt(r.result.BasePoints)}</td>
                    <td className="p-2">{fmt(r.result.AgePull)}</td>
                    <td className="p-2">{fmt(r.result.Layer4Output.FilmEffective)}</td>
                    <td className="p-2">
                      {fmt(r.result.Layer4Output.RASEffective)}
                      {r.rasImputed && <span className="ml-1 text-amber-400">*</span>}
                    </td>
                    <td className="p-2">{fmt(r.result.Layer4Output.BreakoutEffective)}</td>
                    <td className="p-2">{fmt(r.result.Layer4Output.Combined)}</td>
                    <td className="p-2">{fmt(r.result.CapMultiplier)}</td>
                    <td className="p-2">{r.result.CapTier}</td>
                    <td className="p-2 font-bold text-emerald-300">{fmt(r.result.AdjustedScore)}</td>
                  </>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-slate-500">* RAS imputed (player has no RAS; L1 fallback applied).</p>
    </div>
  );
}
