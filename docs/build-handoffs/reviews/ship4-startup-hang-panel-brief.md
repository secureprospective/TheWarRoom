# EXPERT PANEL BRIEF — Ship 4 startup hang / "params store not initialized"

**Date:** 2026-07-05
**Reviewer routing:** GLM 5.2 (primary) + Gemini + DeepSeek, BLIND, independent. Output is **leads, not findings** — triage against source.
**Ask:** Diagnose the most likely root cause of the symptoms below and name the single most decisive next diagnostic. Do NOT assume my hypotheses are correct — challenge them.

---

## The application

TheWarRoom — a Wails v2 (Go backend + React/TS frontend) desktop app. SQLite (modernc.org/sqlite v1.52.0) in WAL mode. Runs on a Beelink (AMD APU, Linux, `wails build -tags webkit2_41`). The Go backend compiles and lints clean; `go test -race ./...` is green.

We just landed **Ship 4** of a salary-ledger migration (commit a708597): it drops three dead columns from the `contracts` table (`annual_salary`, `adjusted_salary`, `adjusted_salary_cents`) via `ALTER TABLE contracts DROP COLUMN`, makes the REAL→cents migration atomic, and adds a fail-loud guard in `load()`.

## The symptoms (reproducible, ≥3 times)

1. On launch, the admin panel shows **"params store not initialized"**.
2. **A few seconds later the whole window locks up and goes black** (frozen).
3. A diagnostic commit (d51b15f) logs `startupErr` to stderr on any startup error. The user reports **no new behavior** — but we are NOT certain the rebuilt binary with d51b15f is what ran, and no `startup failed:` line has been confirmed in terminal scrollback.

## Relevant architecture (verified against source)

- Wails calls `app.startup(ctx)` as **OnStartup**, synchronously, on the UI thread. OnStartup has no error return; `startup` stashes failures in `a.startupErr` (surfaced only via a `Ping` IPC method).
- **The app shell (`App.tsx`) never calls `Ping`.** So `startupErr` is displayed nowhere in the UI. The visible "params store not initialized" comes from the AdminPanel calling `GetParams`, which returns that string whenever `a.params == nil`.
- `startup` → `initStoreFloor(ctx)` wires stores in this order and assigns each field only on success:
  1. `params.New(pools).Initialize` → `a.params` (app.go:163-167) — **no network**
  2. `rulebook.New(pools).Initialize` → `a.rulebook` — on an EXISTING DB, **no network** (loads active version); network only if `activeVersion==0`
  3. `state.New(...).Initialize` → `a.state` — schema+migrations then `load()`; network seed only if `!hasState`
  4. transactions coordinator, then output store
- `a.params` is set FIRST (step 1). So for `GetParams` to report nil, **either step 1 (params.Initialize) failed/blocked, or startup never reached it, or startup is still blocked inside initStoreFloor before it returns** (the panel renders while OnStartup is still running).
- `state.Initialize.initSchema` runs: `baseSchemaDDL` exec → `initLedgerSchema` → `migrateMoneyCents` → `dropLegacyMoneyColumns`.
  - `migrateMoneyCents` opens `pools.Write().BeginTx` (holds the single write conn), ALTER ADD + UPDATE backfill + verify SELECT (on the tx) + Commit.
  - `dropLegacyMoneyColumns` loops 3 columns: `columnExists` (reads `pools.Read()`) then `pools.Write().ExecContext("ALTER TABLE contracts DROP COLUMN <col>")`.
- **DB pools:** Write pool `SetMaxOpenConns(1)` (single writer), `_txlock=immediate`, `busy_timeout` set, WAL. Read pool `mode=ro`, MaxOpenConns 10.
- The DB under test is described as a "fresh Ship-3 DB" — i.e. it HAS seeded state (so no network on startup) but was created by the PRIOR (Ship-3) binary, meaning the Ship-4 DROP-COLUMN migration runs against it for the first time on this launch.

## My three competing hypotheses (challenge them)

- **H1 — Blocking OnStartup:** all of initStoreFloor runs synchronously on the UI thread; something in it blocks → UI freezes black, params never set. But an existing DB does no network, and migrations on ~1700 rows should be ms. What in this chain could block for seconds on an existing DB?
- **H2 — DROP COLUMN / single-writer interaction:** the Ship-4 migration against a real Ship-3 DB deadlocks or stalls the single write connection (WAL checkpoint, `_txlock=immediate`, a lingering read handle, or DROP COLUMN table-rewrite contending). Would this hang rather than error?
- **H3 — The new `load()` fail-loud guard** (rostered salaried player with no PAID current-season cell) errors on real data → startup returns error → but then params WAS already set, so this can't explain a nil params store. So H3 explains a possible error but NOT the "params nil" symptom. Correct?

## Specific questions for the panel

1. Given `a.params` is set before any migration/network, what is the most plausible way `GetParams` still returns nil? (Blocked-still-running OnStartup vs. an early panic vs. the wrong binary running.)
2. Can a Wails OnStartup that panics (vs. returns) leave the app window up with a zero-value App (nil params) and a subsequently frozen UI? Does Wails recover a panicking OnStartup?
3. Is there a deadlock/stall path in the `migrateMoneyCents` (BeginTx holds the 1 write conn) → `dropLegacyMoneyColumns` (columnExists on read pool + Exec on write pool) sequence under WAL + `_txlock=immediate` + MaxOpenConns(1)?
4. Is running the full store floor (DB migrations) inside OnStartup on the UI thread the actual design defect, regardless of the specific stall — i.e. should this be moved off-thread with a loading state?
5. What is the single most decisive diagnostic to run next? (We plan a headless probe: a small `-probe` mode / Go program that opens the ACTUAL Beelink DB file and runs each Initialize step with per-step timing + logging, over SSH, no Wails — so the compositor and the silent-Ping cannot hide the failure.)
