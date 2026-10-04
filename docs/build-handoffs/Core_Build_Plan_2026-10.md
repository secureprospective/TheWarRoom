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
| R3 | ~~Before the Week 9 trade deadline: fresh MFL data, the clock started, and the current engine recomputed on today's data with its limits labelled.~~ **Dropped (Christopher, 2026-10-03):** the app is not used for in-season work until 2027-01-01, so no deadline build is needed. |
| R4 | Claude builds; Bee reviews finished stages when GPT budget allows. |
| R5 | The measurable is **two numbers per player, kept separate**: on-field-now and dynasty value. |
| R6 | Its shape is the **credibility blend**: `measurable = Z·production + (1−Z)·prior`, `Z = e/(e+k)`, on a within-position percentile scale. Parameters are stored, so a fitted model can replace them later without a rewrite. |
| R7 | Scouting caps and weights become **adjustable settings**: fitted now, editable in the Admin Console, learned later. |
| R8 | **Madden is out of the core prior.** |
| R9 | Approved sources added: nflverse, DynastyProcess crosswalk, CFBD and StatRankings routes (`docs/sources/Approved_Sources.md` v1.2). |
| R10 | **History is a measure dictionary.** Every number is saved as `player · season · week · measure`, under a measure named for what it means, never for the source it came from. The blend reads measures, never sources. |
| R11 | **A lost source never stops the board.** It keeps running on the measures still flowing, labelled as running on a reduced set. A rebalance is prepared from history, shown next to the current board, and applied only when Christopher approves it in the Admin Console. |
| R12 | **Claude runs the live gates** on Claude-OS (ruled 2026-10-03): the production build against a snapshot of the live database, with screenshots and logs as evidence. |

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

   **Targets**, over the code that survives Stage 0:
   - Comment share 30% → 15% or less, with no package over 20%. Exception, by name: `scouting`,
     a types-only package whose comments are its data's documentation.
   - Provenance in code comments → 0.

   **Measured at the end of Stage 0:** surviving code 14.3%, no package over 20% apart from
   `scouting` (29.5%), provenance 0. The packages Stages 4, 5 and 7 rewrite or delete
   (`engine/l4`, `scouting/assembly`, `harness`, and the nflverse/CFBD fetchers) sit at 33.6% and
   hold all 7 remaining provenance lines. They were not polished, because the code is going away.
   The ratchet holds them flat, and the gates of Stages 4, 5 and 7 inherit the same targets for
   whatever replaces them.

**Gate:**
- One branch line, pushed.
- `make bloat` is at or under its targets and wired into the pre-push check.
- `go build`, `go vet`, `go test -race` and `golangci-lint` green, plus the frontend build.
- Docs match the code.
- A launch writes a log.

**Gate check, 2026-10-03** (branch `session/m1b-bash`):

| Gate | Result |
|---|---|
| One branch line, pushed | PASS. `origin/main` merged in, pre-push fixed, branch pushed. The stale branches are listed and wait on Christopher's go-ahead to delete (each tagged `archive/<name>` first). |
| Bloat at or under target, in pre-push | PASS. Numbers above; `make bloat` is part of `make lint`, which `make verify` and the pre-push hook run. Ratchet baseline: comment 20, provenance 7, tiny files 28, dupl 38. |
| build, vet, race tests, lint, frontend build | PASS (`make verify`, plus `go build ./...` and `go vet ./...`). |
| Docs match the code | PASS. CLAUDE.md, SYSTEM_MAP (21 IPC methods, every package), AGENTS.md, Build_Tracker and North_Star pointers; Fable's June planning docs moved to `archive/2026-06-pre-build/`. |
| A launch writes a log | PASS: a dev build (`-probe`) and the production build in the live gate each write a timed line per startup step. |

Found and fixed during the check: the DT cushion's Layer 4 half still used the literals 8.00
and 0.90 (item 4). Both halves now read one `engine.CushionGuard` from the params, with outputs
bit-identical at the defaults.

**Stage 0 closed, 2026-10-03:**
1. Stale branches deleted, 19 of them, each kept as an `archive/<name>` tag.
2. christopher-coding-standards PR #33 merged (`93ee82a`), after PR #34 cleared the CVE that
   had failed its dependency scan since September.
3. Live gate PASS on Claude-OS (R12), with the production build `v0.5.0-123-g14fb858` run
   against a snapshot of the live database:
   - migration v3 ran with its `premigration` backup, and a relaunch did not migrate again;
   - the board loads 827 scored players;
   - the log has a timed line per startup step;
   - a second instance shows the startup-failure banner.

   Evidence: `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-2026-10-03/`.

The gate also caught four defects, all fixed before close:
- The version stamp picked up the new archive tags.
- A Wails cache file marked every build dirty.
- The rankings board showed bare franchise ids.
- The filter selects were unreadable on a light GTK theme.

Christopher's real database migrates, with its backup, the first time he runs a production
build.

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

**Design (Claude, R4, 2026-10-03).** How the storage is shaped, and what was rejected.

| # | Decision | Rejected, and why |
|---|---|---|
| S1 | **Two database files, split by lifecycle.** `thewarroom.db` mirrors MFL plus app settings: it can be rebuilt, and Stage 2 replaces it. `history.db` holds everything that can't be rebuilt: the raw archive, the fetch log, observations, the registries and scoring runs. Dev builds use `history-dev.db`. | Raw bodies alone in `history.db`: observations and runs are the clock too, and replacing the league database in Stage 2 would reset them. |
| S2 | **Three packages.** `internal/measures` is a leaf holding the registries (embedded CSV), their validation, the `Fact` shape that fetchers emit, and the dictionary renderer. `internal/store/history` owns every table in `history.db`. `internal/archive` is an HTTP transport that tees each response into the archive. `internal/output` is deleted: scores are history, keyed by run. | A separate runs store: the reduced-set check reads the param set and the measures flowing, so splitting would force a cross-store join for one query. |
| S3 | **Archive at the transport.** One `http.RoundTripper` records every fetch the app makes: MFL, nflverse, DynastyProcess, CFBD. Bodies are content-addressed (`raw_archive`, sha256, gzip). Every attempt is a `fetch_log` row, so the same body fetched twice is stored once but logged twice. `mfl.Client` takes the transport, which also gives it the test seam it lacked. | Archiving per fetcher means editing about 15 packages that Stage 4 deletes anyway. A single `raw_archive` table keyed by sha256 would lose the second fetch, so a source would look stale while it was working. |
| S4 | **Observations append only on change.** `as_of` is the load time. A reload with the same value writes nothing; a different value appends a row. A read "as of T" takes the latest row on or before T. Each load attempt, success or failure, is a `loads` row. | Appending every row on every load: the data grows daily with nothing new. |
| S5 | **"Lost" is observed, not stored.** `sources.csv` says `active` or `retired`, and gives a `max_age_days`. A source is lost when its latest load failed and it has no success within `max_age_days`, or no success at all. | A stored status flag goes stale and needs someone to flip it back. |
| S6 | **Registries are CSV files** in `internal/measures`, embedded in the binary and reloaded into tables at startup. `history.db` is then self-describing, so a fitting script can read features from it alone. `docs/data-layer/Measure_Dictionary.md` is generated (`make measure-dictionary`); a test fails if it is stale. Registry tables mirror the files, so adding a column to one is not a migration. | YAML needs a dependency, and CSV diffs as a table. Keeping registries in Go code would hide them from SQL. |
| S7 | **`player_ids` is data, not a checked-in registry.** Stage 3's crosswalk fills it; MFL ids resolve as themselves. Position and birthdate arrive with the crosswalk. Draft facts are `prior.*` measures, not directory columns. Rows with no matched player wait in `unresolved_observations`, and `LinkPlayerIDs` promotes them, keeping their `as_of`. | |
| S8 | **Param sets are snapshots.** The params store stays the one place params are edited. Each run snapshots it (`params.Set`), the engine reads that snapshot, and the snapshot is stored content-addressed. The engine therefore always uses exactly what the run records, and a param edit always produces a new run. A param set also lists the measures its model reads; today that is one, `outcome.fantasy_points`, the MFL base points. | A second "active param set" pointer in `history.db` would make two places that decide the live params. |
| S9 | **Board and rebalance runs.** `scoring_runs.kind` is `board` or `rebalance`. The board is the latest `board` run. A run records the measures its param set needs that were not flowing, which drives the reduced-set label. A new run is written only when the param set, engine build, inputs, scores or missing measures differ from the latest run of its kind; otherwise ScoreLeague reports "unchanged". The inputs hash covers every player's engine input and every exclusion. | |
| S10 | **Approving and applying a rebalance waits for Stage 6.** It needs the fitting tool and params keyed by measure, so that a refit can drop a lost measure as data. Stage 1 builds everything up to approval: detection, the label, a proposal run readable beside the board, and a board that ignores it. | Building approval now would mean code that has nothing to approve until Stage 6. |
| S11 | The old `season_scores` table stays in `thewarroom.db`, unread. Stage 2's fresh database won't have it. | |

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

**Stage 1 gate, 2026-10-03.** Built on `session/stage1-measure-store`.

| Gate item | Result |
|---|---|
| Fetch → archive and observations; twice is idempotent | PASS. Tests: `TestFetchArchivesOnceAndLoadIsIdempotent`. Live: three MFL loads of 1,650 facts; the second and third added 0. |
| Correction appends; "as of yesterday" returns the old value | PASS. Test: `TestCorrectionAppendsAndAsOfReadsTheOldValue`. |
| Source-loss drill | PASS on a fixture: `TestSourceLossDrill`, `TestSourceIsLostOnlyAfterFailingPastItsMaxAge`. Live with MFL blocked: the board still ran, warned that the points were not refreshed, and scored from the points held. Approval waits for Stage 6 (S10). |
| Source-gain drill | PASS: `TestSourceGainNeedsOnlyMappingRows`. |
| Measure Dictionary matches the registry | PASS: `TestDictionaryIsCurrent`. |
| Changed params → a second board; both readable | PASS. Test: `TestChangedParamsMakeASecondReadableBoard`. Live: `layer3.decay_rate` 0.03 → 0.04 wrote board #2 with older players moved, and board #1 still reads back. |

Live gate on Claude-OS (R12), production build `v0.5.0-126-g7f46782` with a fresh snapshot of the
live database. A first ScoreLeague wrote board #1 (828 players, 3 excluded with reasons). A second
reported "No change" and wrote nothing. Evidence:
`~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage1-2026-10-03/`.

The gate caught two defects, both fixed:
- **RAS drifted between runs** (`7f46782`). The RAS cohort was summed in map order, so RAS, the
  breakout it modulates and the adjusted score differed in their last bits on every pass. The
  first gate attempt's second ScoreLeague therefore wrote a new board. The cohort is now summed
  in id order, and `TestScoreRASIsDeterministic` requires bit-identical results.
- **An unknown argument opened the window** (`a01498e`). `thewarroom -version` opened the
  production app on the Beelink against the real database. Unknown arguments now exit 2, and
  `-version` exists.

### Stage 2 — League truth (MFL refresh)

1. **Refresh** rosters, contracts and ledger, players, standings and injuries from MFL, landing in
   the archive and state. MFL wins (R2).
   - Moves made in the app become what-if scenarios, stored apart from the mirrored state.
   - Refresh the rulebook too. The live database's only version (2026-07-05) predates the
     franchise list, so every screen labels teams "Franchise 0014" (found in the Stage 0 live
     gate).
2. **One season source**, derived from MFL's league year and the phase. Delete the hard-coded
   `ingestion.SeasonYear`.
3. **A fresh live database** from a gated live pull on Christopher's machine.
   - Archive the two test-shaped databases, logged per the move convention. Neither is a faithful
     mirror.

**Scope (Christopher, 2026-10-03).** Transactions, what-if plans and the refresh plumbing are
placeholders until the core (Stages 3–8) makes them relevant. Stage 2 cleans up their
architecture so that MFL truth has one home and one read path. It builds no new features.

**Design (Claude, R4, 2026-10-03).**

| # | Decision | Rejected, and why |
|---|---|---|
| L1 | **One league mirror.** `state.Mirror`, one row in `thewarroom.db`, holds what MFL says: the season, rosters with contracts as MFL states them, and salary adjustments (franchise names live in the rulebook). It implements `state.Reader`, so the board, Power, the inspector and scouting read MFL truth through the interface they already use. Cap follows MFL: salaries (taxi and IR at the league's percentages) plus salary adjustments. | Rewriting the state store's tables on refresh: their append-only ledger triggers exist for app-made moves, and MFL wins makes them wrong for truth. |
| L2 | **Refresh replaces the mirror whole,** in one write, from fetches that all go through the archive. If the content matches what is held, nothing is written and the refresh reports "up to date". `rulebook.Sync` stores and promotes MFL's config only when it differs (MFL wins), which brings in the franchise names. Refresh runs after the window opens (an empty mirror is filled during startup instead) and from a "Refresh from MFL" button. It needs no players list, so MFL's once-a-day limit never blocks it. Host discovery is cached for 15 minutes, so a refresh's fetches share one. | Field-by-field merging: there is nothing local to preserve, because truth is MFL's. |
| L3 | **The season comes from MFL.** It is the newest year in the league export's `history` under this league's id (`league.Discover`). `ingestion.SeasonYear` is deleted. The season is read from the mirror at startup, so a rollover takes effect at the next launch. Deriving the phase from MFL's weeks waits for the core to need it; the what-if league keeps its own phase log. | A configured year: it went stale, and the app ended up with two sources for the season. |
| L4 | **What-if keeps the existing machinery, isolated, at placeholder depth.** The state store and coordinator move to their own file, `whatif.db`, seeded from the mirror by the seed path they already have. Transact and Trade read it, and a refresh leaves it alone. The Reset button and the "built on MFL as of" label wait until what-if becomes relevant; until then, a fresh `whatif.db` reseeds from the mirror. | Named plans (deferred). A scenario column across the coordinator's tables: invasive, for a placeholder. |
| L5 | **A fresh live database.** After the Claude-OS gate, the two test-shaped databases are archived (logged per the move convention) and Christopher's first launch builds the league from MFL. `history.db` is kept. | |

Injuries move to Stage 4 as a measure (R10) rather than into the mirror: they are facts about a
player and a week, which is what history holds.

**Gate:**
- Rosters and cap for all 32 teams match MFL. The app's comparison is spot-checked by Christopher
  on several franchises.
- Every screen shows the 32 franchise names from MFL.
- Re-running the refresh changes nothing.

**Stage 2 gate, 2026-10-03.** Built on `session/stage2-league-truth`.

| Gate item | Result |
|---|---|
| Rosters and cap for all 32 teams match MFL | PASS for rosters: the mirror holds MFL's 1,450 rostered players, and its cap equals salaries plus salary adjustments computed independently from MFL's raw responses for all 32 teams (`cap-check.txt`). Transact shows the same cap (Denver $123.9M). Christopher's spot-check: MFL's cap screen shows Denver at $123.91, the app's figure. MFL counts every adjustment its export lists, including the 148 stamped 2023–2025. |
| Every screen shows the 32 MFL franchise names | PASS: the board, Power, Transact and Trade. |
| Re-running the refresh changes nothing | PASS: two button presses after the launch refresh both read "up to date · 2026". `league_mirror` holds one row, written once, and `rulebook_versions` holds one version. |

Live gate on Claude-OS (R12), production build `v0.5.0-134-g6fe30b9` from an empty config folder,
as Christopher's fresh launch will run. Startup with the first MFL refresh took 4.0 s. Evidence:
`~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage2-2026-10-03/` (earlier attempts in
`attempt1`–`attempt3`).

The gate caught three defects, all fixed:
- **The first screen asked for data before startup finished** (`9fb127e`). On Linux, Wails runs
  startup alongside the page load, so the board opened on "stores not initialized" and season 0.
  Every IPC method now waits on one `ready` gate, which replaces a dozen copied nil checks.
- **Every refresh wrote a new rulebook version** (`9218921`). MFL returns the scoring-rule blocks
  in a different order on each request, so the comparison never matched and the rail said
  "updated". The blocks are now sorted where they are read.
- **The same team showed two caps** (`6fe30b9`). The what-if league was seeded from rosters
  only, so Transact left out MFL's salary adjustments. It now seeds them too, and
  `TestWhatIfSeededFromMirrorHasTheMirrorsCap` requires the two caps to match.

Seen and left for later (outside Stage 2):
- Power shows "FINAL, season complete" in October. It reads the what-if league's own phase log;
  deriving the phase from MFL is deferred under L3.
- Power's all-play columns read 0-0, and Transact's roster rows show no player names.

### Stage 3 — Crosswalk

1. Load DynastyProcess into the player directory.
2. Measure the match rate per position for rostered players and for the free-agent pool.
3. Surface the unmatched players in the admin console.

**Design (Claude, R4, 2026-10-03).** Measured first against the live league: DynastyProcess
(12,518 rows) and MFL's player list (2,715) for the 1,450 rostered players.

| # | Decision | Rejected, and why |
|---|---|---|
| C1 | **Every id DynastyProcess carries is linked.** Each `<x>_id` column (gsis, espn, pfr, sleeper, pff and the rest) becomes id type `<x>` in `player_ids`, pointing at the MFL id. A future source keyed on any of them resolves with no code change, as Stage 1's lifecycle promises. | Linking only gsis, espn and pfr: each new source would then need a code change. |
| C2 | **A clean miss beats a mis-attributed player.** Two guards. (a) An id DynastyProcess ties to two MFL ids links to neither: 10 gsis ids today, all retired players with the same name merged. (b) No link when MFL lists the id under a different name. MFL reuses low ids for team units and commissioner-created players: `0360` is the Steelers unit on MFL and Ronde Barber in DynastyProcess, and `0816` is Stephen Gosnell on MFL and Dre' Bly there. There are 69 such ids, and without the guard their espn and pfr facts would land on the wrong player. Names compare lowercased, letters only, suffixes dropped, MFL's "Last, First" reordered; no real player disagrees. An id MFL doesn't list (a retired player) links on DynastyProcess's word, which keeps history for the Stage 6 fit. | Picking one of two duplicates by heuristic. Trusting DynastyProcess's MFL ids without a check. |
| C3 | **Loads run with ScoreLeague and from a button.** ScoreLeague already fetches DynastyProcess and MFL's player list, so it links at no extra cost. Control → Crosswalk has a "Load from DynastyProcess" button. Each load is a `loads` row for the `dynastyprocess` source, so source health covers it. Links upsert; a link DynastyProcess later drops stays until a source column is needed. | Loading at launch: it would spend MFL's once-a-day player-list call before anyone asks. |
| C4 | **Matched means linked to a gsis id,** the key nflverse (the production source) uses. Rates are per position for the league's ten (QB, RB, WR, TE, PK, DT, DE, LB, CB, S), rostered and free agents (MFL's list minus the rosters). Team units, coaches and punters are left out. Every unmatched rostered player is listed with one reason: not in DynastyProcess, no NFL id yet, DynastyProcess has this MFL id as another player, or the NFL id is shared with another MFL id. The report is held in memory and shown in Control; a summary line goes to the log. | Storing the report: it is recomputed from data that is stored. |
| C5 | **Position and birthdate stay MFL's.** The report uses MFL's position, the league's own. Directory columns from DynastyProcess wait for the stage that reads them. This amends S7. | Copying DynastyProcess's position and birthdate now: nothing reads them yet. |

**Gate:**
- Match rate published per position.
- Every unmatched rostered player listed with a reason.

**Stage 3 gate, 2026-10-03.** Built on `session/stage3-crosswalk`.

| Gate item | Result |
|---|---|
| Match rate published per position | PASS. Control → Crosswalk shows the ten positions, rostered and free agents. Rostered 1,444 of 1,450 (99.6%): QB 90/91, RB 149/149, WR 228/229, TE 105/105, K 45/45, DT 143/144, DE 165/165, LB 192/192, CB 174/174, S 153/156. Free agents 744 of 796 (93.5%). |
| Every unmatched rostered player listed with a reason | PASS. All six: `0816` Gosnell (DynastyProcess has this MFL id as another player, Dre' Bly); `0820` Roberts, `0835` Childress, `0843` Thompson, `0844` Wood (not in DynastyProcess: commissioner-created); `17471` Pavia (no NFL id yet). |

Live gate on Claude-OS (R12), production build `v0.5.0-137-ga091c60`, on the Stage 2 gate's
databases (the state Christopher's machine is in).
- The load wrote 124,906 links across 19 id types for 11,475 players, in about 4 s. The guards
  held: nothing links to `0816` or to the Steelers unit `0360`, and the shared gsis `00-0031636`
  links to no one. A second load (from ScoreLeague) gave the same counts.
- Both loads are `loads` rows for `dynastyprocess`. ScoreLeague still scores (board #2, identical
  values; the new engine build is why a board was written).
- The numbers match a dry run of the same code on the live data, and the Python analysis that
  shaped C1–C4.

Evidence: `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage3-2026-10-03/`.

### Week-9 checkpoint (R3): dropped

Christopher, 2026-10-03: "the app wont be used for inseason work till Jan 1st 2027, if things are
connected right. skip it." Stages 4–8 follow Stage 3 directly.

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
- The MFL standings and schedule caches (`standings_cache`, `league_schedule_cache`) become
  "latest good `raw_archive` row for this feed"; drop both tables. The app already reaches
  them through one generic `liveOrCache`, so only its put/read pair changes.

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

**Gate:** each signal has a coverage table and a freshness check. 2021–2025 history loaded. The
loader meets the Stage 0 comment and provenance targets.

**Stage 4 design (Claude, 2026-10-04):**

| # | Decision | Why |
|---|---|---|
| S1 | `feeds.csv` joins the registry: one row per file (URL with `{season}`, id column and type, period columns, row filter). `internal/ingestion/feeds` reads any of them; a field named `<feed>.<column>` reads that column. | A new signal from a file is CSV rows, never a package. |
| S2 | A `player` grain (season 0, week 0) for facts that belong to no period: birth date, draft slot, combine. | The prior's inputs are one fact per player; forcing them into a season would make every read guess which season. |
| S3 | Week-level zeros are not written. A whole-file load carries a scope, so a week value the file no longer reports is corrected to zero. | Most box-score cells are zero; storing them would multiply the table about fivefold. The scope keeps a correction to zero honest. |
| S4 | `Ingest` reads the directory and the latest held values once per load, not once per fact. | 1.03 million facts from 2021–2026 load in about four minutes, at 210 MB peak memory. |
| S5 | CFBD stays bespoke (bearer key, one long-format call per season), mapped through `source_fields` like any feed. Team totals are summed from the same response for the share denominators. Only players the directory knows are kept. | Most FBS players never reach the NFL; keeping them would leave millions of rows waiting forever. A player who joins the directory later is picked up at the season's next monthly load. |
| S6 | A file is due when it has never loaded, when it is the current season or a single file and is over 20 hours old, or when it is a closed season over 30 days old. A reload of unchanged data writes nothing. | Closed seasons change only by correction. |
| S7 | `standings_cache` and `league_schedule_cache` are dropped (state migration v4). A failed MFL fetch falls back to the newest archived body of the same export, season and league that still parses. | The archive already holds every body; a body that proved bad is passed over. |
| S8 | **The old fetchers stay until their consumer goes.** `agetrajectory`, `collegeshare`, `collegedefense`, `schooltier`, `ras`, `pfrcoverage`, `veteranfilm` and `madden` still feed today's board. | Stage 5's golden test must reproduce today's board exactly, which needs today's inputs. Rewiring an engine that Stage 7 retires would be wasted work. They are deleted with today's engine. |
| S9 | **StatRankings routes are deferred.** The routes page serves 5 rows in its HTML and renders the rest in the browser, so a plain fetch cannot read it. It is current-season only and joins on name. | The plan ranks it lowest, as a gap-filler. Reading it would need a headless browser. |

**Gate check, 2026-10-04** (branch `session/core-stages-4-8`, build `v0.5.0-140-ga365ba3`, live on
Claude-OS against a backup-API snapshot of Christopher's databases; evidence in
`~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage4-2026-10-04/`):

| Gate item | Result |
|---|---|
| Each signal has a coverage table | PASS. Control → Signals: players with data by season (with ids waiting for a match), and rostered coverage by position. 2026 so far: weekly stats 54–83%, snaps 57–88% (the rest have not played), player facts 98–100%, draft 40–92% (undrafted kickers pull K down), combine 53–90%, college 67–93%. |
| Each signal has a freshness check | PASS. Each signal's current file shows fresh or stale against its source's window, with the load time. Source health lists every source. |
| 2021–2025 history loaded | PASS. One launch loaded 35 files in 3 min 42 s: 1,029,669 values. Weekly stats reach 1,613–1,755 players a season, snaps 1,695–1,833. About 1% of weekly ids wait for a directory match. |
| Values right | PASS. Spot-checked against known 2024 leaders: Burrow 4,918 passing yards, Hendrickson 17.5 sacks. |
| A reload writes nothing | PASS. A second Load signals: 35 files, 0 new values. |
| Archive fallback replaces the caches | PASS. Migration v4 ran with its backup. With MFL blocked in the VM, Pulse showed "CACHED · live fetch failed" from the archived body. |
| Loader meets the Stage 0 comment and provenance targets | PASS. `feeds` 8%, `college` 13%, registry `feeds.go` 8%, `signals_app.go` 8%; provenance 0. Ratchet lowered: comment 19 → 18, dupl 38 → 34. |
| Defect found | The Sources table showed nflverse's error from the first launch after later loads succeeded. Fixed: an error shows only while it is newer than the last good load (`sourceView`, tested). |

Incident: on the first launch, Claude-OS's network process (`passt`) segfaulted twice within
three seconds as the downloads began. The kernel log has the same crash address both times, so
it is a bug in passt. A single 8.5 MB download and the full relaunch ran clean afterwards. The
network card was hot-plugged back (live only) without a reboot, so the desktop session survived.

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
- The new rubric code meets the Stage 0 comment and provenance targets.
- The Madden-removal diff is reported.
- Line counts before and after.

**Stage 5 design (Claude, 2026-10-04):**

| # | Decision | Why |
|---|---|---|
| R5-1 | One routine, `l4.Rubric`, and one table, `l4.Defaults`: film, RAS and breakout as capped S-curves, four breakout weights, three curves and the athletic lift per position. A component with cap 0 is off (QB and K read no RAS; K has no breakout). | The ten rubrics differed only in constants. |
| R5-2 | Each adjustable number is a param per position (`l4.film.cap@WR`, …), seeded from the table where the shipped value is not zero. The curves stay in the table. | R7 needs the caps and weights editable. A zero is a mechanic that is off at that position; turning one on is a rubric change. Curves as params would be dozens of breakpoint rows nobody can edit by hand. |
| R5-3 | A run builds its rubrics from its own params snapshot (`composition.Rubrics`). Breakout weights an edit leaves off 1 are rescaled. | A run records exactly the settings it scored with, and a proposed param set scores with its own. |
| R5-4 | The cushion is applied to the age trajectory at every position, and composition hands it only to DT. | One rule for both halves of the guard; it is the same switch L3 already used. |
| R5-5 | The kicker's 0.60/0.40 film blend moved into composition, then went with Madden. | The rubric reads one film composite at every position. |
| R5-6 | The twin files in `scouting/assembly` became one generic routine; `m1_scouting.go`'s merge functions were left as they are. | Both go when today's engine does (Stage 7); the assembly pair was cut because the plan names it. |

**Gate check, 2026-10-04** (branch `session/core-stages-4-8`, live on Claude-OS against the
Stage 4 gate databases; evidence in `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage5-2026-10-04/`):

| Gate item | Result |
|---|---|
| The golden board is identical | PASS, twice. **Unit:** the sha256 of every output bit over 20,000 generated inputs per position, taken from the ten old rubrics before deletion, matches the new routine at all ten positions (`l4/golden_test.go`). **Live:** Score League with the old build (`v0.5.0-140`), the new one (`v0.5.0-144-g1159105`) and the old again: all three runs have scores hash `6268eb7e…` over 1,443 players. The two old runs' inputs hashes match, so the data did not move. |
| Validation cases | PASS: the 13 harness cases run against the new registry with zero FAIL. |
| Comment and provenance targets | PASS. `l4` 12.6% comments, no decision labels. Ratchet: comment 18 → 15, provenance 7 → 3, dupl 34 → 11, tiny files 27 → 24. |
| Line counts | `l4`: 1,522 → 356 production lines, 1,932 → 288 test lines. Madden removal: 30 files, −2,415 lines. Assembly twins: 643 → 245 lines. |
| Madden removal diff (`v0.5.0-145-g17efc6c` against run 3) | 648 of 1,443 scores changed, every one down: Madden's film composite sat above the S-curve's midpoint for every rostered player, so it was a flat lift of up to 5%, not a separator. Mean change by position −1.2% (DT) to −3.1% (QB); largest −5.85% (a CB). CB and S keep the coverage-only film (225 players). Kickers unchanged. Top 100: 97 stay. Largest rank move 32, mean 5.9. Table in `madden-removal-diff.txt`. |
| Defect found | With a CFBD key, Score League failed: it read the current college season, whose players are not in the NFL, so the feed resolved no one. Fixed: college production is read through the last completed season. A second problem remains on that old path: fetching six seasons live from CFBD timed out at 90 s in the VM. Stage 7 replaces it by reading the college data Signals already stores, so it was not patched. Christopher's Beelink has no CFBD key, so his board never took this path. |

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

**Stage 6 design (Claude, 2026-10-04).** The tool is `go run ./cmd/fit -db <history.db>`. Its
code is `internal/model` (the pure model) and `internal/model/fit`. Its outputs are
`internal/store/params/fitted.json` and `docs/fit/Fit_Report.md`.

| # | Decision | Why |
|---|---|---|
| R6-1 | **The scale is a within-position percentile of league fantasy points per game played, among the season's regulars (4+ games).** One k per position on that scale, not one per production component (item 3). | The league's own scoring is what an asset is worth here, and a percentile makes positions and seasons comparable. Per-component k's need an opportunity and an efficiency series that map onto points. IDP has no free opportunity denominators, and on offense it would be a second model. The single-scale blend beats both last season alone and the prior alone at all ten positions on the holdout. **This carries into Stage 7:** on-field-now is built on points per game, not on usage rates (Stage 7 item 1). |
| R6-2 | A game is a week with any snap or a league score, up to the league's last week (17 in every season from 2021 to 2025). | Injured weeks add no trials, and a game played that scored 0 still counts. |
| R6-3 | **The prior is one ridge regression on standardized inputs, with a missing indicator per input.** Inputs: log pick, entry age, the eight combine tests and last-college-season production. Penalty by 3-fold cross-validation, games-weighted, over seasons 1–3 of a career. Coefficients are folded back to raw units: `Intercept`, `Weight`, `Missing`. | Facts that move together are not counted twice (E9e). A missing test scores as its fitted missing term, not as zero (Stage 7 item 3). Raw units mean the app needs no stored means. |
| R6-4 | **k_now comes from a one-way analysis of variance on weekly points**, divided by (1 − R²_true), with R²_true = prior R² ÷ mean reliability. It uses the holdout R² when the holdout has 30+ rows, otherwise the lower of train and holdout. Split-half (odd weeks predicting even weeks) is its holdout check. | ANOVA measures the same within-season reliability as split-half, from every week. Dividing makes k a measure against the prior (item 2). The cautious rule keeps a small, lucky holdout from inflating k (K's first fit did). |
| R6-5 | **Dynasty k and the talent arc are fitted together** on season pairs, as next = Z·this + (1−Z)·prior + one step of the arc. Survivors are weighted by 1/P(survive), capped at 5. Exits are left to the survival arc. | The first attempt, deltas with exits imputed at the 10th percentile (item 6's method), counted regression to the mean and attrition twice: the arc was pulled down by both. It was rejected. |
| R6-6 | **Every optional piece earns its place on the 2025 holdout:** Z's shape, the arc against none, fitted recency against Marcel's 1/0.8/0.6, and the survival model against the base rate. The winner is refitted on all seasons; a tie goes to the simpler piece. | It is one rule for every choice. A piece that cannot beat its simpler alternative on unseen data is not shipped. |
| R6-7 | Recency is fitted after moving each past season along the arc. Evidence is the effective count (Σw·g)²/Σw²·g. | Fitted on raw seasons, recency would count age twice (item 5); the effective count is E7. |
| R6-8 | **Survival is one-year logistic** on age, age², log pick, games/17 and percentile, using only facts known at the season's end. Years 2–5 chain the one-year rate (Stage 7). Injury history enters as games played. | There is no free injury feed in the store, and a lost season shows as missing games. Never conditioning on future career length is E10f. |
| R6-9 | **Athleticism × age was tested and rejected.** The test used a properly scaled measure: the mean within-position z-score of 40 time, vertical, broad jump, 3-cone and shuttle, on players with two or more tests. **DT:** the one position the cushion covered got worse on the holdout (0.247 → 0.251). **DE:** improved 0.003 with the opposite sign (athletic DEs declined faster). **Elsewhere:** under 0.001, coefficients about 0.003 per SD per year. A first try that used the combine part of the prior as the measure gave wild arcs: offsetting height and weight terms with a spread that has nothing to do with athleticism. Both runs are in the evidence. | Item 6 asked for the test. No fitted term replaces the DT cushion; it leaves with today's engine in Stage 7. |
| R6-10 | **Fitted values are params with provenance:** `model.*@POS`, `is_calibrated = 1`, described as "fitted 2026-10-04 on 2021–2025". Shipped defaults upsert at start-up and overrides are untouched. Admin gained a filter and a Source column (fitted or hand-set), and takes any step. | R7: editable, and a run records the values it used. 487 settings do not fit one unfiltered table. |
| R6-11 | **The fit is reproducible.** Points per game was summed over a Go map in random order. The last bit changed between runs, flipped percentile ties, and moved the penalty and grid choices (QB prior holdout R² 0.32 vs 0.31). Weeks are now summed in order; a test fails without the fix. | The plan requires a reproducible fitting tool. |
| R6-12 | **The prior's inputs stay pre-NFL facts.** Only its calibration target is the league-points percentile. | CLAUDE.md's no-leak rule is about inputs: it keeps production from entering the prior and being counted twice. Item 1 asks for the mapping to the percentile, and k measured against the prior (R6-4) keeps the blend from counting the prior's share twice. For Christopher to veto if he reads the rule more strictly. |
| R6-13 | Weekly league scores come from MFL `playerScores` with `W=n`, fetched once per closed season at 0.2 requests per second. The path year must be the season itself; MFL answers `W=18` with week 17. | The k_now fit needs weekly points; the app at runtime needs only year-to-date (Stage 7). |

**Gate check, 2026-10-04.** Branch `session/core-stages-4-8`. Fit on
`~/scratch/twr-stage4/db/history.db`: 417,071 values, MFL year-to-date and weekly 2021–2025.
Evidence is in `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage6-2026-10-04/`.

| Gate item | Result |
|---|---|
| A fit report with holdout scores for every parameter | PASS: `docs/fit/Fit_Report.md`. Each group is trained on seasons before 2025 and scored on predicting 2025. **Prior:** holdout R². **k_now:** split-half; shrunk beats raw at all 10 positions. **Dynasty k and Z's shape:** pair RMSE against last season alone and the prior alone; the blend beats both at all 10. **Recency:** against Marcel; fitted at 7, Marcel at QB, K and DT. **Arc:** against none; kept at 8, none at QB and CB. **Survival:** against the base rate; model at 9, base rate at K. |
| Parameters loaded and shown in the Admin Console | PASS, live on Claude-OS at `v0.5.0-151`: Control → Engine Admin lists 487 settings, and the filter "fitted" gives 380, the `model.*` values for 10 positions. "WR model.arc" shows the four WR arc terms. The CB arc shows 0, as the holdout chose. After the reproducibility fix the VM's seeded values were compared with `fitted.json` at `e8ef223`: 380 of 380 equal. Screenshots 01–04. |
| Reproducible | PASS: three fits gave byte-identical `fitted.json` (sha256 `e43b22de…`) and report (`3c89c5c2…`). |
| Findings to carry | **DT's prior** explains 40% of training seasons but 3% of 2025's, so DT k_now barely leans on it. **Kickers:** the prior explains about nothing, last season alone predicts worst (RMSE 0.403), and dynasty k is 149 games; in this league's scoring a kicker's season barely predicts his next. **Noise:** choices decided by under 0.002 (several arc, recency and shape calls) are within one season's noise, as the report says. **Arc level:** the level term is negative at most offensive positions because each year's rookies push incumbents down the percentile scale. That is real and separate from the dynasty discount, which is time preference only. |

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

**Stage 7 design (Claude, 2026-10-04).** Code is in `internal/model` (`Project`, `Value`),
`internal/modelrun` (the pass over the league and the case set), the `model_runs` and
`model_scores` tables in history, and `ScoreLeague`.

| # | Decision | Why |
|---|---|---|
| R7-1 | **On-field-now is built on league points per game (R6-1), not on usage rates** (item 1). This season's games blend by k_now with the projection from earlier seasons. Evidence is games played, so an injured week adds nothing. | It is the scale everything was fitted and checked on. A usage model has no free IDP denominators and would be a second, unfitted scale. |
| R7-2 | **One projection, `model.Project`, for the fit and the app.** The last three seasons are each moved along the arc to the season before the target, weighted by recency and games, and blended with the prior by the dynasty k on the effective count. One more arc step reaches the target. With one season it is exactly the season-pair prediction the dynasty k and the arc were fitted on. The recency fit was rerun through it (DE 0.3 → 0.2). | The fits had applied the arc in two slightly different places. The app must score with the form that was fitted. |
| R7-3 | **Dynasty** = Σ over seasons t = 0…H−1 of dᵗ × P(on the field) × level, ÷ Σ dᵗ. This season's level is on-field-now; next season's is the projection with this season included, then the arc carries it. P chains the survival arc, with a projected season counted as full. **H and d are hand-set params (`dynasty.seasons`, `dynasty.discount`) for Christopher: shipped at 5 and 0.85, set by him on 2026-10-04 to 3 and 0.75 (win-now).** It is reported both as a percentile and in league points per game, read through last season's scale at the position. | Item 2. Cap and contract stay outside it (separable). The discount is time preference, which is a product call. Points per game make positions comparable on the board. |
| R7-4 | **A rookie's chance of taking the field is fitted (`model.debut.*`):** a logistic of becoming a regular in his first season, on draft slot, per position. It beats the base rate on 2025's rookies at all 10 positions. The prior is the rookie's level, so no arc step applies before his first season. | Prior and survival were fitted on players who played, so at first every rookie was assumed to play. That put a #1-pick QB who had not taken a snap third on the dynasty board. |
| R7-5 | **Missing measures are normal** (item 3). The prior scores a missing input by its fitted missing term. A player with no birth date gets no arc step and an age-27 survival. Each score lists the inputs that fed it. | A player is valued on what exists. |
| R7-6 | **Age enters once, in the arcs** (item 4). Today's board keeps its L3 decay until it is retired; the model reads no age except through the arcs and the prior's entry age. | Item 4. |
| R7-7 | **A model run has its own append-only tables** (`model_runs`, `model_scores`) and shares `param_sets` and the board's write path. It records only `model.*` and `dynasty.*` params, so a board edit makes no new model run. | `scoring_runs` only allows the kinds board and rebalance. Widening that rule means rebuilding a table in Christopher's real history.db. |
| R7-8 | **Score League loads MFL season totals for every season since 2021.** Earlier seasons are loaded once; last season and the current one are reloaded every pass. Last season and the current one come through the current league, under today's scoring rules; older ones are only served under their own year's rules. | The board's base has always been last season under today's rules (Maye: 448.35 under 2026 rules, 428.95 under 2025). Percentiles are within a season, so older rule changes barely move them. |
| R7-9 | **Today's board reads its college share and breakout age from history, not from CFBD live.** The `collegeshare`, `collegedefense` and `agetrajectory` fetchers are gone; only school tier still calls CFBD. | This fixes the Stage 5 defect: with a key, Score League fetched six seasons live and timed out. Christopher's board, which has no key, now gets the college signals too. |
| R7-10 | **The harness is retired.** `internal/harness`, the Rookie Sandbox and Architectural Tests tabs, their bindings, and the l4 hooks only the harness read are deleted. The Admin endpoints moved to `admin_app.go`. The frontend store, which is the app's IPC gateway, is now `store/app.ts`. | The case set in `internal/modelrun/cases_test.go` replaces the harness's cases on the real params. |

**Gate check, 2026-10-04.** Branch `session/core-stages-4-8`, live on Claude-OS at
`v0.5.0-166-g7fe4ab1`, against the gate databases, with a CFBD key set. Evidence is in
`~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage7-2026-10-04/`.

| Gate item | Result |
|---|---|
| Rookies are non-zero | PASS. Model run #3 valued all 1,450 rostered players; the lowest dynasty value is 0.04 points per game. All 298 players today's board scores at zero base have a positive on-field-now and dynasty value. The 85 with no NFL season are valued on the prior alone. |
| Injured veterans keep discounted evidence | PASS (case set): a WR with 3 games in his last season keeps on-field-now 0.849 against a healthy twin's 0.851. His evidence drops from 34.7 to 27.0 effective games, and his dynasty value falls through survival (0.649 vs 0.718). |
| Case set | PASS, on the shipped params: rookies by draft slot (0.86 / 0.52 / 0.32, with dynasty ordered by debut chance); the year-2 jump (0.559 vs 0.532); a peak WR (0.85 now, 0.70 dynasty); RB vs QB at 31 (dynasty 0.48 vs 0.68); the injured starter; an athletic DT at 31 (separated only by the prior, as R6-9 found); and a backup with 3 games at the 98th percentile, shrunk to 0.63. |
| Holdout: 2025 from data through 2024 beats today's board | **Mostly.** `docs/fit/Fit_Report.md`, "Against today's board", with params fitted before 2025. **Talent** (2025 regulars, points per game): the model wins at 9 of 10 positions, e.g. RB 0.76 vs 0.53 and TE 0.80 vs 0.56; it loses at K (0.03 vs 0.25). **Value** (everyone who played in 2024, 2025 total points): even, within ±0.02 at 9 positions, the model ahead at DT (0.64 vs 0.58); it loses at K (0.07 vs 0.46). **Rookies:** the model ranks 2025's rookie regulars at 0.41–0.86; today's board scores them all 0. **Kickers fail:** their prior is noise, and the fit shrinks toward it. |
| Retire the harness | PASS: deleted, golden rubric hashes unchanged, the Control module shows four tabs (screenshot 04). Total −2,326 lines. |
| Spearman against today's board | Model run #3 against board #8, 1,443 players: on-field-now 0.58, dynasty 0.62. By position, on-field-now runs 0.21 (K) to 0.76 (TE) and dynasty 0.54 (K) to 0.77 (QB). Moderate by design: the board ranks last season's total points, so injuries and rookies sink, while the model is per game and reads this season. |
| Board change from R7-9 | Board #5 → #7 with the college signals now from history: 1,098 of 1,443 scores changed, within ±5%. The top 100 is unchanged; the mean rank move is 3.9. |
| Known limits | **In-season k:** in-season, k_now blends this season against the projection, but it was fitted against the pre-NFL prior, so this season weighs somewhat too much (Maye, three 2026 games, Now 17.6). **Rookies who sit:** a rookie with no game by week w keeps his draft-day debut chance; fitting the chance given no game through week w needs the weekly history. **Kickers:** see above. The app is not used in-season until 2027-01-01, so these are follow-ups. |

### Stage 8 — M2 on the new numbers

1. Two views: **this season** (on-field-now) and **the franchise** (dynasty).
2. Recompute via `scoring_runs`.

**Gate:**
- A param edit → a new run → a changed board, with the old board still readable.
- Both views shown.

**Stage 8 design (Claude, 2026-10-04).** Code is in `m2_app.go` (`GetPowerRankings`),
`internal/m2service` (the board) and `PowerRankingsBoard.tsx`.

| # | Decision | Why |
|---|---|---|
| R8-1 | **Two views** (item 1). **This season** sums each roster's on-field-now and blends its z-score with the season's results at the free weight (default 60/40). **The franchise** sums each roster's dynasty value. Both read the season's latest model run in league points per game; the roster aggregates as the full sum or the top-N starters, as before. | Points per game add up across positions into what a lineup scores. Percentiles are within a position and do not. |
| R8-2 | **The franchise ranks on the roster alone** (weight fixed at 1, no slider). | This season's record says nothing about the roster three seasons out. |
| R8-3 | **Recompute reads the model run, not `scoring_runs`** (item 2 changed by R7-7). Δ is each team's rank move against the board built from the previous model run, with the same standings and today's ownership, so it isolates what the model changed. | The model run is the stored, append-only unit; the old one stays readable and is what Δ is computed from. |
| R8-4 | **The results side reads what MFL reports:** all-play win% when the standings carry it, otherwise points for ÷ the league's best, and nothing before the first week. This league's standings carry no all-play. The board names the result it read in its banner, the slider and the column. | Points for is as free of schedule luck as all-play. A 0-0 all-play gave every team the same results score and quietly made the blend roster-only. |
| R8-5 | **The season's phase comes from MFL's standings:** the weeks scored are counted from the head-to-head records against `lastRegularSeasonWeek`, giving FINAL or NOT STARTED. | The old FINAL read all-play, which this league never reports, so the board said FINAL mid-season. |
| R8-6 | **A run records only the params it reads** (found in the gate). `params.Set` splits into `Board()` and `Model()`, the model's part being `model.*` and `dynasty.*`; the board run records one and the model run the other. | The board run recorded all 509 params, so a dynasty edit wrote a board identical to the last one and M1's Δ compared against it. |
| R8-7 | **Admin shows which settings are pinned, and Reset clears them** (found applying Christopher's horizon). A row is amber when it has an override, even one equal to the default, and Reset returns it to the shipped default. Live on `v0.5.0-178-g74c15d5`: the gate VM's 0.85 showed pinned against the new 0.75 default; after Reset, model run #9 scored on 3 seasons at 0.75. Evidence: `live-gate-horizon-2026-10-04/`. | An override wins over every later shipped default. A value set back to its old default looked untouched yet stayed pinned, and would hide a refit's new values the same way. |

**Gate check, 2026-10-04.** Branch `session/core-stages-4-8`, live on Claude-OS at
`v0.5.0-172-g5eb0e1b`, then `v0.5.0-173-geb005c6` for R8-6, against the gate databases.
Evidence is in `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage8-2026-10-04/`.

| Gate item | Result |
|---|---|
| Both views shown | PASS (screenshots 01, 02). This season on model run #3: the banner and the slider name points for. The franchise ranks in order of roster value (412.5, 356.8, 351.3 …), and Δ against model run #2 shows moves. |
| A param edit → a new run → a changed board | PASS. `dynasty.discount` 0.85 → 0.6 in Engine Admin (03), then Score League wrote model run #4 (04). Now is unchanged and Dyn moved as a nearer horizon should: young RBs up (Robinson 19.7 → 21.8), a rising young QB down (Maye 20.3 → 19.8). The franchise view on run #4 shows Δ against run #3 (05): Lions 6th → 4th (+2), Giants 9th → 13th (−4). |
| The old board stays readable | PASS. `model_scores` holds model runs 1–4 at 1,450 rows each, behind the append-only triggers. Restoring 0.85 wrote run #5 on run #3's param set, with a scores hash identical to run #3's (a43ed59b): the same params give the same values. |
| R8-6, live | PASS on `eb005c6`: through a dynasty edit and its restore, M1 reported "No change … match board #11, so nothing new was written" while model runs #7 and #8 were written (12, 14). Before the fix, the same edit had written board #10, identical to #9. |
| Found and fixed in the gate | The slider label, the results column and the all-play record still said all-play: they now name the result the blend read, and the record shows a dash (ce690b5, with the points-for test that was missing). Right-aligned numbers touched the next cell ("0Minnesota Vikings"). On an unmaximized window M1's player names fell to two letters, and the Transact roster showed no names at all. The board grid now has a column gap, and name columns a 150 px floor (5eb0e1b). |
| Known limits | The Transact roster's middle pane is narrower than its table, so status and salary scroll. That is periphery layout, left as is. |

## Open items that need Christopher

- ~~The dynasty horizon~~ decided 2026-10-04: 3 seasons at 0.75 (win-now), now the shipped
  default. The case set passes on it.
- ~~CFBD key on the Beelink~~ set 2026-10-04: `~/.config/cfbd/api_key` (mode 600), exported from
  `~/.profile` (menu launches, after the next login) and `~/.bashrc` (terminals).
- **All-play:** Christopher is turning it on in MFL. The blend switches to it on its own (R8-4)
  and it moves only This season; the franchise view ranks on the roster alone (R8-2).

- The fresh live pull is gated (`TWR_LIVE_*`) and runs on his machine (Stage 2.3).
- Spot-check the MFL comparison (Stage 2 gate).
- The README's "How It Gets Built" section is rewritten after the last stage (deferred by
  Christopher 2026-10-03). It still credits the retired GLM/Gemini/DeepSeek/Ornith council.
