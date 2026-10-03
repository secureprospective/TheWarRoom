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
| R10 | **History is a measure dictionary.** Every number is saved as `player · season · week · measure`, under a measure named for what it means, never for the source it came from. The blend reads measures, never sources. |
| R11 | **A lost source never stops the board.** It keeps running on the measures still flowing, labelled as running on a reduced set. A rebalance is prepared from history, shown next to the current board, and applied only when Christopher approves it in the Admin Console. |

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

5. **Standards drift and bloat.** Measured 2026-10-03 on non-test Go: 25,247 lines, of which
   7,787 (30%) are comment lines, against 15,403 lines of code.

   **Fix the rules that cause the bloat**, in `docs/agent-codex.md`, and the same change
   upstream in `christopher-coding-standards`:

   | Rule today | Bloat it causes | New rule |
   |---|---|---|
   | Hard 400-line file cap (`filelen` gate; target 250), and "one exported job per file" | 19 files cite the cap or an "own file" split. 34 files have under 40 lines of code. Cohesive code is scattered into sibling files. | **Split by responsibility, never by size.** Keep the function limits (`funlen`, `gocyclo`), because they measure complexity. The file cap becomes a report: over 600 lines, say why in the PR. |
   | Comments "explain why… the spec citation" | Label soup and history in code: 47 files carry review or agent provenance ("GLM review m3", "DeepSeek"). L4 rubrics are 47% comments and carry 124 decision labels in one package. | **Comments say what the code can't, in the fewest lines.** One spec link per unit, at most. No review history, decision IDs or session notes in code: those belong in commits and docs. |
   | First-Instance Template Review (copy a reviewed pattern) | The pattern was copied as code, not as data: 10 rubrics, and about 15 near-identical ingestion packages. | **A second near-copy is the signal to turn it into data** (M12). Add to the §4 slop catalogue: "parallel units that differ only in constants" and "comment walls restating spec docs". |

   **Clean the code to the new rules:**
   - Re-merge files that were split only for size.
   - Fold sub-40-line files into their natural home unless they mark a real boundary.
   - Strip provenance and label soup from comments.
   - Convert parallel test suites to table-driven tests (21,411 test lines today).

   **A ratchet so the bloat can't return.** `make bloat` reports four numbers, and the pre-push
   check fails if any rises:
   - comment share
   - provenance hits
   - files under 40 lines of code
   - near-duplicate files

   **Targets:**
   - Comment share 30% → 15% or less, with no package over 20%.
   - Provenance in code comments → 0.

**Gate:**
- One branch line, pushed.
- `make bloat` is at or under its targets and wired into the pre-push check.
- `go build`, `go vet`, `go test -race` and `golangci-lint` green, plus the frontend build.
- Docs match the code.
- A launch writes a log.

### Stage 1 — Start the clock (storage), as a measure dictionary (R10, R11)

Design and reasoning: `Core_Build_Reasoning_2026-10.md` §5a.

**Three layers.** Each has one job. The engine reads only the top one.

| Layer | Table(s) | Job |
|---|---|---|
| Raw | `raw_archive` (sha256, source, url, fetched_at, compressed body), in a separate `history.db` | Every fetch, exactly as received. Append-only, content-addressed. A parser fix replays it. |
| Observations | `observations` (player_id, season, week, measure, value, source, as_of) | Every fact, under its measure name. Append-only. |
| Features | `season_features`, `week_features` (views) | One clean row per player · period · measure, chosen by source priority. **The only thing the engine and UI read.** |

**Registries.** Each is a checked-in file loaded into a table, so changes are reviewed in git.
- **`measures`:** measure, family, grain, unit, positions and a one-line meaning. The families
  are exposure, opportunity, outcome, prior, availability and context.
  - Examples: `exposure.snaps_def`, `opportunity.targets`, `outcome.tackles_solo`,
    `prior.draft_pick`, `availability.game_status`.
  - `docs/data-layer/Measure_Dictionary.md` is **generated** from this registry, so the doc can't
    drift from the data.
- **`source_fields`:** source + field → measure, with transform and priority. This is the only
  place a source's vocabulary appears.
- **`sources`:** status (active / lost / retired), last good fetch, and the measures it feeds.
- **`player_ids`:** id_type + id_value → player_id. The player_id is the MFL id: a string with
  leading zeros. The directory adds position, birthdate and draft year, round and pick.

**Rules.** These keep it low-maintenance and its output clean.
1. **The key is always `player · season · week · measure`.** Week 0 means season-level. Pre-NFL
   facts (draft, combine, college) sit at the draft season, week 0.
2. **Store facts and counts, never derived values.** Rates, shares, percentiles and blends are
   computed in the features layer or the engine. A formula change never rewrites history.
3. **One measure, one meaning.** A source that defines a stat differently gets its own measure:
   solo tackles are not combined tackles. Measures never silently mix.
4. **Corrections append; nothing is updated.**
   - A corrected value is a new row with a later `as_of`.
   - Reads take the latest `as_of` on or before the date asked.
   - So "what did we know on 2026-10-03" can always be rebuilt. The calibration needs this,
     and it is what stops future data leaking into a fit.
5. **No row is dropped for a missing match.** Observations without a matched player wait in
   `unresolved_observations` with their source ID. They resolve when the crosswalk matches.
6. **Several sources for one measure:**
   - All rows are kept with their source.
   - The features layer picks one by priority.
   - A daily agreement report flags sources that disagree.

**Lifecycle (R11).**
- **Gaining a source** means a fetcher plus `source_fields` rows.
  - A source that supplies an existing measure needs no other change: better coverage for free.
  - A new meaning adds one `measures` row plus a refit.
  - Neither needs a schema change.
- **Losing a source:** its status becomes `lost`, and its measures stop updating. History stays
  intact. Then, in order:
  1. Scoring runs carry on with the measures still flowing.
  2. The board is labelled "running on a reduced set".
  3. A rebalance (a refit on history restricted to the measures that remain) is prepared and
     shown next to the current board.
  4. It is applied only on approval.
- **Param sets record the measure set** they were fitted on (a hash). Scoring runs record their
  param set. A mismatch between the param set and the measures now flowing is what raises the
  flag.

**Scoring runs.**
- A `scoring_runs` row holds run_id, as_of, param set and an inputs hash. Scores are keyed by
  run_id.
- This keeps the append-only rule (AD-04) and makes recomputing normal.
- Today's `season_scores` key is `(season, config, mfl_id)` and its triggers forbid change, so a
  board can never be recomputed.

**Gate:**
- A fetch writes an archive row and observation rows, and running it twice is idempotent.
- A correction appends, and a read "as of yesterday" still returns the old value.
- **Source-loss drill on a fixture:**
  - Mark a source lost.
  - The board still runs and shows the reduced-set label.
  - A rebalance is prepared, and nothing changes until it is approved.
- **Source-gain drill:** add a second source for an existing measure with mapping rows only, and
  no code change.
- The generated Measure Dictionary matches the registry.
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

Each signal follows the same path: fetcher → `raw_archive` → `source_fields` mapping →
`observations` under its measures → coverage report by position → source health and freshness
shown in the app. Adding a signal adds registry rows; it never adds a table.

**Ingestion collapses into one loader.** Today about 15 packages under `internal/ingestion/`
(touchshare, kicking, nflproduction, ras, madden, pfrcoverage, pfrpassrush and others) each
hand-parse one CSV into their own struct.
- One table-driven loader reads `source_fields`: which column, which ID type, which measure.
- Per-season sums and shares move to the features layer.
- Each package is deleted as its source moves over.
- What stays bespoke is only real protocol work: MFL's API, CFBD's key, and HTML for
  StatRankings.
- This also fixes a silent gap: `touchshare` reads only offense snap columns, while nflverse
  carries `defense_snaps` for IDP.

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
5. The 13 validation cases in `internal/harness` (3A–3M, run with every real rubric
   registered) are the regression net for step 1 alongside the golden board. They stay until
   Stage 7.

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
3. **Missing measures are normal, not errors.**
   - The production composite re-weights over the components present for that player and
     period.
   - The prior is fitted with missing-value indicators, so a player without combine data is
     scored on what exists, not zeroed.
   - Each output records which measures fed it.
4. **Age enters once.** Today age is counted twice: the L3 decay and an age-trajectory
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
- **Retire the harness** once the case set replaces it: delete `internal/harness`, its two
  dev tabs (Rookie Sandbox, Architectural Tests) and their bindings. The sandbox scores
  synthetic "QB Alpha" fixtures, which has no place in the GM's app.
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
