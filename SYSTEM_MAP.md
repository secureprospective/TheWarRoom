# System Map

What exists in TheWarRoom and where new code belongs. Update it in the same commit as any new
package, IPC method or external service. Current as of 2026-10-03 (Stage 0 of
`docs/build-handoffs/Core_Build_Plan_2026-10.md`).

## Layers and packages

Import rules marked **(depguard)** are build errors in `.golangci.yml`, not conventions.

| Layer | Package | Job |
|---|---|---|
| App | repo root (`package main`) | Wails entrypoint (`main.go`), the `App` composition root (`app.go`) and the IPC adapters (`*_app.go`). Adapters validate, route and format; no business logic, no SQL. |
| Transport | `internal/mfl` | MFL HTTP client: rate limit, host discovery, 429 backoff. No domain types. |
| Layer 1 | `internal/ingestion` | Fetchers returning raw `Raw*` records. Shared helpers in the root package (`LeagueExport`, `FetchLeagueExport`, the CSV and CFBD plumbing); one subpackage per source. |
| Layer 1 | `internal/normalize` | Raw records → domain types: the players lookup and roster join. |
| Leaf | `internal/domain`, `internal/playerid`, `internal/numeric`, `internal/scouting` | Value types. `playerid.New` is the only way to build a `PlayerID`. `scouting` holds the Layer 4 input types. |
| Store | `internal/db` | SQLite pools: one write connection, many read-only ones, one WAL file. |
| Store | `internal/store/rulebook` | League rules from MFL as immutable versions with one active pointer, plus commissioner overrides. |
| Store | `internal/store/params` | Engine calibration: shipped defaults plus admin overrides. |
| Store | `internal/store/state` | Rosters, contracts, the contract-year ledger, dead cap, cap relief, phases, feed, calendar. Append-only ledgers; the transaction coordinator holds the only `Writer`. |
| Store | `internal/output` | Engine scores, frozen per (season, scoring config). Append-only, enforced by triggers. |
| Engine | `internal/engine` | The scoring pipeline as pure functions (L1, L3, L4 dispatch, L5, L6). |
| Engine | `internal/engine/l4/{offense,defense,kicker,curve}` | The ten position rubrics and the shared S-curve. |
| Composition | `internal/composition` | Engine inputs from the stores plus per-player facts. |
| Composition | `internal/rankings` | M1: scores every rostered player and writes the batch to `output`. |
| Composition | `internal/m2service`, `internal/powerrankings` | M2: franchise aggregation and the z-score blend with MFL standings. |
| Composition | `internal/scouting/assembly` | Builds scouting profiles from the Layer 1 feeds. |
| Mutation | `internal/transactions` | The `Coordinator`: every league-state change runs here, in one transaction. Handler subpackages (`acquisitions`, `contracts`, `deadcap`, `freeagency`) are reachable only through it. |
| Dev | `internal/harness` | The 13 architectural cases and the rookie sandbox; retired in Stage 7. |
| Tooling | `tools/ifaceguard` | Vet tool: no `interface{}`/`any` in exported signatures. |

**(depguard)**
- `mfl`, `ingestion` and `normalize` never import `engine`, `store`, `transactions`, `output` or
  `database/sql`.
- `engine` imports no store, transport, ingestion, normalize, `database/sql`, `net` or `os`.
- The four stores (`rulebook`, `state`, `params`, `output`) never import each other.
- `database/sql` is confined to `db`, `store` and `output`.
- `transactions/*` handler packages are imported only by `transactions`.

## IPC surface (21 methods on `App`)

| File | Methods |
|---|---|
| `version.go` | `AppInfo` (version, commit, startup error for the banner) |
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

- **Database:** `~/.config/TheWarRoom/thewarroom.db`. A dev build (no version stamp) uses
  `thewarroom-dev.db`, so development never touches the real league. One instance at a time
  (`.lock` file).
- **Migrations:** `internal/store/state/migrations.go`, forward-only and versioned; a
  `VACUUM INTO` backup is taken before any pending migration runs.
- **Logs:** one file per launch in `~/.config/TheWarRoom/logs/`, also on stderr. Startup logs
  each step and its time; a failed startup shows a banner in the app.
- **Headless check:** `thewarroom -probe` runs the full startup without a window and exits
  non-zero on failure.

## External services

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
- `standings_cache` and `league_schedule_cache`: become reads of `raw_archive` (Stage 4).
- `internal/harness` and its two dev tabs: deleted when the Stage 7 case set lands.

## What does not exist, on purpose

- No ORM: parameterized `database/sql` only, inside the data layer.
- No DI framework: constructor injection from `app.go`.
- No package-level mutable state (`gochecknoglobals`).
- No cgo SQLite: `modernc.org/sqlite`.
- No `interface{}`/`any` in exported signatures (`ifaceguard`).
- No logging framework: the standard library.
- No inbound HTTP server.
