import { useHarnessStore } from '../store/harness';

// ValidationBoard is Module 3: the 12 architectural cases with three honest states. PENDING
// is rendered distinctly from FAIL (amber, not red) — an encoded spec awaiting its B5b block,
// not a failure. Today only 3L passes; the rest are PENDING by design.
//
// B-2 restyle: on the Session-C token contract (semantic ramps), off the old emerald/rose/
// amber Tailwind. Each tile carries a semantic LEFT edge (the banner idiom) — green pass,
// amber pending, red fail — so state reads at a glance without tinted fills shouting.
const stateColor: Record<string, string> = {
  PASS: 'var(--green-base)',
  FAIL: 'var(--red-base)',
  PENDING: 'var(--amber-base)',
};

function Pill({ label, color }: { label: string; color: string }) {
  return (
    <span
      style={{
        fontFamily: 'var(--mono)',
        fontSize: 11,
        padding: '3px 10px',
        border: '1px solid var(--hairline)',
        borderLeft: `2px solid ${color}`,
        background: 'var(--surface-tile)',
        color,
      }}
    >
      {label}
    </span>
  );
}

export function ValidationBoard() {
  const validation = useHarnessStore((s) => s.validation);
  if (!validation) return <p style={{ color: 'var(--text-secondary)' }}>Loading validation suite…</p>;
  if (!validation.ok) return <div className="twr-banner twr-banner--warn">Validation suite error.</div>;

  const { cases, summary } = validation;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        <Pill label={`${summary.pass} pass`} color="var(--green-base)" />
        <Pill label={`${summary.fail} fail`} color="var(--red-base)" />
        <Pill label={`${summary.pending} pending`} color="var(--amber-base)" />
        <Pill label={`${summary.total} total`} color="var(--text-tertiary)" />
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
          gap: 8,
        }}
      >
        {cases.map((c) => {
          const color = stateColor[c.state] ?? 'var(--text-tertiary)';
          return (
            <div
              key={c.id}
              style={{
                border: '1px solid var(--hairline)',
                borderLeft: `2px solid ${color}`,
                background: 'var(--surface-tile)',
                padding: 12,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span style={{ fontFamily: 'var(--mono)', fontWeight: 700, color: 'var(--text-primary)' }}>
                  {c.id}
                </span>
                <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color }}>{c.state}</span>
              </div>
              <p style={{ margin: '6px 0 0', fontSize: 13, color: 'var(--text-primary)' }}>{c.name}</p>
              <p style={{ margin: '4px 0 0', fontSize: 11, color: 'var(--text-secondary)' }}>{c.detail}</p>
              <p style={{ margin: '4px 0 0', fontSize: 11, color: 'var(--text-tertiary)' }}>→ {c.b5bBlock}</p>
            </div>
          );
        })}
      </div>
    </div>
  );
}
