HANDOFF — Session 11: B3c — League State Store (second Layer-2 store)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B3b — League Rulebook is COMPLETE + merged to main (2026-06-26, squash `12afc3c`).
  It is the FIRST Layer-2 config store and SET THE STORE TEMPLATE. Built:
  - `internal/store/rulebook/` — immutable versioned config snapshots + an explicit
    active pointer; `Reload()` SIDE-LOADS (stores a candidate version, returns a
    `ChangeSet`, never auto-promotes — stability over freshness); `Promote(ver)` =
    confirm-new OR rollback-to-prior; commissioner overrides as SEPARATE layered
    records with a `validateOverride` gate. All SQL parameterized; `db.Pools`
    read/write split; concurrency hardened (deep-copied slice reads; admin writes
    serialized under a write mutex — both from the GLM 5.2 review).
  - `internal/ingestion/league/` — Layer-1 fetcher; corrected the handoff premise
    that scoring is in the `league` export (it is NOT — scoring is the separate
    `rules` export; two calls). RAW `$t`-unwrapped, MFLList collapse-tolerant decode.
- B1/B2/B3 done: mfl transport client; ingestion boundary + fetchers; `internal/domain`
  + `internal/normalize` (Layer-1 typed records — `PlayerRecord`/`Roster`; LOCKED).
- The whole B2b-Fetch (Layer-1 scouting/production) arc is DONE + merged.
- THIS SESSION builds B3c, the SECOND Layer-2 store. Branch fresh off main:
  git checkout main && git pull && git checkout -b session/b3c-league-state-store.
  Confirm scope with Christopher first.

== READ FIRST ==
- /root/.claude/plans/very-good-now-i-replicated-feigenbaum.md → "Wireframe 2 — Layer 2
  Config Stores", the **B3c row** (the writer/reader-split constraints — clone the
  FILE SHAPE from rulebook, NOT its version/override machinery; see below).
- docs/backend/Backend_Architecture.md §8 — the `league_state` + `league_calendar`
  tables, and the `rosters`/`contracts` tables (the 32-team mutable state surface).
- internal/store/rulebook/ — the built template (file order, parameterized SQL,
  db.Pools split, injected Source for testability, concurrency idioms).
- internal/normalize + internal/domain — B3c's seed source (`Initialize` PULLS these).
- /root/.claude/plans/session-3-audit-build-sequencing.md → AD-02, AD-05.

== RECON (Haiku fan-out — run before design/build) ==
Spin a Haiku Explore subagent over the READ FIRST docs; ask for: the exact
`league_state`/`league_calendar`/`rosters`/`contracts` schemas verbatim; what
"32-team mutable state" comprises (live rosters + contracts + cap usage vs
phase/flags — distinguish what B3c OWNS from what B6/calendar own); how
`internal/normalize` exposes its typed records (the seed shape); and any OQ touching
league state / phase transitions. Claude VERIFIES load-bearing claims against source
before any code (the rulebook handoff's "scoring is in `league`" premise was wrong —
do not trust a handoff claim over the live schema/source).

== GATE CHECK (confirm with Christopher before writing code) ==
1. Scope of "32-team mutable state" for B3c v1.0: rosters + contracts + cap usage,
   and/or the `league_state` phase/flags row? (Recommend: the live roster/contract
   state B7 mutates, seeded from B3; phase/flags may be a thin add or deferred.)
2. Confirm the StateReader surface the engine/M1 need (read methods) vs the
   StateWriter surface B7 needs (mutations) — design both interfaces this session.

== WHAT THIS SESSION BUILDS (Build_Tracker row 10) ==
B3c — League State Store: the 32-team MUTABLE runtime state, SQLite-backed. It is a
Layer-2 store but DIFFERS from B3b's config store:
  - **Writer/reader interface split (the defining B3c constraint):** expose a
    `StateWriter` interface (the mutations) injected via DEPENDENCY INJECTION to B7
    ONLY, and a read-only `StateReader` interface to every other consumer (engine,
    modules, handlers). B7 is the SOLE runtime mutator (AD-02 — B7a is built before
    any mutation path is exercised).
  - **`Initialize()` PULLS from B3** (normalize's typed records). B3 NEVER pushes —
    inject the seed source the way rulebook injected `league.Source`.
  - **NO `Reload()`.** Config can re-pull MFL; runtime state cannot be blindly
    re-pulled (B7's mutations are the source of truth at runtime). This is the key
    divergence from the rulebook template — do not clone `Reload`/versioning/overrides.
  - Clone from rulebook ONLY: file shape (state.go/types.go/state_test.go), file
    order, parameterized SQL, db.Pools split, injected seed source, the concurrency
    idioms (slice-copy on reads, serialized writes).

== LOCKED DECISIONS (do not relitigate) ==
- AD-02: B7a (transaction foundation + coordinator) is built FIRST among the
  transaction work and is the sole holder of `StateWriter`. B3c only DEFINES the
  writer interface + the atomic parameterized write path; it does not wire B7.
- Layer law: B3c is Layer 2 — it imports `domain`/`normalize` (downward) and `db`;
  it does NOT import another store (no rulebook import), and modules NEVER write B3c
  except through B7.
- The single-writer SQLite pool (`db.Pools.Write()`, MaxOpenConns=1, _txlock=immediate)
  is given ONLY to the StateWriter path — this is the driver-level enforcement of
  "B7 is the sole writer."

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b3c-league-state-store. Never git --no-verify.
- CT105 build: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first);
  then go test -race ./... . Go 1.26.4 at /usr/local/go/bin.
- Every custom gate proven by a planted failure (M3). Shared logic extracted, not
  copy-pasted (M17). File < 400 lines (filelen); pre-split if needed (AD-17).
- Review gate: GLM 5.2 (Z.ai Coding Plan, via OpenCode on bird). agy/Gemini RETIRED
  as code reviewers. Reviewer works BLIND; output is LEADS, not findings — TRIAGE
  every finding against source. (B3b's GLM review found 2 REAL concurrency defects —
  it earns its keep; still triage.)

== CARRIED FROM LAST SESSION (B3b) ==
- The store concurrency pattern is now proven: reads deep-copy nested slices; admin
  mutations serialize under a write mutex separate from the reader RWMutex. Clone it.
- The injected-source pattern (Source interface + APISource adapter + fake in tests)
  is the testability seam — reuse for B3c's B3 seed.
- OPEN (not B3c-blocking): OQ-013 created→official player-id reconciliation ramp
  (refresh/sync layer) — relevant when B3c seeds from B3's normalized records, since
  commissioner-created players carry placeholder ids. Note it; do not solve here.

== CLOSE GATE FOR THIS SESSION ==
- Build: make lint 0 + go test -race ./... green.
- StateReader/StateWriter split is real: a planted test confirms a non-B7 consumer
  cannot reach a mutation (compile-time via the interface it is handed).
- Initialize seeds 32 franchises' state from a B3 (normalize) source; a planted
  bad/empty seed fails loud (no silent empty-state wipe — the rosters errEmpty lesson).
- The store template divergence is clean (no Reload/version/override leaked in).
- GLM 5.2 BLIND review; triage every finding vs source.
- Squash-merge to main after Christopher confirms; write the next handoff (B4 —
  Admin Parameter Store, row 11) before clearing.
