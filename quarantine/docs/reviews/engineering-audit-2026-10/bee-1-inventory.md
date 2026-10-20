# Sector 1 — Inventory

All paths below are relative to this run directory. The project snapshot was read only. Database queries cover both immutable copies: `data/thewarroom.db` (the release-path-named copy) and `data/thewarroom-dev.db` (the dev-path-named copy). Both contain in-app test activity; neither is assumed to be a faithful live-league mirror. The run-directory app-log copies are ten zero-byte files (listed in §1c); there is no log content to infer from them.

## 1a. External inputs

MFL transport constructs `https://<host>.myfantasyleague.com/<year>/export?...`; `src/internal/mfl/client.go` quotes `q.Set("TYPE", endpoint)` and `q.Set("JSON", "1")`. League-scoped fetchers first call `DiscoverHost`, which requests `TYPE=league` with `L=<league id>`; API host discovery reads `league.baseURL`. `SeasonYear`/`LeagueID` are app constants in `src/internal/ingestion/`.

| Remote input | Fetcher and URL / TYPE quote | Identifier key / output identity | Production caller / reachability |
|---|---|---|---|
| MFL league metadata | `internal/ingestion/league.Fetch` → `call(..., "league", ...)`; `league.go`: `call(ctx, c, "league", year, leagueID)`; shared endpoint also used by discovery | `L` league ID; franchise IDs from export | Startup `App.initStoreFloor` → `rulebook.Initialize` → `league.APISource.Fetch`; reachable from `main.go` startup. |
| MFL scoring rules | Same fetcher; `league.go`: `call(ctx, c, "rules", year, leagueID)` | `L` league ID; scoring rule identifiers / values | Same startup path; combined with `league` export. |
| MFL rosters | `internal/ingestion/rosters.Fetch`; `fetcher.go`: `Type: "rosters"` | Franchise `id`, player `id` (MFL ID), roster status, salary | `App.rosterSeedSource.Rosters` → `rosters.Fetch`, only when `state.Initialize` finds no state rows for league+season. Reachable from startup; not a routine refresh. |
| MFL player database | `internal/ingestion/players.Fetch`; `fetcher.go`: `Type: "players"`, `DETAILS=1` | MFL `id`; facts include name, position, birthdate, draft year, college | `App.directory` → `players.Fetch`; reachable lazily from startup seeding, M1, M4, Inspector, feed, and transaction verbs. Cached once per process. |
| MFL player scores | `internal/ingestion/playerscores.Fetch`; `fetcher.go`: `Type: "playerScores"`, `W=YTD`, `YEAR=<score season>` | MFL player `id`; YTD score is raw string | `App.basePoints` → `ScoreLeague`; reachable from M1 scoring on explicit user action unless output already exists. |
| MFL salary adjustments | `internal/ingestion/salaryadjustments.Fetch`; `fetcher.go`: `Type: "salaryAdjustments"` | `franchise_id`, adjustment `id`; amount | No non-test caller found (`grep` for the package's `Fetch` has no production call). Fetcher and live test exist, but it does not enter current startup or transaction state. |
| MFL league standings | `internal/ingestion/leaguestandings.Fetch`; `fetcher.go`: `Type: "leagueStandings"` | Franchise `id`; standings report fields incl. all-play, PF/PA, Pwr | `App.standingsOrCache` → `GetPowerRankings`; reachable through M2 IPC; successful response cached in `standings_cache`. |
| MFL fantasy schedule | `internal/ingestion/leagueschedule.Fetch`; `fetcher.go`: `Params: map[string]string{"L": leagueID}` and endpoint `schedule` | Franchise IDs and MFL schedule week/matchup IDs | `App.leagueScheduleOrCache` → `GetLeagueSchedule`; reachable via Calendar summon; successful response cached in `league_schedule_cache`. |
| MFL NFL schedule | `internal/ingestion/schedule.Fetch`; `fetcher.go`: `Type: "nflSchedule"` | NFL team code and kickoff/game identifiers (not player IDs) | No production caller found; its live test is env-gated. Distinct from fantasy schedule above. |
| DynastyProcess player-ID crosswalk CSV | `internal/ingestion/crosswalk.Fetch`; `fetcher.go`: `SourceURL = "https://raw.githubusercontent.com/dynastyprocess/data/master/files/db_playerids.csv"` | `mfl_id`→`gsis_id`; optional `espn_id`, `pfr_id`, plus name+DOB join | `App.buildScoutingDirectory` in M1; reachable via ScoreLeague. One crosswalk fetch is shared by the active scouting assemblers. |
| nflverse combine / RAS CSV | `internal/ingestion/ras.Fetch`; `fetcher.go`: `SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/combine/combine.csv"` | CSV `pfr_id` bridged to GSIS; output keyed GSIS | `assembly.BuildRAS` → `App.buildScoutingDirectory` → ScoreLeague. |
| nflverse player bio CSV | `internal/ingestion/agetrajectory.Fetch`; `fetcher.go`: `SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/players/players.csv"` | `gsis_id`; extracts DOB | `mergeCFBDScouting` only, after `CFBD_API_KEY` is nonblank; then `assembly.BuildBreakoutAge` and `BuildBreakoutAgeIDP`. |
| CFBD team list | `internal/ingestion/schooltier.Fetch`; `fetcher.go`: `TeamsURL = "https://api.collegefootballdata.com/teams"` plus `?year=<year>` | MFL players-DB college string joins to CFBD school/team identity; returned tier is school-keyed | `mergeSchoolTier` in the CFBD-keyed block of M1; skipped if `CFBD_API_KEY` is unset. |
| CFBD player-season statistics (offense) | `internal/ingestion/collegeshare.Fetch`; `SeasonStatsURL = "https://api.collegefootballdata.com/stats/player/season"`; `cfbd.go` adds `?year=<year>&category=<category>` (rushing / receiving) | CFBD `playerId` (ESPN-style) → crosswalk `espn_id`→GSIS | `mergeCollegeShare` and multi-year `BuildBreakoutAge`; CFBD key required. |
| CFBD player-season statistics (defense) | `internal/ingestion/collegedefense.Fetch`; same quoted `SeasonStatsURL`, categories defensive and interceptions | CFBD `playerId` → ESPN→GSIS | `mergeCollegeDefense` and `BuildBreakoutAgeIDP`; CFBD key required. |
| EA Madden ratings | `internal/ingestion/madden.Fetch`; `fetcher.go`: `RatingsURL = "https://ratings-api.ea.com/v2/entities/m24-ratings"`; paginated `?limit=...&offset=...` | Madden name + birthdate through injected resolver → crosswalk's normalized name+DOB→GSIS | `assembly.BuildIDPFilm` and `BuildOffenseFilm`, reached from ScoreLeague unconditionally. Current URL is explicitly m24; no CFBD key required. |
| nflverse PFR advanced defense CSV.GZ — coverage | `internal/ingestion/pfrcoverage.Fetch`; `SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/pfr_advstats/advstats_season_def.csv.gz"` | `pfr_id`→crosswalk PFR→GSIS | `assembly.BuildCoverage` from ScoreLeague; CB/S coverage anchor. |
| nflverse PFR advanced defense CSV.GZ — pass rush | `internal/ingestion/pfrpassrush.Fetch`; same `SourceURL` constant and dataset | `pfr_id`→PFR→GSIS | No production call found (only package tests). It is a fetcher, not an active M1 signal. |
| nflverse player season stats CSV | `internal/ingestion/nflproduction.Fetch`; generated pattern `"https://github.com/nflverse/nflverse-data/releases/download/player_stats/player_stats_season_" + year + ".csv"` | `player_id` (GSIS), regular season | No production caller found; package tests only. |
| nflverse kicker season stats CSV | `internal/ingestion/kicking.Fetch`; generated pattern `"https://github.com/nflverse/nflverse-data/releases/download/player_stats/player_stats_kicking_season_" + year + ".csv"` | `player_id` (GSIS), regular season | No production caller found; package tests only. |
| nflverse snap counts CSV | `internal/ingestion/touchshare.Fetch`; generated pattern `"https://github.com/nflverse/nflverse-data/releases/download/snap_counts/snap_counts_" + year + ".csv"` | `pfr_player_id`→PFR→GSIS | No production caller found; package tests only. |
| nflverse FTN charting CSV | `internal/ingestion/veteranfilm.SeasonSources`; `fetcher.go`: `"https://github.com/nflverse/nflverse-data/releases/download/ftn_charting/ftn_charting_" + y + ".csv"` | GSIS identifiers on charted plays; receiver/passer roles | `assembly.BuildOffenseFilm` via M1; fetches completed prior season. |
| nflverse play-by-play CSV | Same `SeasonSources`; `"https://github.com/nflverse/nflverse-data/releases/download/pbp/play_by_play_" + y + ".csv"` | GSIS player IDs in play-by-play; receiver/passer roles | `assembly.BuildOffenseFilm` via M1; accompanies FTN charting. |

**Fetchers with no production caller:** MFL `salaryadjustments`, NFL `schedule`, plus `kicking`, `nflproduction`, `pfrpassrush`, `touchshare`. `grep -R` over non-test Go call sites found none; the packages' tests call their own functions. This is not a claim that all are useless: it records the current reachable call graph.

**Live-test gates (found by `grep -R TWR_LIVE_`):** `TWR_LIVE_MFL` gates `internal/mfl` smoke; ingestion `players`, `rosters`, `schedule`, `playerscores`, `salaryadjustments`, `league`, `leaguestandings` has its own coverage where defined; and normalize's live checks. `TWR_LIVE_CFBD` gates `schooltier`, `collegeshare`, `collegedefense` (also require `CFBD_API_KEY`). `TWR_LIVE_EA` gates `madden` (which also retrieves crosswalk data). `TWR_LIVE_NFLVERSE` gates `crosswalk`, `ras`, `agetrajectory`, `nflproduction`, `kicking`, `pfrcoverage`, `pfrpassrush`, `touchshare`, `veteranfilm`. No live gate was set or test run in this audit. Runtime M1 additionally checks `CFBD_API_KEY` in `src/m1_scouting.go`: absent means the CFBD signal block is skipped, not a failure.

## 1b. IPC surface

All 24 Go `*App` exported methods below are present in both generated files `src/frontend/wailsjs/go/main/App.js` and `App.d.ts`; comparison of method names found no binding without a Go method and no Go method without a binding. Callers means actual reachable call sites, not comment-only name matches. Table names are persistence access through the named store; `—` means no database table access.

| Method (signature quote; Go file) | Reads / writes | Generated JS binding | Frontend caller / status |
|---|---|---|---|
| `Ping()` — `src/app.go`: `func (a *App) Ping() PingResult` | DB pool health + journal mode; no table | Yes | No caller in `frontend/src` (available binding but unused). |
| `ScoreRookies()` — `src/harness_app.go`: `ScoreRookies() RookiesResult` | Reads effective params and active rulebook/cap through composition; sample fixture only; no writes | Yes | `store/harness.ts` → App startup `loadAll`; `RookieTable` under Control. |
| `RunValidationSuite()` — `src/harness_app.go`: `RunValidationSuite() ValidationResult` | In-memory fixtures / rubrics; no table | Yes | `store/harness.ts` → App startup; `ValidationBoard` under Control. |
| `GetParams()` — `src/harness_app.go`: `GetParams() ParamsResult` | `param_defaults`, `param_overrides` via params store | Yes | `store/harness.ts` → AdminPanel under Control. |
| `SetParam(key string, value float64)` — `src/harness_app.go` | Writes `param_overrides` | Yes | `store/harness.ts` → AdminPanel; then reloads harness data. |
| `GetLeagueSchedule()` — `src/leagueschedule_app.go`: `GetLeagueSchedule() LeagueScheduleResult` | Reads/writes `league_schedule_cache`; MFL schedule fetch | Yes | `CalendarBoard` (summoned from shell); visible matchup pane. |
| `ScoreLeague()` — `src/m1_app.go`: `ScoreLeague() ScoreLeagueResult` | Reads rulebook, state, params, directory; writes `season_scores` unless already scored | Yes | `store/harness.ts` → `RankingsBoard` user action. |
| `GetRankings()` — `src/m1_app.go`: `GetRankings() RankingsResult` | Reads `season_scores` plus state; players-DB lookup is display enrichment | Yes | `store/harness.ts` → `RankingsBoard` (and retained board data may be used by Home). |
| `GetPlayerScore(mflID string)` — `src/m1_player_score_app.go` | Reads one `season_scores` row + state and players-DB directory | Yes | `store/inspector.ts` → `InspectorContent`, selected by M1 row. |
| `GetPowerRankings(weight float64, aggMode string)` — `src/m2_app.go` | Reads `season_scores`, rulebook, state; reads/writes `standings_cache`; MFL standings | Yes | `store/harness.ts` → `PowerRankingsBoard`. |
| `GetRoster(franchiseID string)` — `src/m4_app.go` | Reads current state (`rosters`, `contracts`, `contract_years`, ledgers as state loads); directory enrichment | Yes | `store/transactions.ts` → `TransactionWorkspace` / `TradeBuilder`. |
| `GetFreeAgentPool()` — `src/m4_app.go` | Reads latest `player_status_events` plus `rosters`; directory enrichment | Yes | `store/transactions.ts` → `TransactionWorkspace`. |
| `GetFranchises()` — `src/m4_app.go` | Reads state/roster view + active rulebook franchise names | Yes | `store/transactions.ts` → M4 transaction and trade views. |
| `GetLegalOps()` — `src/m4_app.go` | Reads `season_phases` then applies `transactions.LegalOps`; no writes | Yes | `store/transactions.ts` → transaction/trade surfaces. |
| `PreviewTransaction(req TransactionRequest)` — `src/m4_app.go` | Routes applicable state/store writes inside a rollback-only coordinator transaction; no durable write. Reads state/rulebook/player directory. | Yes | Direct callers: `TransactionWorkspace`, `TradeBuilder`, `LeagueControls`, `CalendarBoard`; staged confirmation. |
| `GetLeagueSetting(key string)` — `src/rulebook_app.go` | Reads active rulebook + `rulebook_overrides`; no writes | Yes | `LeagueControls` (taxi / IR controls). |
| `SetLeagueSettingOverride(key, value, note string)` — `src/rulebook_app.go` | Writes `rulebook_overrides` directly (admin path, not Coordinator) | Yes | `LeagueControls`. |
| `ExecuteTransaction(req TransactionRequest)` — `src/transactions_app.go` | Coordinator commits applicable state writes across `rosters`, `contracts`, `contract_years`, `contract_year_changes`, `dead_cap_ledger`, `cap_relief_ledger`, `player_status_events`, `season_phases`, `calendar_events`, `trade_notes`, `transaction_counts`; exact op-specific writes in Sector 2 | Yes | Direct callers: `TransactionWorkspace`, `TradeBuilder`, `LeagueControls`, `CalendarBoard`. |
| `GetFranchiseState(franchiseID string)` — `src/transactions_app.go` | Reads in-memory state loaded from `rosters`/`contracts`/ledger tables | Yes | No actual caller in `frontend/src` (legacy/dev surface). |
| `GetCurrentPhase()` — `src/transactions_app.go` | Reads `season_phases` | Yes | `store/home.ts` and `store/transactions.ts`. |
| `GetFreeAgents()` — `src/transactions_app.go` | Reads latest `player_status_events` and `rosters` | Yes | No actual caller in `frontend/src`; M4 uses named `GetFreeAgentPool` instead. |
| `GetCalendarEvents()` — `src/transactions_calendar_app.go` | Reads `calendar_events` | Yes | `store/home.ts` and `CalendarBoard`. |
| `GetFeed()` — `src/transactions_feed_app.go` | Reads `trade_notes`, `player_status_events`, `dead_cap_ledger`, `cap_relief_ledger`, `contract_year_changes`, then `transaction_corrections` | Yes | `FeedBoard` (summoned from shell). |
| `AppInfo()` — `src/version.go` | Link-time version / commit / build date; no table | Yes | `store/appInfo.ts` → `NavRail` build stamp. |

**IPC mismatches:** three exposed Go+binding methods have no frontend invocation: `Ping`, `GetFranchiseState`, `GetFreeAgents` (the last is superseded in UI by `GetFreeAgentPool`). No binding/Go-surface mismatch exists. The README-level comment in `SYSTEM_MAP.md` claiming only `App.Ping()` is currently IPC is stale; see S1-F1.

## 1c. Persistence

### DDL inventory versus the immutable database copy

DDL table names were obtained from all non-test `CREATE TABLE` statements in `src/internal/store`, `src/internal/output`, and `src/internal/db`. SQL below was run against both immutable DB copies using Python `sqlite3` read-only URIs and `sqlite_master`/`PRAGMA table_info`. The main table gives code anchor and release-path copy values; the comparison immediately below gives both databases side by side. `newest` is `MAX` of available timestamp columns (`created_at`, `last_updated`, `changed_at`, `updated_at`, `as_of`, `at`, `applied_at`, `fetched_at`); `—` means table absent or timestamp column has no values.

| Table | DDL anchor (quoted) | Writes / reads | DB rows; newest timestamp |
|---|---|---|---|
| `rosters` | `src/internal/store/state/schema.go`: `CREATE TABLE IF NOT EXISTS rosters (` | state seed + Coordinator state writer / state reader, transaction ops, M1 & M4 | 831; `as_of=2026-07-18T22:34:00Z` |
| `contracts` | `state/schema.go`: `CREATE TABLE IF NOT EXISTS contracts (` | state seed + Coordinator / state loader, migration and transaction logic | 831; `last_updated=2026-07-18T22:34:00Z` |
| `dead_cap_ledger` | `state/schema.go`: `CREATE TABLE IF NOT EXISTS dead_cap_ledger (` | Coordinator/deadcap / state cap derivation, ledger ops, feed | 6; `created_at=2026-07-12T02:07:34Z` |
| `transaction_counts` | `state/schema.go`: `CREATE TABLE IF NOT EXISTS transaction_counts (` | Coordinator state writer / transaction limit check | 9; no timestamp column |
| `contract_years` | `state/ledger_schema.go`: `CREATE TABLE IF NOT EXISTS contract_years (` | state seed and transaction ledger writers / state cap calculation, rollover, contract ops | 3,865; `last_updated=2026-07-12T02:08:09Z` |
| `contract_year_changes` | `state/ledger_schema.go`: `CREATE TABLE IF NOT EXISTS contract_year_changes (` | contract/ledger transaction writes / activity feed | 3,893; `changed_at=2026-07-12T02:08:09Z` |
| `season_phases` | `state/season_phase.go`: `CREATE TABLE IF NOT EXISTS season_phases (` | Coordinator phase, rollover, deadline/window writers / phase readers and transaction gates | 13; `at=2026-07-18T22:35:50Z` |
| `cap_relief_ledger` | `state/cap_relief.go`: `CREATE TABLE IF NOT EXISTS cap_relief_ledger (` | Coordinator cap-relief writer / cap usage, feed | 0; `created_at=NULL` |
| `player_status_events` | `state/player_status.go`: `CREATE TABLE IF NOT EXISTS player_status_events (` | Coordinator status/sign/cut/special-event writes / latest-status/free-agent readers and feed | 397; `at=2026-07-18T22:35:30Z` |
| `calendar_events` | `state/calendar.go`: `CREATE TABLE IF NOT EXISTS calendar_events (` | Coordinator schedule/reschedule/cancel / calendar board and home | absent; startup DDL should create it |
| `standings_cache` | `state/standings_cache.go`: `CREATE TABLE IF NOT EXISTS standings_cache (` | App M2 cache write / M2 fallback read | absent; startup DDL should create it |
| `league_schedule_cache` | `state/league_schedule_cache.go`: `CREATE TABLE IF NOT EXISTS league_schedule_cache (` | App schedule cache write / schedule fallback read | absent; startup DDL should create it |
| `trade_notes` | `state/trade_notes_schema.go`: `CREATE TABLE IF NOT EXISTS trade_notes (` | Coordinator trade-note write / feed UNION | absent; code `Feed` reads it, so direct feed query against this un-migrated snapshot fails `no such table` until state initialization runs its DDL |
| `transaction_corrections` | `state/correction.go`: `CREATE TABLE IF NOT EXISTS transaction_corrections (` | correction Coordinator writer / feed correction projection | absent; startup DDL should create it |
| `rulebook_versions` | `store/rulebook/rulebook.go`: `CREATE TABLE IF NOT EXISTS rulebook_versions (` | rulebook initialize/reload / active config readers | 1; `created_at=2026-07-05T19:33:17Z` |
| `rulebook_active` | `rulebook/rulebook.go`: `CREATE TABLE IF NOT EXISTS rulebook_active (` | rulebook initialize/promote / all active-version consumers | 1; no timestamp column |
| `rulebook_overrides` | `rulebook/rulebook.go`: `CREATE TABLE IF NOT EXISTS rulebook_overrides (` | rulebook admin override / effective setting/config reads | 0; `created_at=NULL` |
| `param_defaults` | `store/params/writes.go`: `CREATE TABLE IF NOT EXISTS param_defaults (` | params seed-once / parameter definition/effective-value reads | 5; no timestamp column |
| `param_overrides` | `params/writes.go`: `CREATE TABLE IF NOT EXISTS param_overrides (` | admin/harness override / effective parameter reads | 0; `updated_at=NULL` |
| `season_scores` | `internal/output/output.go`: `CREATE TABLE IF NOT EXISTS season_scores (` | output writer via rankings Runner / M1 boards, Inspector, M2 | 827; `created_at=2026-07-20T14:03:21Z` |
| `schema_migrations` | `state/migrations.go`: `CREATE TABLE IF NOT EXISTS schema_migrations (` | state migration runner / state migration runner | 2; `applied_at=2026-07-24 11:07:34` |

**Code creates 21 application tables; the release-path-named DB has 16 non-internal tables and the dev-path-named DB has all 21 (20 data tables plus `schema_migrations`).** Five current-code tables are absent from the release-path copy: `calendar_events`, `standings_cache`, `league_schedule_cache`, `trade_notes`, `transaction_corrections`; all five exist in dev. Neither DB contains an application table absent from current code. `sqlite_sequence` is SQLite's internal autoincrement table and is excluded. Counts derive from `sqlite_master` and per-table `COUNT(*)`, not an estimate.

**Both-DB table/count/timestamp comparison** (table set is the union of code DDL and both DB schemas):

| Table | `thewarroom.db` rows; newest timestamp | `thewarroom-dev.db` rows; newest timestamp |
|---|---:|---:|
| `rosters` | 831; `as_of=2026-07-18T22:34:00Z` | 1,375; `as_of=2026-07-25T22:42:18Z` |
| `contracts` | 831; `last_updated=2026-07-18T22:34:00Z` | 1,375; `last_updated=2026-07-25T22:42:18Z` |
| `dead_cap_ledger` | 6; `created_at=2026-07-12T02:07:34Z` | 3; `created_at=2026-07-28T00:48:06Z` |
| `transaction_counts` | 9; no timestamp column | 0; no timestamp column |
| `contract_years` | 3,865; `last_updated=2026-07-12T02:08:09Z` | 4,525; `last_updated=2026-07-28T00:48:06Z` |
| `contract_year_changes` | 3,893; `changed_at=2026-07-12T02:08:09Z` | 4,532; `changed_at=2026-07-28T00:48:06Z` |
| `season_phases` | 13; `at=2026-07-18T22:35:50Z` | 4; `at=2026-07-27T00:31:59Z` |
| `cap_relief_ledger` | 0; `created_at=NULL` | 0; `created_at=NULL` |
| `player_status_events` | 397; `at=2026-07-18T22:35:30Z` | 3; `at=2026-07-28T00:48:06Z` |
| `calendar_events` | absent | 3; `created_at=2026-07-25T10:17:36Z` |
| `standings_cache` | absent | 1; `fetched_at=2026-07-28T00:48:55Z` |
| `league_schedule_cache` | absent | 1; `fetched_at=2026-07-27T00:40:52Z` |
| `trade_notes` | absent | 1; `created_at=2026-07-25T22:42:18Z` |
| `transaction_corrections` | absent | 0; `created_at=NULL` |
| `rulebook_versions` | 1; `created_at=2026-07-05T19:33:17Z` | 1; `created_at=2026-07-24T21:58:44Z` |
| `rulebook_active` | 1; no timestamp column | 1; no timestamp column |
| `rulebook_overrides` | 0; `created_at=NULL` | 0; `created_at=NULL` |
| `param_defaults` | 5; no timestamp column | 5; no timestamp column |
| `param_overrides` | 0; `updated_at=NULL` | 0; `updated_at=NULL` |
| `season_scores` | 827; `created_at=2026-07-20T14:03:21Z` | 1,299; `created_at=2026-07-25T10:52:15Z` |
| `schema_migrations` | 2; `applied_at=2026-07-24 11:07:34` | 2; `applied_at=2026-07-24 21:58:44` |

**Why roster counts differ (query evidence):**

```sql
SELECT league_id, season, count(*) AS rows,
       count(DISTINCT franchise_id) AS franchises,
       count(DISTINCT mfl_id) AS players
FROM rosters GROUP BY league_id, season;
```

`thewarroom.db` → `('14432', 2027, 831, 32, 831)`; dev → `('14432', 2026, 1375, 32, 1375)`. Both have zero duplicate `(league_id,season,mfl_id)` groups. Release-path roster statuses are 794 `ROSTER` + 37 `TAXI_SQUAD`; dev statuses 1,314 `ROSTER` + 61 `TAXI_SQUAD`. These are not comparable live-season observations: the release-path copy's phase log ends at a 2027 rollover, and the dev copy ends in 2026. Both databases contain in-app transaction activity (provenance and rollover analysis below); the season values therefore establish test-exercised DB state, not the real league's current season or roster.

### Seed provenance, test operations, and rollover (both copies)

The state seed's ledger writer explicitly tags its change rows `source='seed'` and reason `seed: flat-fill from MFL annual salary + expiration` (`src/internal/store/state/ledger.go`, `insertCell`). Transaction paths append later ledger changes with `source='op'`, `source='extension'`, or `source='signing'`; the rows do not contain an explicit test marker, so their test-exercise interpretation is supported by the test-like operation/phase sequence, not by a dedicated `is_test` column.

Read-only query:

```sql
SELECT source, count(*) AS change_rows, count(DISTINCT mfl_id) AS players
FROM contract_year_changes WHERE league_id='14432' GROUP BY source;
```

| DB copy | `source` | Change rows | Distinct players |
|---|---|---:|---:|
| `thewarroom.db` | `seed` | 3,859 | 1,232 |
| | `op` | 34 | 13 |
| `thewarroom-dev.db` | `seed` | 4,522 | 1,377 |
| | `op` | 10 | 3 |

`seed` is the ledger state originally derived from MFL roster salary/expiration facts. `op` rows are in-app transactions, not imported MFL updates. The additional signing-side categories are visible in `contract_years`: release-path has 8 `extension` rows (6 PAID + 2 UFA); dev has 5 `signing` rows (4 PAID + 1 UFA). `transaction_counts` has nine release-path rows totaling 9 operations: BUYOUT=1, EXTENSION=2, RESTRUCTURE=3, TAG=3. Dev has no `transaction_counts` rows. That table is a limit-counter for selected op kinds, not a comprehensive transaction log.

`player_status_events` groups by `(status,reason)`:

| DB copy | Status/reason | Events (distinct player IDs) |
|---|---|---:|
| `thewarroom.db` | `FREE_AGENT` / `ufa-expiry §14` | 395 (395) |
| | `FREE_AGENT` / `buyout §12` | 1 (1) |
| | `FREE_AGENT` / `waiver-cut §8` | 1 (1) |
| `thewarroom-dev.db` | `FREE_AGENT` / `waiver-cut §8` | 3 (3) |

`dead_cap_ledger` is transaction-written, not a seed source. Release-path reasons/counts are waiver=3, buyout=1, `gaines-adams §13`=1, retirement=1 (6 total); dev has three `waiver-cut §8` rows. The `contract_year_changes` `op` reasons additionally enumerate release-path extensions, restructures, tags, buyout, waiver, retirement, and death/`gaines-adams`; dev enumerates waiver and free-agency signing (including closing prior cells). Therefore neither DB is an untouched first MFL seed.

The row-count difference is **not** all explained by rollover. The test-exercised release-path phase log contains phase changes seconds apart on July 11 and July 18, then `PLAYOFFS→OFFSEASON` at season 2027 on `2026-07-18T22:35:30Z`. In `src/internal/store/state/season_phase.go`, `RolloverSeason` calls `promoteExpiredContracts(next)` before advancing the snapshots. The exact expiration predicate is not the mutable `contracts.expiration_year` column: `readExpiredRosterIDs` selects roster IDs for which there is **no** `contract_years` row with `year_status='PAID' AND league_year >= next`. It then records `FREE_AGENT`/`ufa-expiry §14` and releases the player.

Query comparing seed IDs, current roster IDs, and those expiry events:

```sql
WITH seed_ids AS (
  SELECT DISTINCT mfl_id FROM contract_year_changes
  WHERE league_id='14432' AND source='seed'
),
roster_ids AS (
  SELECT DISTINCT mfl_id FROM rosters WHERE league_id='14432'
),
exp_ids AS (
  SELECT DISTINCT mfl_id FROM player_status_events
  WHERE league_id='14432' AND reason='ufa-expiry §14'
)
SELECT (SELECT count(*) FROM seed_ids),
       (SELECT count(*) FROM roster_ids),
       (SELECT count(*) FROM exp_ids),
       (SELECT count(*) FROM seed_ids s JOIN exp_ids e USING(mfl_id)),
       (SELECT count(*) FROM seed_ids s JOIN roster_ids r USING(mfl_id)),
       (SELECT count(*) FROM exp_ids e JOIN roster_ids r USING(mfl_id)),
       (SELECT count(*) FROM seed_ids s LEFT JOIN exp_ids e USING(mfl_id)
        LEFT JOIN roster_ids r USING(mfl_id)
        WHERE e.mfl_id IS NULL AND r.mfl_id IS NULL);
```

Output: release-path `(1232, 831, 395, 395, 831, 0, 6)`; dev `(1377, 1375, 0, 0, 1375, 0, 2)`. A second query against all 395 release-path `ufa-expiry §14` IDs found all 395 are not currently rostered, have no PAID cell at or after 2027, and have a maximum PAID `league_year` below 2027. The 831 surviving 2027 contracts all have a PAID cell at/after 2027 and `expiration_year >= 2027`. This matches the rollover implementation's rule.

Arithmetic: the cross-copy roster gap is `1375 − 831 = 544`. Of it, 395 release-path players are directly evidenced as 2027 rollover expiries (72.6%). The seed-ID populations themselves differ by `1377 − 1232 = 145`; the remaining net difference is 4, because 6 release-path seed IDs and 2 dev seed IDs are not in their respective current roster sets and are not in the release-path expiry set. The DBs do not preserve the original `rosters` rows as a roster history, so the remaining 149 cannot be assigned player-by-player to a shared MFL seed cohort from these copies. `contracts.expiration_year` confirms the release-path survivors are years 2027–2031; dev currently has 384 contracts whose expiration year is before the *next* season 2027, which the current 2026 DB has not rolled into. Thus rollover explains most, not all, of the observed roster count difference.

**Conclusion:** neither DB currently establishes a faithful mirror of the real league as of its last MFL seed. The release-path-named DB has test-like phase-gate activity, an in-app 2027 rollover, 395 test-path UFA expiry events, and other transaction writes; the dev-named DB has in-app waivers/signings and phase-gate changes. The MFL seed history is not a refresh feed, and these copies contain no post-seed MFL roster reconciliation. They are not trustworthy current league-state inputs without a fresh MFL export and explicit reconciliation. This is **HIGH**: there is no evidence-backed database copy that can safely stand in for the live league today.

**Which DB today's code opens:** `src/version.go` defaults to `version = "dev"`; `src/app.go` states “A DEV build uses a SEPARATE `-dev` database” and `dbFileName` returns `thewarroom-dev.db` when `devBuild` is true. So an un-stamped build of this source opens `thewarroom-dev.db`; a stamped release (`version != "dev"`) opens `thewarroom.db`. The code itself does not uniquely identify an installed binary's link-time stamp; its `AppInfo()` reports it. The filename identifies the code path, not whether the DB contents are faithful.

DB parameter rows, queried with `SELECT param_key,position,default_val,min_val,max_val FROM param_defaults ORDER BY param_key,position;`: `captier.cold_ceiling_pct=1.2`, `captier.hot_floor_pct=4.8`, `cushion_guard.ras_threshold=8`, `cushion_guard.reduction=0.1`, `layer3.decay_rate=0.03` (all global position `''`). No overrides exist.

### Migration markers

`schema_migrations` query output:

```text
owner  version  applied_at           method
state  1        2026-07-24 11:07:34  reconciled
state  2        2026-07-24 11:07:34  reconciled
```

`src/internal/store/state/migrations.go` quotes: `method distinguishes a migration we actually RAN ('migrated') from one we found already in its target state ... and merely stamped ('reconciled')`; registry v1 is `migrateMoneyCents`, v2 `dropLegacyMoneyColumns`. Therefore these rows say the DB already satisfied both migration predicates when current tracking stamped them; they do not claim the migration transformations were executed during that launch. Five later-created feature tables have no separate migration marker: `state.initSchema` calls their `CREATE TABLE IF NOT EXISTS` initializers before `runMigrations`.

### App-log files

`find data/app-logs -type f -printf '%P %s bytes'` returned ten files, all 0 bytes: timestamps in names range 2026-07-25 through 2026-07-28. They contain no errors, warnings, migrations, timing records, or other messages. Any claims about last-run activity from the logs are **could not establish**; empty files settle only that the copied logs are blank.

## 1d. Frontend stores and components

All five Zustand store files are imported by a component or shell path and reachable from `App.tsx`:

| Zustand store | IPC actions | Usage / reachability |
|---|---|---|
| `src/frontend/src/store/appInfo.ts` | `AppInfo` | `NavRail` BuildStamp; `App.tsx` loads at startup. |
| `store/harness.ts` | `ScoreRookies`, `RunValidationSuite`, `GetParams`, `SetParam`, `GetRankings`, `GetPowerRankings`, `ScoreLeague` | App startup `loadAll`; M1, M2, Admin, Rookie Sandbox, Validation boards. |
| `store/home.ts` | `GetCurrentPhase`, `GetCalendarEvents` | `HomeBoard`, mounted in Home module. |
| `store/inspector.ts` | `GetPlayerScore` | M1 row selection → InspectorContent overlay. |
| `store/transactions.ts` | `GetFranchises`, `GetCurrentPhase`, `GetLegalOps`, `GetRoster`, `GetFreeAgentPool` | M4 transaction workspace / trade builder / controls; each selected tab mounts one flow. |

`App.tsx` is mounted by `frontend/src/main.tsx`, calls harness + version stores at startup, and chooses 6 nav modules (`home`, `assets`, `pulse`, `txn`, `trade`, `control`). Under the control module its tabs mount League Controls, Engine Admin, Rookie Sandbox, and Architectural Tests. Calendar and feed are conditional shell summons, not orphan files. Components/direct file status:

| Component / file | IPC or store | App reachability / visible surface |
|---|---|---|
| `components/AdminPanel.tsx` | `useHarnessStore` | Control → Engine Admin tab. |
| `components/board/freshness.ts` | None (pure helper) | Imported by `board/primitives.tsx`; reachable. |
| `components/board/keys.ts` | None (keyboard hook; selection callback supplied by RankingsBoard) | Imported by `RankingsBoard`; reachable. |
| `components/board/primitives.tsx` | None | Shared Engrave/Skeleton/Freshness/phase/delta elements imported by boards, calendar, home and other components; reachable. |
| `components/calendar/CalendarBoard.tsx` | Direct `GetCalendarEvents`, `GetLeagueSchedule`, `PreviewTransaction`, `ExecuteTransaction` | `App.tsx` calendar summon; visible only when summoned. |
| `components/feed/FeedBoard.tsx` | Direct `GetFeed` | `App.tsx` feed summon; visible only when summoned. |
| `components/home/HomeBoard.tsx` | `useHomeStore`, `useHarnessStore` | Home nav module. |
| `components/home/SeasonalCard.tsx` | None | Imported/rendered conditionally by HomeBoard; reachable. |
| `components/home/seasonal.ts` | None | Pure seasonal selector used by HomeBoard/Card; reachable. |
| `components/inspector/InspectorContent.tsx` | `useInspectorStore` | AppShell Inspector child when an M1 player is selected. |
| `components/PowerRankingsBoard.tsx` | `useHarnessStore` | M2 Pulse nav module. |
| `components/RankingsBoard.tsx` | harness + inspector stores | M1 Assets nav module. |
| `components/RookieTable.tsx` | `useHarnessStore` | Control → Rookie Sandbox tab. |
| `components/shell/AppShell.tsx` | None | Root shell in `App.tsx`; mounts NavRail, Workspace, CommsStrip, Inspector. |
| `components/shell/CommsStrip.tsx` | None | Always mounted shell strip; its comms summon is a placeholder, not an IPC flow. |
| `components/shell/Inspector.tsx` | None | Always mounted overlay container; body conditional on selection. |
| `components/shell/NavRail.tsx` | appInfo store | Always mounted navigation + build stamp. |
| `components/shell/Workspace.tsx` | None | Always mounted active module region. |
| `components/shell/types.ts` | None | Module/density types and nav definitions used by App/Shell; reachable. |
| `components/shell/useDensity.ts` | None | Hook imported by App; reachable. |
| `components/transactions/ConfirmModal.tsx` | Receives submit callbacks; no direct IPC | Nested staged confirmation component in mounted M4 flows. |
| `components/transactions/format.tsx` | None | Shared display formatting imported by transaction surfaces; reachable. |
| `components/transactions/LeagueControls.tsx` | direct league setting + preview/execute IPC; transactions store | Control tab and commissioner workspace paths; rendered from App control module. |
| `components/transactions/TradeBuilder.tsx` | direct preview/execute; transactions store | Trade nav module. |
| `components/transactions/TransactionWorkspace.tsx` | direct preview/execute; transactions store | Transaction nav module. |
| `components/ValidationBoard.tsx` | `useHarnessStore` | Control → Architectural Tests tab. |

No component/store file under `frontend/src/components` is orphaned by static import tracing. `Ping`/`GetFranchiseState`/`GetFreeAgents` are unused bindings as noted in §1b, not unmounted components. `GetFreeAgents` is superseded by M4's name-enriched pool call.

## Sector 1 findings

| ID | Severity | Status | Evidence / finding |
|---|---|---|---|
| S1-F1 | MEDIUM | `DRIFT` | `src/SYSTEM_MAP.md` says “Currently: `App.Ping()`” under IPC; root App files expose 24 methods and `App.d.ts` binds 24. The public-facing surface inventory is stale. |
| S1-F2 | MEDIUM | `SCAFFOLD` | Six implemented external fetchers have no production call site: `salaryadjustments`, NFL `schedule`, `kicking`, `nflproduction`, `pfrpassrush`, `touchshare`. Evidence: fetcher functions exist in the named `src/internal/ingestion/<pkg>/fetcher.go`; whole-tree non-test caller search returned no production references. |
| S1-F3 | MEDIUM | `PARTIAL` | The release-path-named copy is missing five tables defined by current code, including `trade_notes`; dev has all five. `src/internal/store/state/feed.go` reads `FROM trade_notes WHERE league_id = ?1`, while `trade_notes_schema.go` creates it during `state.Initialize`. Thus a direct feed read against the raw release-path copy before startup DDL fails; normal current-code startup initializes this table. Sector 3 verifies this on copies of both DBs. |
| S1-F4 | LOW | `PARTIAL` | `Ping`, `GetFranchiseState`, and `GetFreeAgents` remain generated IPC bindings but have no frontend caller. `src/frontend/wailsjs/go/main/App.d.ts` defines all three; grep of invocation sites in `frontend/src` returns none. |
| S1-F5 | LOW | `DRIFT` | `src/SYSTEM_MAP.md` calls `internal/scouting` types-only and marks ingestion planned; current `src/m1_scouting.go` calls `crosswalk.Fetch`, RAS, Madden, PFR coverage, veteranfilm, and CFBD assemblers from ScoreLeague. The live implementation has advanced beyond this hand-maintained map. |
| S1-F6 | HIGH | `BROKEN` | Neither database copy is an evidence-backed faithful current league mirror. The release-path-named copy contains test-like rapid phase changes, a test-path rollover to 2027 and 395 §14 expiry releases; the dev copy contains in-app waiver/signing rows and phase changes. `state.Initialize` seeds only once, and there is no MFL roster refresh path. §1c shows source-separated counts, rollover checks, and the unreconciled seed cohorts. A fresh MFL export and explicit reconciliation are required before either is trusted for current league state. |
