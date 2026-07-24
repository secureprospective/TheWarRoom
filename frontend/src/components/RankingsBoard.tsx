import { useEffect, useMemo, useState } from 'react';
import { useHarnessStore } from '../store/harness';
import { main } from '../../wailsjs/go/models';
import { SortHeader, EngraveState, type SortDir } from './board/primitives';

// RankingsBoard is the M1 module view: the REAL 32-team ranked board read back
// from the B6 output store. Three client-side lenses over the same persisted
// rows (the backend never re-sorts; B6's order IS the canonical ranking, carried
// by each row's `rank` — client sorting only reorders the display):
//   - global ranked list, with a position filter
//   - per-team drill-down (AD-20): pick a franchise, see its players ranked
//   - cap-efficiency: chip-filter to rows WITH a defined Adj/$M, sort by it
// Every score surface carries the proxy label — BasePoints is the labeled MFL
// YTD placeholder until the real L2 block ships. Debuggability over polish.
//
// B-2 restyle: migrated onto the Session-B board grammar (docs/ui/wireframes/
// session-b) — Adjusted-dominant facet map, typographic sort, four row states,
// honest data-states. The locked facet map is Rank·Player·Pos·Franchise·Base·
// Adjusted·Salary; the engine-internal diagnostics (AgePull, L4, CapTier, Adj/$M)
// are retained as a RECESSED trailing group in Narrative/Tactical and dropped in
// Matrix — a deliberate Phase-1 deviation from the strict 7-col lock so the tool
// keeps its debuggability until B-4 relocates layer detail into the Inspector.

const POSITIONS = ['QB', 'RB', 'WR', 'TE', 'K', 'DT', 'DE', 'LB', 'CB', 'S'] as const;

// Sortable numeric facets (Command Ledger B1: assets.sort col=<...>). The rank
// column is not sortable — it is the canonical B6 order, always ascending.
type SortKey = 'base' | 'adjusted' | 'salary' | 'capEff';

// Grid templates (Session-B §2·3). Tactical carries the facet map + the recessed
// diagnostic group; Matrix collapses to the pure scan (Rank·Player·Pos·Adj·Sal),
// the extra cells hidden via .twr-hide-mtx so they vacate their grid tracks.
const COLS = '34px 1fr 42px 148px 66px 92px 80px 60px 54px 66px 72px';
const COLS_MTX = '24px 1fr 36px 72px 72px';

export function RankingsBoard() {
  const rankings = useHarnessStore((s) => s.rankings);
  const scoreReport = useHarnessStore((s) => s.scoreReport);
  const scoring = useHarnessStore((s) => s.scoring);
  const loadRankings = useHarnessStore((s) => s.loadRankings);
  const scoreLeague = useHarnessStore((s) => s.scoreLeague);

  const [position, setPosition] = useState<string>('ALL');
  const [franchise, setFranchise] = useState<string>('ALL');
  const [capEffOnly, setCapEffOnly] = useState(false);
  const [sortKey, setSortKey] = useState<SortKey>('adjusted');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [selected, setSelected] = useState<string | null>(null);

  useEffect(() => {
    void loadRankings();
  }, [loadRankings]);

  const rows = useMemo(() => rankings?.rows ?? [], [rankings]);

  // Franchise list is derived from the persisted rows — the drill-down offers
  // exactly the teams that actually hold scored players.
  const franchises = useMemo(
    () => Array.from(new Set(rows.map((r) => r.franchiseID).filter(Boolean))).sort(),
    [rows],
  );

  function onSort(key: SortKey) {
    if (key === sortKey) {
      setSortDir((d) => (d === 'desc' ? 'asc' : 'desc'));
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  }

  const visible = useMemo(() => {
    let v = rows;
    if (position !== 'ALL') v = v.filter((r) => r.position === position);
    if (franchise !== 'ALL') v = v.filter((r) => r.franchiseID === franchise);
    // Cap-efficiency filter: rows WITH a defined efficiency only, never faking a
    // $0-salary row as 0 (matches the prior 'capeff view' exclusion).
    if (capEffOnly) v = v.filter((r) => r.capEffOK);
    const pick = (r: main.RankRow): number => {
      switch (sortKey) {
        case 'base': return r.basePoints;
        case 'salary': return r.salary;
        case 'capEff': return r.capEffOK ? r.capEff : -Infinity; // undefined sorts last
        default: return r.adjustedScore;
      }
    };
    const sorted = v.slice().sort((a, b) => pick(b) - pick(a));
    if (sortDir === 'asc') sorted.reverse();
    return sorted;
  }, [rows, position, franchise, capEffOnly, sortKey, sortDir]);

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Command strip — score action + provenance. */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <button type="button" className="twr-btn" onClick={() => void scoreLeague()} disabled={scoring}>
          {scoring ? 'Scoring…' : 'Score League'}
        </button>
        {rankings && (
          <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-tertiary)' }}>
            season {rankings.season} · rulebook v{rankings.configVersion} · {rows.length} scored
          </span>
        )}
      </div>

      {/* Honest-labeling banner: this is a validation surface, not a published
          ranking, until real L2 base scoring lands. Amber = caution, not alarm. */}
      <div className="twr-banner twr-banner--caution">
        {rankings?.label ?? 'BasePoints proxy — L2 pending'}
      </div>
      {rankings?.warning && (
        <div className="twr-banner twr-banner--warn">{rankings.warning}</div>
      )}

      {scoreReport && <ScoreReportPanel report={scoreReport} />}

      {/* Filters — position + franchise selects, cap-eff chip. */}
      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8 }}>
        <select className="twr-select" value={position} onChange={(e) => setPosition(e.target.value)}>
          <option value="ALL">All positions</option>
          {POSITIONS.map((p) => (
            <option key={p} value={p}>{p}</option>
          ))}
        </select>
        <select className="twr-select" value={franchise} onChange={(e) => setFranchise(e.target.value)}>
          <option value="ALL">All teams</option>
          {franchises.map((f) => (
            <option key={f} value={f}>Franchise {f}</option>
          ))}
        </select>
        <button
          type="button"
          className={`twr-chip${capEffOnly ? ' is-on' : ''}`}
          aria-pressed={capEffOnly}
          onClick={() => setCapEffOnly((v) => !v)}
        >
          Cap-eff only
        </button>
      </div>

      {rows.length === 0 ? (
        <EngraveState
          lines={['M1 Asset Rankings', '— awaiting scored data —', 'run “Score League”']}
        />
      ) : (
        <div
          className="twr-board"
          style={{ ['--twr-cols' as string]: COLS, ['--twr-cols-mtx' as string]: COLS_MTX }}
        >
          <div className="twr-board__sub">
            <span>#</span>
            <span>Player</span>
            <span>Pos</span>
            <span className="twr-hide-mtx">Franchise</span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader label="Base" sortKey="base" activeKey={sortKey} dir={sortDir} onSort={onSort} numeric />
            </span>
            <span className="twr-r">
              <SortHeader label="Adj" sortKey="adjusted" activeKey={sortKey} dir={sortDir} onSort={onSort} numeric />
            </span>
            <span className="twr-r">
              <SortHeader label="Salary" sortKey="salary" activeKey={sortKey} dir={sortDir} onSort={onSort} numeric />
            </span>
            <span className="twr-r twr-hide-mtx">Age</span>
            <span className="twr-r twr-hide-mtx">L4</span>
            <span className="twr-hide-mtx">Tier</span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader label="Adj/$M" sortKey="capEff" activeKey={sortKey} dir={sortDir} onSort={onSort} numeric />
            </span>
          </div>
          {visible.map((r) => (
            <div
              key={r.mflID}
              className={`twr-board__row${selected === r.mflID ? ' is-selected' : ''}`}
              onClick={() => setSelected((cur) => (cur === r.mflID ? null : r.mflID))}
            >
              <span className="twr-c-rank">{r.rank}</span>
              <span className="twr-c-name">{r.name}</span>
              <span className="twr-c-pos">{r.position}</span>
              <span className="twr-c-fr twr-hide-mtx">{r.franchiseID || '—'}</span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.basePoints.toFixed(2)}</span>
              <span className="twr-c-adj twr-r">{r.adjustedScore.toFixed(2)}</span>
              <span className="twr-c-num twr-r">${r.salary.toFixed(2)}</span>
              <span className="twr-c-diag twr-r twr-hide-mtx">{r.agePull.toFixed(3)}</span>
              <span className="twr-c-diag twr-r twr-hide-mtx">{r.l4Combined.toFixed(3)}</span>
              <span className="twr-c-diag twr-hide-mtx">{r.capTier}</span>
              <span className="twr-c-diag twr-r twr-hide-mtx">{r.capEffOK ? r.capEff.toFixed(2) : '—'}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ScoreReportPanel renders the last scoring pass's outcome: skip-if-present,
// the zero-base count, and EVERY exclusion with its reason — the policy is that
// an unscored player is visible, never silently missing from the board.
function ScoreReportPanel({ report }: { report: main.ScoreLeagueResult }) {
  const rep = report.report;
  if (!report.ok || !rep) return null;
  return (
    <div className="twr-panel">
      <p style={{ margin: 0 }}>
        {rep.skippedExisting
          ? `Already scored under rulebook v${rep.configVersion} — ${rep.existing} persisted rows served as-is (append-only; bump the rulebook to re-score).`
          : `Scored ${rep.scored} players under rulebook v${rep.configVersion} (${rep.zeroBase} with no ${rep.season - 1} YTD record` +
            (rep.negativeBase > 0 ? `; ${rep.negativeBase} negative totals floored to 0 — check the proxy data` : '') +
            ').'}
      </p>
      {rep.excluded && rep.excluded.length > 0 && (
        <details style={{ marginTop: 4 }}>
          <summary style={{ cursor: 'pointer', color: 'var(--amber-base)' }}>
            {rep.excluded.length} excluded (unscored — reasons)
          </summary>
          <ul style={{ margin: '4px 0 0', paddingLeft: 18, color: 'var(--text-secondary)' }}>
            {rep.excluded.map((e) => (
              <li key={e.mflID}>
                {e.name || e.mflID} (team {e.franchiseID}): {e.reason}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}
