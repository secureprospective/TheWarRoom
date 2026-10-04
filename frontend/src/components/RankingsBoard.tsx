import { useEffect, useMemo, useState } from 'react';
import { useAppStore } from '../store/app';
import { main } from '../../wailsjs/go/models';
import {
  SortHeader,
  EngraveState,
  DeltaRank,
  FreshnessBar,
  type SortDir,
} from './board/primitives';
import { useBoardKeys, useScrollCursorIntoView } from './board/keys';
import { useInspectorStore } from '../store/inspector';

// RankingsBoard is the M1 module view: the REAL 32-team ranked board, read back
// from the latest board run in history. Three client-side lenses over the same
// persisted rows (the backend never re-sorts; the run's order IS the canonical ranking, carried
// by each row's `rank` — client sorting only reorders the display):
//   - global ranked list, with a position filter
//   - per-team drill-down (AD-20): pick a franchise, see its players ranked
//   - cap-efficiency: chip-filter to rows WITH a defined Adj/$M, sort by it
// Every score surface carries the proxy label — BasePoints is the labeled MFL
// YTD placeholder until the real L2 block ships. Debuggability over polish.
//
// B-2 restyle: migrated onto the Session-B board grammar (docs/ui/wireframes/
// session-b) — Adjusted-dominant facet map, typographic sort, row hover feedback,
// honest empty state. Row selection + keyboard nav landed in B-4a.
//
// B-4b: the board now holds the locked facet map EXACTLY —
// Rank·Player·Pos·Franchise·Base·Adjusted·Salary. The B-2 recessed diagnostic
// group is gone: AgePull / L4 / CapTier are engine internals that the B-4a
// Inspector now carries per player, so keeping them on the board duplicated the
// Inspector at the cost of the 7-col lock.
//
// Adj/$M is the exception, and deliberately so: it is not a diagnostic but the
// locked "Cap efficiency view" (UI_Direction_Document §12.2), a BOARD-level lens
// (rank the league by efficiency) that the per-player Inspector cannot express.
// So it is bound to that lens — the column exists only while the cap-eff chip is
// on, and never in Matrix (which stays the pure Rank·Player·Pos·Adj·Sal scan).

const POSITIONS = ['QB', 'RB', 'WR', 'TE', 'K', 'DT', 'DE', 'LB', 'CB', 'S'] as const;

// Sortable numeric facets (Command Ledger B1: assets.sort col=<...>). The rank
// column is not sortable — it is the run's canonical order, always ascending.
type SortKey = 'base' | 'adjusted' | 'now' | 'dynasty' | 'salary' | 'capEff';

// Grid templates (Session-B §2·3). Narrative/Tactical carry the locked facet map plus the
// Adj/$M track ONLY while the cap-efficiency lens is on; Matrix collapses to the pure scan,
// the extra cells hidden via .twr-hide-mtx so they vacate their grid tracks.
//
// TRACK COUNTS MUST BALANCE — tsc and the linter CANNOT see a violation here, so count by
// hand on every change. B-5 added the §1 Δ track next to #; it is carried in ALL densities
// because movement is the point of a scan, not a detail to drop from one.
//   COLS        10: # · Δ · Player · Pos · Franchise · Base · Adj · Now · Dyn · Sal
//   COLS_CAPEFF 11: the above + Adj/$M (cap-efficiency lens only)
//   COLS_MTX     6: # · Δ · Player · Pos · Adj · Sal   (Franchise, Base, Now and Dyn are .twr-hide-mtx)
// Now and Dyn are the model run's measurables in league points per game.
const COLS = '34px 44px 1fr 42px 148px 66px 92px 58px 58px 80px';
const COLS_CAPEFF = `${COLS} 72px`;
const COLS_MTX = '24px 40px 1fr 36px 72px 72px';

export function RankingsBoard() {
  const rankings = useAppStore((s) => s.rankings);
  const scoreReport = useAppStore((s) => s.scoreReport);
  const scoring = useAppStore((s) => s.scoring);
  const storeError = useAppStore((s) => s.error);
  const loadRankings = useAppStore((s) => s.loadRankings);
  const scoreLeague = useAppStore((s) => s.scoreLeague);
  const select = useInspectorStore((s) => s.select);
  const selectedMflID = useInspectorStore((s) => s.selectedMflID);

  const [position, setPosition] = useState<string>('ALL');
  const [franchise, setFranchise] = useState<string>('ALL');
  const [capEffOnly, setCapEffOnly] = useState(false);
  const [sortKey, setSortKey] = useState<SortKey>('adjusted');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  useEffect(() => {
    void loadRankings();
  }, [loadRankings]);

  const rows = useMemo(() => rankings?.rows ?? [], [rankings]);

  // Franchise list is derived from the persisted rows — the drill-down offers
  // exactly the teams that actually hold scored players.
  const franchises = useMemo(() => {
    const byID = new Map<string, string>();
    for (const r of rows) if (r.franchiseID) byID.set(r.franchiseID, r.franchiseName);
    return Array.from(byID, ([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name));
  }, [rows]);

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
    // Rows without a defined Adj/$M are always parked LAST (in either direction),
    // never faked as a value — partition them out before sorting the rest.
    const hasVal = (r: main.RankRow): boolean =>
      sortKey === 'capEff' ? r.capEffOK : sortKey === 'now' || sortKey === 'dynasty' ? r.modelOK : true;
    const pick = (r: main.RankRow): number => {
      switch (sortKey) {
        case 'base': return r.basePoints;
        case 'now': return r.nowPPG;
        case 'dynasty': return r.dynastyPPG;
        case 'salary': return r.salary;
        case 'capEff': return r.capEff;
        default: return r.adjustedScore;
      }
    };
    const ranked = v.filter(hasVal).sort((a, b) => pick(b) - pick(a));
    if (sortDir === 'asc') ranked.reverse();
    return [...ranked, ...v.filter((r) => !hasVal(r))];
  }, [rows, position, franchise, capEffOnly, sortKey, sortDir]);

  // Session-B keyboard map: J/K travel the board, Enter opens the Inspector. The ids are
  // the DISPLAYED order, so the cursor follows the user's current sort and filters rather
  // than the underlying rank. Enter is the only thing that fires an IPC fetch — travelling
  // is free.
  const visibleIDs = useMemo(() => visible.map((r) => r.mflID), [visible]);
  const { cursorID, setCursorID } = useBoardKeys(visibleIDs, (id) => void select(id));
  useScrollCursorIntoView(cursorID);

  // A click both selects AND plants the cursor, so J/K continue from where the user
  // clicked rather than jumping back to the top of the board.
  const pickRow = (id: string) => {
    setCursorID(id);
    void select(id);
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Command strip — score action + provenance. */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <button type="button" className="twr-btn" onClick={() => void scoreLeague()} disabled={scoring}>
          {scoring ? 'Scoring…' : 'Score League'}
        </button>
        {rankings && (
          <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-tertiary)' }}>
            season {rankings.season} ·{' '}
            {rankings.runID > 0
              ? `board #${rankings.runID} · scored ${new Date(rankings.asOf).toLocaleString()} · ${rows.length} players`
              : 'no board yet'}
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
      {/* Reduced set (R11): a lost source never stops the board, but the board must say what it
          scored without. */}
      {rankings && rankings.missingMeasures?.length > 0 && (
        <div className="twr-banner twr-banner--warn">
          Reduced set: this board scored without{' '}
          {rankings.missingMeasures.map((m) => m.meaning || m.name).join('; ')} — no active source
          feeds {rankings.missingMeasures.length === 1 ? 'it' : 'them'}.
        </div>
      )}

      {/* M1's board is local SQLite, so this is normally silent (local data reads live).
          It is wired anyway so every board answers the freshness question the same way —
          a module that simply omits the signal is indistinguishable from one that is fine. */}
      <FreshnessBar freshness={rankings?.freshness} board="Rankings" />
      {/* No PhaseBar here, deliberately: M2's standings are FINAL once the season ends, but
          an M1 board is scoring output that can legitimately be re-scored under new settings
          at any time. Labelling it "final" would be false. */}

      {/* A failed score must NEVER be silent — ScoreReportPanel renders nothing on !ok, which left an
          empty board with no reason (the score's network fetch can fail, or it can score zero). Surface
          the engine's rejection detail (ok:false) or an IPC throw (store error) so the cause is visible. */}
      {scoreReport && !scoreReport.ok && (
        <div className="twr-banner twr-banner--warn" style={{ display: 'block' }}>
          Scoring failed — {scoreReport.error || 'the engine returned no detail.'}
        </div>
      )}
      {!scoreReport && storeError && (
        <div className="twr-banner twr-banner--warn" style={{ display: 'block' }}>
          {storeError}
        </div>
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
            <option key={f.id} value={f.id}>{f.name}</option>
          ))}
        </select>
        <button
          type="button"
          className={`twr-chip${capEffOnly ? ' is-on' : ''}`}
          aria-pressed={capEffOnly}
          onClick={() => {
            // The chip IS the cap-efficiency lens (B-4b): it filters to rows with a
            // defined Adj/$M, reveals the Adj/$M column, and sorts by it — one action,
            // as the original capeff view was. Turning it off must also drop a capEff
            // sort, or the board would stay ordered by a column that is no longer
            // rendered (an invisible sort state with no header to undo it).
            if (!capEffOnly) {
              setSortKey('capEff');
              setSortDir('desc');
            } else if (sortKey === 'capEff') {
              setSortKey('adjusted');
              setSortDir('desc');
            }
            setCapEffOnly((v) => !v);
          }}
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
          style={{
            ['--twr-cols' as string]: capEffOnly ? COLS_CAPEFF : COLS,
            ['--twr-cols-mtx' as string]: COLS_MTX,
          }}
        >
          <div className="twr-board__sub">
            <span>#</span>
            <span className="twr-r" title="Movement since the previous scoring run">
              Δ
            </span>
            <span>Player</span>
            <span>Pos</span>
            <span className="twr-hide-mtx">Franchise</span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader label="Base" sortKey="base" activeKey={sortKey} dir={sortDir} onSort={onSort} />
            </span>
            <span className="twr-r">
              <SortHeader label="Adj" sortKey="adjusted" activeKey={sortKey} dir={sortDir} onSort={onSort} />
            </span>
            <span className="twr-r twr-hide-mtx" title="On-field-now: expected league points per game he plays this season">
              <SortHeader label="Now" sortKey="now" activeKey={sortKey} dir={sortDir} onSort={onSort} />
            </span>
            <span className="twr-r twr-hide-mtx" title="Dynasty: expected league points per game over the coming seasons, discounted, counting a season off the field as 0">
              <SortHeader label="Dyn" sortKey="dynasty" activeKey={sortKey} dir={sortDir} onSort={onSort} />
            </span>
            <span className="twr-r">
              <SortHeader label="Salary" sortKey="salary" activeKey={sortKey} dir={sortDir} onSort={onSort} />
            </span>
            {capEffOnly && (
              <span className="twr-r twr-hide-mtx">
                <SortHeader label="Adj/$M" sortKey="capEff" activeKey={sortKey} dir={sortDir} onSort={onSort} />
              </span>
            )}
          </div>
          {visible.map((r) => (
            // Row selection (B-4a): click (or Enter/Space) selects the player → the
            // inspector store fetches the breakdown and App auto-opens the Inspector.
            // `.is-selected` paints the neutral achromatic axis (Session-C selection).
            <div
              key={r.mflID}
              className={`twr-board__row${r.mflID === selectedMflID ? ' is-selected' : ''}${
                r.mflID === cursorID ? ' is-cursor' : ''
              }`}
              data-row-id={r.mflID}
              role="button"
              tabIndex={0}
              aria-pressed={r.mflID === selectedMflID}
              // aria-current exposes the KEYBOARD CURSOR, which is a different fact from
              // aria-pressed (selection) — without it a screen-reader user pressing J/K
              // hears nothing change at all.
              aria-current={r.mflID === cursorID ? true : undefined}
              onClick={() => pickRow(r.mflID)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  pickRow(r.mflID);
                }
              }}
            >
              <span className="twr-c-rank">{r.rank}</span>
              <span className="twr-r">
                <DeltaRank delta={r.rankDelta} ok={r.deltaOK} />
              </span>
              <span className="twr-c-name">{r.name}</span>
              <span className="twr-c-pos">{r.position}</span>
              <span className="twr-c-fr twr-hide-mtx">{r.franchiseName || '—'}</span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.basePoints.toFixed(2)}</span>
              <span className="twr-c-adj twr-r">{r.adjustedScore.toFixed(2)}</span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.modelOK ? r.nowPPG.toFixed(1) : '—'}</span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.modelOK ? r.dynastyPPG.toFixed(1) : '—'}</span>
              <span className="twr-c-num twr-r">${r.salary.toFixed(2)}</span>
              {capEffOnly && (
                // The '—' arm is unreachable while the column is lens-bound (the lens filters to
                // capEffOK rows), and is kept only so decoupling the column from the filter later
                // cannot print a bare NaN. It is a guard, not a second contract. (GLM lead M2.)
                <span className="twr-c-diag twr-r twr-hide-mtx">
                  {r.capEffOK ? r.capEff.toFixed(2) : '—'}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ScoreReportPanel renders the last scoring pass's outcome: the run it wrote (or
// that nothing changed), the zero-base count, any source that failed, and EVERY
// exclusion with its reason — the policy is that
// an unscored player is visible, never silently missing from the board.
function ScoreReportPanel({ report }: { report: main.ScoreLeagueResult }) {
  const rep = report.report;
  if (!report.ok || !rep) return null;
  return (
    <div className="twr-panel">
      <p style={{ margin: 0 }}>
        {rep.unchanged
          ? `No change: settings, inputs and scores match board #${rep.runID}, so nothing new was written.`
          : `Scored ${rep.scored} players as board #${rep.runID} (${rep.zeroBase} with no ${rep.season - 1} YTD record` +
            (rep.negativeBase > 0 ? `; ${rep.negativeBase} negative totals floored to 0 — check the proxy data` : '') +
            ').'}
      </p>
      {report.model && report.model.runID > 0 && (
        <p style={{ margin: '4px 0 0' }}>
          {report.model.unchanged
            ? `Measurables unchanged: model run #${report.model.runID}.`
            : `Valued ${report.model.scored} players as model run #${report.model.runID} (${report.model.rookies} on their prior alone` +
              (report.model.excluded?.length ? `; ${report.model.excluded.length} excluded` : '') +
              ').'}
        </p>
      )}
      {report.warning && (
        <p style={{ margin: '4px 0 0', color: 'var(--amber-base)' }}>{report.warning}</p>
      )}
      {rep.excluded && rep.excluded.length > 0 && (
        <details style={{ marginTop: 4 }}>
          <summary style={{ cursor: 'pointer', color: 'var(--amber-base)' }}>
            {rep.excluded.length} excluded (unscored — reasons)
          </summary>
          <ul style={{ margin: '4px 0 0', paddingLeft: 18, color: 'var(--text-secondary)' }}>
            {rep.excluded.map((e) => (
              <li key={e.mflID}>
                {e.name || e.mflID} ({e.franchiseName}): {e.reason}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}
