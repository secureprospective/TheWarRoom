# System Map

What exists in TheWarRoom and where new code belongs. Update it in the same commit as any new
package, IPC method or external service. Current as of 2026-10-03 (Stage 2 of
`docs/build-handoffs/Core_Build_Plan_2026-10.md`).

## Layers and packages

Import rules marked **(depguard)** are build errors in `.golangci.yml`, not conventions.

| Layer | Package | Job |
|---|---|---|
| App | repo root (`package main`) | Wails entrypoint (`main.go`), the `App` composition root (`app.go`) and the IPC adapters (`*_app.go`). Adapters validate, route and format; no business logic, no SQL. |
| Transport | `internal/mfl` | MFL HTTP client: rate limit, host discovery, 429 backoff. No domain types. |
| Transport | `internal/archive` | The HTTP transport every outbound request goes through: it records each response body (sha256, gzip) and each attempt to a `Sink`. Leaf. |
| Layer 1 | `internal/ingestion` | Fetchers returning raw `Raw*` records. Shared helpers in the root package (`LeagueExport`, `FetchLeagueExport`, the CSV and CFBD plumbing); one subpackage per source. |
| Layer 1 | `internal/normalize` | Raw records → domain types: the players lookup and roster join. |
| Leaf | `internal/domain`, `internal/playerid`, `internal/numeric`, `internal/scouting` | Value types. `playerid.New` is the only way to build a `PlayerID`. `scouting` holds the Layer 4 input types. |
| Leaf | `internal/measures` | The measure registry: `measures.csv`, `sources.csv` and `source_fields.csv`, embedded and validated. Generates `docs/data-layer/Measure_Dictionary.md` (`make measure-dictionary`). Adding a source is a CSV row, not code. |
| Store | `internal/db` | SQLite pools: one write connection, many read-only ones, one WAL file. |
| Store | `internal/store/rulebook` | League rules from MFL as immutable versions with one active pointer, plus commissioner overrides. |
| Store | `internal/store/params` | Engine calibration: shipped defaults plus admin overrides. |
| Store | `internal/store/state` | Two things behind one `Reader`. **`Mirror`**: the league as MFL states it (season, rosters with contracts, salary adjustments), replaced whole by a refresh; every score surface reads it. **`Store`**: the what-if league in `whatif.db`, seeded from the mirror: rosters, contracts, the contract-year ledger, dead cap, cap relief, phases, feed, calendar. Append-only ledgers; the transaction coordinator holds the only `Writer`. |
| Store | `internal/store/history` | `history.db`: everything the app cannot rebuild. The fetch archive (`raw_archive`, `fetch_log`), facts per measure appended on change (`observations`, read as of a date through `Features`), source health, and scoring runs with the param set, engine and inputs they used. Append-only, enforced by triggers. |
| Engine | `internal/engine` | The scoring pipeline as pure functions (L1, L3, L4 dispatch, L5, L6). |
| Engine | `internal/engine/l4/{offense,defense,kicker,curve}` | The ten position rubrics and the shared S-curve. |
| Composition | `internal/composition` | Engine inputs from the stores plus per-player facts. |
| Composition | `internal/rankings` | M1: scores every rostered player from a params snapshot and history features, and writes one scoring run. |
| Composition | `internal/m2service`, `internal/powerrankings` | M2: franchise aggregation and the z-score blend with MFL standings. |
| Composition | `internal/scouting/assembly` | Builds scouting profiles from the Layer 1 feeds. |
| Mutation | `internal/transactions` | The `Coordinator`: every league-state change runs here, in one transaction. Handler subpackages (`acquisitions`, `contracts`, `deadcap`, `freeagency`) are reachable only through it. |
| Dev | `internal/harness` | The 13 architectural cases and the rookie sandbox; retired in Stage 7. |
| Tooling | `tools/ifaceguard` | Vet tool: no `interface{}`/`any` in exported signatures. |

**(depguard)**
- `mfl`, `ingestion` and `normalize` never import `engine`, `store`, `transactions` or
  `database/sql`.
- `engine` imports no store, transport, ingestion, normalize, `database/sql`, `net` or `os`.
- The four stores (`rulebook`, `state`, `params`, `history`) never import each other.
- `measures` imports nothing of the app's but `domain`; `archive` imports nothing of the app's.
- `database/sql` is confined to `db` and `store`.
- `transactions/*` handler packages are imported only by `transactions`.

## IPC surface (22 methods on `App`)

| File | Methods |
|---|---|
| `version.go` | `AppInfo` (version, commit, startup error for the banner) |
| `refresh_app.go` | `RefreshLeague` (pull the league from MFL into the mirror) |
| `m1_app.go`, `m1_player_score_app.go` | `ScoreLeague`, `GetRankings`, `GetPlayerScore` |
| `m2_app.go` | `GetPowerRankings` |
| `leagueschedule_app.go` | `GetLeagueSchedule` |
| `m4_app.go` | `GetFranchises`, `GetRoster`, `GetFreeAgentPool`, `GetLegalOps` |
| `transactions_app.go` | `ExecuteTransaction`, `PreviewTransaction`, `GetCurrentPhase` |
| `transactions_feed_app.go`, `transactions_calendar_app.go` | `GetFeed`, `GetCalendarEvents` |
| `rulebook_app.go` | `GetLeagueSetting`, `SetLeagueSettingOverride` |
| `harness_app.go` | `GetParams`, `SetParam`, `ScoreRookies`, `RunValidationSuite` |

Every method that takes frontend input validates it before acting. Bindings in
`frontend/wailsjs/` are generated (`wails generate module`); never hand-edit them.

## Data and logs

- **Databases,** in `~/.config/TheWarRoom/`, split by lifecycle:
  - `thewarroom.db`: the MFL mirror and app settings. Rebuildable from MFL.
  - `history.db`: the fetch archive, observations and scoring runs. Not rebuildable; back it up.
  - `whatif.db`: the what-if league the transaction screens work on (R2). Deleting it reseeds it
    from the mirror at the next launch.
  - A dev build (no version stamp) uses `thewarroom-dev.db`, `history-dev.db` and
    `whatif-dev.db`, so development never touches the real league. One instance at a time
    (`.lock` file).
- **Season:** read from MFL (`league.Discover`: the newest year in the league's history), held in
  the mirror. Nothing in the code names a year.
- **Refresh:** after the window opens, and from "Refresh from MFL" in the rail. An empty mirror is
  filled during startup.
- **Migrations:** `internal/store/state/migrations.go`, forward-only and versioned; a
  `VACUUM INTO` backup is taken before any pending migration runs.
- **Logs:** one file per launch in `~/.config/TheWarRoom/logs/`, also on stderr. Startup logs
  each step and its time; a failed startup shows a banner in the app.
- **Headless check:** `thewarroom -probe` runs the full startup without a window and exits
  non-zero on failure. `-version` prints the build label. Any other argument exits 2, so a
  mistyped flag never opens a window on the real database.

## External services

- Every outbound request goes through `internal/archive`, so every body received is kept in
  `history.db`. A URL with no row in `sources.csv` is logged as `unregistered`.
- **MFL API** (league 14432), outbound only, through `internal/mfl`. The league host is
  discovered at runtime. The players endpoint is limited to once a day.
- **nflverse and DynastyProcess** (static CSVs on GitHub), **CFBD** (bearer token in
  `CFBD_API_KEY`), **EA Madden** ratings. See `docs/sources/Approved_Sources.md`.

## Scheduled for replacement

These work and are tested, but the core plan rewrites or deletes them. Do not extend them.

- `internal/engine/l4/*`: becomes one routine plus a per-position settings table (Stage 5).
- `internal/scouting/assembly`, `m1_scouting.go` and the CSV/CFBD fetchers (`agetrajectory`,
  `collegedefense`, `collegeshare`, `madden`, `pfrcoverage`, `ras`, `schooltier`,
  `veteranfilm`): replaced by one table-driven loader into the measure store (Stage 4).
- `standings_cache` and `league_schedule_cache`: become reads of `raw_archive` (Stage 4). Until
  then they sit in `whatif.db` with the state store that owns them.
- `internal/harness` and its two dev tabs: deleted when the Stage 7 case set lands.

## What does not exist, on purpose

- No ORM: parameterized `database/sql` only, inside the data layer.
- No DI framework: constructor injection from `app.go`.
- No package-level mutable state (`gochecknoglobals`).
- No cgo SQLite: `modernc.org/sqlite`.
- No `interface{}`/`any` in exported signatures (`ifaceguard`).
- No logging framework: the standard library.
- No inbound HTTP server.
