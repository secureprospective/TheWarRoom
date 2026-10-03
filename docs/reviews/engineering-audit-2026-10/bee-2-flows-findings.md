# Sector 2 — Flow findings

## Findings

### S2-F1 — `HIGH` — Real-player scouting is only a multiplier on prior-season fantasy points, not an independent measurable

**Demonstrating evidence:** the production `rankings.scorePlayer` looks up `r.base[p.MFLID]`; a missing Go map key gives BasePoints=0. It then sends the same player through composition and the position rubric. `engine.Pipeline.Score` computes `ScoutingAdjusted = BasePoints × AgePull × L4.Combined`, then applies the cap multiplier. Thus any player with no 2025 MFL score has final `AdjustedScore=0` regardless of RAS, film, breakout, age, or contract. A positive-base player still has only one mixed L4 multiplier. `season_scores` contains no separate on-field-now and dynasty-asset result; M2 receives AdjustedScore. Flow 5 traces this in detail; Sector 3b quantifies the actual rank/score effect separately for both test-exercised copies.

**Anchor quotes:**
> `basePts, hasBase := r.base[p.MFLID]` — `src/internal/rankings/rankings.go`, `scorePlayer`; an absent key reads as zero.
>
> `ScoutingAdjusted = p.BasePoints * agePull * l4.Combined` — `src/internal/engine/pipeline.go`.
>
> `Combined is the product ... and is the only field the accumulation reads` — `src/internal/engine/types.go`, `Layer4Output`.

This inverts Christopher's locked direction: under the existing formula, last season's fantasy points decide whether the player has a nonzero value, and real-player measurements can only nudge that value. Rookies and any 2025 non-contributor with no row are mathematically unresponsive to the measurable. The intended two values—on-field-now and dynasty asset—are neither stored nor independently selectable by M2.

**Fix direction:** give each player independently computed and persisted on-field-now and dynasty-asset values; keep MFL fantasy points as one current on-field input while Layer-2 computation remains deferred. Let M2 choose/use those values rather than multiplying every real-player measure onto one prior-year fantasy score.

### S2-F2 — `HIGH` — MFL roster changes do not refresh local runtime state after the first seed

**Demonstrating evidence:** `state.Initialize` fetches `rosterSeedSource.Rosters` only when it finds no rows for the current state season. `rosterSeedSource.Rosters` is the sole non-test caller of `rosters.Fetch`. Every later startup loads SQLite without reseeding. There is no roster-refresh IPC, MFL transaction-import fetcher, or MFL write-back path. `App.directory` does fetch the MFL players database once per process, but that updates only the in-memory name/position/DOB lookup; it does not write `rosters`, `contracts`, or `contract_years`. The only later state writes are local `Coordinator` operations, so trades, cuts, signings, contract changes, or new roster assignments made on MFL are not imported. An unrostered MFL player being present in the directory does not add him to the runtime state or free-agent pool.

**Anchor quotes:**
> `On a fresh database ... it seeds ONCE from src; on an existing database it loads what is there WITHOUT reseeding` — `src/internal/store/state/state.go`, `Store.Initialize`.
>
> `rosters.Fetch` — `src/app.go`, `rosterSeedSource.Rosters`; this is the only non-test production call site.
>
> `directory ... fetching it on first use` / `players.Fetch` — `src/app.go`; this is the process-cached player lookup, not a roster writer.

On 2026-10-03, mid-season, no-refresh means the local roster can remain at its first seed while the actual MFL league changes. The DB copies are already test-exercised and cannot establish whether either seed matched the live league at inception. Without a refresh/reconciliation path, the app has no evidence-backed current roster state. This is a `HIGH` core data-integrity finding.

**Fix direction:** provide a deliberate MFL roster reconciliation path that reports adds, drops, ownership/status, and contract deltas before applying them to local state; do not overwrite coordinator-managed ledger history blindly.

### S2-F3 — `HIGH` — The test-exercised release-path copy pairs 2027 state with 2026 scoring/output year

**Demonstrating evidence:** `src/internal/ingestion/schema.go` sets `SeasonYear = "2026"`; `App.startup` parses it once into `a.season`. Separately, `state.refreshSeason` re-derives the runtime state year from the newest phase row. The release-path-named copy has a 2027 phase/roster state (831 rows) and 2026 `season_scores` (827 rows, config 1, created `2026-07-20T14:03:21Z`, after its July 18 rollover). `App.ScoreLeague` would use the unchanged 2026 `a.season` as its score/output key while `Runner.Run` reads the current state roster; `basePoints` asks for `a.season-1=2025`. M2 similarly requests MFL standings for `SeasonYear=2026` and attaches franchise ownership from runtime state. Inventory §1c and MAP Flow 6 give the query and test-state provenance.

**Anchor quotes:**
> `season is parsed once` — `src/app.go`, `App.startup`.
>
> `refreshSeason re-derives s.season from the phase log` — `src/internal/store/state/season_phase.go`.
>
> `playerscores.Fetch(..., strconv.Itoa(a.season-1))` — `src/m1_app.go`, `App.basePoints`.
>
> `Run scores every rostered player for season` — `src/internal/rankings/rankings.go`, `Runner.Run`.

For this copied DB state, the season/roster/output inputs disagree, so the reachable board path is `BROKEN`: `ScoreLeague` sees 827 existing rows and skips rather than recomputing, while `GetRankings` reads 2026 output and joins display/ownership from 2027 state. If a fresh pass were needed against this same state, the compile-time 2026 score key and 2025 YTD request would also disagree with the 2027 runtime roster. The rapid phase sequence, in-app rollover, and expiry operations establish that this copy is test-exercised. This does **not** establish that the real league is on season 2027 or prove the existing score rows were produced by this exact source revision; neither DB is a trustworthy live mirror (S1-F6). The code still has two season authorities, so a similarly mixed state can recur if they diverge.

**Fix direction:** use one authoritative runtime season for state, MFL requests, score/output keys, and M2. Re-score only under an identity that represents all inputs; do not merely relabel this output or infer the real league year from the test database.

### S2-F4 — `HIGH` — Layer 2 consumes an MFL YTD proxy; the stored MFL scoring matrix does not produce BasePoints

**Demonstrating evidence:** for `a.season=2026`, `App.basePoints` calls MFL `playerScores` on league/path year 2026 with `W=YTD&YEAR=2025`, then maps each returned string directly to BasePoints. As of 2026-10-03 the active 2026 NFL season is underway, but the app still requests the prior completed 2025 YTD aggregate. A player omitted from that export has no map entry and `rankings.scorePlayer` supplies zero. `engine.Pipeline.Score` accepts that value and multiplies it; it does not calculate league fantasy points from `RawConfig.ScoringRules`. Non-test `GetScoringRules` usage search returns only its definition. The UI labels the value “proxy / L2 pending.” Christopher has deferred L2 scoring computation, so the important current seam is that the real-player measurable must work independently while MFL points remain the fantasy input; S2-F1 documents that it currently does not.

**Anchor quotes:**
> `playerscores.Fetch(..., strconv.Itoa(a.season-1))` — `src/m1_app.go`, `basePoints`.
>
> `W=YTD` and `YEAR: scoreYear` — `src/internal/ingestion/playerscores/fetcher.go`.
>
> `basePts, hasBase := r.base[p.MFLID]` — `src/internal/rankings/rankings.go`; a missing entry reads zero.
>
> `GetScoringRules returns the active position-additive scoring config` — `src/internal/store/rulebook/rulebook.go`; there is no non-test caller.

Separately, MFL does not populate `RawConfig` with the salary floors used by transactions: §6 `MinSalaryFloor` and §10 `PositionFloor` are code switch tables (`src/internal/transactions/freeagency/freeagency.go`, `pricing.go`). The scoring-matrix cutover and contract-rule machinery are deferred by the 2026-10-03 scope decision; they are dependency seams, not the first core implementation step.

### S2-F5 — `HIGH` — M1's always-fetched Madden film backbone is pinned to stale 2023-edition ratings and has no freshness guard

**Demonstrating evidence:** `madden.RatingsURL` is fixed to `/v2/entities/m24-ratings`; the package comment calls m24 “the 2023-season ratings” and records that, as of its 2026-06-21 live check, m25 returned empty and m26 returned HTTP 500. On 2026-10-03 the production source still pins m24. The active path calls Madden in both `mergeIDPFilm` and `mergeOffenseFilm`; the fetcher reads EA rating values without a season/date field or freshness validation. Stale-but-successful records are therefore accepted and blended. A missing player match is an ordinary per-player miss and L4 is Data-Parity neutral; an empty/dead slug, HTTP error, or parse failure is returned by `madden.Fetch`, bubbles through `BuildIDPFilm` / `BuildOffenseFilm`, and makes `App.ScoreLeague` return an error before the scoring runner persists output. It does not silently degrade the whole league to neutral.

**Anchor quotes:**
> `RatingsURL = "https://ratings-api.ea.com/v2/entities/m24-ratings"` — `src/internal/ingestion/madden/fetcher.go`.
>
> `m24 (the 2023-season ratings, two seasons stale)` / `m25 is empty, m26 500s` — same file, package comment with its 2026-06-21 verification date.
>
> `return nil, err` on the Madden fetch — `src/internal/scouting/assembly/idpfilm.go`, `BuildIDPFilm`; `src/m1_scouting.go` wraps and returns the failure from the unconditional merge.

Madden is skill-grade/dynasty-ability context, not current NFL on-field performance. The separate FTN/PBP and PFR signals request completed 2025, so they are the freshest complete NFL-season data in this path but still one season behind the in-progress 2026 NFL season. This is a core signal-freshness limitation, distinct from the intentionally deferred L2 MFL scoring cutover.

**Fix direction:** pin and freshness-check a supported Madden source, or keep Madden explicitly on the dynasty-asset side and do not label it current film; define whether a Madden-source outage should stop all M1 scoring or preserve the other measured sources.

### S2-F6 — `HIGH` — K’s declared active Madden/NFLProduction rubric has no scorer input path

**Demonstrating evidence:** `App.buildScoutingDirectory` merges Madden into `Profile.IDPFilm` or `Profile.OffenseFilm`; neither profile group is present for `PosK`. `rankings.applyScouting` copies RAS/school/college/breakout, then sets `HasFilm` only for `IDPFilm` or `OffenseFilm`; it never sets `PlayerSpec.MaddenFilm`/`HasMaddenFilm` or `NFLProduction`/`HasNFLProduction`. `madden`’s fetch is active for the other positions, but there is no K profile assembler. Consequently the real kicker rubric receives neither K input and its Data-Parity path returns neutral film for every K.

**Anchor quotes:**
> `Film ACTIVE — S-curve over a Madden 0.60 / NFLProduction 0.40 composite` — `src/internal/engine/l4/kicker/k.go`.
>
> `hasMadden := profile.IDPFilm != nil` and `else if profile.OffenseFilm != nil` — `src/internal/rankings/rankings.go`, `applyScouting`.
>
> `MaddenFilm ... no fetcher populates them today` — `src/internal/scouting/types.go`.

The code’s `App.rubrics` does register `kicker.NewK`, so this is a reachable broken input handoff, not an absent rubric. The K-layer film remains the neutral value even though the K rubric itself is non-identity.

### S2-F7 — `HIGH` — Score changes do not invalidate or recompute persisted M1 output

**Demonstrating evidence:** after any state transaction, parameter edit, or same-version rulebook override, `season_scores` is not invalidated. Frontend `SetParam` refreshes only `ScoreRookies`, `RunValidationSuite`, and `GetParams`. If the user presses Score League, the App fast path returns `SkippedExisting` as soon as **any** rows exist for `(a.season, active rulebook version)`, before it fetches MFL scores or reads the changed state/calibration; `Runner.Run` repeats the any-row guard. It neither compares current roster IDs with saved IDs nor checks that the batch is complete. The table's identity is `(season, scoring_config_id, mfl_id)`, not a roster or parameter version. A new active rulebook version would create a different key, but Reload/Promote have no App IPC/frontend surface; `SetLeagueSettingOverride` changes an override without changing the active version. Old immutable rows remain, and same-season admin `SetParam` cannot update an already-scored M1 board. A later season key can trigger a new pass, which reads then-current parameters; that is not a same-season invalidation.

**Anchor quotes:**
> `loadAll ... ScoreRookies(), RunValidationSuite(), GetParams()` — `src/frontend/src/store/harness.ts`.
>
> `if existing, serr := a.output.Reader().Scores(...); serr == nil && len(existing) > 0` — `src/m1_app.go`, `App.ScoreLeague`.
>
> `if len(existing) > 0 { rep.SkippedExisting = true ... return rep, nil }` — `src/internal/rankings/rankings.go`, `Runner.Run`.
>
> `(season, scoring_config_id, mfl_id)` primary key and UPDATE/DELETE-denial triggers — `src/internal/output/output.go`.

The AdminPanel copy says parameter edits cause the board to move, but the existing league board is frozen under a key that omits the changed parameters and roster state. Fix direction: score identity and invalidation must encode all score-affecting inputs and define recomputation/versioning; do not mutate immutable output rows in place.

### S2-F8 — `MEDIUM` — Two editable cushion parameters are stored but ignored; several North Star tunables are not exposed

**Demonstrating evidence:** `param_defaults` has five rows: two L5 boundaries, L3 decay, cushion RAS threshold, and cushion reduction. `composition.Assembler.Calibration` reads only `GetCapTiers` and `KeyLayer3DecayRate`; the DT cushion is obtained from `composition.cushionGuard`, which returns literal `(8.00, 0.90)`. `SetParam` can persist overrides for the stored cushion rows, but no effective engine read uses them. The current AdminPanel also has no Layer-2 score-value, per-position L3 peak, L4 weight, or L6 scarcity editor despite the North Star list.

**Anchor quotes:**
> `GetCapTiers` and `GetGlobal(params.KeyLayer3DecayRate)` — `src/internal/composition/composition.go`.
>
> `return 8.00, 0.90` — `src/internal/composition/defaults.go`, `cushionGuard`.
>
> `Layer 2 scoring value overrides`, `Layer 3 peak limits`, `Layer 6 positional scarcity matrix` — `src/North_Star.md`, Admin Console tunables.

`param_defaults` query and all five values are in `INVENTORY.md` §1c. Those values are real persisted/admin-visible rows, not live calibrations. This is separate from S2-F7: even the sandbox cannot tune these two cushion behaviors via the advertised override values.

### S2-F9 — `MEDIUM` — Commissioner calendar stores planned actions but has no execution path

**Demonstrating evidence:** `ScheduleEvent`, `RescheduleEvent`, and `CancelEvent` append intent rows to `calendar_events`; the frontend reads/renders the latest event and can schedule/reschedule/cancel. A whole-source search finds no production due-event runner and no `FIRED` write path. The board header explicitly states there is no fire-now IPC; no worker is started by `main.go`.

**Anchor quotes:**
> `It records intent only — the op runs when the event is fired, not now` — `src/frontend/src/components/calendar/CalendarBoard.tsx`.
>
> `no fire-now IPC yet` — same file.
>
> `w.AppendCalendarEvent(ctx, row)` — `src/internal/transactions/request_calendar.go`.

A scheduled timestamp currently does not trigger the payload or mark an event executed. The rendered calendar is a plan ledger, not an automation path.

### S2-F10 — `MEDIUM` — Activity Feed is a five-ledger projection, not a complete transaction history

**Demonstrating evidence:** `state.Feed` unions exactly `trade_notes`, `player_status_events`, `dead_cap_ledger`, `cap_relief_ledger`, and `contract_year_changes`. `ROSTER_STATUS` writes only `rosters`; `ADVANCE_PHASE`/window/deadline ops append `season_phases`; calendar requests append `calendar_events`. None of those mutations is included in the feed query.

**Anchor quotes:**
> `GetFeed reads ... the append-only ledger tables (trade_notes, player_status_events, dead_cap_ledger, cap_relief_ledger, contract_year_changes)` — `src/transactions_feed_app.go`.
>
> `UPDATE rosters SET roster_status` — `src/internal/store/state/writes.go`.
>
> `AppendPhaseTransition` inserts `season_phases` — `src/internal/store/state/season_phase.go`.

The feed is reachable and useful for the ledgers it projects, but its name/chronological-spine presentation must not be treated as a complete audit of all successful operations. The raw production copy lacks `trade_notes` before startup; current state initialization creates it before normal IPC use (migration verification remains Sector 3).

### S2-F11 — `MEDIUM` — Coordinator `CORRECT` operation has no frontend/App request mapping

**Demonstrating evidence:** `KindCorrect` is a request type; `Correction.validate/apply` append `transaction_corrections`, and `phasePolicy` explicitly classifies it. But `src/transactions_app.go:buildRequest` and `buildMoneyRequest` have no `CORRECT` case; no frontend component constructs or invokes it. The generated binding exposes only generic `ExecuteTransaction(TransactionRequest)` and that builder rejects this discriminator as unknown.

**Anchor quotes:**
> `func (Correction) Kind() Kind { return KindCorrect }` — `src/internal/transactions/request_correction.go`.
>
> `unknown transaction kind %q` — `src/transactions_app.go`, `buildMoneyRequest`.
>
> `KindCorrect` appears in `phasePolicy` — `src/internal/transactions/phase_gate.go`.

The handler is callable by code that constructs `transactions.Correction` and calls the coordinator directly, but that route is not reachable through the app’s IPC/UI.

### S2-F12 — `MEDIUM` — M4 signable free-agent pool is limited to explicit status-event history

**Demonstrating evidence:** initial `state.seed` writes players from MFL rosters and ledger cells; it does not write a player-master/free-agent table. `state.FreeAgents` and M4 `GetFreeAgentPool` derive the candidate set from latest `player_status_events` rows equal to `FREE_AGENT`, excluding rostered IDs. `freeagency.Sign` then requires the same latest status to be signable. A never-rostered MFL player with no status event has no route into this pool or through the signability check.

**Anchor quotes:**
> `only if no roster rows exist` / `seed ... rosters + contracts` — `src/internal/store/state/state.go`.
>
> `latest status is FREE_AGENT and who are on no roster` — `src/transactions_app.go`, `FreeAgentsResult`.
>
> `not a signable free agent` — `src/internal/transactions/freeagency/freeagency.go`.

The MFL player lookup includes the full league player feed but is not persisted as an eligibility source. Thus current signings can cover released/expired status-event players; a general unrostered-player intelligence pool (M5) is absent.

### S2-F13 — `MEDIUM` — Reachable validation harness is mislabeled as Module 3; matchup predictions are absent

**Demonstrating evidence:** `RunValidationSuite` returns architectural engine cases from `harness.RunValidationSuite`; `ValidationBoard` renders PASS/FAIL/PENDING test cases in Control. North Star Module 3 is “Matchups” and specifies matchup score prediction; there is no production matchup prediction service/IPC/component. The schedule board only renders MFL matchup schedule and scores.

**Anchor quotes:**
> `Module 3 — Matchups` — `src/North_Star.md`.
>
> `ValidationBoard is Module 3` — `src/frontend/src/components/ValidationBoard.tsx`.
>
> `RunValidationSuite` — `src/harness_app.go`; returns `harness.RunValidationSuite(a.rubrics())`.

The harness is reachable and useful, but it does not close the Module-3 user-facing feature.

## Flow status ledger

| Flow | Status | Stop / scope |
|---:|---|---|
| 1 Rulebook | `PARTIAL` | Cap/roster/starter/name branches are consumed; MFL scoring matrix and MFL contract floors do not drive the engine/transaction floors. |
| 2 Players/rosters/contracts/year ledger | `PARTIAL` | First MFL roster seed and local transaction writes are wired; external MFL roster/trade/cut/sign/contract changes do not refresh local state after seed (F2). |
| 3 Parameters | `PARTIAL` | Cap boundaries + decay consumed; cushion rows inert, no M1 recalculation/versioning, remaining North Star tunables absent. |
| 4 Scouting | `PARTIAL` | Active signal subset reaches L4; K input gap, CFBD key gate, and three unused fetchers listed in Flow 4. |
| 5 AdjustedScore through six layers | `PROXY` | Full calculation runs, but BasePoints is 2025 MFL YTD; missing per-player row→0, so real-player L4/L3 values only multiply that proxy and cannot move a zero-base player's score (F1/F4). No separate on-field-now/dynasty values. |
| 6 M1 score/persist/read/render | `BROKEN` | In the test-exercised release-path copy, runtime state is 2027 while source/output key is 2026; any existing rows freeze the batch. This does not establish real league season (F3/F7). |
| 7 M2 power rankings | `BROKEN` | In the test-exercised release-path copy only: 2026 score/standings path is combined with the 2027 state ownership. M2 otherwise receives one proxy-backed aggregate, not two measurables. |
| 8 Schedule/calendar | `PARTIAL` | MFL schedule/calendar plans render; due calendar payloads have no execution path. |
| 9 Coordinator transactions | `PARTIAL` | The 18 mapped kinds have preview/execute paths; `CORRECT` has no App builder/UI path. |
| 10 Season/FA/contract ops | `PARTIAL` | Local Coordinator contract ops are wired; no MFL-state refresh exists, and free-agent candidates remain status-event-derived. |
| 11 Feed | `PARTIAL` | Feed runs after state DDL, but omits roster-status/phase/calendar mutations; the release-path copy lacks `trade_notes` before startup DDL. |
| 12 Harness/Inspector/Home | `PARTIAL` | Sandbox, validation, inspector, and seasonal Home card are reachable; M3 prediction and other empty Home modules are absent. |
| 13 Infrastructure | `WIRED` | Startup, DB/stores/migrations, lock/logs, version path, freshness, and headless probe are connected in source; Sector 3 exercises copies. |
