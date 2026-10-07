# System Map

What exists in TheWarRoom and where new code belongs. Update it in the same commit as any new
package, IPC method or external service. Current as of 2026-10-04 (Stage 7 of
`docs/build-handoffs/Core_Build_Plan_2026-10.md`).

## Layers and packages

Import rules marked **(depguard)** are build errors in `.golangci.yml`, not conventions.

| Layer | Package | Job |
|---|---|---|
| App | repo root (`package main`) | Wails entrypoint (`main.go`), the `App` composition root (`app.go`) and the IPC adapters (`*_app.go`). Adapters validate, route and format; no business logic, no SQL. |
| Transport | `internal/mfl` | MFL HTTP client: rate limit, host discovery, 429 backoff. No domain types. |
| Transport | `internal/archive` | The HTTP transport every outbound request goes through: it records each response body (sha256, gzip) and each attempt to a `Sink`. Leaf. |
| Layer 1 | `internal/ingestion` | Fetchers returning raw `Raw*` records. Shared helpers in the root package (`LeagueExport`, `FetchLeagueExport`, the CSV and CFBD plumbing); one subpackage per source. |
| Layer 1 | `internal/ingestion/feeds` | The table-driven loader: reads any file in `feeds.csv` and maps its columns through `source_fields.csv` into a batch for the measure store. A new signal from such a file is registry rows, not code. |
| Layer 1 | `internal/ingestion/college` | CFBD season stats as a batch for the measure store: one call per college season, each team's totals summed for the share denominators. `Map` reads an archived body, so a season is re-read when the registry maps a measure it lacks. |
| Layer 1 | `internal/ingestion/contracts` | NFL contracts (OverTheCap via nflverse, Parquet) as a batch: each player's contract terms under the season signed. Read every 30 days. |
| Layer 1 | `internal/normalize` | Raw records → domain types: the players lookup and roster join. |
| Leaf | `internal/domain`, `internal/playerid`, `internal/numeric`, `internal/scouting` | Value types. `playerid.New` is the only way to build a `PlayerID`. `scouting` holds the Layer 4 input types. |
| Leaf | `internal/measures` | The measure registry: `measures.csv`, `sources.csv`, `source_fields.csv` and `feeds.csv`, embedded and validated. Generates `docs/data-layer/Measure_Dictionary.md` (`make measure-dictionary`). Adding a source is a CSV row, not code. |
| Store | `internal/db` | SQLite pools: one write connection, many read-only ones, one WAL file. |
| Store | `internal/store/rulebook` | League rules from MFL as immutable versions with one active pointer, plus commissioner overrides. |
| Store | `internal/store/params` | Engine calibration: shipped defaults plus admin overrides, league-wide or per position. The Layer 4 settings are seeded from `l4.Defaults`; the model's fitted values from the embedded `fitted.json`, marked calibrated. Shipped defaults are upserted at start-up; overrides are never touched. A set splits into the board's part and the model's (`model.*`, `dynasty.*`); each run records only its own. |
| Store | `internal/store/state` | Two things behind one `Reader`. **`Mirror`**: the league as MFL states it (season, rosters with contracts, salary adjustments), replaced whole by a refresh; every score surface reads it. **`Store`**: the what-if league in `whatif.db`, seeded from the mirror (rosters and MFL salary adjustments, so its cap starts equal): rosters, contracts, the contract-year ledger, dead cap, cap relief, phases, feed, calendar. Append-only ledgers; the transaction coordinator holds the only `Writer`. |
| Store | `internal/store/history` | `history.db`: everything the app cannot rebuild. The fetch archive (`raw_archive`, `fetch_log`), facts per measure appended on change (`observations`, read as of a date through `Features`), source health, and scoring runs with the param set, engine and inputs they used. Append-only, enforced by triggers. |
| Engine | `internal/engine` | The scoring pipeline as pure functions (L1, L3, L4 dispatch, L5, L6). |
| Engine | `internal/model` | The measurable's model: a player's league-points percentile blended with a prior from pre-NFL facts, the talent and survival arcs, and `Params`, the fitted values per position (`model.*@POS`). Pure. |
| Engine | `internal/model/fit` | Fits `model.Params` from stored seasons, scoring every choice on a holdout season. Pure; `cmd/fit` runs it. |
| Engine | `internal/engine/l4` | Layer 4: one rubric routine driven by a per-position settings table. The adjustable numbers in the table are params (`l4.film.cap@WR` and so on); composition reads them back for each run. |
| Composition | `internal/composition` | Engine inputs from the stores plus per-player facts. |
| Composition | `internal/rankings` | M1: scores every rostered player from a params snapshot and history features, and writes one scoring run. |
| Composition | `internal/m2service`, `internal/powerrankings` | M2: counts each roster's model values (on-field-now or dynasty, league points per game) as the best legal lineup, the top N or the whole roster, and z-blends them: this season with the results (all-play when MFL reports it, otherwise points for) at 4 ÷ (4 + weeks played) or the slider's weight; the franchise with the roster's age. `m2service.Outlooks` adds the context columns (cap room, schedule luck, projected record), never in the score. |
| Composition | `internal/scouting/assembly` | Builds today's board's scouting profiles: RAS, coverage and school tier from their feeds, college share and breakout age from history. |
| Composition | `internal/modelrun` | The measurables: values every rostered player with `internal/model` and the run's params, and writes a model run beside the board. `AtLeaguePositions` puts every player MFL lists at MFL's position before the scales are built; `cmd/fit` does the same from MFL's players export. |
| Composition | `internal/snapshot` | Pure, deterministic target UI snapshot builder over a read-only source; shared freshness is `domain.Freshness`. |
| Tooling | `cmd/fixtures` | Copies league and history snapshots to disposable databases, initializes the app stores offline and exports the shared snapshot. |
| Mutation | `internal/transactions` | The `Coordinator`: every league-state change runs here, in one transaction. Handler subpackages (`acquisitions`, `contracts`, `deadcap`, `freeagency`) are reachable only through it. |
| Tooling | `cmd/fit` | Reads a history database, runs `model/fit`, and writes `internal/store/params/fitted.json` (shipped as calibrated defaults) and `docs/fit/Fit_Report.md`. |
| Tooling | `tools/ifaceguard` | Vet tool: no `interface{}`/`any` in exported signatures. |

**(depguard)**
- `mfl`, `ingestion` and `normalize` never import `engine`, `store`, `transactions` or
  `database/sql`.
- `engine` and `model` import no store, transport, ingestion, normalize, `database/sql`, `net` or `os`.
- The four stores (`rulebook`, `state`, `params`, `history`) never import each other.
- `measures` imports nothing of the app's but `domain`; `archive` imports nothing of the app's.
- `database/sql` is confined to `db` and `store`.
- `transactions/*` handler packages are imported only by `transactions`.

## IPC surface (26 methods on `App`)

| File | Methods |
|---|---|
| `target_app.go` | `TargetSnapshot` (initialized mirror and rulebook, cached live directory with archive fallback; shared fixture/live contract); `TargetClock` (phase log and commissioner calendar from the what-if store; no network) |
| `version.go` | `AppInfo` (version, commit, startup error for the banner) |
| `refresh_app.go` | `RefreshLeague` (pull the league from MFL into the mirror) |
| `crosswalk_app.go` | `LoadCrosswalk`, `GetCrosswalkReport` (the player directory and its match rates) |
| `signals_app.go` | `LoadSignals`, `GetSignals` (load every due signal file; source health, freshness and coverage) |
| `m1_app.go`, `model_app.go`, `m1_player_score_app.go` | `ScoreLeague` (board run and model run), `GetRankings` (with the measurables), `GetPlayerScore` |
| `m2_app.go` | `GetPowerRankings` |
| `leagueschedule_app.go` | `GetLeagueSchedule` |
| `m4_app.go` | `GetFranchises`, `GetRoster`, `GetFreeAgentPool`, `GetLegalOps` |
| `transactions_app.go` | `ExecuteTransaction`, `PreviewTransaction`, `GetCurrentPhase` |
| `transactions_feed_app.go`, `transactions_calendar_app.go` | `GetFeed`, `GetCalendarEvents` |
| `rulebook_app.go` | `GetLeagueSetting`, `SetLeagueSettingOverride` |
| `admin_app.go` | `GetParams`, `SetParam` (the Engine Admin console) |

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
  `CFBD_API_KEY`). EA Madden was removed from the prior (R8); its `sources.csv` row is retired so
  its archived fetches stay labelled. See `docs/sources/Approved_Sources.md`.

## Scheduled for replacement

These work and are tested, but the core plan rewrites or deletes them. Do not extend them.

- `internal/scouting/assembly`, `m1_scouting.go` and the CSV/CFBD fetchers (`pfrcoverage`, `ras`,
  `schooltier`): today's board still reads them. The college share and breakout age already read
  the college seasons `college` stores in history, not CFBD live. They go when their consumer,
  today's engine, does.

## What does not exist, on purpose

- No ORM: parameterized `database/sql` only, inside the data layer.
- No DI framework: constructor injection from `app.go`.
- No package-level mutable state (`gochecknoglobals`).
- No cgo SQLite: `modernc.org/sqlite`.
- No `interface{}`/`any` in exported signatures (`ifaceguard`).
- No logging framework: the standard library.
- No inbound HTTP server.

The target UI data contract and fixture/live providers live in `frontend/src/app/data/`;
`components/` remains the current harness. `snapshot.NewSource` composes the initialized mirror and rulebook; the directory carries its own fetch provenance. Money in the snapshot is exact cents; absent held fields are omitted.

The target shell lives in `frontend/src/app/shell/`; its six-node table and hash routes are data.
`frontend/src/app/commands/` owns the immutable verb registry, sole dispatch path, keyboard
bindings and in-memory recent-command log. `Act` is the only clickable primitive in the target
UI. `frontend/src/TargetMount.tsx` switches between the target shell and the unchanged harness.

`docs/ui/registry/normalize.py` normalizes and validates `docs/ui/endpoint-registry.csv`.
`frontend/scripts/gen-endpoints.mjs` generates its frozen TypeScript endpoint map;
`frontend/src/app/registry/` owns resolution, indexed placements, compact not-wired rows and the
pure two-route gate. `endpoint.open` navigates and highlights; merged rows target kept ids or places.
The harness is lazy-loaded only on `harness.open`; it is not in the target entry chunk.

`internal/leagueclock` is a pure reading over the held phase log and commissioner calendar:
deadlines soonest first with U0–U3 and the pin/promote ordering rule; league windows stay explicitly
unknown. MFL's schedule has no dates, so there is no current week until a dated source exists.
`internal/envelope` owns immutable move states (one transition map), checks, observation predicates
and an in-memory audit log; neither core fetches, persists or reads the clock.
`snapshot.BuildClock` is the one clock builder: `TargetClock` (in `target_app.go`) passes the what-if store, and
`cmd/fixtures -whatif` passes a backup of it. Move fixtures stay outside the entry chunk.
