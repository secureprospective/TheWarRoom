# Sector 2 — Data-flow map

Paths are relative to the run directory. Status means the route reachable from `src/main.go`, not merely that a package exists. `PARTIAL` names the stop; `PROXY` names the substitute; `BROKEN` names the demonstrated wrong result. Both database copies are test-exercised states, not evidence of the real league's current season: the release-path-named `data/thewarroom.db` ends on 2027 after test-like phase changes, and `data/thewarroom-dev.db` ends on 2026 with in-app transaction writes. Source `SeasonYear` and `App.season` are 2026. See `INVENTORY.md` §1c for provenance and rollover arithmetic.

## 1. League configuration / rulebook — `PARTIAL`

- MFL `TYPE=league` + `TYPE=rules` → `src/internal/ingestion/league.Fetch` (`call(..., "league", ...)`, `call(..., "rules", ...)`) → JSON envelope decode and `RawConfig` validation (`league.Fetch`, `mapRules`) → `league.APISource.Fetch` → `rulebook.Store.Initialize` → `insertVersion` into `rulebook_versions`, active pointer in `rulebook_active`, raw config JSON → `loadActive` → in-memory active config plus `rulebook_overrides`.
- This is reached from `src/main.go:main` → Wails `OnStartup: app.startup` → `src/app.go:App.startup` → `initStoreFloor` → `rulebook.Initialize`. On a fresh database only, `Initialize` fetches and promotes the first version. On an existing database it loads the current active version without network; refresh/promote are separate operations and neither has a frontend IPC surface in this snapshot.
- Cap branch: `rulebook.Store.GetSalaryCap` → `composition.Assembler.leagueCap` → `engine.ApplyCapScaling` (Layer 5); state branch `rulebook.Store.TaxiCapPercent` / `IRCapPercent` → `state.loadCellCap` applies roster-status cap discounts. Cap override comes from `rulebook_overrides`.
- Roster limits branch: `rulebook.Store.GetSetting` + `ActiveConfig().RosterLimits` → root `rosterPolicyAdapter.RosterSize`, `TaxiSquad`, `InjuredReserve`, `PositionLimit` → `transactions.Coordinator.Execute/Preview` roster-limit gate. Position enforcement reads the parsed **maximum**, not the MFL minimum. M2 branch: `m2service.Service.starterCount` reads only `ActiveConfig().Starters.Count` for the top-N mode. Franchise names flow through `FranchiseNames` into M2, schedule, transactions, and feed display DTOs.
- **Stops:** `GetScoringRules` returns the stored raw position-additive MFL scoring matrix, but there is no non-test caller. `src/internal/engine.Pipeline.Score` takes `PlayerInput.BasePoints` as already supplied. The contract floors are not in `RawConfig`: `transactions.PositionFloor` and `freeagency.MinSalaryFloor` are hard-coded rule tables. `Starters.Positions`, `IOPStarters`, `IDPStarters`, `UsesSalaries`, `UsesContractYear`, week bounds, and `IncludeTaxiWithContractYear` have no current production consumer found by whole-tree grep. Rulebook setting UI (`GetLeagueSetting` / `SetLeagueSettingOverride`) is used for selected taxi/IR controls, not a complete rules-governance surface.

Anchors:
> `On an existing database it simply loads the current active version without a network call` — `src/internal/store/rulebook/rulebook.go`, `Store.Initialize`.
>
> `GetScoringRules returns the active position-additive scoring config` — `src/internal/store/rulebook/rulebook.go`; whole-tree non-test search for `GetScoringRules` finds only its definition.
>
> `PositionFloor` / `MinSalaryFloor` each contain switch-based tables — `src/internal/transactions/pricing.go`; `src/internal/transactions/freeagency/freeagency.go`.

## 2. Players, rosters, contracts, and annual salary ledger — `PARTIAL`

- MFL `TYPE=players&DETAILS=1` → `players.Fetch` → `normalize.NewLookup` → `App.directory` caches `normalize.Lookup` for the process. It supplies names, positions, DOB/draft year, college, and crosswalk joins; it is not a persistent player-master table.
- MFL `TYPE=rosters` → `rosters.Fetch` → `normalize.Rosters(raws, lookup)` → `rosterSeedSource.Rosters` → `state.Store.Initialize` seeds **only if no roster rows exist for its league/current phase-log season**. Seed transaction inserts `rosters` + `contracts` + `contract_years` + the initial `contract_year_changes` audit rows. Existing state is loaded, never refreshed from MFL.
- `state.seedLedgerPlayer` lays one PAID cell per year from current season through the normalized contract end, then a trailing zero UFA cell. The cap source of truth is the current-season PAID `contract_years` cell, not legacy salary columns: `state.loadCellCap` joins those cells to current `rosters`, applies TAXI/IR percentages, rounds each contribution to $10k, and returns per-player `CapSalary` and per-franchise `CapUsed`. `state.Store.load` joins `rosters` to legacy `contracts` for identity/contract metadata, then loads money from the ledger.
- `transactions.Coordinator` is the sole runtime writer. Examples: `contracts.Tag/Extend/Restructure` change `contract_years` and append `contract_year_changes`; dead-cap releases void PAID cells and append changes; signing lays new cells; rollover advances roster/contract snapshots but preserves annual ledger history. These tables are read by cap/state readers, M1/M4, transaction logic, and the feed.
- `App.GetRoster` / `GetFreeAgentPool` → `store/transactions.ts` → `TransactionWorkspace` / `TradeBuilder`; after commit the transaction store reloads. State has no player-master row for an unrostered MFL player; free-agent eligibility is derived from latest `player_status_events`, not the whole MFL directory (see Flow 10).
- **Refresh seam (no route after first seed):** the only production caller of `rosters.Fetch` is `rosterSeedSource.Rosters` in `src/app.go`; `state.Initialize` calls it only when `hasState` is false. Later startup only derives the season and loads SQLite. `App.directory` can fetch the MFL players database once per process, but that is non-persistent identity enrichment and never reconciles `rosters`/`contracts`/`contract_years`. `playerscores.Fetch`, MFL standings, and fantasy schedule fetches have separate consumers and cannot update roster state. Local `Coordinator` transactions are the only state mutations after seed; they do not ingest changes made on MFL. No other non-test `rosters.Fetch` caller, MFL transaction-export fetcher, roster-refresh IPC, or MFL write-back call exists in the source search. Therefore MFL trades, cuts, signings, contract changes, and newly rostered players made externally do not reach local runtime state after first seed. Local state can drift while current MFL player-directory labels refresh on a later process; those paths are not a roster sync.

Anchors:
> `state.Initialize ... On an existing DB it loads what is there WITHOUT reseeding` — `src/internal/store/state/state.go`.
>
> `The ledger cell is the money source of truth (KING)` — `src/internal/store/state/helpers.go`, `loadCellCap`.
>
> `insertCell writes one contract_years cell and its INIT change-log row atomically` — `src/internal/store/state/ledger.go`.

## 3. Calibration parameters — `PARTIAL`

- No external source → `params.defaultParams` seeds `param_defaults` once → `params.Store.Initialize` loads defaults and `param_overrides` → `GetCapTiers` / `GetGlobal` → `composition.Assembler.Calibration` → `engine.Pipeline.Score`.
- **Consumed:** `captier.cold_ceiling_pct` and `captier.hot_floor_pct` → L5 boundary percentages; `layer3.decay_rate` → L3. L5 factors remain code constants: Cold `1.15`, Neutral `1.00`, Hot `0.85` in `src/internal/engine/capscaling.go`. Composition hard-codes per-position peak limits (QB 32, RB 25, WR/TE/LB 29, DE/DT/K 30, CB/S 28), RAS fallback 5, salary floor 0, and L6 scarcity rank 0. Layer-4 weights/curves are in position rubric code.
- **Present but unread:** the two seeded `cushion_guard` values. `params.KeyCushionGuardRAS` / `KeyCushionGuardReduct` are not read by composition; `composition.cushionGuard(PosDT)` hard-codes threshold `8.00` and decline factor `0.90`. The admin panel can write overrides, but these writes do not move the engine value.
- **Intent checklist gap:** North Star tunables not represented by this store/UI include MFL Layer-2 scoring-value overrides, per-position Layer-3 peak limits, Layer-4 component weights/parameters, and Layer-6 scarcity matrix. The five current defaults are the two cap-tier boundaries, decay, and two cushion values.
- `GetParams` / `SetParam` → `harness_app.go` → `store/harness.ts` → `AdminPanel`; successful `SetParam` calls `loadAll`, which runs `ScoreRookies`, `RunValidationSuite`, and `GetParams`. It does **not** run M1 `ScoreLeague` or reload `GetRankings`; the stored league board has a separate append-only identity stamped only with active rulebook version (see Flow 6).

Anchors:
> `The engine passes a Key* constant` — `src/internal/store/params/params.go`, `GetGlobal`.
>
> `cushionRAS, cushionDecline := cushionGuard(pos)` — `src/internal/composition/composition.go`; `8.00, 0.90` — `src/internal/composition/defaults.go`.
>
> `loadAll` runs `ScoreRookies(), RunValidationSuite(), GetParams()` — `src/frontend/src/store/harness.ts`; no M1-score IPC is in that call.

## 4. Scouting signals — all eleven SYSTEM_MAP fetchers, RAS assembly, and school-tier

`WIRED` means a source reaches a stored M1 score through `main.go`; CFBD wires only when nonblank `CFBD_API_KEY` is present. No environment values were inspected. All M1 signal fetches happen at ScoreLeague time; no signal snapshots or per-run coverage rates are persisted. Unless a documented rate is called out below, exact roster-to-signal match percentages **could not establish** from code/tests. If a CFBD key is absent, its four signal groups are skipped; a per-player miss is normally Data-Parity-neutral. A genuine fetch failure on an unconditional source aborts `ScoreLeague` before output is written.

| Signal / source | Source period and freshness at 2026-10-03 | Join path / available rate evidence | Rubric use and missing-data fallback | Meaning / status |
|---|---|---|---|---|
| `crosswalk` — DynastyProcess `db_playerids.csv` (`src/internal/ingestion/crosswalk/fetcher.go`) | `master` URL, no season/version pin; fetched at scoring time. Source revision is not recorded. | Foundation MFL ID→GSIS, plus optional ESPN→GSIS, PFR→GSIS, normalized (name, birthdate)→GSIS. Feeds RAS, CFBD, Madden, PFR and nflverse IDs. The package comments document 4/~7,900 ambiguous ESPN IDs and 3/~7,800 ambiguous PFR IDs dropped; no MFL-roster match rate is recorded. | Join substrate, not a rubric signal. A per-player bridge miss means the dependent profile field is absent; an entirely empty foundation map fails loud. | Identity infrastructure; `WIRED`. |
| `ras` — nflverse `combine.csv` (`ingestion/ras.Fetch`) | One all-season file, no year filter or pinned release. Contains combine measurables, a pre-NFL event; current feed contents are not version-stamped, so exact newest combine season **could not establish** offline. | PFR ID→GSIS through crosswalk, then roster MFL ID→GSIS; RAS assembler also resolves MFL position. Ambiguous duplicate GSIS combine rows are dropped (fetcher comment: ~17 PFR IDs have multi-rows); no roster match rate is persisted. | `BuildRAS` forms roster/position cohorts and averages available-measurable z-scores into RAS-equivalent 0–10. Missing record/drills → `HasRAS=false`; L4 uses its presence-neutral path, while L1/tiebreak hygiene falls back to RAS 5.0. | Athletic potential / dynasty asset; `WIRED`. |
| RAS **assembly** (`assembly.BuildRAS`) | Runs each ScoreLeague pass against that all-season combine feed. | `App.buildScoutingDirectory` → `BuildRAS` → crosswalk + `normalize.Lookup.Position(MFLID)` → `Profile{RAS,HasRAS}` keyed by MFL ID → `rankings.scorePlayer` → `applyScouting` → L4. | Per-position, per-measurable cohort z-equivalent; no combine row or no measurable means absent, not fabricated zero. RAS 5.0 is the L1/L6 fallback; rubrics gate RAS L4 on `HasRAS`. | Athleticism / dynasty asset; `WIRED`. |
| `madden` — EA `m24-ratings` (`ingestion/madden.Fetch`) | Fixed `m24` slug. Source comment identifies it as 2023-season ratings and says on 2026-06-21 `m25` was empty and `m26` returned 500; by 2026-10-03 the configured URL remains the 2023-edition feed, stale against the 2025 completed NFL season and in-progress 2026 season. No freshness check exists. | Madden name+birthdate→GSIS via crosswalk resolver; roster MFL ID→GSIS. Code comments report ~77% of the Madden feed resolves through name+DOB (feed match, not a roster coverage rate). Fetched separately by IDP and offense assemblers. | K1 IDP and K3 offense Madden composites use curated per-position EA attributes. An individual miss leaves no profile; other film seats stay neutral. If the source is stale-but-200, its ratings flow as current because no age check. If HTTP/parse/empty fails, the active assembler returns an error and `ScoreLeague` stops (not a neutral board). K has no Madden assembly, so its Madden flag remains false. | Static athlete-skill ratings, not current on-field production; stale dynasty/ability proxy. `PARTIAL` (active for IDP/offense, absent for K). |
| `veteranfilm` — nflverse FTN charting + PBP (`SeasonSources`) | `SeasonYear−1=2025` for both files. This is the latest completed NFL season, but not current for 2026-10-03: the 2026 NFL season is underway and the active season's partial charting is not requested. | Raw charted rows are GSIS-keyed; roster MFL→GSIS through crosswalk. No per-roster match rate is persisted. | `BuildOffenseFilm` uses FTN quality as a bounded percentile overlay on the Madden backbone; only players over role-specific charting floors get the overlay. It cannot create a standalone offense film row without a Madden backbone. Below FTN floors is pure Madden; absent final profile → L4 film neutral (1.0). | Prior-season NFL on-field performance (not current 2026 in-season); `WIRED` for QB/RB/WR/TE. |
| `pfrcoverage` — nflverse PFR advanced defense (`BuildCoverage`) | Selects season `2025` (`SeasonYear−1`) from the all-season CSV.GZ. Latest complete season but one behind the live 2026 season. | PFR ID→GSIS, then roster MFL→GSIS; positions gate CB/S only. No roster match rate is recorded. Rows below 10 targets, absent rating, non-CB/S or join miss are skipped. | Passer-rating-allowed is inverted/normalized to `[0,1]` and receives a 0.20 CB/S film seat. If absent, the seat is omitted and Madden/neutral seats remain; if no other profile film exists, L4 film is neutral. Empty feed/fetch failure is loud. | Prior-season on-field outcome, not current 2026; `WIRED` CB/S only. |
| `collegeshare` — CFBD offense stats (`BuildCollegeShare`) | `year=2026`, current in-progress college season as of Oct 3. It is the requested season, but live API row recency cannot be verified without network. | CFBD ESPN-style `playerId`→crosswalk ESPN→GSIS; roster MFL→GSIS; position lookup. No per-roster match rate is recorded. | WR/TE use receiving-yard share; RB uses 0.70 rushing + 0.30 receiving share. QB/K/defense absent here. `HasCollegeProductionShare=false` → rubric's `curve.SubSignal` neutral, not zero; share is an input to position `BreakoutEffective`. | College production (not current NFL production); prospect/dynasty asset; `WIRED` when CFBD key exists. |
| `collegedefense` — CFBD defensive stats (`BuildCollegeDefense`) | `year=2026`, current in-progress college season; data arrival/currentness on Oct 3 is not verified offline. | Same ESPN→GSIS and MFL→GSIS bridges. No per-roster match rate is recorded. | CB/S/LB/DT/DE get rubric-defined means of defensive within-team shares; offense/K absent. Missing share → `HasCollegeShare=false` and L4 neutral; share feeds `BreakoutEffective`. | College production (not current NFL production); prospect/dynasty asset; `WIRED` when CFBD key exists. |
| `agetrajectory` — nflverse `players.csv` DOB (`agetrajectory.Fetch`) | Full player file, no season argument or pinned release. DOB is stable but source snapshot age/version is not stored. | `players.csv` GSIS ID joins through roster MFL→GSIS; combined with the roster player's position and CFBD season shares to derive breakout age. No per-roster match rate recorded. | It supplies a birth date, not age or a current-production grade. Missing GSIS/DOB prevents breakout-age derivation; `HasBreakoutAge=false` is neutral. | DOB input to dynasty breakout-age; not on-field-now; `WIRED` behind CFBD key. |
| Derived `BreakoutAge` — `BuildBreakoutAge` + `BuildBreakoutAgeIDP` (not a separate fetcher) | Scans the six college seasons ending in `SeasonYear`: 2021–2026. As of Oct 3, the 2026 college season is in progress, so latest season may be partial; the API freshness cannot be verified offline. | CFBD ESPN ID→GSIS + roster MFL→GSIS, plus GSIS→DOB from `players.csv`, and MFL position. Offense WR/TE first receiving-yard share ≥0.20, RB rushing-yard share ≥0.20; IDP CB/S/LB/DT/DE first averaged defensive-share ≥0.12; age is calculated at Sept 1 of the crossing year. No roster match rate recorded. | The raw breakout age feeds each position's age curve into `BreakoutEffective`; no crossing, DOB or row → `HasBreakoutAge=false` and neutral. | College-career dominator timing / dynasty asset; `WIRED` when CFBD key exists. |
| `schooltier` — CFBD `/teams?year=2026` (`BuildSchoolTier`) | Requests the current 2026 CFBD team list. Whether every returned tier row is current on Oct 3 cannot be verified offline. | MFL players-DB college name→exact CFBD school string, then a small alias table. Assembler comment reports exact-string coverage ~94%, rising to ~98%+ for FBS skill-position players with aliases (verified against the July 20 CFBD list; not a live run coverage count). No all-roster rate. | Tier is normalized by composition for position and enters `BreakoutEffective`. Unknown/unmatched school → `SchoolUnset` and rubric neutral. Missing CFBD key skips all CFBD signals. | College competition pedigree / dynasty asset; `WIRED` behind CFBD key. |
| `kicking` — nflverse kicker season file (`kicking.Fetch`) | URL takes a season argument, but no production caller supplies any year. No runtime freshness to classify. | Fetcher data does not have a production assembly/join to a roster MFL ID or `PlayerSpec`; no match rate. | No kicker data reaches K L4. K's actual `MaddenFilm` and `NFLProduction` flags are both false, so `kicker.K.Apply` returns neutral film and fixed-neutral RAS/breakout. | Would be on-field-now kicking production if wired; `SCAFFOLD`. |
| `nflproduction` — nflverse player stats (`nflproduction.Fetch`) | URL takes a season argument, but no production caller selects a year; no currentness in this app path. | Raw GSIS player IDs could bridge by crosswalk, but no production assembler/roster join exists; no match rate. | In offense/IDP `applyScouting` reserves a 0.05 NFLProduction film seat at neutral 0.50; K's dedicated input remains unset. No fetched production reaches any `PlayerSpec`. | Would be on-field-now NFL production if wired; `SCAFFOLD`. |
| `touchshare` — nflverse snap counts (`touchshare.Fetch`) | URL takes a season argument, but no production caller selects a year. | Fetcher uses PFR player ID→GSIS bridge, but no production assembler and no population path to `PlayerSpec.TouchShare`; no match rate. | RB TouchShare remains absent; no active rubric input to receive it. | Would be on-field-now snaps/usage if wired; `SCAFFOLD`. |

The named eleven are `agetrajectory`, `collegeshare`, `collegedefense`, `crosswalk`, `kicking`, `madden`, `nflproduction`, `pfrcoverage`, `ras`, `touchshare`, `veteranfilm`. `schooltier` is a twelfth active source and RAS assembly is listed separately as requested. One additional fetcher, `pfrpassrush`, is not one of those eleven and has no non-test production caller; it is also scaffold. `pfrcoverage` is a different signal and is the only PFR defense feed assembled into M1.

**Where today's code merges the two measurables:** `rankings.applyScouting` sends recent FTN/PFR film-style NFL data together with RAS, college production, college breakout age, school tier, and current player age/contract context into position rubrics. Each rubric collapses film, RAS, and breakout-derived values to one `Layer4Output.Combined`; `engine.Pipeline.Score` then multiplies that value and L3 age decay into the MFL BasePoints proxy. There is no separate `on-field-now` or `dynasty-asset` output field. M2 receives the same `AdjustedScore`, then combines it with MFL all-play. This is the central architectural inversion detailed in Flow 5 and finding S2-F1.

Anchors:
> `if key := strings.TrimSpace(os.Getenv(cfbdEnvVar)); key != "" { mergeCFBDScouting(...) }` — `src/m1_scouting.go`.
>
> `The only populated EA slug is m24 (the 2023-season ratings...)` — `src/internal/ingestion/madden/fetcher.go`, package comment and `RatingsURL`.
>
> `SeasonSources(year - 1)` — `src/m1_scouting.go`, `mergeOffenseFilm`; `coverageSeason := strconv.Itoa(year - 1)` — same file, `mergeCoverage`.
>
> `profile.HasCollegeProductionShare` / `profile.HasBreakoutAge` are copied, but `MaddenFilm`, `NFLProduction`, and `TouchShare` are not — `src/internal/rankings/rankings.go`, `applyScouting`.
>
> `ScoutingAdjusted = BasePoints * AgePull * L4.Combined` — `src/internal/engine/pipeline.go`.

## 5. The number a user sees: `AdjustedScore` backwards through six layers — `PROXY`

M1's reachable calculation is `App.ScoreLeague` → MFL YTD score map → `rankings.Runner.Run` → `scorePlayer` → `composition.Assembler.Assemble` → `engine.Pipeline.Score` → `output.Writer.Write` → `season_scores` → `GetRankings` / `RankingsBoard` and `GetPlayerScore` / `InspectorContent`.

1. **L1 — data hygiene / identity:** MFL `players.Fetch` supplies MFL ID, position, DOB, draft year, and college through `App.directory`; state supplies roster and contract/cap values; scouting RAS arrives via MFL→GSIS crosswalk and combine. Missing player record, FLAG position, missing DOB, or implausible age excludes the player. Missing RAS gives L1/L6 the fallback 5.0; salary floor is 0. No L2 score is derived here.
2. **L2 — base points:** `App.basePoints` uses the compile-time league path year `ingestion.SeasonYear=2026`, but passes `scoreYear=a.season-1=2025` to `playerscores.Fetch` (`TYPE=playerScores`, `W=YTD`, `YEAR=2025`). On an offseason score this is the last completed 2025 season YTD total; on 2026-10-03 it still requests 2025 and ignores the in-progress 2026 YTD. It parses those MFL fantasy points directly, in league scoring, not the stored MFL scoring-rule matrix and not a projection. `playerscores` omits players with no scored record; in `scorePlayer`, `basePts, hasBase := r.base[p.MFLID]` leaves a missing map value at Go's `0`. Negative values are also floored to 0 and counted separately. A rookie/no-2025-contribution with no record therefore enters L2 as exactly 0.
3. **L3 — age value:** `rankings.scorePlayer` derives fractional age from MFL DOB at scoring `time.Now()`; `ApplyDecay` uses position peak limits hard-coded in composition and global `layer3.decay_rate` (default 0.03). DT's late-career RAS cushion uses constants 8.0/0.90, not the editable cushion parameters. This is a dynasty-age adjustment, not a second on-field score.
4. **L4 — real-player scouting:** `applyScouting` copies the per-player Profile's RAS, film, breakout age, school tier, and college share into `PlayerSpec` presence-gated fields; the registered position rubric collapses them to `FilmEffective`, `RASEffective`, `BreakoutEffective`, and `Combined`. The inputs are heterogeneous: FTN/PFR 2025 NFL performance, Madden 2023-edition grades, combine athletic potential, college production/pedigree/breakout, and age-derived signals. Missing sub-signals use Data-Parity-neutral rubric components. The active code does not populate NFLProduction, RB TouchShare, or kicker Madden/NFLProduction. There is no separate on-field-now/dynasty output.
5. **L5 — cap multiplier:** state ledger `CapSalary` in $M divided by active rulebook cap selects a cap tier; thresholds are parameter-backed but tier multipliers are fixed Cold=1.15, Neutral=1.00, Hot=0.85. This is a local contract/roster input, not an MFL-refresh result.
6. **L6 — tie-break:** `BuildTiebreaker` uses veteran status, cleaned RAS, and supplied scarcity rank. Composition supplies scarcity rank 0 for every position; on equal adjusted score, stored order falls through to stable MFL ID after veteran/RAS.

**Exact leverage and failure case:** `ScoutingAdjusted = BasePoints × AgePull × L4.Combined`; `AdjustedScore = ScoutingAdjusted × CapMultiplier`. With BasePoints=0, the final score is 0 regardless of a player's L3 age or L4 real-NFL measurable; scouting is only a multiplier on prior-season fantasy points, not a measurable that stands on its own. Players without a 2025 score therefore have no score response to scouting. The saved row stores one `combined` L4 multiplier and one `adjusted_score`; M1 does not emit separate on-field-now and dynasty-asset values. M2 consumes this same AdjustedScore. This is the highest-value architecture seam (S2-F1); Sector 3b measures its realized effect in both test-exercised score tables.

Anchors:
> `playerscores.Fetch(..., strconv.Itoa(a.season-1))` — `src/m1_app.go`, `basePoints`.
>
> `basePts, hasBase := r.base[p.MFLID]` — `src/internal/rankings/rankings.go`, `scorePlayer`; unset map keys return 0, with missing origin counted as `baseAbsent`.
>
> `ScoutingAdjusted = BasePoints * agePull * l4.Combined` — `src/internal/engine/pipeline.go`.
>
> `playerScores` and `YEAR=scoreYear` with `W=YTD` — `src/internal/ingestion/playerscores/fetcher.go`; the fetcher explicitly receives score year separately from path year.

## 6. M1 asset rankings: score, persist, read, render — `BROKEN` in the test-exercised release-path copy

- User action `RankingsBoard` → `useHarnessStore.scoreLeague` → Wails `ScoreLeague` → `App.ScoreLeague` → `App.directory` + `App.basePoints` + `App.buildScoutingDirectory` + `App.assembler` → `rankings.New` → `Runner.Run` loops current `state.Reader.Franchises/Roster` → `scorePlayer`/composition/engine → `output.Writer.Write` appends rows to `season_scores`, keyed `(season, scoring_config_id, mfl_id)`.
- Read: `App.GetRankings` reads `(a.season, ActiveVersion)` from output store and joins current state (salary/franchise) and cached players directory (name/position). `GetPlayerScore` reads a saved anatomy row. `store/harness.ts` stores results; `RankingsBoard` renders rank/player/position/franchise/base/adjusted/salary; selecting a row goes `store/inspector.ts` → `GetPlayerScore` → `InspectorContent` score bars.
- **Demonstrating input in the copied release-path state, not proof of actual live league season:** source `ingestion.SeasonYear` and `App.season` are 2026; `state.refreshSeason` derives 2027 from that DB's latest phase row. Its `rosters` has 831 rows for 2027, while `season_scores` has 827 rows stamped 2026/config 1 at `2026-07-20T14:03:21Z`, after the July 18 test-path rollover. On a current `ScoreLeague` call, the existing 827 rows trigger the skip guard **before** data is refetched or scores are rerun; `GetRankings` reads those 2026-stamped rows and joins display/ownership from 2027 state. If the current code had to perform a fresh pass against that same state with no existing output, it would use `a.season=2026` and ask for 2025 YTD while iterating the 2027 roster. This establishes an observable year/state/output mismatch in a test-exercised snapshot; it does **not** establish the real league's season or roster. Inventory §1c concludes neither copy is a trustworthy mirror.
- **No score invalidation after roster, params or override changes:** output schema is append-only and identifies scores only by `(season, scoring_config_id, mfl_id)`. State transaction commits do not delete or invalidate `season_scores`. `App.ScoreLeague` checks `len(existing)>0` for `(a.season, ActiveVersion)` **before** fetching scores or assembling profiles and returns `SkippedExisting`; `Runner.Run` repeats that any-row guard. It does not compare the saved MFL-ID set with the live roster or detect missing rows. `SetParam` does not change `scoring_config_id`; it refreshes only the sandbox/validation/admin values. `rulebook.SetOverride` also keeps the same active version. Only a new active rulebook version yields a different output key, but `Reload`/`Promote` have no App IPC/frontend path in this snapshot. Old rows remain frozen and no automatic rescore is triggered.

Anchors:
> `season is parsed once` — `src/app.go`; `refreshSeason` re-derives the state season from the latest phase row — `src/internal/store/state/season_phase.go`.
>
> `if existing, serr := a.output.Reader().Scores(...); serr == nil && len(existing) > 0 { ... SkippedExisting: true ... }` — `src/m1_app.go`, `App.ScoreLeague`.
>
> `if len(existing) > 0 { rep.SkippedExisting = true ... return rep, nil }` — `src/internal/rankings/rankings.go`, `Runner.Run`.
>
> `PRIMARY KEY (season, scoring_config_id, mfl_id)` and BEFORE UPDATE/DELETE triggers — `src/internal/output/output.go`.
>
> Read-only DB query: `SELECT season,scoring_config_id,count(*),min(created_at),max(created_at) FROM season_scores GROUP BY season,scoring_config_id;` → release-path `2026,1,827,2026-07-20T14:03:21Z,2026-07-20T14:03:21Z`; `SELECT season,count(*) FROM rosters GROUP BY season;` → release-path `2027,831`. These are test-exercised database values, not evidence about actual live league season.

## 7. M2 power rankings (standings / all-play source) — `BROKEN` in the test-exercised release-path copy

- `PowerRankingsBoard` → `store/harness.ts.loadPowerRankings` → `App.GetPowerRankings` reads M1 scores for `(a.season, active rulebook version)` and calls `standingsOrCache` → `leaguestandings.Fetch` requests `SeasonYear=2026`/league 14432; validated response is saved to `standings_cache`, or failed fetch falls back to last-known-good cached JSON with explicit stale `Freshness`.
- `m2service.Service.BuildBoard` reads the league’s current `state.Reader` and active rulebook starter count; joins each M1 score MFL ID to the **current** state franchise; aggregates all AdjustedScores or top-N; parses MFL all-play W/L/T into all-play win%; `powerrankings.Blend` standardizes scouting with median/MAD, performance with mean/std, applies caller weight (default 0.60), min-maxes display score; DTO joins MFL standings and franchise names.
- `PowerRankingsBoard` renders blended score, scouting z, all-play rate, MFL standings/points columns, freshness/phase, and a proxy banner. There is one aggregated M1 `AdjustedScore` input per franchise, not separate on-field-now and dynasty-value inputs; the default blend is 60% standardized M1 score and 40% MFL all-play performance (the caller weight is configurable). The M1 component inherits the prior-season MFL fantasy proxy and mixes NFL current-ish data with dynasty signals. In the test-exercised release-path DB, M1/output/MFL path year is 2026 while current state/phase is 2027; this demonstrates that copy's mismatch, not the actual league season. Neither database copy is a trusted live-state reference (S1 §1c).

Anchors:
> `the all-play component is the whole 40% of the blend` — `src/m2_app.go`; cache fallback path in `src/m2_standings_source.go`.
>
> `a player's owning franchise comes from runtime state` — `src/internal/m2service/m2service.go`, `buildBlendInputs`.
>
> `median + MAD` scouting and `mean/std` all-play — `src/internal/powerrankings/blend.go`.

## 8. League schedule and season calendar — `PARTIAL`

- MFL fantasy schedule `leagueschedule.Fetch` → boundary validation → `App.leagueScheduleOrCache` → JSON in `league_schedule_cache` on success / stale fallback on failure → `GetLeagueSchedule` resolves franchise names → `CalendarBoard` renders weekly matchup pane.
- Commissioner-authored events are a different source: `SCHEDULE_EVENT` / `RESCHEDULE_EVENT` / `CANCEL_EVENT` → sealed transaction request → `calendar_events` append-only rows → `state.CalendarEvents` latest row per logical event → `GetCalendarEvents` → `CalendarBoard` and `store/home.ts` → Home seasonal card/calendar list.
- Calendar blobs record opaque eventual-operation payloads, scheduled timestamp, and PLANNED/CANCELLED status. **No worker/scheduler executes due payloads; no current fire-now IPC/UI branch marks them FIRED.** Thus matchups and plan history render; scheduled acts do not execute at their timestamp.

Anchors:
> `It records intent only — the op runs when the event is fired, not now` — `src/frontend/src/components/calendar/CalendarBoard.tsx`.
>
> `no fire-now IPC yet` — same file header; backend request code appends calendar rows only — `src/internal/transactions/request_calendar.go`.

## 9. Transactions: every `Coordinator.Execute` operation — `PARTIAL`

Common path: mounted `TransactionWorkspace`, `TradeBuilder`, `LeagueControls`, or `CalendarBoard` → DTO → `App.buildRequest` / `buildMoneyRequest` (TAG/EXTENSION/SIGN use directory-aware verbs) → sealed `transactions.Request.validate` → `Coordinator.Execute` → one `state.Writer.WriteTx` → `gatePhase` first, roster-policy check where applicable, request handler, atomic commit/reload → `Receipt` → `App.receiptResult` / `TransactionResult` → frontend staged `PreviewTransaction` rollback quote, then re-sent intent through `ExecuteTransaction` for commit and reload. Preview follows the same gate + apply but returns the dry-run sentinel so the tx rolls back. It does not trust the preview result as execution input.

Every concrete request kind accepted by the coordinator is listed below. Table names are the state-side writes; preview rolls all of them back.

| Kind | Handler / state writes (transaction tables) | Frontend reachability |
|---|---|---|
| `TRADE` | `acquisitions.Trade` → `rosters`, `contracts`; `LogTradeNote` → `trade_notes` | `TradeBuilder` |
| `ROSTER_STATUS` | `acquisitions.SetStatus` → `rosters` | `TransactionWorkspace` |
| `WAIVER` | `deadcap.Waive` → delete `rosters`/`contracts`, append `player_status_events`, `dead_cap_ledger`, void `contract_years`, append `contract_year_changes` | `TransactionWorkspace` |
| `RESTRUCTURE` | `contracts.Restructure` → contract-year cells + change log, contract metadata, per-op count (`transaction_counts`) | `TransactionWorkspace` |
| `TAG` | `contracts.Tag` → salary ledger/change log + contract metadata/count | `TransactionWorkspace`; price resolved server-side |
| `EXTENSION` | `contracts.Extend` → future `contract_years` + change rows, contract metadata/count | `TransactionWorkspace`; floor resolved server-side |
| `BUYOUT` | release (`rosters`,`contracts`,`player_status_events`), `dead_cap_ledger`, void cells/change log, `transaction_counts` | `LeagueControls` / `TransactionWorkspace` |
| `ADVANCE_PHASE` | append `season_phases` | `LeagueControls` |
| `ROLLOVER_SEASON` | append `season_phases`; advance active roster/contract snapshot; release expired players with status events; annual ledgers retained | `LeagueControls` |
| `SET_SIGNING_WINDOW` | append same-phase `season_phases` row with meta | `LeagueControls` |
| `RETIREMENT` | release rows, `player_status_events`, `dead_cap_ledger`, void cells/change log | `LeagueControls` |
| `DEATH` | same release/status/cell path, zero `dead_cap_ledger` audit row | `LeagueControls` |
| `CAP_RELIEF` | append `cap_relief_ledger` | `LeagueControls` |
| `SIGN` | `SignContract` → `rosters`, `contracts`, `contract_years`, `contract_year_changes`; replaces prior cells | `LeagueControls` / `TransactionWorkspace` |
| `SET_TRADE_DEADLINE` | append same-phase `season_phases` row with deadline in `meta` | `LeagueControls` |
| `SCHEDULE_EVENT` | append `calendar_events` PLANNED | `CalendarBoard` |
| `RESCHEDULE_EVENT` | append `calendar_events` PLANNED head row; keeps history | `CalendarBoard` |
| `CANCEL_EVENT` | append `calendar_events` CANCELLED head row | `CalendarBoard` |
| `CORRECT` | append `transaction_corrections` | **No `buildRequest` branch or frontend action; request exists in the package but is not callable through App IPC** |

Anchors:
> `Every transaction runs inside state's spanning transaction` — `src/internal/transactions/coordinator.go`.
>
> `return errDryRun // fully applied and valid — roll it all back` — `Coordinator.Preview`.
>
> `KindCorrect` is declared and classified in `phasePolicy`, but `src/transactions_app.go:buildRequest` and `buildMoneyRequest` have no `CORRECT` case; `Correction.apply` is only reachable through a direct internal request.

## 10. Phase, rollover, free agency / UFA, tag, extension, restructure, buyout — `PARTIAL`

- Current phase is the latest `season_phases` transition. `state.refreshSeason` derives the current runtime season from that same phase log at startup and after commit. `GetCurrentPhase` → Home, M4 workspace, and commissioner controls. `ADVANCE_PHASE` may append any valid phase; `ROLLOVER_SEASON` is separately gated from PLAYOFFS and increments year to OFFSEASON.
- UFA signing requires OFFSEASON or REGULAR_SEASON plus an open commissioner signing window; `SET_SIGNING_WINDOW` records a same-phase directive. `SIGN` resolves draft year from MFL players, derives experience, checks current signable status / buyout lockout / hard-coded §6 floor, snaps salary, and creates a flat 1–4-year deal. The M4 free-agent pool comes from latest `FREE_AGENT` status events; initial state seed writes rostered players only, so an unrostered MFL player with no release/expiry event is not made signable by the seed. Full M5 free-agent intelligence / source pool is absent.
- `TAG` price = top five current salaries by position, floored at 120% of current salary; `EXTENSION` = 150% of highest remaining paid year, raised to a hard-coded position floor; `RESTRUCTURE` moves cap dollars between annual cells under hard-coded tier/season limits; `BUYOUT` is offseason-only, rate by 2/3/4 remaining years, capped at two per franchise/season, with dead cap + voided cells. All resolve money against committed ledger state inside one tx. `WAIVER`, `RETIREMENT`, `DEATH` use the same append-only status/dead-cap/cell-history pattern.
- `SET_TRADE_DEADLINE` is a commissioner-authored RFC3339 instant stored in `season_phases.meta`; trades are blocked after that instant. It is **not** the North Star’s automatic Week-9 rule derived from MFL week. The `TradeDeadlinePassed` check uses wall clock; absent an explicit setting, no deadline blocks trade.

Anchors:
> `The season int moves ONLY through this op` — `src/internal/transactions/request_season.go`, RolloverSeason.
>
> `the free-agency signing window is closed by the commissioner` — `src/internal/transactions/phase_gate.go`.
>
> `no directive ... → false (no block — the v1 default)` — `src/internal/store/state/trade_deadline.go`.

## 11. Activity feed — `PARTIAL` (readable after startup schema initialization; not a complete event log)

- `state.Store.Feed` unions `trade_notes`, `player_status_events`, `dead_cap_ledger`, `cap_relief_ledger`, and `contract_year_changes` in reverse-chronological order; `transactions_feed_app.GetFeed` adds latest-per-`tx_id` corrections, cached MFL player lookup, and rulebook franchise-name joins → `FeedBoard` summoned from the shell.
- On the release-path-named DB copy `trade_notes` and `transaction_corrections` are absent before current startup DDL. `state.Initialize` runs `initSchema` including `trade_notes_schema`/`correction` DDL before IPC is served; a raw direct `Feed` query against the uninitialized copy fails at `trade_notes`. Sector 3 migration test is still required to exercise the current initializer on copies.
- The projection is not every transaction: `ROSTER_STATUS` writes `rosters` only; phase transitions and signing-window/deadline directives write `season_phases`; calendar operations write `calendar_events`. Those writes do not appear in the UNION. `CORRECT` rows are readable if written, but no IPC can write them.

Anchors:
> `GetFeed reads ... (trade_notes, player_status_events, dead_cap_ledger, cap_relief_ledger, contract_year_changes)` — `src/transactions_feed_app.go`.
>
> `SetRosterStatus ... UPDATE rosters SET roster_status` — `src/internal/store/state/writes.go`.

## 12. Harness, validation board, inspector, home — `PARTIAL`

- **Rookie harness:** `App.ScoreRookies` → sample fixtures + real rubric registry + `composition.Assembler`/engine → `RookiesResult` → `store/harness.ts.loadAll` → `RookieTable` under Control. It is a sample/sandbox, not the 32-team league board.
- **Validation:** `RunValidationSuite` → `harness.RunValidationSuite(App.rubrics)` → 12 cases/summary → same harness store → `ValidationBoard` under Control. This executable architectural calibration suite is not M3 matchup prediction. North Star M3 prediction has no current matchup scoring/prediction implementation.
- **Inspector:** M1 click → `store/inspector.ts.select` → `GetPlayerScore` reads one persisted `season_scores` row + state and resolves names → `InspectorContent` renders layer bars, composite, contract/cap. It is a complete read-side projection of the stored score, but inherits the proxy and season-stamp defect.
- **Home:** `store/home.ts` calls `GetCurrentPhase` + `GetCalendarEvents`; `HomeBoard` renders a phase/calendar seasonal card and reads harness M1 rows already in memory. Activity, trade block, and chat cards are explicit engravings; comms summon is a placeholder. The home screen does not fetch standings or M1 on its own.
- **Module implementation boundary (North Star’s M1–M8 plus M9a/M9b):**

  | Element | Current reachable implementation |
  |---|---|
  | M1 Asset Rankings | Board + Inspector are reachable; scoring is proxy-backed. In the test-exercised release-path copy, output year and phase-derived roster season disagree (Flows 5–6); that is not evidence of the real league season. |
  | M2 Power Rankings | Reachable, but mixes the single proxy-backed M1 AdjustedScore aggregate with MFL all-play; the test-exercised release-path copy also has a 2026/2027 source-state split (Flow 7). |
  | M3 Matchup Predictions | `ABSENT`; ValidationBoard is harness cases, schedule pane is MFL matchup schedule only. |
  | M4 Transaction System | Coordinator + staged M4 UI are reachable; request/status/pool limitations are Flows 9–11. |
  | M5 Free Agency Intelligence | `ABSENT`; current M4 free-agent picker is status-event-derived, not an intelligence/pool module (Flow 10). |
  | M6 Rookie Draft Intelligence | `ABSENT`; RookieTable scores fixtures and is a sandbox, not draft rankings. |
  | M7 Trade Analyzer | `ABSENT`; TradeBuilder submits executable player moves and does not compute trade values. |
  | M8 Commissioner Dashboard | `PARTIAL`; LeagueControls exposes selected actions/settings, not the specified dashboard/health surface. |
  | M9a Engine Calibration UI | `PARTIAL`; AdminPanel edits five global param rows, but two cushion rows are inert and the North Star tunables are not covered (Flow 3). |
  | M9b Commissioner Rules UI | `PARTIAL` only for selected setting overrides; no current MFL Reload/change-review/Promote governance interface is mounted. |

Anchors:
> `ValidationBoard is Module 3` — `src/frontend/src/components/ValidationBoard.tsx` (label in UI code); North Star defines Module 3 as matchup predictions, which are absent.
>
> `Home reads LOCAL state only` — `src/frontend/src/store/home.ts`; no M1/M2 IPC in `load`.

## 13. Infrastructure: stamp, freshness, probe, lock, logs, DB/migrations — `WIRED` (static path)

- `src/version.go` link-time `version/commit/buildDate` → `App.AppInfo` → `store/appInfo.ts` → NavRail. Default `version="dev"`; `src/app.go:dbFileName` chooses `thewarroom-dev.db` for unstamped builds, `thewarroom.db` for stamped releases.
- `main.go:main` → `App.startup`: resolve OS config directory, set up rotating file log (stderr + file; nonfatal logging failure), acquire nonblocking advisory `.lock`, `db.Open` (single writer, read-only pool, WAL), parse compile-time `SeasonYear`, create MFL client, initialize params → rulebook → state → coordinator → output. `state.Initialize` ensures DDL, performs registered schema predicates/migrations, derives runtime season, seeds only if current-season state absent, loads memory. Shutdown closes pools and releases lock.
- `main.go` handles `-probe` before Wails and routes to `runProbe`: repeats store-floor initialization with per-step 20s timeout, no GUI. Probe still opens the DB selected by build stamp; it is not an immutable/read-only check.
- Freshness is explicit for M2 standings and league schedule (live success; stale cached fallback; fail if no cache). M1 persisted score reads report local data as live; the board still shows the proxy label. `setupLogging(dir)` creates `<os.UserConfigDir()>/TheWarRoom/logs/thewarroom-<UTC>.log` and sends the standard Go logger to `io.MultiWriter(os.Stderr,file)`; it is not configured to write nowhere. Source has no normal-success or routine scoring/transaction log calls: current calls are startup failure/logging warning, `-probe` step output, and one state ledger-drift warning. Therefore a zero-byte file is consistent with an ordinary launch that emitted no such warning/probe output, not proof that logger setup is broken. Ten copied files under `data/app-logs/` are all 0 bytes; Christopher's feedback reports seven 0-byte files in the original location, which this audit did not access (the brief forbids reading `~/.config`). Empty files cannot establish what happened on prior runs.

Anchors:
> `var (version = "dev" ...)` — `src/version.go`.
>
> `Single-instance guard BEFORE opening the DB` — `src/app.go`, `App.startup`.
>
> `probeStep` bounds each startup step — `src/probe.go`; `db.Open` verifies `journal_mode == "wal"` — `src/internal/db/pools.go`.

## System diagram as implemented

```text
CAVEAT: both local DB copies contain in-app test activity; neither establishes today's live league state.
main.go / Wails OnStartup
  └─ App.startup → build stamp → dev DB OR release DB → lock → SQLite pools/WAL
       ├─ params defaults/overrides ── cap tiers + L3 decay ─────────────────────────┐
       │                 └─ cushion params stored but unread                        │
       ├─ MFL league + rules → rulebook versions/overrides                           │
       │    ├─ cap + taxi/IR → local state cap → transactions/M1                     │
       │    ├─ roster limits → Coordinator gate                                      │
       │    ├─ starter count → M2 top-N                                             │
       │    └─ scoring matrix ──X no BasePoints consumer; contract floors hard-coded │
       ├─ MFL players ──> process-cached directory (not roster sync)                 │
       ├─ MFL rosters ──> normalize ──> state seed ONLY when empty                    │
       │                         └──X no subsequent MFL roster refresh               │
       │    local Coordinator operations ──> rosters/contracts/contract_years        │
       │       └─ ledgers/status/phase → M4 + partial Feed                           │
       │    current local roster + cap ────────────────────────────────────────┐     │
       ├─ MFL playerScores: path YEAR=2026, score YEAR=2025, W=YTD ──[PROXY]─┐  │     │
       │    missing per-player score → BasePoints=0                          │  │     │
       ├─ crosswalk + sources → scouting profiles                            │  │     │
       │    ├─ RAS / college / breakout / school tier → dynasty signals       │  │     │
       │    ├─ FTN 2025 + PFR coverage 2025 → prior-season NFL film/outcomes   │  │     │
       │    ├─ Madden m24 (2023-season) → stale skill-grade film              │  │     │
       │    ├─ CFBD branches only if CFBD_API_KEY                             │  │     │
       │    ├─ NFLProduction / kicking / touchshare ──X no caller              │  │     │
       │    └─ K Madden/NFLProduction ──X absent inputs, neutral K film        │  │     │
       └─ ScoreLeague → composition → L1 → L2 proxy → L3 age → L4 mixed inputs ┘  │
            L4.Combined = one multiplier (no on-field/dynasty outputs)              │
            ScoutingAdjusted = BasePoints × AgePull × L4.Combined                  │
            AdjustedScore = ScoutingAdjusted × CapMultiplier                       │
            └─> season_scores [append-only; key lacks params/roster/input versions] │
                 └─ any existing row → skip, no rescore after state/param changes   │
                 └─ GetRankings → harness Zustand → RankingsBoard / Inspector       │
       MFL standings (YEAR=2026) → cache/stale fallback → M2                       │
            M2 = default 60% M1 AdjustedScore aggregate + 40% MFL all-play          │
            no separate on-field-now/dynasty choice → PowerRankingsBoard            │
       MFL schedule → schedule cache → CalendarBoard                               │
       calendar intent → calendar_events ──X no due-event worker/fire               │
       Coordinator → transaction ledgers → partial Feed UNION → FeedBoard          │
            ├─ CORRECT handler exists but no App/request-builder caller              │
            └─ Feed omits roster-status + phase + calendar events                    │
       ScoreRookies / RunValidationSuite → Control sandbox/validation               │
            └─ NOT M3 matchup prediction; M5–M9 remain absent/partial                │

       Release-path DB evidence: state phase/rosters are 2027, but compile-time
       App/output/standings year is 2026. This is a test-exercised-copy observation,
       not a finding of the real league's current season (see INVENTORY §1c).
```
