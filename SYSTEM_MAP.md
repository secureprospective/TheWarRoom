# System Map

What exists in TheWarRoom and where new code belongs, so agents do not reinvent
utilities or violate the layer boundaries. **The tree below is locked at B0**
(AD-03, `internal/`-rooted layout from `Fable_TheWarRoom_code_plan.md` §3.1).
After B0 the tree is law — a new package means updating this file.

> Hand-maintained. Update when a package, utility, or external service is added.
> Status tags: `[built]` exists now · `[planned: Bn]` lands in that build session.

---

## Directory invariants

Repo root holds the Wails entrypoints and the toolchain config; all backend
logic lives under `internal/`. The import rules below mirror the `depguard`
rules in `.golangci.yml` — they are build errors, not conventions.

- `main.go` — Wails bootstrap only (`wails.Run`, embed assets, lifecycle hooks). `[built]`
- `app.go` — the `App` composition root. IPC-bound methods are thin adapters: validate → route → format. **No business logic, no direct SQL here.** `[built]`
- `frontend/` — React + Tailwind + Zustand (Vite, pnpm). `[built]`
  - `frontend/src/store/` — Zustand slices. The IPC call lives in the store, never the component. Slices never import each other.
  - `frontend/wailsjs/` — **generated** Go↔JS bindings. Never hand-edit; regenerate with `wails generate module`.
- `internal/playerid/` — `PlayerID` newtype (struct-wrap, AD-06). Single source of truth for MFL IDs. Imports nothing internal. `[built]`
- `internal/schema/` — hand-written boundary validation for external input (MFL JSON, CSV, IPC). `Decode*` + `Validate()`. No reflection/struct-tags. `[built]`
- `internal/db/` — SQLite access. **One of only two packages allowed to import `database/sql` / the sqlite driver** (with `internal/store`). Split read/write pools. `[built]`
- `internal/mfl/` — Layer 1 HTTP transport only. One `Do()` + `DiscoverHost()`. No domain types. Never imports up (engine/store/transactions/api). `[built — friction-test client; formalized at B1]`
- `internal/ingestion/` — Layer 1 fetchers → `Raw*` records. Schema-validate, transform nothing. Never imports up. `[planned: B2/B2b]`
- `internal/normalize/` — `Raw*` → domain types. Pure transformation. `[planned: B3]`
- `internal/domain/` — leaf shared types (`PlayerRecord`, `EngineRecord`) + `constants.go` (all Section 5 numbers). Imports nothing internal. `[planned: B3]`
- `internal/store/` — three sibling stores; **none imports another** (depguard). `[planned: B3b/B3c/B4]`
  - `rulebook/` (B3b) · `state/` (B3c — `StateReader`/`StateWriter` split; only B7 gets the writer) · `params/` (B4)
- `internal/engine/` — pure-function scoring pipeline. **Imports no store, no DB, no I/O** — all state arrives as parameters (depguard `engine-is-pure`). `[planned: B5a]`
  - `l4/{offense,defense,kicker}/` — position rubrics (B5b-*); `mathx/` — the shared S-curve.
- `internal/output/` — B6 per-season output store; append-only (no Update/Delete API + SQLite trigger). `[planned: B6]`
- `internal/transactions/` — B7. Root package = the sole-writer `Coordinator`. Handler subpackages (`acquisitions`/`contracts`/`deadcap`) are reachable only via `Coordinator.Execute` (depguard). `[planned: B7a–d]`
- `tools/ifaceguard/` — custom go/analysis vettool (separate Go module). Flags `interface{}`/`any` in exported signatures. `[built]`
- `docs/` — architecture, rubrics, build handoffs. `docs/agent-codex.md` = build doctrine.

## Existing utilities (reuse, do not duplicate)

- `playerid.New(raw string) (PlayerID, error)` — validates + zero-pads MFL IDs to 4 digits. The **only** way to build a `PlayerID`. Also `String()`, `IsZero()`, JSON marshal/unmarshal through `New`.
- `db.Open(ctx, path) (*Pools, error)` — opens + verifies WAL; returns the split pools. `Pools.Read()`, `Pools.Write()`, `Pools.Health(ctx)`, `Pools.JournalMode(ctx)`, `Pools.Close()`.
- `schema.DecodePlayerRecord(r io.Reader)` + `RawPlayerRecord.Validate()` — the boundary-validation pattern to copy for every external input shape.
- `internal/mfl` — the HTTP transport client (`Do`, `DiscoverHost`). Use it; do not write a second HTTP client.
- `app.PingResult` / `App.Ping()` — the IPC health round-trip (B0 reference for a bound method).

## External services

- **MyFantasyLeague (MFL) API** — outbound HTTP only, via `internal/mfl`. League ID `14432`; host discovered at runtime (never hardcode `www47`). Always append `JSON=1`. Respect cache discipline (companion plan §5.7); back off on 429.
- **SQLite (WAL)** — local file at `~/.config/TheWarRoom/thewarroom.db`, via `internal/db`. Single writer, many readers.

## Configuration sources

- **Database path** — `app.databasePath()` (user config dir). Not env-driven.
- **Engine calibration params** — shipped defaults in `internal/store/params` (B4), tunable via the M9a admin UI. Never hardcode calibration numbers in engine code.
- **League rules** — `internal/store/rulebook` (B3b), MFL-sourced + delta overrides.

## Deferred / unwired scaffolding (built ahead of wiring — NOT dead code)

Some code ships in `main` ahead of the phase that switches it on. It is designed,
unit-tested scaffolding with a documented wiring trigger — retained by ruling
(2026-07-20) rather than carved out. Flagged here so a reviewer does not read it as rot.

- **Scouting sub-system (~2000 lines).** `internal/scouting` (`Profile`, `OffenseFilm`,
  `IDPFilm`, `NGSCoverage`, `SafetyRole`) + the ~11 scouting fetchers under
  `internal/ingestion` (`agetrajectory`, `collegeshare`, `collegedefense`, `crosswalk`,
  `kicking`, `madden`, `nflproduction`, `pfrcoverage`, `ras`, `touchshare`,
  `veteranfilm`). The types + `Fetch()` exist and are tested; the production wiring
  (fetch → `Profile` → engine Layer 4) is **not switched on** — Layer 4 runs the
  identity/neutral path (M1: "Data-Parity ABSENT … no fetcher wired yet"). **Wiring
  trigger:** the scouting data-integration sprint (Option D hybrid). **Why retained:**
  the source maps (`docs/data-layer/{Offense,Defense}_Scouting_Source_Map.md`) + the
  per-type SOURCE-DRIFT notes in `internal/scouting/types.go` capture eliminated-source
  research that would be expensive to rediscover. See the `internal/scouting` package doc.
- **`NewCFBDClient`** (`internal/ingestion/cfbd.go`) — extracted ahead of the CFBD
  orchestration; wire or drop when that fetcher path lands.

## What does not exist (intentionally)

- **No ORM.** Raw parameterized `database/sql`, confined to `internal/db` + `internal/store`.
- **No DI framework.** Constructor injection; dependencies passed explicitly from `app.go`.
- **No package-level mutable state / globals** (`gochecknoglobals`). Only sanctioned globals: sentinel errors and the `go:embed` assets var.
- **No `mattn/go-sqlite3`.** `modernc.org/sqlite` (pure Go, CGo-free) — DSN uses `_pragma=` params.
- **No `interface{}`/`any` at exported boundaries** (`ifaceguard`). Use concrete types; `//ifaceguard:allow` only for a deliberate generic boundary.
- **No logger framework.** Standard library.

## Public-facing surfaces (for security review)

- **IPC** — exported methods on `App` (`app.go`), called from the frontend via generated bindings. Every method that accepts frontend input must validate before acting. Currently: `App.Ping()`.
- **Outbound HTTP** — `internal/mfl` to the MFL API. No inbound HTTP server.
- **Local filesystem** — the SQLite file under the user config dir.
