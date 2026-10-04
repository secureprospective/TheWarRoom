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

// PowerRankingsBoard is the M2 module view: the 32 franchises ranked in one of two views.
// This season counts each roster's on-field-now (league points per game) and blends its z-score
// with the season's results: MFL's all-play record when the league reports it, otherwise points
// for as a share of the league's best. The roster's weight follows the weeks played (Auto: 4 ÷
// (4 + weeks)) until the slider sets it. The franchise counts each roster's dynasty value, with an
// older roster counting against it. The roster counts as the lineup the league's rules allow
// (this season's default), the top N by value, or the whole roster (the franchise's default).
// Values come from the latest model run; Δ is each team's move since the board built from the
// previous one. Age, Proj, Luck and Cap room are context: they never enter the score. MFL's
// report columns come with the same standings call.

type SortKey =
  | "rank"
  | "rosterValue"
  | "rosterZ"
  | "results"
  | "pf"
  | "pa"
  | "pp"
  | "pwr"
  | "altPwr"
  | "age"
  | "proj"
  | "luck"
  | "capRoom";

// Tactical carries the context and the full MFL report; Matrix collapses to the blend essentials.
const COLS =
  "34px 40px minmax(150px, 1fr) 88px 70px 66px 74px 52px 76px 56px 72px 66px 66px 58px 58px 58px 66px 60px";

// AGG labels the three roster counts, keyed by m2service's Agg* values.
const AGG: { mode: string; label: (n: number) => string; tip: string }[] = [
  {
    mode: "lineup",
    label: () => "Lineup",
    tip: "The best lineup the league's starter rules allow",
  },
  {
    mode: "topn",
    label: (n) => `Top ${n || "N"} by value`,
    tip: "The most valuable players, whatever their positions",
  },
  {
    mode: "sum",
    label: () => "Whole roster",
    tip: "Every rostered player, bench included",
  },
];
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
  const powerAuto = useAppStore((s) => s.powerAuto);
  const powerAgg = useAppStore((s) => s.powerAgg);
  const powerAggChoice = useAppStore((s) => s.powerAggChoice);
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
    void loadPowerRankings(
      powerWeight,
      powerAuto,
      powerAggChoice[powerView] ?? "",
      powerView,
    );
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

  // applyWeight commits the current slider on release, which turns Auto off. It
  // skips a redundant fetch when the value hasn't changed (a no-op click on the
  // track), and clears the interacting flag so the echo-sync can resume.
  const applyWeight = () => {
    interacting.current = false;
    if (slider === lastApplied.current) return;
    lastApplied.current = slider;
    void loadPowerRankings(slider, false, powerAgg, powerView);
  };

  // setAgg re-fetches at the CURRENTLY-APPLIED weight with the new roster count.
  const setAgg = (mode: string) => {
    if (mode === powerAgg) return;
    void loadPowerRankings(lastApplied.current, powerAuto, mode, powerView);
  };

  // setView switches between this season and the franchise, each with its own roster count.
  const setView = (view: string) => {
    if (view === powerView) return;
    void loadPowerRankings(
      lastApplied.current,
      powerAuto,
      powerAggChoice[view] ?? "",
      view,
    );
  };

  const onSort = (key: SortKey) => {
    if (key === sortKey) {
      setAsc(!asc);
    } else {
      setSortKey(key);
      // Rank and PA are "lower is better" → ascending; every other metric is
      // "higher is better" → descending.
      setAsc(key === "rank" || key === "pa" || key === "age");
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
            season {powerRankings.season}
            {rows.length > 0 && ` · ${rows.length} franchises`}
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
        weeks={powerRankings?.ok ? powerRankings.weeksScored : -1}
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
          : "the franchise ranks on the roster's dynasty value, an older roster counting against it at half the roster's weight, then scales 0–1."}{" "}
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
        {AGG.map((a) => (
          <button
            key={a.mode}
            type="button"
            title={a.tip}
            className={`twr-chip${powerAgg === a.mode ? " is-on" : ""}`}
            aria-pressed={powerAgg === a.mode}
            onClick={() => setAgg(a.mode)}
          >
            {a.label(powerRankings?.starterN ?? 0)}
          </button>
        ))}
      </div>

      {/* Weight control: Auto follows the weeks played; the slider (fires on release) overrides
          it. This season only. */}
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
            className={`twr-chip${powerAuto ? " is-on" : ""}`}
            aria-pressed={powerAuto}
            title="The roster's weight shrinks as weeks are played: 80% after week 1, 50% after week 4"
            onClick={() => {
              if (powerAuto) return;
              interacting.current = false;
              void loadPowerRankings(
                lastApplied.current,
                true,
                powerAgg,
                powerView,
              );
            }}
          >
            Auto
          </button>
          {/* After the controls, so the slider does not move when Auto turns off. */}
          <span
            style={{
              fontFamily: "var(--mono)",
              fontSize: 11,
              color: "var(--text-tertiary)",
            }}
          >
            {powerAuto
              ? `4 ÷ (4 + ${powerRankings?.weeksScored ?? 0} weeks played)`
              : "set by the slider"}
          </span>
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
            <span
              className="twr-r twr-hide-mtx"
              title="The counted players' age, weighted by value: context only, except in the franchise view"
            >
              <SortHeader
                label="Age"
                sortKey="age"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span
              className="twr-r twr-hide-mtx"
              title="Projected regular-season record: wins so far plus each remaining game's win chance. Context only"
            >
              <SortHeader
                label="Proj"
                sortKey="proj"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span
              className="twr-r twr-hide-mtx"
              title="Head-to-head wins above what the all-play rate would have earned: schedule luck. Context only"
            >
              <SortHeader
                label="Luck"
                sortKey="luck"
                activeKey={sortKey}
                dir={dir}
                onSort={onSort}
              />
            </span>
            <span
              className="twr-r twr-hide-mtx"
              title="Cap room: the cap less what MFL charges, dead cap included. Context only"
            >
              <SortHeader
                label="Cap room"
                sortKey="capRoom"
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
              <span className="twr-c-num twr-r">{fixed(r.rosterZ, 2)}</span>
              <span className="twr-c-num twr-r">
                {(r.results * 100).toFixed(1)}%
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.age > 0 ? r.age.toFixed(1) : "—"}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.hasProj
                  ? `${r.projW.toFixed(1)}-${r.projL.toFixed(1)}`
                  : "—"}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.hasLuck ? signed(r.luck) : "—"}
              </span>
              <span
                className="twr-c-num twr-r twr-hide-mtx"
                title={r.hasCap ? `dead cap $${r.deadCap.toFixed(2)}M` : ""}
              >
                {r.hasCap ? millions(r.capRoom) : "—"}
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

// fixed is toFixed without the "-0.00" a tiny negative prints.
function fixed(v: number, digits: number): string {
  const s = v.toFixed(digits);
  return Number(s) === 0 ? (0).toFixed(digits) : s;
}

// signed shows one decimal with its sign, and a value that rounds to zero as plain 0.0.
function signed(v: number): string {
  const r = Math.round(v * 10) / 10;
  if (r === 0) return "0.0";
  return `${r > 0 ? "+" : "−"}${Math.abs(r).toFixed(1)}`;
}

// millions shows $M to one decimal, a shortfall with a leading minus, never "$-0.0M".
function millions(v: number): string {
  const r = Math.round(v * 10) / 10;
  return `${r < 0 ? "−" : ""}$${Math.abs(r).toFixed(1)}M`;
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
    case "age":
      return r.age > 0 ? r.age : Number.MAX_VALUE;
    case "proj":
      return r.hasProj ? r.projW : -1;
    case "luck":
      return r.hasLuck ? r.luck : -Number.MAX_VALUE;
    case "capRoom":
      return r.hasCap ? r.capRoom : -Number.MAX_VALUE;
    default:
      return r.rank;
  }
}
