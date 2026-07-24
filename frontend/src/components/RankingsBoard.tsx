import { useEffect, useMemo, useState } from 'react';
import { useHarnessStore } from '../store/harness';
import { main } from '../../wailsjs/go/models';

// RankingsBoard is the M1 module view: the REAL 32-team ranked board read back
// from the B6 output store. Three client-side views over the same persisted rows
// (the backend never re-sorts; B6's order IS the ranking):
//   - global ranked list, with a position filter
//   - per-team drill-down (AD-20): pick a franchise, see its players ranked
//   - cap-efficiency: the same rows ordered by AdjustedScore per $M of salary
// Every score surface carries the proxy label — BasePoints is the labeled MFL
// YTD placeholder until the real L2 block ships. Debuggability over polish.

const POSITIONS = ['QB', 'RB', 'WR', 'TE', 'K', 'DT', 'DE', 'LB', 'CB', 'S'] as const;

type ViewMode = 'rank' | 'capeff';

export function RankingsBoard() {
  const rankings = useHarnessStore((s) => s.rankings);
  const scoreReport = useHarnessStore((s) => s.scoreReport);
  const scoring = useHarnessStore((s) => s.scoring);
  const loadRankings = useHarnessStore((s) => s.loadRankings);
  const scoreLeague = useHarnessStore((s) => s.scoreLeague);

  const [position, setPosition] = useState<string>('ALL');
  const [franchise, setFranchise] = useState<string>('ALL');
  const [view, setView] = useState<ViewMode>('rank');

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

  const visible = useMemo(() => {
    let v = rows;
    if (position !== 'ALL') v = v.filter((r) => r.position === position);
    if (franchise !== 'ALL') v = v.filter((r) => r.franchiseID === franchise);
    if (view === 'capeff') {
      // Cap-efficiency view: rows WITH a defined efficiency, ordered by it. Rows
      // without one ($0 salary) are excluded from this view, never faked as 0.
      v = v.filter((r) => r.capEffOK).slice().sort((a, b) => b.capEff - a.capEff);
    }
    return v;
  }, [rows, position, franchise, view]);

  return (
    <div>
      <div className="mb-3 flex items-center gap-3">
        <h2 className="text-lg font-semibold">M1 — Asset Rankings (live league)</h2>
        <button
          type="button"
          onClick={() => void scoreLeague()}
          disabled={scoring}
          className="border border-emerald-600 px-3 py-1 text-sm text-emerald-400 hover:bg-emerald-950 disabled:opacity-50"
        >
          {scoring ? 'Scoring…' : 'Score League'}
        </button>
        {rankings && (
          <span className="text-xs text-slate-400">
            season {rankings.season} · rulebook v{rankings.configVersion} · {rows.length} scored
          </span>
        )}
      </div>

      {/* The honest-labeling banner: the board is a validation surface, not a
          published ranking, until real L2 base scoring lands. */}
      <div className="mb-3 border border-amber-700 bg-amber-950 px-3 py-2 text-xs text-amber-300">
        {rankings?.label ?? 'BasePoints proxy — L2 pending'}
      </div>

      {rankings?.warning && (
        <div className="mb-3 border border-orange-700 bg-orange-950 px-3 py-2 text-xs text-orange-300">
          {rankings.warning}
        </div>
      )}

      {scoreReport && <ScoreReportPanel report={scoreReport} />}

      <div className="mb-3 flex flex-wrap items-center gap-2 text-sm">
        <select
          value={position}
          onChange={(e) => setPosition(e.target.value)}
          className="border border-slate-600 bg-slate-800 px-2 py-1"
        >
          <option value="ALL">All positions</option>
          {POSITIONS.map((p) => (
            <option key={p} value={p}>{p}</option>
          ))}
        </select>
        <select
          value={franchise}
          onChange={(e) => setFranchise(e.target.value)}
          className="border border-slate-600 bg-slate-800 px-2 py-1"
        >
          <option value="ALL">All teams</option>
          {franchises.map((f) => (
            <option key={f} value={f}>Franchise {f}</option>
          ))}
        </select>
        <div className="flex">
          <ViewButton active={view === 'rank'} onClick={() => setView('rank')}>By rank</ViewButton>
          <ViewButton active={view === 'capeff'} onClick={() => setView('capeff')}>Cap efficiency</ViewButton>
        </div>
      </div>

      {rows.length === 0 ? (
        <p className="text-sm text-slate-400">
          No persisted scores for the active config yet — run “Score League”.
        </p>
      ) : (
        <table className="w-full text-left text-xs">
          <thead className="text-slate-400">
            <tr className="border-b border-slate-700">
              <th className="py-1 pr-2">#</th>
              <th className="py-1 pr-2">Player</th>
              <th className="py-1 pr-2">Pos</th>
              <th className="py-1 pr-2">Team</th>
              <th className="py-1 pr-2 text-right">Base (proxy)</th>
              <th className="py-1 pr-2 text-right">AgePull</th>
              <th className="py-1 pr-2 text-right">L4</th>
              <th className="py-1 pr-2">CapTier</th>
              <th className="py-1 pr-2 text-right">Salary</th>
              <th className="py-1 pr-2 text-right">Adjusted</th>
              <th className="py-1 pr-2 text-right">Adj/$M</th>
            </tr>
          </thead>
          <tbody>
            {visible.map((r) => (
              <tr key={r.mflID} className="border-b border-slate-800 hover:bg-slate-800">
                <td className="py-1 pr-2 text-slate-400">{r.rank}</td>
                <td className="py-1 pr-2">{r.name}</td>
                <td className="py-1 pr-2">{r.position}</td>
                <td className="py-1 pr-2">{r.franchiseID || '—'}</td>
                <td className="py-1 pr-2 text-right">{r.basePoints.toFixed(2)}</td>
                <td className="py-1 pr-2 text-right">{r.agePull.toFixed(3)}</td>
                <td className="py-1 pr-2 text-right">{r.l4Combined.toFixed(3)}</td>
                <td className="py-1 pr-2">{r.capTier}</td>
                <td className="py-1 pr-2 text-right">{r.salary.toFixed(2)}</td>
                <td className="py-1 pr-2 text-right font-semibold">{r.adjustedScore.toFixed(2)}</td>
                <td className="py-1 pr-2 text-right">{r.capEffOK ? r.capEff.toFixed(2) : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
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
    <div className="mb-3 border border-slate-700 bg-slate-800 px-3 py-2 text-xs">
      <p>
        {rep.skippedExisting
          ? `Already scored under rulebook v${rep.configVersion} — ${rep.existing} persisted rows served as-is (append-only; bump the rulebook to re-score).`
          : `Scored ${rep.scored} players under rulebook v${rep.configVersion} (${rep.zeroBase} with no ${rep.season - 1} YTD record` +
            (rep.negativeBase > 0 ? `; ${rep.negativeBase} negative totals floored to 0 — check the proxy data` : '') +
            ').'}
      </p>
      {rep.excluded && rep.excluded.length > 0 && (
        <details className="mt-1">
          <summary className="cursor-pointer text-amber-400">
            {rep.excluded.length} excluded (unscored — reasons)
          </summary>
          <ul className="mt-1 list-inside list-disc text-slate-300">
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

function ViewButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`border px-2 py-1 text-sm ${
        active
          ? 'border-sky-600 bg-sky-950 text-sky-300'
          : 'border-slate-600 text-slate-400 hover:bg-slate-800'
      }`}
    >
      {children}
    </button>
  );
}
