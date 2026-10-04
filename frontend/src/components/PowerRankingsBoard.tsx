import { useEffect, useMemo, useRef, useState } from "react";
import { useAppStore } from "../store/app";
import { main } from "../../wailsjs/go/models";
import {
  SortHeader,
  EngraveState,
  SkeletonState,
  FreshnessBar,
  PhaseBar,
  DeltaRank,
} from "./board/primitives";

// DEFAULT_ROSTER_WEIGHT mirrors Go's powerrankings.DefaultRosterWeight (0.60).
// Kept in sync by hand — the Go const is the source of truth; if it moves, move this.
const DEFAULT_ROSTER_WEIGHT = 0.6;

// PowerRankingsBoard is the M2 module view: the 32 franchises ranked in one of two views.
// This season sums each roster's on-field-now (league points per game) and blends its z-score
// with the season's results at a free 0–100% weight (default 60/40): MFL's all-play record when
// the league reports it, otherwise points for as a share of the league's best. The franchise sums each
// roster's dynasty value and ranks on the roster alone. Values come from the latest model run;
// Δ is each team's move since the board built from the previous one. The roster aggregates as
// the full sum or the top-N starters. MFL's report columns come with the same standings call.

type SortKey =
  | "rank"
  | "rosterValue"
  | "rosterZ"
  | "results"
  | "pf"
  | "pa"
  | "pp"
  | "pwr"
  | "altPwr";

// Tactical carries the full MFL report; Matrix collapses to the blend essentials.
const COLS =
  "34px 40px minmax(150px, 1fr) 88px 70px 66px 74px 66px 66px 58px 58px 58px 66px 60px";
const COLS_MTX = "24px 36px minmax(120px, 1fr) 76px 62px 70px";

// RESULTS names the result the season blend read, keyed by m2service's Perf* values.
const RESULTS: Record<
  string,
  { banner: string; short: string; column: string; tip: string }
> = {
  "all-play": {
    banner: "all-play record",
    short: "all-play",
    column: "AllPlay%",
    tip: "All-play win %: the season blend's results side",
  },
  "points for": {
    banner: "points for (MFL reports no all-play for this league)",
    short: "points for",
    column: "PF%",
    tip: "Points for as a share of the league's best: the season blend's results side",
  },
  none: {
    banner: "results (none yet)",
    short: "results",
    column: "Results",
    tip: "No week scored yet, so results do not move the blend",
  },
};

export function PowerRankingsBoard() {
  const powerRankings = useAppStore((s) => s.powerRankings);
  const powerWeight = useAppStore((s) => s.powerWeight);
  const powerAgg = useAppStore((s) => s.powerAgg);
  const powerView = useAppStore((s) => s.powerView);
  const powerLoading = useAppStore((s) => s.powerLoading);
  const error = useAppStore((s) => s.error);
  const loadPowerRankings = useAppStore((s) => s.loadPowerRankings);

  // Local slider value for instant display; the network fetch fires only on release
  // (onPointerUp/onKeyUp), never on every drag tick.
  const [slider, setSlider] = useState<number>(powerWeight);
  const [sortKey, setSortKey] = useState<SortKey>("rank");
  const [asc, setAsc] = useState<boolean>(true); // rank ascending = best first

  // interacting suppresses the powerWeight→slider echo while the user is dragging,
  // so a resolving fetch can't snap the thumb out from under an in-progress drag.
  const interacting = useRef(false);
  // lastApplied is the weight the current rows were fetched for — a release that
  // didn't change the value skips a redundant live fetch.
  const lastApplied = useRef(powerWeight);

  useEffect(() => {
    void loadPowerRankings(powerWeight, powerAgg, powerView);
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
    return rows
      .slice()
      .sort((a, b) => (getSortVal(a, sortKey) - getSortVal(b, sortKey)) * dir);
  }, [rows, sortKey, asc]);

  // applyWeight commits the current slider on release. It skips a redundant fetch
  // when the value hasn't changed (a no-op click on the track), and clears the
  // interacting flag so the echo-sync can resume.
  const applyWeight = () => {
    interacting.current = false;
    if (slider === lastApplied.current) return;
    lastApplied.current = slider;
    void loadPowerRankings(slider, powerAgg, powerView);
  };

  // setAgg re-fetches at the CURRENTLY-APPLIED weight with the new aggregation.
  const setAgg = (mode: string) => {
    if (mode === powerAgg) return;
    void loadPowerRankings(lastApplied.current, mode, powerView);
  };

  // setView switches between this season and the franchise at the season slider's weight.
  const setView = (view: string) => {
    if (view === powerView) return;
    void loadPowerRankings(lastApplied.current, powerAgg, view);
  };

  const onSort = (key: SortKey) => {
    if (key === sortKey) {
      setAsc(!asc);
    } else {
      setSortKey(key);
      // Rank and PA are "lower is better" → ascending; every other metric is
      // "higher is better" → descending.
      setAsc(key === "rank" || key === "pa");
    }
  };
  const dir = asc ? "asc" : "desc";
  const results = RESULTS[powerRankings?.performance ?? "none"] ?? RESULTS.none;
  const allPlay = powerRankings?.performance === "all-play";

  return (
    <div
      style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        {powerRankings?.ok && (
          <span
            style={{
              fontFamily: "var(--mono)",
              fontSize: 11,
              color: "var(--text-tertiary)",
            }}
          >
            season {powerRankings.season} · {rows.length} franchises
            {powerLoading && " · refreshing…"}
          </span>
        )}
      </div>

      {error && <div className="twr-banner twr-banner--warn">{error}</div>}

      {/* B-5 degradation contract: an MFL standings outage now serves the last-known-good
          board with a CACHED edge rather than blanking M2. The bar states the age; the
          rows below are untouched and fully readable. PhaseBar is separate and neutral:
          an offseason board is final, not degraded. */}
      <FreshnessBar
        freshness={powerRankings?.freshness}
        board="Power Rankings"
      />
      <PhaseBar
        weeks={powerRankings?.weeksScored ?? -1}
        seasonWeeks={powerRankings?.seasonWeeks ?? 0}
      />

      <div className="twr-banner twr-banner--caution">
        {powerRankings?.label || "No model run yet"}
        {powerRankings?.previousRunID
          ? ` · Δ against model run #${powerRankings.previousRunID}`
          : ""}
        . Roster value is z-scored with median and MAD, so one stacked roster
        cannot move the scale;{" "}
        {powerView === "season"
          ? `this season blends it with MFL's ${results.banner}, then scales 0–1.`
          : "the franchise ranks on the roster alone, scaled 0–1."}{" "}
        Roster z of 0 = a typical team.
      </div>

      <div
        className="twr-panel"
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "center",
          gap: 12,
        }}
      >
        <span style={{ fontWeight: 500, color: "var(--text-primary)" }}>
          View
        </span>
        <button
          type="button"
          className={`twr-chip${powerView === "season" ? " is-on" : ""}`}
          aria-pressed={powerView === "season"}
          onClick={() => setView("season")}
        >
          This season
        </button>
        <button
          type="button"
          className={`twr-chip${powerView === "franchise" ? " is-on" : ""}`}
          aria-pressed={powerView === "franchise"}
          onClick={() => setView("franchise")}
        >
          The franchise
        </button>
        <span
          style={{
            marginLeft: 12,
            fontWeight: 500,
            color: "var(--text-primary)",
          }}
        >
          Roster
        </span>
        <button
          type="button"
          className={`twr-chip${powerAgg === "sum" ? " is-on" : ""}`}
          aria-pressed={powerAgg === "sum"}
          onClick={() => setAgg("sum")}
        >
          Roster sum
        </button>
        <button
          type="button"
          className={`twr-chip${powerAgg === "topn" ? " is-on" : ""}`}
          aria-pressed={powerAgg === "topn"}
          onClick={() => setAgg("topn")}
        >
          Top-{powerRankings?.starterN || "N"} starters
        </button>
      </div>

      {/* Weight control: free 0–100% roster-value weight, fires on release. This season only. */}
      {powerView === "season" && (
        <div
          className="twr-panel"
          style={{
            display: "flex",
            flexWrap: "wrap",
            alignItems: "center",
            gap: 12,
          }}
        >
          <span style={{ fontWeight: 500, color: "var(--text-primary)" }}>
            Blend weight
          </span>
          <span
            style={{
              fontFamily: "var(--mono)",
              fontSize: 11,
              color: "var(--text-tertiary)",
            }}
          >
            roster {(slider * 100).toFixed(0)}% / {results.short}{" "}
            {((1 - slider) * 100).toFixed(0)}%
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
            className="twr-slider"
            style={{ width: 220 }}
            aria-label="roster-value weight"
          />
          <button
            type="button"
            className="twr-chip"
            onClick={() => {
              interacting.current = false;
              setSlider(DEFAULT_ROSTER_WEIGHT);
              lastApplied.current = DEFAULT_ROSTER_WEIGHT;
              void loadPowerRankings(
                DEFAULT_ROSTER_WEIGHT,
                powerAgg,
                powerView,
              );
            }}
          >
            Reset 60/40
          </button>
        </div>
      )}

      {rows.length === 0 ? (
        powerLoading ? (
          <SkeletonState />
        ) : (
          <EngraveState
            lines={
              error
                ? [
                    "M2 Power Rankings",
                    "— could not load standings —",
                    "see the error above",
                  ]
                : [
                    "M2 Power Rankings",
                    "— no model run yet —",
                    "run Score League on the Assets board, then reload",
                  ]
            }
          />
        )
      ) : (
        <div
          className="twr-board"
          style={{
            ["--twr-cols" as string]: COLS,
            ["--twr-cols-mtx" as string]: COLS_MTX,
          }}
        >
          <div className="twr-board__sub">
            <SortHeader
              label="#"
              sortKey="rank"
              activeKey={sortKey}
              dir={dir}
              onSort={onSort}
            />
            <span
              className="twr-r"
              title="Move since the board built from the previous model run"
            >
              Δ
            </span>
            <span>Team</span>
            <span className="twr-r">Power</span>
            <span
              className="twr-r"
              title="The roster's summed league points per game in this view"
            >
              <SortHeader
                label="Value"
                sortKey="rosterValue"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r">
              <SortHeader
                label="Roster z"
                sortKey="rosterZ"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r" title={results.tip}>
              <SortHeader
                label={results.column}
                sortKey="results"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r twr-hide-mtx">Record</span>
            <span className="twr-r twr-hide-mtx">AllPlay</span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader
                label="PF"
                sortKey="pf"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader
                label="PA"
                sortKey="pa"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader
                label="PP"
                sortKey="pp"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader
                label="MFL Pwr"
                sortKey="pwr"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span className="twr-r twr-hide-mtx">
              <SortHeader
                label="AltPwr"
                sortKey="altPwr"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
          </div>
          {sorted.map((r) => (
            <div key={r.franchiseID} className="twr-board__row">
              <span className="twr-c-rank">{r.rank}</span>
              <span className="twr-r">
                <DeltaRank delta={r.rankDelta} ok={r.deltaOK} />
              </span>
              <span className="twr-c-name">{r.name}</span>
              <span className="twr-c-adj twr-r">{r.powerScore.toFixed(3)}</span>
              <span className="twr-c-num twr-r">
                {r.rosterValue.toFixed(1)}
              </span>
              <span className="twr-c-num twr-r">{r.rosterZ.toFixed(2)}</span>
              <span className="twr-c-num twr-r">
                {(r.results * 100).toFixed(1)}%
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.h2hW}-{r.h2hL}
                {r.h2hT > 0 ? `-${r.h2hT}` : ""}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {allPlay
                  ? `${r.allPlayW}-${r.allPlayL}${r.allPlayT > 0 ? `-${r.allPlayT}` : ""}`
                  : "—"}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.pf.toFixed(1)}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.pa.toFixed(1)}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.pp.toFixed(1)}
              </span>
              <span className="twr-c-diag twr-r twr-hide-mtx">
                {r.pwr.toFixed(2)}
              </span>
              <span className="twr-c-diag twr-r twr-hide-mtx">
                {r.altPwr.toFixed(1)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// getSortVal maps a row + key to a comparable number. Rank ascends (1 = best); every
// other column is a magnitude where higher is better, handled by the sort direction.
function getSortVal(r: main.PowerRow, key: SortKey): number {
  switch (key) {
    case "rank":
      return r.rank;
    case "rosterValue":
      return r.rosterValue;
    case "rosterZ":
      return r.rosterZ;
    case "results":
      return r.results;
    case "pf":
      return r.pf;
    case "pa":
      return r.pa;
    case "pp":
      return r.pp;
    case "pwr":
      return r.pwr;
    case "altPwr":
      return r.altPwr;
    default:
      return r.rank;
  }
}
