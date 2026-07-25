import { useMemo, useState } from 'react';
import { useHarnessStore } from '../store/harness';
import { EngraveState } from './board/primitives';

// RookieTable is Module 1: the scored rookie board with EVERY engine intermediate visible
// (the debuggability bar). A position filter mirrors the spec's position-group tabs. The
// L4 columns are all 1.000 under identity L4 — the header labels that honestly.
//
// B-2 restyle: migrated onto the Session-B board grammar (.twr-board*) + the Session-C
// token contract, off the old slate/emerald Tailwind. Adjusted is the hero column; the
// engine intermediates are a RECESSED diagnostic group dropped in Matrix — the same
// debuggability-over-polish deviation M1 (RankingsBoard) carries as a dev/validation surface.
const fmt = (n: number) => (Number.isFinite(n) ? n.toFixed(3) : '—');

// Tactical carries the full intermediate ledger; Matrix collapses to the pure scan
// (#·Player·Pos·Adjusted), the .twr-hide-mtx cells vacating their grid tracks.
const COLS = '34px 1fr 42px 40px 66px 66px 60px 62px 56px 60px 62px 52px 80px';
const COLS_MTX = '24px 1fr 36px 72px';

export function RookieTable() {
  const rookies = useHarnessStore((s) => s.rookies);
  const [pos, setPos] = useState('ALL');

  const positions = useMemo(() => {
    const set = new Set<string>();
    rookies?.rows.forEach((r) => set.add(r.position));
    return ['ALL', ...Array.from(set).sort()];
  }, [rookies]);

  if (!rookies) return <p style={{ color: 'var(--text-secondary)' }}>Loading rankings…</p>;
  if (!rookies.ok)
    return <div className="twr-banner twr-banner--warn">Error: {rookies.error}</div>;

  const rows = rookies.rows.filter((r) => pos === 'ALL' || r.position === pos);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8 }}>
        <span className="twr-banner">Layer 4: {rookies.l4Mode}</span>
        <select
          className="twr-select"
          aria-label="Filter rookies by position"
          value={pos}
          onChange={(e) => setPos(e.target.value)}
        >
          {positions.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
      </div>

      {rookies.rows.length === 0 ? (
        // Genuine no-data only. A position filter narrowing to zero still renders the
        // board (header + no rows), matching the pre-restyle behavior — it does not claim
        // "no scored rookies" when rookies exist at other positions.
        <EngraveState lines={['Rookie sandbox', '— no scored rookies —']} />
      ) : (
        <div
          className="twr-board"
          style={{ ['--twr-cols' as string]: COLS, ['--twr-cols-mtx' as string]: COLS_MTX }}
        >
          <div className="twr-board__sub">
            <span>#</span>
            <span>Player</span>
            <span>Pos</span>
            <span className="twr-r twr-hide-mtx">Age</span>
            <span className="twr-r twr-hide-mtx">Base</span>
            <span className="twr-r twr-hide-mtx">AgePull</span>
            <span className="twr-r twr-hide-mtx">Film</span>
            <span className="twr-r twr-hide-mtx">RAS</span>
            <span className="twr-r twr-hide-mtx">Brk</span>
            <span className="twr-r twr-hide-mtx">L4</span>
            <span className="twr-r twr-hide-mtx">CapMlt</span>
            <span className="twr-hide-mtx">Tier</span>
            <span className="twr-r">Adj</span>
          </div>
          {rows.map((r, i) => (
            <div key={r.mflID} className="twr-board__row">
              <span className="twr-c-rank">{i + 1}</span>
              <span className="twr-c-name">{r.name}</span>
              <span className="twr-c-pos">{r.position}</span>
              <span className="twr-c-diag twr-r twr-hide-mtx">{r.age}</span>
              {r.err ? (
                // Error spans the remaining tracks to the row end (faithful to the old
                // colSpan-to-end behavior — no Adjusted value is shown for an errored row).
                <span
                  className="twr-c-diag twr-hide-mtx"
                  style={{ gridColumn: '5 / -1', color: 'var(--red-base)' }}
                >
                  {r.err}
                </span>
              ) : (
                <>
                  <span className="twr-c-num twr-r twr-hide-mtx">{fmt(r.result.BasePoints)}</span>
                  <span className="twr-c-diag twr-r twr-hide-mtx">{fmt(r.result.AgePull)}</span>
                  <span className="twr-c-diag twr-r twr-hide-mtx">
                    {fmt(r.result.Layer4Output.FilmEffective)}
                  </span>
                  <span className="twr-c-diag twr-r twr-hide-mtx">
                    {fmt(r.result.Layer4Output.RASEffective)}
                    {r.rasImputed && <span style={{ color: 'var(--amber-base)' }}> *</span>}
                  </span>
                  <span className="twr-c-diag twr-r twr-hide-mtx">
                    {fmt(r.result.Layer4Output.BreakoutEffective)}
                  </span>
                  <span className="twr-c-diag twr-r twr-hide-mtx">
                    {fmt(r.result.Layer4Output.Combined)}
                  </span>
                  <span className="twr-c-diag twr-r twr-hide-mtx">{fmt(r.result.CapMultiplier)}</span>
                  <span className="twr-c-diag twr-hide-mtx">{r.result.CapTier}</span>
                  <span className="twr-c-adj twr-r">{fmt(r.result.AdjustedScore)}</span>
                </>
              )}
            </div>
          ))}
        </div>
      )}
      <p style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>
        * RAS imputed (player has no RAS; L1 fallback applied).
      </p>
    </div>
  );
}
