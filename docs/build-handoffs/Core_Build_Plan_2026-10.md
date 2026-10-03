# TheWarRoom — Core Build Plan (October 2026)
**Version:** 1.0 — 2026-10-03
**Status:** Approved direction (Christopher, 2026-10-03). Supersedes `Build_Tracker.md` for the core.
**Why this plan exists:** `Core_Build_Reasoning_2026-10.md`, in this folder.
**Evidence:** `docs/reviews/engineering-audit-2026-10/`.

## Scope

**In scope.** Cleanup that puts code, branches, data and docs on one timeline. Then the core:
a measurable on the real NFL player → power rankings (M2). Each stage is built to its intention
and feeds the next.

**Out of scope.** These are kept working but not extended:
- The Commissioner Suite.
- The Layer 2 fantasy-points engine. Points stay MFL's.
- Contract and transaction machinery beyond keeping it working.
- M3, M5, M6 and M7.
- The Vision_2026 horizons.

**Rulings that bind every stage (2026-10-03):**

| # | Ruling |
|---|---|
| R1 | The first user is Christopher as a GM. |
| R2 | **MFL wins.** A refresh overwrites the app's league and ledger state. Moves made in the app are what-if plans until made on MFL. |
| R3 | **Before the Week 9 trade deadline:** fresh MFL data, the clock started, and the current engine recomputed on today's data with its limits labelled. The full blend lands after the deadline. |
| R4 | Claude builds; Bee reviews finished stages when GPT budget allows. |
| R5 | The measurable is **two numbers per player, kept separate**: on-field-now and dynasty value. |
| R6 | Its shape is the **credibility blend**: `measurable = Z·production + (1−Z)·prior`, `Z = e/(e+k)`, on a within-position percentile scale. Parameters are stored, so a fitted model can replace them later without a rewrite. |
| R7 | Scouting caps and weights become **adjustable settings**: fitted now, editable in the Admin Console, learned later. |
| R8 | **Madden is out of the core prior.** |
| R9 | Approved sources added: nflverse, DynastyProcess crosswalk, CFBD and StatRankings routes (`docs/sources/Approved_Sources.md` v1.2). |

**Effect on CLAUDE.md's Hard Constraints.** Christopher reopened two locked rules on purpose.

| Rule | What changes |
|---|---|
| "Layer 4 structural mechanics never exposed in Admin UI" | **Superseded by R7.** |
| "Layer 2 / Layer 4 zero scoring leaks" | **Kept for the scouting prior and narrowed in scope.** The prior must not reference fantasy points, volume or scoring config. That rule is what keeps it an independent measure and stops it double-counting production. The production side of the blend uses usage and fantasy points by design, so the leak rule no longer covers it. |

CLAUDE.md is updated to say so in Stage 0.2.

**Standing bars.** Where a better architecture exists, explore it before moving on. Find and
remove AI slop on sight: volume from repetition, parallel units that differ only in constants.

## Stages

Every stage ends at a gate. A stage is done only when its gate passes on real output, not on an
exit code. Work happens on a session branch; never on main.

### Stage 0 — One timeline

1. **Branches.**
   - Merge `origin/main` into `session/m1b-bash`. That brings in three commits: the live-gate PASS
     record, the CLAUDE.md session close and the pnpm guardrail.
   - Fix the failing pre-push `ifaceguard` check, then push.
   - List the stale branches for Christopher to confirm before anything is deleted: `b7c-buyout`,
     the original `session-0/1/2` branches, `warden-pr2`, `migration/pnpm`, and about 20 remote heads.
2. **Docs to today.**
   - CLAUDE.md workflow: Claude builds and Bee reviews; Beelink paths. Remove OpenCode, GLM and
     DeepSeek. Bring the Hard Constraints in line with R6/R7 (see "Effect on CLAUDE.md's Hard
     Constraints" above).
   - SYSTEM_MAP: the 24 IPC methods; the built packages without "[planned]" tags.
   - Build_Tracker: point to this plan.
   - Copy the CT105-only Commissioner Suite plan into the repo, marked DEFERRED.
   - Add the smell "parallel near-identical units that differ only in constants" to
     `agent-codex.md` §4.
3. **Observability.** Every launch writes a non-empty app log. Today all 10 log files are empty.
4. **Dead code.**
   - Delete the `nflproduction` and `kicking` fetchers. Both point at URLs that now return 404.
   - Decide each item: `Ping`, `GetFranchiseState`, `GetFreeAgents` (no caller), and the `CORRECT`
     operation (no App mapping).
   - Make the DT cushion read its stored params; delete the literals 8.00 and 0.90.

**Gate:**
- One branch line, pushed.
- `go build`, `go vet`, `go test -race` and `golangci-lint` green, plus the frontend build.
- Docs match the code.
- A launch writes a log.

### Stage 1 — Start the clock (storage)

1. **Raw fetch archive.** Every network fetch stores: source, URL, fetched_at, sha256 and the
   compressed body.
   - It is append-only and content-addressed.
   - Nothing fetched is ever lost, and a parser fix can replay history.
2. **Typed observation tables** derived from the archive, per source family.
   - Covers production, snaps, injuries, draft, combine and college. Each row carries season,
     week, as_of and its source.
3. **Player directory.** MFL id ↔ GSIS/PFR/ESPN, plus position, birthdate and draft year, round
   and pick, kept with history.
4. **Scoring runs.**
   - A `scoring_runs` row holds run_id, as_of, a params snapshot and an inputs hash. Scores are
     keyed by run_id.
   - This keeps the append-only rule (AD-04) and makes recomputing normal.
   - Today's `season_scores` key is `(season, config, mfl_id)` and its triggers forbid change, so
     a board can never be recomputed.

**Gate:**
- A fetch writes an archive row plus typed rows, and running it twice is idempotent.
- A second scoring run with changed params produces a second board, and both remain readable.

### Stage 2 — League truth (MFL refresh)

1. **Refresh** rosters, contracts and ledger, players, standings and injuries from MFL, landing in
   the archive and state. MFL wins (R2).
   - Moves made in the app become what-if scenarios, stored apart from the mirrored state.
2. **One season source**, derived from MFL's league year and the phase. Delete the hard-coded
   `ingestion.SeasonYear`.
3. **A fresh live database** from a gated live pull on Christopher's machine.
   - Archive the two test-shaped databases, logged per the move convention. Neither is a faithful
     mirror.

**Gate:**
- Rosters and cap for all 32 teams match MFL. The app's comparison is spot-checked by Christopher
  on several franchises.
- Re-running the refresh changes nothing.

### Stage 3 — Crosswalk

1. Load DynastyProcess into the player directory.
2. Measure the match rate per position for rostered players and for the free-agent pool.
3. Surface the unmatched players in the admin console.

**Gate:**
- Match rate published per position.
- Every unmatched rostered player listed with a reason.

### Week-9 checkpoint (R3)

Stages 0–3, then today's engine recomputed through `scoring_runs` on fresh data. M1 and M2 show
their known limits: points-based, scouting capped, rookies at zero. **Deliver before the league's
Week 9 trade deadline.**

### Stage 4 — Signals into the store, one at a time

Each signal follows the same path: fetcher → archive → typed table → coverage report by position
→ freshness shown in the app.

1. **Production:**
   - nflverse `stats_player`, the replacement for `player_stats`.
   - Snap counts, including defense: wire the existing `touchshare`.
   - Injuries.
   - Backfill 2021–2025.
2. **Priors:**
   - nflverse draft picks.
   - Combine.
   - CFBD college stats (key held).
   - StatRankings routes last: name/team join, gap-filler only.

**Gate:** each signal has a coverage table and a freshness check. 2021–2025 history loaded.

### Stage 5 — Rubric as data (pure refactor first)

1. Replace the 10 parallel rubric structs (`internal/engine/l4/*`, 1,540 lines, 44% comments)
   with one routine and a per-position settings table stored as params (R7).
2. **Golden test:** with today's settings, the new code reproduces today's board exactly.
3. Only then remove Madden (R8), as a separate, reported change.
4. Collapse the twin scouting files in `internal/scouting/assembly`, measuring before cutting.

**Gate:**
- The golden board is identical.
- The Madden-removal diff is reported.
- Line counts before and after.

### Stage 6 — Fit the parameters from 2021–2025

A reproducible fitting tool in the repo. Its outputs are stored as params with provenance.
The full career-and-age design and its evidence are in the reasoning file, §4a.

1. **The prior mapping first:** draft capital, combine and college in **one joint regression**
   → expected within-position percentile (E9a–E9e).
   - Today's engine multiplies film × RAS × breakout as if they were independent. Joint fitting
     stops correlated signals counting twice (E9e).
   - Record the prior's R² per position.
2. **k against the prior, not the league mean.**
   - `k = σ²/τ²_resid`, where `τ²_resid = τ²·(1 − R²_prior)`. Here τ² is the true-talent spread
     around the prior, not around the league average (E0.3).
   - A better prior means a larger k, so the prior keeps its weight longer.
   - Fitting k against the league mean would under-weight scouting.
3. **Two k's, one per number** (E0.2):
   - **On-field-now** k comes from within-season split-half reliability: how sure we are about
     him *this season*.
   - **Dynasty** k comes from season-pair stability, which carries role, team and age change:
     how sure we are about him *next year and after*.
   - k is per production component (opportunity vs efficiency; E3, E5), not one per position.
4. **The shape of Z is tested, not assumed:** `e/(e+k)` against `1 − exp(−e/a)`, by holdout
   (E6).
5. **Recency weights** by temporal holdout, against the Marcel 1/0.8/0.6 baseline (E8).
   - Use the effective-count form when discounting (E7).
   - Fit **after** age-adjusting each past season (item 6), so recency measures loss of
     information, not age decline. Fitting it on raw seasons would count age twice.
6. **The talent arc per position**, rising and falling, not a flat 3% past a peak (E10b–E10e).
   - Fitted by the delta method with survivorship imputation (E10a, Schuckers/Lopez/Macdonald).
   - Experience (NFL season count) is fitted separately from age, for the year-1-to-year-2 jump
     (E8b).
   - Athleticism × age is tested as a fitted interaction, replacing the DT-only literal cushion
     (E9b, E10a).
7. **The survival arc per position:** the probability a player is still on an NFL field
   1–5 years out, by age, draft capital and injury history.
   - Fitted from nflverse roster and snap history.
   - Dynasty value needs it: release, retirement and injury are part of asset value (E10a).
   - Never condition on future career length (E10f).

**Gate:** a fit report with holdout scores for every parameter. Parameters loaded and shown in
the Admin Console.

### Stage 7 — The two measurables

1. **On-field-now:** a blend of usage and opportunity rates (research E2–E5) with the prior.
   - Evidence is snaps, exposure-weighted; injured weeks add no trials.
   - Fantasy points are one production input, not the base.
   - IDP uses snaps plus box-score rates per snap. No free source has pressures, routes or
     coverage snaps.
2. **Dynasty:**
   ```
   Σ over years t = 1..H of: discount_t × P(on field at t) × talent(age + t, experience + t)
   ```
   - The talent starting point is the dynasty blend, which uses the dynasty k, so the prior
     weighs more.
   - Talent is carried along the position's talent arc. Survival is the survival arc.
   - Cap and contract are applied after and stay separable.
3. **Age enters once.** Today age is counted twice: the L3 decay and an age-trajectory
   sub-signal inside 9 of the 10 L4 rubrics. In the new design age appears only in the arcs.
   The prior's inputs are pre-NFL facts.

**Gate:**
- Rookies are non-zero.
- Injured veterans keep discounted evidence.
- **Case set:**
  - A rookie by draft round.
  - A year-2 jump.
  - A peak veteran.
  - An age-31 RB and an age-31 QB, so the arcs visibly differ.
  - An injured starter.
  - A high-athleticism late-career lineman.
  - A backup with a small, efficient sample, who must be shrunk.
- A holdout check: 2025 predicted from data through 2024 beats today's board.
- Spearman correlation vs today's board reported.

### Stage 8 — M2 on the new numbers

1. Two views: **this season** (on-field-now) and **the franchise** (dynasty).
2. Recompute via `scoring_runs`.

**Gate:**
- A param edit → a new run → a changed board, with the old board still readable.
- Both views shown.

## Open items that need Christopher

- Confirm the stale-branch deletion list (Stage 0.1).
- The fresh live pull is gated (`TWR_LIVE_*`) and runs on his machine (Stage 2.3).
- Spot-check the MFL comparison (Stage 2 gate).
