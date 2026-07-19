import { useEffect, useMemo, useRef, useState } from 'react';
import { useHarnessStore } from '../store/harness';
import { main } from '../../wailsjs/go/models';

// DEFAULT_SCOUTING_WEIGHT mirrors Go's powerrankings.DefaultScoutingWeight (0.60).
// Kept in sync by hand — the Go const is the source of truth; if it moves, move this.
const DEFAULT_SCOUTING_WEIGHT = 0.6;

// PowerRankingsBoard is the M2 module view: the 32 franchises ranked by the blended
// PowerScore, in MFL's report column layout. The headline metric standardizes our
// forward-looking scouting engine (the 60) and MFL's luck-adjusted all-play result
// (the 40) to z-scores, weights them w / (1−w), and min-max's the result to [0,1];
// w is a free 0–100% slider, default 0.60. Scouting talent aggregates as the full
// roster sum OR the top-N starters (toggle). The raw MFL columns (pwr, altpwr,
// all-play, PF/PA/PP) come free in the same standings call and are shown as sortable
// context. Read-only: no writes, the live MFL fetch is the only network touch.

type SortKey =
  | 'rank'
  | 'scoutingZ'
  | 'mflPerfZ'
  | 'allPlayWinPct'
  | 'pf'
  | 'pa'
  | 'pp'
  | 'pwr'
  | 'altPwr';

export function PowerRankingsBoard() {
  const powerRankings = useHarnessStore((s) => s.powerRankings);
  const powerWeight = useHarnessStore((s) => s.powerWeight);
  const powerAgg = useHarnessStore((s) => s.powerAgg);
  const powerLoading = useHarnessStore((s) => s.powerLoading);
  const error = useHarnessStore((s) => s.error);
  const loadPowerRankings = useHarnessStore((s) => s.loadPowerRankings);

  // Local slider value for instant display; the network fetch fires only on release
  // (onPointerUp/onKeyUp), never on every drag tick.
  const [slider, setSlider] = useState<number>(powerWeight);
  const [sortKey, setSortKey] = useState<SortKey>('rank');
  const [asc, setAsc] = useState<boolean>(true); // rank ascending = best first

  // interacting suppresses the powerWeight→slider echo while the user is dragging,
  // so a resolving fetch can't snap the thumb out from under an in-progress drag.
  const interacting = useRef(false);
  // lastApplied is the weight the current rows were fetched for — a release that
  // didn't change the value skips a redundant live fetch.
  const lastApplied = useRef(powerWeight);

  useEffect(() => {
    void loadPowerRankings(powerWeight, powerAgg);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadPowerRankings]);

  // Sync the slider to the backend-echoed (clamped) weight — but never mid-drag.
  useEffect(() => {
    if (!interacting.current) {
      setSlider(powerWeight);
      lastApplied.current = powerWeight;
    }
  }, [powerWeight]);

  const rows = useMemo(() => powerRankings?.rows ?? [], [powerRankings]);

  const sorted = useMemo(() => {
    const dir = asc ? 1 : -1;
    return rows.slice().sort((a, b) => (getSortVal(a, sortKey) - getSortVal(b, sortKey)) * dir);
  }, [rows, sortKey, asc]);

  // applyWeight commits the current slider on release. It skips a redundant fetch
  // when the value hasn't changed (a no-op click on the track), and clears the
  // interacting flag so the echo-sync can resume.
  const applyWeight = () => {
    interacting.current = false;
    if (slider === lastApplied.current) return;
    lastApplied.current = slider;
    void loadPowerRankings(slider, powerAgg);
  };

  // setAgg re-fetches at the CURRENTLY-APPLIED weight with the new aggregation.
  const setAgg = (mode: string) => {
    if (mode === powerAgg) return;
    void loadPowerRankings(lastApplied.current, mode);
  };

  const onSort = (key: SortKey) => {
    if (key === sortKey) {
      setAsc(!asc);
    } else {
      setSortKey(key);
      // Rank and PA are "lower is better" → ascending; every other metric is
      // "higher is better" → descending.
      setAsc(key === 'rank' || key === 'pa');
    }
  };

  return (
    <div>
      <div className="mb-3 flex items-center gap-3">
        <h2 className="text-lg font-semibold">M2 — Power Rankings (live blend)</h2>
        {powerRankings?.ok && (
          <span className="text-xs text-slate-400">
            season {powerRankings.season} · {rows.length} franchises
            {powerLoading && ' · refreshing…'}
          </span>
        )}
      </div>

      {error && (
        <div className="mb-3 border border-red-700 bg-red-950 px-3 py-2 text-xs text-red-300">
          {error}
        </div>
      )}

      {/* Scouting rides the BasePoints proxy — carry the same honest label M1 shows. */}
      <div className="mb-3 border border-amber-700 bg-amber-950 px-3 py-2 text-xs text-amber-300">
        {powerRankings?.label ?? 'BasePoints proxy — L2 pending'} · scouting rides the same
        proxy; MFL all-play is live-season actual. Scouting is robust-standardized (median +
        MAD, outlier-resistant) and all-play z-standardized before the weighted blend, then
        scaled 0–1 for display. Scout z of 0 = a typical team.
      </div>

      {/* Weight control: free 0–100% scouting weight, fires on release. */}
      <div className="mb-4 flex flex-wrap items-center gap-3 border border-slate-700 bg-slate-800 px-3 py-2 text-sm">
        <span className="font-medium">Blend weight</span>
        <span className="text-xs text-slate-400">
          scouting {(slider * 100).toFixed(0)}% / all-play {((1 - slider) * 100).toFixed(0)}%
        </span>
        <input
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={slider}
          onChange={(e) => {
            interacting.current = true;
            setSlider(Number(e.target.value));
          }}
          onPointerUp={applyWeight}
          onKeyUp={applyWeight}
          className="h-1 w-56 cursor-pointer accent-sky-500"
          aria-label="scouting weight"
        />
        <button
          type="button"
          onClick={() => {
            interacting.current = false;
            setSlider(DEFAULT_SCOUTING_WEIGHT);
            lastApplied.current = DEFAULT_SCOUTING_WEIGHT;
            void loadPowerRankings(DEFAULT_SCOUTING_WEIGHT, powerAgg);
          }}
          className="border border-slate-600 px-2 py-1 text-xs text-slate-300 hover:bg-slate-700"
        >
          Reset 60/40
        </button>

        <span className="ml-4 font-medium">Scouting</span>
        <div className="flex">
          <AggButton active={powerAgg === 'sum'} onClick={() => setAgg('sum')}>
            Roster sum
          </AggButton>
          <AggButton active={powerAgg === 'topn'} onClick={() => setAgg('topn')}>
            Top-{powerRankings?.starterN || 'N'} starters
          </AggButton>
        </div>
      </div>

      {rows.length === 0 ? (
        <p className="text-sm text-slate-400">
          {powerLoading
            ? 'Loading live standings…'
            : error
              ? 'Could not load power rankings — see the error above.'
              : 'No power rankings yet — score the M1 board (Asset Rankings tab) first, then reload.'}
        </p>
      ) : (
        <table className="w-full text-left text-xs">
          <thead className="text-slate-400">
            <tr className="border-b border-slate-700">
              <SortTh label="#" k="rank" sortKey={sortKey} asc={asc} onSort={onSort} />
              <th className="py-1 pr-2">Team</th>
              <th className="py-1 pr-2 text-right">Power</th>
              <SortTh label="Scout z" k="scoutingZ" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
              <SortTh label="AllPlay%" k="allPlayWinPct" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
              <th className="py-1 pr-2 text-right">Record</th>
              <th className="py-1 pr-2 text-right">AllPlay</th>
              <SortTh label="PF" k="pf" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
              <SortTh label="PA" k="pa" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
              <SortTh label="PP" k="pp" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
              <SortTh label="MFL Pwr" k="pwr" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
              <SortTh label="AltPwr" k="altPwr" sortKey={sortKey} asc={asc} onSort={onSort} align="right" />
            </tr>
          </thead>
          <tbody>
            {sorted.map((r) => (
              <tr key={r.franchiseID} className="border-b border-slate-800 hover:bg-slate-800">
                <td className="py-1 pr-2 text-slate-400">{r.rank}</td>
                <td className="py-1 pr-2">{r.name}</td>
                <td className="py-1 pr-2 text-right font-semibold text-sky-300">
                  {r.powerScore.toFixed(3)}
                </td>
                <td className="py-1 pr-2 text-right">{r.scoutingZ.toFixed(2)}</td>
                <td className="py-1 pr-2 text-right">{(r.allPlayWinPct * 100).toFixed(1)}%</td>
                <td className="py-1 pr-2 text-right">
                  {r.h2hW}-{r.h2hL}
                  {r.h2hT > 0 ? `-${r.h2hT}` : ''}
                </td>
                <td className="py-1 pr-2 text-right">
                  {r.allPlayW}-{r.allPlayL}
                  {r.allPlayT > 0 ? `-${r.allPlayT}` : ''}
                </td>
                <td className="py-1 pr-2 text-right">{r.pf.toFixed(1)}</td>
                <td className="py-1 pr-2 text-right">{r.pa.toFixed(1)}</td>
                <td className="py-1 pr-2 text-right">{r.pp.toFixed(1)}</td>
                <td className="py-1 pr-2 text-right text-slate-400">{r.pwr.toFixed(2)}</td>
                <td className="py-1 pr-2 text-right text-slate-400">{r.altPwr.toFixed(1)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// getSortVal maps a row + key to a comparable number. Rank ascends (1 = best); every
// other column is a magnitude where higher is better, handled by the sort direction.
function getSortVal(r: main.PowerRow, key: SortKey): number {
  switch (key) {
    case 'rank':
      return r.rank;
    case 'scoutingZ':
      return r.scoutingZ;
    case 'mflPerfZ':
      return r.mflPerfZ;
    case 'allPlayWinPct':
      return r.allPlayWinPct;
    case 'pf':
      return r.pf;
    case 'pa':
      return r.pa;
    case 'pp':
      return r.pp;
    case 'pwr':
      return r.pwr;
    case 'altPwr':
      return r.altPwr;
    default:
      return r.rank;
  }
}

function SortTh({
  label,
  k,
  sortKey,
  asc,
  onSort,
  align,
}: {
  label: string;
  k: SortKey;
  sortKey: SortKey;
  asc: boolean;
  onSort: (k: SortKey) => void;
  align?: 'right';
}) {
  const active = sortKey === k;
  return (
    <th
      className={`cursor-pointer select-none py-1 pr-2 hover:text-slate-200 ${
        align === 'right' ? 'text-right' : ''
      } ${active ? 'text-sky-300' : ''}`}
      onClick={() => onSort(k)}
    >
      {label}
      {active ? (asc ? ' ▲' : ' ▼') : ''}
    </th>
  );
}

function AggButton({
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
      className={`border px-2 py-1 text-xs ${
        active
          ? 'border-sky-600 bg-sky-950 text-sky-300'
          : 'border-slate-600 text-slate-400 hover:bg-slate-800'
      }`}
    >
      {children}
    </button>
  );
}
