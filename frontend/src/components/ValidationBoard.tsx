import { useHarnessStore } from '../store/harness';

// ValidationBoard is Module 3: the 12 architectural cases with three honest states. PENDING
// is rendered distinctly from FAIL (amber, not red) — an encoded spec awaiting its B5b block,
// not a failure. Today only 3L passes; the rest are PENDING by design.
const stateStyle: Record<string, string> = {
  PASS: 'border-emerald-700 bg-emerald-950/40 text-emerald-300',
  FAIL: 'border-rose-700 bg-rose-950/40 text-rose-300',
  PENDING: 'border-amber-800 bg-amber-950/30 text-amber-300',
};

export function ValidationBoard() {
  const validation = useHarnessStore((s) => s.validation);
  if (!validation) return <p className="text-slate-400">Loading validation suite…</p>;
  if (!validation.ok) return <p className="text-rose-400">Validation suite error.</p>;

  const { cases, summary } = validation;

  return (
    <div className="space-y-4">
      <div className="flex gap-4 text-sm">
        <span className="rounded bg-emerald-950/40 px-3 py-1 text-emerald-300">
          {summary.pass} pass
        </span>
        <span className="rounded bg-rose-950/40 px-3 py-1 text-rose-300">{summary.fail} fail</span>
        <span className="rounded bg-amber-950/30 px-3 py-1 text-amber-300">
          {summary.pending} pending
        </span>
        <span className="rounded bg-slate-800 px-3 py-1 text-slate-400">{summary.total} total</span>
      </div>

      <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
        {cases.map((c) => (
          <div key={c.id} className={`rounded-lg border p-3 ${stateStyle[c.state] ?? ''}`}>
            <div className="flex items-center justify-between">
              <span className="font-mono font-bold">{c.id}</span>
              <span className="rounded bg-black/30 px-2 py-0.5 text-xs">{c.state}</span>
            </div>
            <p className="mt-1 text-sm text-slate-200">{c.name}</p>
            <p className="mt-1 text-xs text-slate-400">{c.detail}</p>
            <p className="mt-1 text-xs text-slate-500">→ {c.b5bBlock}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
