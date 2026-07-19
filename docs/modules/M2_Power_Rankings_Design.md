# M2 — Power Rankings: Design + Research (pre-build)

**Status:** DESIGN LOCKED (pending 2 small confirms), NO CODE WRITTEN YET.
**Branch:** `session/m2-power-rankings` (off main, clean).
**Date:** 2026-07-18.
**Spec source:** `docs/modules/Module_Specifications.md` → "Module 2: Weekly Power Rankings".

---

## Core decision

M2 is **NOT** "invent a power formula." MFL already computes rich power-ranking
data and exposes it via the API. M2 = **fetch MFL's report data + blend it with
our scouting engine's roster-strength, displayed in MFL's report column layout.**

> **UPDATE 2026-07-19 (post-build, GLM-5.2 two-stage math panel + Christopher decision):**
> Normalization changed from min-max to **z-score standardization**, and scouting
> aggregation is now **both sum AND top-N starters (user toggle)**. See "Locked
> decisions" at the bottom. The min-max description below is SUPERSEDED — kept for the
> reasoning trail.

**Blend (the headline metric) — CURRENT (z-score):**
```
scoutingZ = (scoutingAgg − mean) / std   ; mflPerfZ = (allPlayWin% − mean) / std
blend     = w · scoutingZ + (1 − w) · mflPerfZ          default w = 0.60
PowerScore = minmax(blend across 32 teams) → [0,1]   (display only)
```
Z-score (not min-max) so w controls each component's true influence on the spread,
and a dynasty super-team / tanked roster can't compress the field. scoutingAgg =
franchise roster **sum** OR **top-N** (N = league `Starters.Count`), user toggle.

**SUPERSEDED original (min-max):**
```
PowerScore = w · scoutingNorm + (1 − w) · mflPerfNorm      default w = 0.60
```
- `scoutingNorm` = per-franchise roster **Adjusted Score** (from M1's persisted
  board, `output.Reader().Scores`), aggregated per franchise, min-max normalized
  across the 32 teams. (The "Both, sortable" earlier answer: compute BOTH total
  Adjusted Score AND top-N starters sum; show both, sortable.)
- `mflPerfNorm` = MFL in-season performance, normalized [0,1] across 32 teams.
- **w is USER-ADJUSTABLE in the UI**, default 0.60 (scouting is the 60). 60/40 is
  the opinionated base, not a hardcode — spec calls for "configurable, testable"
  weights. User can also sort by any raw column and ignore the blend entirely.

### OPEN CONFIRM #1 — what `mflPerfNorm` resolves to
**Recommendation: all-play win% = `all_play_w ÷ 527`** (computed by us), NOT MFL's
`pwr`. Reasoning (from the math research below): MFL's `pwr` is unbounded and
league-specific (bakes in roster sizes + raw point scales), so min-max'ing it
across 32 teams is arbitrary; all-play win% is already a clean, continuous,
monotonic, transparent [0,1] luck-adjusted result signal — the ideal complement
to the forward-looking scouting 60%. `pwr`/`altpwr` still DISPLAYED as columns
(they come free in the same API call), just not used as the blend input.

### OPEN CONFIRM #2 — weight slider bounds
Free 0–100%, or snapped presets? (Default 0.60 either way.)

---

## MFL API research (2 Haiku bots, 2026-07-18) — treat as well-cited LEADS; confirm field names on a live pull

### API availability (bot 1)
- **`TYPE=leagueStandings`** returns **11 of 16 report columns DIRECTLY** in one
  call. Confirmed field names (via ffscrapr `mfl_standings.R` + python-mfl):
  - `h2hw`/`h2hl`/`h2ht` — head-to-head W-L-T
  - `all_play_w`/`all_play_l`/`all_play_t` — all-play record
  - `pf`, `pa`, `avgpf`, `avgpa` — points for/against + averages
  - `pp` — **potential points** (optimal-lineup sum)
  - `pwr` — **Power Rank** (direct!)
  - `altpwr` — **Alternate Power Rank** (direct!)
  - `salary` — dynasty cap
- Params: `L`=league ID (required), `JSON=1`. All-play needs no special param
  (league-setting; fields null/0 only if the league disabled all-play).
- **No native `powerRankings`/`allPlay`/`pointsHistory` export** — pwr/altpwr/
  all-play are FIELDS inside leagueStandings, not separate exports.
- **5 columns are NOT in the API** → require a `TYPE=weeklyResults` per-week loop
  (`W`=week or `YTD`, `JSON=1`) + local optimal-lineup math:
  Bench Points, Max PF, Min PF, Coulda Won, Woulda Lost.
  - Bench = Σ non-starter scores per week. Max/Min PF = max/min weekly started sum.
  - Coulda Won = weeks actual<opp but optimal>opp. Woulda Lost = weeks actual>opp
    but opponent's optimal>actual... (opponent-optimal vs our actual).

### Power-rank math (bot 2) — documented on MFL/DLF forums
- **Power Rank** = A+B+C+D+E: pts-per-week-per-starter + pts-per-week-per-bench +
  all-play-wins-per-week + actual wins + division wins. Unbounded, unnormalized
  (~42–51 in this league). Tracks a PF-per-week + actual-wins blend.
- **Alternate Power Rank** = rank-sum `(N−RecordRank+1)+(N−PFRank+1)+(N−BreakdownRank+1)`,
  Breakdown = all-play-sorted standings. Scale 3 → 3N = 96 (32 teams). `.5` values
  = tie-averaged ranks. Coarse — discards magnitude.
- **All-play games/team = 527 = 31 opponents × 17 weeks.** CONFIRMED.
- All-play-alone as AltPR driver is **REJECTED**: Washington `all_play_w=438`
  (more) → AltPR 86.0, but Atlanta `all_play_w=421` (fewer) → AltPR 87.0 (higher);
  Atlanta's better actual record (12-1 vs 10-3) lifts it. AltPR is non-monotonic
  in all-play wins.

---

## Build plan

**slice-1 (this branch):** new Layer-1 `internal/ingestion/leaguestandings` fetcher —
inherits B1 transport, DiscoverHost-first, league-scoped, `MFLList` collapse-tolerant
decode (rosters template). Normalize → per-franchise standings rows. Compose with
M1 output (aggregate Adjusted Score per franchise) → blended PowerScore + weight
control. React "Power Rankings" view in the MFL column layout, sortable, weight
slider (default 0.60). Pure aggregation package + depguard, app-level IPC `GetPowerRankings()`.

**slice-2 (later):** `weeklyResults` per-week loop → the 5 optimal-lineup columns
(Bench, Max/Min PF, Coulda-Won, Woulda-Lost). Movement indicators (week-over-week
Δ) need a persisted history table — also slice-2+.

## Data plumbing already confirmed present
- `output.Reader().Scores(ctx, season, cfgVersion)` → per-player `SeasonScore`
  incl. `AdjustedScore`, `MFLID`. (`internal/output/output.go:176`, `helpers.go:21`)
- `state.Reader().Player(mflID)` → `FranchiseID` for per-franchise aggregation.
- `rulebook.FranchiseNames()` → id→name map (M4 slice-3 added/uses this;
  `internal/store/rulebook/helpers.go:21`). Rail/rows show real team names, id fallback.
- M1 IPC pattern to mirror: `m1_app.go` `GetRankings()` (degrade-safe name resolution,
  proxy label, timeout).

## Gotchas (from project CLAUDE.md header)
- Wails nil slice → JSON `null`: guard every list `?? []` at the React edge (D9).
- Numbers never from the model / staged-confirm — read-only view, so N/A here (no writes).
- lint GOMEMLIMIT / `run.concurrency: 1` (CT105 ~2GB). Beelink build: `-tags webkit2_41`.
- BasePoints is a LABELED MFL 2025-YTD proxy (L2 pending) — scouting Adjusted Score
  rides that proxy; label it in the view too.

---

## Locked decisions (2026-07-19) + GLM-5.2 two-stage review

Slice-1 built, then reviewed by TWO independent blind GLM-5.2 passes (correctness +
math/design), triaged against source. Both confirmed on `zai/glm-5.2`.

**Correctness pass** — 3 major + 7 minor leads. Adopted: error surfaced + empty-state
branched on `.ok` (was misleading "score M1 first" on any MFL failure); NaN/Inf
rejected at the fetcher boundary (was poisoning the table sort); PA sorts ascending
(lower=better); out-of-order response guard (seq counter) + slider-snap suppression
while dragging; `Season` set on error returns; tie-averaged all-play % `(W+0.5T)/G`.
Verified-not-a-bug: FranchiseID format ("0001"–"0032") agrees between state + MFL.

**Math pass** — 68% as-shipped, ~85% with the normalizer fix. Two root leads (min-max
range≠variance so 60/40 label ≠ 60/40 influence; min-max fragile to dynasty
super-teams/tankers) → one remedy. **Christopher's calls:**
1. **Normalizer = z-score + display-normalize** (standardize each component, weight,
   then min-max the blend to [0,1]). Honest w (60/40 by amplitude), preserves magnitude.
   **Scouting uses a ROBUST center/scale — median + MAD·1.4826** (3rd GLM verify pass,
   2026-07-19): plain mean/std still let a lone super-team / empty roster shift the
   scale (50% breakdown for median+MAD vs 0% for min/max). All-play win% stays classic
   mean/std — it's bounded [0,1], can't throw an extreme tail. This is the change that
   took the panel from ~92% → ship-confident 95%+.
2. **Scouting aggregation = BOTH sum + top-N starters, user toggle** (matches original
   design intent; slice-1 first shipped sum-only). N = league `Starters.Count`;
   top-N degrades to sum if the count is unreadable (labeled by the AggMode echo).

**Deferred (minor, not math-soundness — Phase 2 candidates):** early-season all-play
low-confidence badge (Lead #5) — also covers the near-zero-variance all-play z spike
when only 1 team has played (verify-pass Q4b: a min-games threshold before including a
franchise in the perf-z population); dense-rank tie display "T7" (Lead #6); tier
buckets vs raw rank for false-precision (Lead #7); forward-vs-backward blend tooltip
(Lead #4); roster-size context next to top-N aggregates (verify-pass Q4g); server-side
standings cache so a weight/agg change re-blends without re-fetching MFL.

**Verify pass 3 (2026-07-19, blind glm-5.2 on the REVISED code):** Lead #1 fully
resolved; Lead #2 resolved by the median+MAD adoption; no math bugs; two safety edges
fixed (non-finite weight fallback; runtime-state freshness comment). Empty-roster → 0
scouting is left as-is (defensible: an empty roster IS the worst roster in dynasty).

## Implementation (slice-1, branch `session/m2-power-rankings`)
- `internal/ingestion/leaguestandings/fetcher.go` — Layer-1 leagueStandings fetch (+NaN/Inf reject).
- `internal/powerrankings/blend.go` — pure z-score blend + display min-max.
- `m2_app.go` — `GetPowerRankings(weight, aggMode)` IPC; sum/top-N aggregation; z-score row join.
- `frontend/src/components/PowerRankingsBoard.tsx` + store slice + tab — slider, agg toggle, sortable table.
