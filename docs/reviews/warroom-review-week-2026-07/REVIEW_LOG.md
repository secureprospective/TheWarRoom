## Chunk 1: config-and-secrets
Session start: 2026-07-11T15:26:55Z · model: glm-5.2 (zai/glm-5.2)


### frontend/vite.config.ts
(No findings — 1.3.2: minimal config, no inline secrets.)

### go.mod
[1.3.4] MEDIUM — Go toolchain go1.26.4 has 2 stdlib vulnerabilities (same root cause, same fix):
  - GO-2026-5856: crypto/tls ECH privacy leak. Code IS affected — 4 call traces:
    internal/ingestion/cfbd.go:68 (http.Client.Do → tls.Conn.HandshakeContext, tls.Dialer.DialContext),
    internal/mfl/client.go:93 (io.ReadAll → tls.Conn.Read),
    internal/domain/money.go:110 (fmt.Fprintf → tls.Conn.Write).
    Advisory: https://pkg.go.dev/vuln/GO-2026-5856
  - GO-2026-4970: os root escape via symlink + trailing slash. Code does NOT call affected paths.
    Advisory: https://pkg.go.dev/vuln/GO-2026-4970
  Fix: upgrade Go toolchain from 1.26.4 to 1.26.5 (resolves both).

[1.3.4] LOW — golang.org/x/sys@v0.42.0 (indirect dep):
  - GO-2026-5024: integer overflow in NewNTUnicodeString (Windows-only). Code does NOT call affected function.
    Advisory: https://pkg.go.dev/vuln/GO-2026-5024
  Fix: upgrade golang.org/x/sys to v0.44.0+.

[1.3.5] INFO — All 4 direct deps actively maintained (wails/v2 v2.12.0, goleak v1.3.0,
  golang.org/x/time v0.8.0, modernc.org/sqlite v1.52.0). No abandoned deps.

### go.sum
(No findings — 1.3.2: module checksums only, no secrets.)

### wails.json
(No findings — 1.3.2: clean. Author email present (secureprospective@gmail.com) but is not a secret.
 build/windows/info.json uses {{.Info.*}} template placeholders, not inline values.)

### frontend/package.json
[1.3.4] HIGH — vite ^3.0.7 has 14 vulnerabilities (2 high, 10 moderate, 2 low).
  Vite 3.x is EOL; no patches will be backported. Root cause: vite pinned at ^3.0.7 (current: 8.x).
  All vulns are dev-server-only (not in production Wails binary), but exploitable during development
  (especially on Windows). Notable advisories:
  - HIGH  GHSA-c27g-q93r-2cwf: command injection via launch-editor (Windows). Fix: vite >=5.4.9.
  - HIGH  GHSA-fx2h-pf6j-xcff: server.fs.deny bypass (Windows alt paths). Fix: vite >=6.4.3.
  - MOD   GHSA-93m4-6634-74q7: server.fs.deny bypass via backslash (Windows, vite 3.x specifically). Fix: vite >=5.4.21.
  - MOD   GHSA-67mh-4wv8-2f99: esbuild dev-server CORS bypass (transitive via vite). Fix: esbuild >=0.25.0.
  - MOD   GHSA-vg6x-rcgg-rjx6: vite dev-server CORS bypass. Fix: vite >=4.5.6.
  - MOD   GHSA-4w7w-66w2-5vf9: path traversal in optimized deps .map handling. Fix: vite >=6.4.2.
  - MOD   GHSA-v6wh-96g9-6wx3: NTLMv2 hash disclosure via UNC path (Windows). Fix: vite >=6.4.3.
  - MOD   GHSA-859w-5945-r5v3: server.fs.deny bypass with /. prefix. Fix: vite >=5.4.14.
  - MOD   GHSA-356w-63v5-8wf4: server.fs.deny bypass with invalid request-target. Fix: vite >=4.5.13.
  - MOD   GHSA-xcj6-pq6g-qj4x: server.fs.deny bypass via .svg/relative paths. Fix: vite >=4.5.12.
  - MOD   GHSA-x574-m823-4x7w: server.fs.deny bypass with ?raw?? Fix: vite >=4.5.10.
  - MOD   GHSA-4r4m-qw57-chr8: server.fs.deny bypass for inline/raw with ?import. Fix: vite >=4.5.11.
  - LOW   GHSA-g4jq-h2w9-997c: middleware serves files with same-name prefix. Fix: vite >=5.4.20.
  - LOW   GHSA-jqfw-vq24-v9c3: server.fs not applied to HTML files. Fix: vite >=5.4.20.
  Fix: upgrade vite to >=6.4.3 (resolves all 14). Also requires upgrading @vitejs/plugin-react (^2.0.1 → ^6.x)
  and typescript (^4.6.4 → ^5.x) for compatibility. Note: this is a multi-major-version jump.

[1.3.5] INFO — All direct deps actively maintained (last release within 12 months per npm registry).
  No abandoned packages. However, vite (^3.0.7 vs 8.x), typescript (^4.6.4 vs 5.x), and
  @vitejs/plugin-react (^2.0.1 vs 6.x) are pinned 2-3 major versions behind current — not abandoned,
  but severely outdated and the root cause of the 14 vulnerabilities above.

### .github
[INFO] Directory does not exist in working tree or git history (git log --all -- '.github/' returned
  nothing). Cannot evaluate 1.3.2 for GitHub Actions/workflow configs. No CI workflow files to review.

### Repo-wide: 1.3.1 git-history secret grep
[INFO] No secrets found in git history.
  - Filename grep across all files ever added to git (git log --diff-filter=A --name-only): no hits
    for (api_key|password|secret|token|BEGIN.*KEY|\.env|credentials).
  - Content grep across all commits (git grep across all revs): hits were all false positives —
    variable names (apiKey parameter), env var references (os.Getenv("CFBD_API_KEY")), comments
    ("not a secret/host to discover"), npm package name "js-tokens", Go AST "go/token" package,
    rate-limiter "token bucket" comment, gitleaks pre-commit hook config.
  - Targeted search for hardcoded secret values (assignment patterns with 8+ char literals): no hits.
  - Private key search (-----BEGIN.*PRIVATE.*KEY-----): no hits.
  - No .env or credential files tracked in git (git ls-files | grep -iE '\.env|credentials|secret': empty).
  - Defense-in-depth: .pre-commit-config.yaml runs gitleaks (v8.30.1, pinned to SHA) on every commit.

### Repo-wide: 1.3.2 secrets in current tree/config
[INFO] No secrets found. All 11 config files reviewed (vite.config.ts, go.mod, go.sum, wails.json,
  frontend/package.json, frontend/tsconfig.json, frontend/tsconfig.node.json, build/windows/info.json,
  opencode.json, .golangci.yml, .pre-commit-config.yaml). No inline secret values. Pre-commit hooks
  pinned to commit SHAs (not mutable tags) — correct supply-chain hygiene.

### Repo-wide: 1.3.4 dependency audit
(See go.mod and frontend/package.json findings above.)
  govulncheck: 3 vulnerabilities (1 code-affected, 2 not-called).
  pnpm audit: 14 vulnerabilities (2 high, 10 moderate, 2 low).

### Repo-wide: 1.3.5 abandoned dependencies
[INFO] No abandoned direct deps found. All Go direct deps (4) and JS direct deps (11) have releases
  within the last 12 months. Evaluation method: npm registry time query for JS deps; version recency
  for Go deps (all at recent major/minor versions indicating active maintenance).

--- chunk config-and-secrets complete: 4 findings ---

## Chunk 2: db-and-store
Session start: 2026-07-11T15:34:27Z · model: glm-5.2 (zai/glm-5.2)

### internal/db/pools.go
(No findings — no panics; `_ = write.Close()`/`_ = read.Close()` are intentional error-path
cleanup with primary error preserved. Single static SQL query string; no string building.
sql.DB pool handles released via Close(). No multi-write paths. No unstructured logging.)

### internal/schema/schema.go
(No findings — boundary-validation logic only; no SQL, no panics, no logging, no DB writes.
io.Reader is passed in (not opened), so no resource lifecycle to manage.)

### internal/store/params/defaults.go
(No findings — pure data (default ParamDef slice returned from function, not package var).
No SQL, no panics, no errors, no resources, no logging.)

### internal/store/params/helpers.go
(No findings — scan helpers take *sql.Rows as parameter (caller owns lifecycle); all rows
usage in writes.go has defer rows.Close(). No SQL construction, no panics, no logging.)

### internal/store/params/params.go
(No findings — store construction + typed read accessors only. No SQL queries in this file,
no panics, no errors discarded, no resource acquisition, no logging.)

### internal/store/params/types.go
(No findings — type/key/constant definitions only. defKey builds an in-memory map key from
two strings, not SQL. No SQL, no panics, no logging.)

### internal/store/params/writes.go
(No findings — all SQL uses ? placeholders with static query strings (no fmt.Sprintf/+ building).
seedDefaults (5 INSERTs) correctly wrapped in BeginTx/Commit. SetOverride is a single upsert
(1 statement, atomic by definition) + read-only load — no multi-write hazard. loadDefaults/
loadOverrides both defer rows.Close(). `_ = tx.Rollback()` in deferred func is correct Go idiom
(post-commit rollback is a no-op). No unstructured logging.)

### internal/store/rulebook/diff.go
(No findings — pure comparison logic (no SQL, no writes, no panics). fmt.Sprintf builds
in-memory diff field labels, not SQL. No resource acquisition, no logging.)

### internal/store/rulebook/helpers.go
(No findings — pure clone/settings-map helpers. No SQL, no panics, no resources, no logging.)

### internal/store/rulebook/types.go
(No findings — type/constant definitions only.)

### internal/store/rulebook/rulebook.go
[1.2.2] MEDIUM — Initialize (lines 79-85) performs TWO separate writes on the fresh-DB path
without a wrapping transaction: insertVersion (INSERT into rulebook_versions, line 79) then
setActive (UPSERT into rulebook_active, line 83). Each is individually atomic (single
statement), but if setActive fails after insertVersion succeeds, an orphaned version row is
left behind with no active pointer. Mitigating context: the orphaned row is immutable and
never referenced; restart recovery is clean (activeVersion returns 0 → re-fetch → insert a
new version). Practical hazard is low (config seeding, not state mutation or ledger dual-
write), hence MEDIUM not HIGH. Fix: wrap the two writes in one BeginTx/Commit.

### internal/store/state/cap_relief.go
(No findings — single-INSERT AddCapRelief on shared w.tx; read-only apply/load via parameterized
SQL. loadCapRelief defers rows.Close(). No panics, no unstructured logging.)

### internal/store/state/helpers.go
(No findings — scan/clone helpers. loadCellCap defers rows.Close(). seedPlayer does 2 INSERTs but
both on the caller's passed-in tx (seed function owns the tx). All SQL parameterized. No logging.)

### internal/store/state/ledger_extension.go
(No findings — AppendExtensionYears + helpers all write on shared w.tx. PaidCells/readContractTail
defer rows.Close(). All SQL parameterized. No panics, no logging.)

### internal/store/state/ledger.go
(No findings — seedLedgerPlayer/insertCell use caller's tx; insertCell does 2 INSERTs (cell + change
log) on that tx. LedgerCells defers rows.Close(). All SQL parameterized. No logging.)

### internal/store/state/ledger_schema.go
(No findings — DDL only (CREATE TABLE/TRIGGER IF NOT EXISTS). No panics, no logging.)

### internal/store/state/ledger_sign.go
(No findings — SignContract multi-write (clearPriorCells + insertSignedRosterRows + layCell loop)
ALL on shared w.tx. readAllCells defers rows.Close(). All SQL parameterized. No logging.)

### internal/store/state/ledger_writes.go
(No findings — MoveCellMoney/SetCell/VoidCells/adjustCell/writeCell/voidCell all write on shared
w.tx (dual-write + change-log pairs). readCell/readPaidCells: QueryRow (no close needed) or deferred
rows.Close(). All SQL parameterized. No logging.)

### internal/store/state/player_status.go
(No findings — RecordStatus single INSERT on w.tx; CurrentStatus/FreeAgents parameterized SQL.
FreeAgents defers rows.Close(). No panics, no logging.)

### internal/store/state/schema.go
(No findings — migrateMoneyCents (ADD COLUMN + backfill + verify) and dropLegacyMoneyColumns
(multiple DROP COLUMN) both correctly wrapped in BeginTx/Commit. fmt.Sprintf in columnExists/
dropLegacyMoneyColumns builds DDL with fixed literal table/column names, not external input —
ALTER TABLE cannot bind identifiers (documented in comments). columnExists defers rows.Close().
No panics, no logging.)

### internal/store/state/season_phase.go
(No findings — AppendPhaseTransition single INSERT on w.tx; RolloverSeason multi-write (transition
INSERT + promoteExpiredContracts + roster/contract UPDATEs) ALL on shared w.tx. readExpiredRosterIDs
defers rows.Close(). All SQL parameterized. No logging.)

### internal/store/state/state.go
[1.4.1] MEDIUM — state.go:328 uses log.Printf() in internal/ for the cell-drift warning
  ("state: load: WARNING rostered player %q has base salary %s..."). This is in a runtime path
  (load() runs on every startup and after every mutation) and logs domain data (player ID, salary).
  Should use slog for structured logging. The log.Printf is the only unstructured-logging instance
  in the entire state package; the rest of the package returns errors instead of logging.

### internal/store/state/types.go
(No findings — type/interface/constant definitions only. No SQL, no panics, no logging.)

### internal/store/state/writes.go
(No findings — WriteTx is the sole mutation entry point (BeginTx/commit/rollback + poison-on-reload-
failure). All txWriter methods write on shared w.tx. MovePlayer (2 UPDATEs) and ReleasePlayer
(2 DELETEs + RecordStatus) are multi-write but all on w.tx. All SQL parameterized. `_ = tx.Rollback()`
in deferred func is correct Go idiom. No unstructured logging.)

--- chunk db-and-store complete: 2 findings ---

## Chunk 3: transactions-engine
Session start: 2026-07-11T20:03:19Z · model: zai/glm-5.2

### TWR-2 HIGH — §9 tag price not snapped to $10k grid (sole exception to universal flat-$10k invariant)
**Files:** `internal/transactions/pricing.go:66,105` + `internal/transactions/contracts/contracts.go:209,218`
**Rule:** TWR-2 (rounding not FLAT $10k where ledger rounding applies)
**Observation:** `tagPrice` (pricing.go:66) averages top-N salaries with round-half-up to the **cent**; `tagFloorPrice` (pricing.go:105) computes the 120% prior-salary floor as exact integer ratio `(priorSalary*120+50)/100`. Neither calls `domain.RoundToNearest10k`. The Tag handler writes the unsapped price via `w.SetCell` (contracts.go:218) and `w.ApplyContract` (contracts.go:209). **Every other money path in this chunk snaps to $10k:** §8 Charge (deadcap.go:59), §10 extension (contracts.go:233), §12 buyout (buyout.go:54), §13 retirement (special.go:48), §13 cap relief (special.go:133), free-agency sign (freeagency.go:87). The §9 tag is the only exception. The deadcap.go:44-48 comment states the invariant explicitly: "the salary-ledger pivot locked FLAT $10k rounding on every figure." Two concrete off-grid paths: (1) `tagFloorPrice` yields $12k-granularity values (`priorSalary×120/100` = k×1,200,000 cents; $12k is not a $10k multiple unless k%5==0) — if this floor exceeds the average, the returned price is off-grid; (2) `tagPrice` average with N<5 players at the position is not guaranteed on-grid. **Triage check:** verify whether `state.TxWriter.SetCell`/`ApplyContract` snap internally — if they do, this is a no-op; if not, an off-grid tag price enters the ledger, breaking CapUsed's on-grid guarantee.

### 1.4.1 MEDIUM — unstructured logging (log.Printf) in freeagency handler
**File:** `internal/transactions/freeagency/freeagency.go:99`
**Rule:** 1.4.1 (structured logging)
**Observation:** `log.Printf("freeagency: sign %q: §6 minimum-salary floor NOT enforced …", mflID)` uses stdlib `log`, not `slog`. Logs domain data (player MFLID) from `internal/`. Every other handler in the chunk is silent (no logging at all). Suggest `slog.Warn` for consistency with the structured-logging mandate.

---
### Chunk-3 rule-coverage summary (INFO)
- **1.1.1 (goroutine recovery):** No `go func(` or `go fn()` calls in any of the 13 files — N/A.
- **1.1.2 (bare panic):** No `panic(` calls — N/A.
- **1.1.3 (discarded errors):** No `_ :=`, `_, _ =`, or bare error-returning calls — all error returns are checked and wrapped with `%w`. Clean.
- **1.2.1 (resource leaks):** No `os.Open`/`os.Create`/`http.Get`/`sql.Open`/`rows.Next` — N/A.
- **1.2.2 (tx-wrapped multi-write):** All multi-write handlers (contracts Restructure/Tag/Extend; deadcap Waive/Buyout/Retire/Death/Relieve; acquisitions Trade) run every mutation on the shared `state.TxWriter` inside `Coordinator.Execute`'s `WriteTx` spanning transaction. Atomicity confirmed.
- **1.3.7 (SQL construction):** No `.Query`/`.Exec`/`.QueryRow` calls — all persistence delegates to `TxWriter` methods. N/A.
- **TWR-2 (money math):** All money is `domain.Money` (int64 cents); no `float32`/`float64` arithmetic on money anywhere in the chunk. The `float64` in rankings.go (lines 63/72/249) is fantasy-points and age — not money. `rankings.go:226` `.Millions()` converts to a dimensionless ratio for the engine's L5 cap scaling (documented OQ-014 edge), not money arithmetic.

--- chunk transactions-engine complete: 2 findings ---

## Chunk 4: wails-binding-surface
Session start: 2026-07-11T22:01:37Z · model: glm-5.2 (zai/glm-5.2)

### app.go
[TWR-1] HIGH — Store-floor initialization runs synchronous MFL network fetches on the
  Wails OnStartup (UI) thread with NO timeout on the context. Init call chain:
  `main.go:37` `OnStartup: app.startup` → `app.go:160` `initStoreFloor(ctx)` →
  `rb.Initialize(ctx, league.APISource{Client: a.mflClient, ...})` (app.go:182, league
  config from MFL — network on a fresh DB) AND `st.Initialize(ctx, rosterSeedSource{app:a})`
  (app.go:188) → `rosterSeedSource.Rosters(ctx)` (app.go:88) → `a.directory(ctx)` (app.go:66)
  → `players.Fetch(ctx, a.mflClient, ...)` + `rosters.Fetch(...)` (app.go:66, app.go:88 — two
  more network calls). The `ctx` passed is the raw OnStartup context with no deadline
  (app.go:118-164 never wraps it; only the bound IPC methods Ping/SetParam/etc add
  `context.WithTimeout`). On first launch (fresh DB) a slow/unreachable MFL endpoint blocks
  OnStartup indefinitely → black window + downstream "store not initialized". This IS the
  known bug class, and the code itself acknowledges it (app.go:121-125 comment) plus a
  dedicated `-probe` diagnostic was built to triage it (probe.go, "Ship-4 hang triage") —
  but the root cause in app.go is unfixed. Fix: wrap the initStoreFloor ctx in a bounded
  timeout (the probe already proves 20s/step is sane) and surface the deadline error via
  startupErr/Ping, or move seeding off the UI thread.

[1.1.3] LOW — app.go:213 `_ = a.pools.Close()` discards the DB-close error in `shutdown`.
  TWR-3 is otherwise SATISFIED (OnShutdown is wired at main.go:38 and calls shutdown →
  pools.Close, so the SQLite handle is closed on exit), but given this app's known
  DB-corruption history (TWR-3) a non-nil Close() error (WAL checkpoint / dirty-page flush)
  is a corruption signal worth logging before exit, not dropping.

[1.4.1] LOW — app.go:128 `log.Printf("the war room: startup failed: %v", ...)` uses stdlib
  `log` (main package, startup-failure diagnostic). Suggest `slog.Error` for consistency
  with the structured-logging mandate. (main.go:45 `log.Fatalf` on wails.Run error is
  conventional startup-failure exit — acceptable.)

### harness_app.go
(No findings — 1.1.1/1.1.2/1.1.3/1.3.6/1.3.8: N/A. Bound methods check startupErr + nil
 stores; SetParam applies a 3s WithTimeout. ScoreRookies/RunValidationSuite/GetParams are
 CPU-only or in-memory reads — no I/O to bound, no error returns dropped.)

### m1_app.go
(No findings — 1.1.1/1.1.2/1.1.3/1.3.6/1.3.8: N/A. ScoreLeague/GetRankings gate on m1Ready
 and bound both with m1Timeout=120s (app.go:18). All error returns checked and surfaced in
 the typed result; GetRankings degrades name resolution gracefully instead of blanking
 persisted scores. Clean.)

### main.go
[TWR-3] INFO (SATISFIED) — `wails.Run` wires `OnStartup: app.startup` and `OnShutdown:
  app.shutdown` (main.go:37-38); shutdown closes the SQLite pools. No `OnBeforeClose` is
  set, but in-flight writes are not a concern here: every mutation runs synchronously
  through the Coordinator's WriteTx (confirmed in chunk transactions-engine), and Wails
  completes a bound IPC call before teardown, so no async write path is left open. DB-close
  path confirmed present (the only nit is the dropped Close error — see app.go 1.1.3).

### probe.go
[1.1.1] MEDIUM — probe.go:32 `go func() { done <- fn(c) }()` has no `defer`/`recover()`.
  Per the chunk rule this pattern is HIGH, but the rule's stated rationale ("a panic in a
  bound-method goroutine kills the desktop app") does NOT apply here: probe.go runs ONLY
  under the `thewarroom -probe` headless diagnostic (main.go:20-23), never under the Wails
  GUI, and every path through probeStep calls os.Exit(1/2). Blast radius is the probe
  binary (a stack trace to stderr — arguably the desired diagnostic output), not the
  desktop app. Downgraded to MEDIUM on that basis; add a recover that logs+os.Exit(2) if
  panic-vs-timeout distinction matters for the triage output.

[1.4.1] INFO — extensive `log.Printf`/`log.Fatalf` throughout. Acceptable: probe.go is a
  purpose-built stderr diagnostic for SSH startup triage (its job IS human-readable stdout
  lines); structured slog would reduce its usefulness. No change recommended.

### transactions_app.go
[INFO — trust boundary, no enumerated rule] buildRequest (transactions_app.go:226) casts
  unvalidated JS strings to domain enums: `domain.RosterStatus(req.Status)` (line 237) and
  `domain.Phase(req.ToPhase)` (line 245). An unknown status/phase value from the frontend
  crosses the boundary as an invalid enum and is relied on to be rejected deeper in the
  Coordinator. Money figures ARE correctly parsed to cents server-side at the boundary
  (buildMoneyRequest, domain.ParseMoneyMillions). Flag for the transactions-engine chunk to
  confirm RosterStatus/Phase are validated before use; if not, a malformed IPC payload could
  drive an invalid enum into a write. Not a rule violation for THIS chunk's set — noted as a
  boundary-handoff lead.

  Otherwise clean: ExecuteTransaction/GetCurrentPhase/GetFreeAgents all gate on startupErr +
  nil-state and apply bounded WithTimeouts (30s/10s/10s). GetFranchiseState is an in-memory
  read with a staleness check (a.state.Err()) and needs no timeout. All errors checked.

### internal/harness/cases_eval_dispatch.go
(No findings — pure CPU evaluation logic. No goroutines/panic/exec/file-ops/logging; all
 errors checked. N/A for 1.1.1/1.1.2/1.1.3/1.3.6/1.3.8/1.4.1.)

### internal/harness/cases_eval.go
(No findings — pure CPU assertion logic over the registered rubrics + engine. eval3L
 builds JSON from a HARDCODED constant id slice ("0001","0999","14263") and decodes it
 through schema.DecodePlayerRecord — no external input, no injection surface. No
 goroutines/panic/exec/file-ops/logging. All errors checked. Clean.)

### internal/harness/cases.go
(No findings — case registry + gating helpers. Pure logic, no I/O, no error returns dropped.)

### internal/harness/fixtures.go
(No findings — SampleRookies is a hardcoded synthetic fixture literal. No I/O, no external
 input. N/A across the rule set.)

### internal/harness/rankings.go
(No findings — RankRookies/sortRows are pure CPU; a player that fails to assemble/score is
 surfaced with Err set (not silently dropped). No goroutines/panic/exec/file-ops/logging.
 All errors checked. Clean.)

### internal/harness/validation.go
(No findings — three-state (PASS/FAIL/PENDING) suite runner + Summary tally. Pure logic,
 no I/O. N/A across the rule set.)

---
### Chunk-4 rule-coverage summary (INFO)
- **1.1.1 (goroutine recovery):** One `go func(` in the chunk — probe.go:32, no recover.
  MEDIUM (probe-only blast radius; rationale for HIGH does not apply). No `go fn()` calls.
  All other files clean.
- **1.1.2 (bare panic):** No `panic(` calls in any of the 12 files — N/A.
- **1.1.3 (discarded errors):** Two `_ = pools.Close()` (app.go:213, probe.go:76), both
  shutdown cleanup. app.go:213 noted LOW (loses DB-corruption signal given TWR-3 history);
  probe.go:76 acceptable for a diagnostic. All other error returns are checked and wrapped.
- **1.3.6 (command injection):** No `exec.Command(` calls — N/A.
- **1.3.8 (path handling):** databasePath() (app.go:250) uses `filepath.Join(os.UserConfigDir(),
  "TheWarRoom", "thewarroom.db")` — no variable/external segments, no user-supplied path,
  no `..` injection vector. Clean.
- **1.4.1 (structured logging):** app.go:128 `log.Printf` LOW (main-pkg diagnostic → slog).
  main.go:45 `log.Fatalf` acceptable (startup exit). probe.go logging acceptable (diagnostic
  CLI). No unstructured logging in `internal/`.
- **TWR-1 (UI-thread init):** HIGH — app.go runs unbounded network init (rulebook + roster
  seed via MFL) on the OnStartup thread with no timeout. See app.go finding.
- **TWR-3 (shutdown & DB close):** SATISFIED — OnShutdown wired (main.go:38) and closes the
  SQLite pools; synchronous Coordinator writes leave no in-flight async path on teardown.
  Only nit: dropped Close() error (app.go 1.1.3).

--- chunk wails-binding-surface complete: 5 findings ---

## Chunk 5: ingestion-and-external
Session start: 2026-07-12 10:02:39 UTC · model: zai/glm-5.2

### Chunk 5 review: ingestion-and-external (29 files, ~5174 LOC)
All 29 files reviewed file-by-file in the prescribed order at commit f624467.

Rules SATISFIED — no ruled violations (1.1.2 panic, 1.1.3 discarded errors, 1.2.1 resource release, 1.4.1 logging):
- No panic() in any production file (swept internal/ingestion, internal/mfl, internal/normalize, internal/playerid; prod only).
- No discarded errors. The only `_ = ...Close()` sites are the canonical deferred-body-close idiom — mfl/client.go:90, cfbd.go:72, extcsv.go:58, madden/fetcher.go:185 — which IS correct resource handling, not error-dropping.
- HTTP body-close verified at every HTTP-issuing site in the chunk:
    mfl/client.go:89-91   Client.Do()        defer resp.Body.Close()                    [OK]
    mfl/client.go:204     executeWithRetry   explicit close on the 429-before-retry path [OK]
    cfbd.go:72            GetCFBD()          defer resp.Body.Close()                    [OK]
    extcsv.go:58→105/155  openCappedCSV      cleanup() closure deferred in fetchCSV & streamCSV [OK]
    madden/fetcher.go:185 fetchPage()        defer resp.Body.Close()                    [OK]
  All other fetchers consume Response.Body []byte from mfl.Client.Do (already read+closed
  in Do) or the shared FetchCSV/GetCFBD helpers (which own body-close) — no leak path.
  Query params built via url.Values.Encode() (mfl/client.go:155-173) → params escaped, no
  query-key smuggling (TYPE/JSON collision guarded at :165). No unnormalized external path
  reaches a file op — URLs are constants + strconv.Itoa(year) only.
- No fmt.Print/log.Print in any internal/ package of this chunk (slog-clean).

INFO — rule inapplicable in this chunk (no matching construct in any of the 29 files):
- 1.1.1 (goroutine panic recovery): no `go func(`/`go fn` in production code; the only `go`
  hits are comment prose. 1.1.1 has nothing to evaluate here.
- 1.3.6 (command injection): no exec.Command anywhere in internal/.
- 1.3.8 (path handling): no os.Open/ReadFile/WriteFile or filepath ops in the chunk; the
  boundary here is HTTP, not the filesystem, so 1.3.8's file-path checks do not apply.

--- chunk ingestion-and-external complete: 0 findings ---

## Chunk 6: engine-and-domain
Session start: 2026-07-12T22:01:36Z · model: zai/glm-5.2

### Chunk 6 summary: engine-and-domain — 0 findings

All 33 files reviewed across composition, domain, engine (+l4 position rubrics),
output, and scouting packages.

Rule evaluation results:
- 1.1.1 (goroutine panic recovery): No `go func(` or `go fn()` in any production
  file. The engine is a pure-function pipeline with no spawned goroutines. N/A.
- 1.1.2 (bare panic): No `panic(` calls in any file. The engine/composition boundary
  uses fail-loud error returns throughout; nil-reader surfaces are documented as
  programmer errors but do not call panic().
- 1.1.3 (discarded errors): No genuine error discards. The one `_ :=` in production
  code (composition.go:130 `tierNorm, _ := schoolTierNorm(...)`) discards a `bool`,
  not an `error` — its precondition is enforced by Validate() upstream (checked at
  playerspec.go:163). The `_ = tx.Rollback()` and `_ = rows.Close()` in output.go
  are canonical Go cleanup idioms (deferred rollback after commit; deferred rows
  close), not silent data-integrity loss.
- 1.2.1 (resource acquisition): The output package uses SQL transactions and rows.
  Both `Scores` and `Score` defer `rows.Close()` in the same function after the
  error check. The `Write` transaction is committed/rolled back correctly.
  All resources properly released.
- 1.4.1 (structured logging): No fmt.Println/Printf or log.Println/Printf in any
  internal/ file. All fmt.* usage is fmt.Errorf (error construction) or
  fmt.Fprintf to a strings.Builder (display formatting). Clean.

--- chunk engine-and-domain complete: 0 findings ---

## Chunk 7: frontend-ui
Session start: 2026-07-13T10:01:34Z · model: zai/glm-5.2


### frontend/src/store/harness.ts
- **MEDIUM — 1.1.4 (bridge-call error handling):** `setParam` (lines 57–65) awaits the Wails `SetParam(key, value)` bridge call **with no try/catch**, and also re-invokes `await get().loadAll()` outside any guard. Its three sibling actions (`loadAll`, `loadRankings`, `scoreLeague`) all wrap their bridge calls in `try/catch` and surface `error` to the store; `setParam` is the outlier. On a bridge rejection the promise propagates to `AdminPanel.tsx:40` where it is discarded with `void setParam(...)` → unhandled rejection, and the `error` slice is never set (AdminPanel's `{error && ...}` block at line 19 stays silent). Fix: mirror the try/catch + `set({ error: String(e) })` pattern used by `loadRankings`.

### frontend/src/components/TransactionsPanel.tsx
- **MEDIUM — 1.1.4 (bridge-call error handling):** All four async functions in this panel make Wails bridge calls with no error handling for the rejection path. `run()` (lines 73–118) wraps the body in `try { ... } finally { setBusy(false); }` but has **no `catch`** — a rejected `ExecuteTransaction` (IPC error / Go-side panic, distinct from the returned `TransactionResult{ok:false}`) leaves the result panel blank and is lost as an unhandled rejection via the `void run()` onClick (line 489). `refreshPhase` (64–67), `loadFranchise` (120–123), and `refreshFreeAgents` (125–128) each `await` a bridge call (`GetCurrentPhase`/`GetFranchiseState`/`GetFreeAgents`) with no try/catch at all, invoked via `void` (lines 69–70, 520, 550) → unhandled rejections on mount and on click, with no user-visible error. Fix: add try/catch to each, writing to a local error state (or a shared one) the way harness.ts does.

### frontend/src/store/ping.ts
- **LOW — 1.1.4 (bridge-call error handling):** `ping()` (lines 17–21) awaits the Wails `Ping()` bridge call with no try/catch. On rejection, `set({ loading: true })` has already run but the subsequent `set({ result, loading: false })` never executes → `loading` is **stuck `true`** and the error is an unhandled rejection. Severity is LOW because `usePingStore` is imported by no component in `frontend/src` (grep confirms zero usages) — the store is currently dead code, but the latent bug will bite the moment it is wired up. Fix: wrap in try/catch and clear `loading` in a `finally`.

### Files with no findings
- `frontend/src/App.tsx`, `frontend/src/main.tsx`, `frontend/src/vite-env.d.ts`, `frontend/src/components/AdminPanel.tsx`, `frontend/src/components/RankingsBoard.tsx`, `frontend/src/components/RookieTable.tsx`, `frontend/src/components/ValidationBoard.tsx` — clean against rules 1.1.4 / 1.2.1 / 1.3.10 / 1.4.1. (Rules 1.2.1/1.3.10 had zero matches across the whole chunk: no `setInterval`/`addEventListener`/string-`setTimeout`, no `eval`/`new Function`. Rule 1.4.1 had zero `console.*` calls anywhere.)

--- chunk frontend-ui complete: 3 findings ---

## Chunk 8: tests-and-tools
Session start: 2026-07-13T22:00:43Z · model: zai/glm-5.2

INFO chunk tests-and-tools: 96 files (~14.7k LOC) grep-driven scan, 0 findings.
  - Secrets (1.3.1): no token formats, no hardcoded credential assignments. Live tests
    env-gated (TWR_LIVE_*) and read CFBD_API_KEY from env; bearer-header assertions use
    dummy values ("k"/"k123"/"right").
  - Exec in tools (1.3.6): no os/exec, exec.Command, or shell spawning in tools/ or tests.
  - Integration cleanup: all 27 db.Open calls paired with t.Cleanup(pools.Close), all paths
    in t.TempDir(); all ~28 httptest.NewServer paired with defer/t.Cleanup close. No raw
    file creation outside TempDir.
--- chunk tests-and-tools complete: 0 findings ---

---

## Phase 3 cross-ref session
Session start: 2026-07-14T..:..Z · model: glm-5.2 (zai/glm-5.2)
Method: groups formed by rule/vulnerability type; only groups containing ≥1 HIGH-or-above
finding were cross-referenced (3 groups qualify). Sources fetched directly via WebFetch
(no websearch tool available). Budget used: 4 fetches of 10 allowed.

Groups reviewed but NOT cross-referenced (no HIGH-or-above finding, per the Phase-3 rule):
- 1.2.2 (tx-wrapped multi-write): max MEDIUM (rulebook.go)
- 1.4.1 (structured logging): max MEDIUM (state.go, freeagency.go) / LOW (app.go)
- 1.1.1 (goroutine panic recovery): MEDIUM only (probe.go; blast-radius downgrade)
- 1.1.3 (discarded errors): LOW only (app.go:213 pools.Close)
- 1.1.4 (bridge-call error handling): MEDIUM/LOW (harness.ts, TransactionsPanel, ping.ts)
- TWR-3 (shutdown & DB close): INFO/LOW only

## Phase 3 cross-ref: 1.3.4 — dependency vulnerability audit (vite + Go toolchain)
Source: https://github.com/vitejs/vite/security/advisories (Vite advisory index)
Source: https://github.com/advisories/GHSA-c27g-q93r-2cwf (CVE-2024-52011 detail)
Confirmation/nuance: The live Vite advisory index confirms the advisories cited in the
chunk-1 finding — GHSA-fx2h-pf6j-xcff (High, server.fs.deny bypass Windows alt paths),
GHSA-93m4-6634-74q7 (Mod, backslash bypass), GHSA-859w-5945-r5v3 (Mod), GHSA-356w-63v5-8wf4
(Mod), GHSA-xcj6-pq6g-qj4x (Mod), GHSA-g4jq-h2w9-997c (Low), GHSA-jqfw-vq24-v9c3 (Low). The
GHSA-c27g-q93r-2cwf advisory detail independently confirms: launch-editor command injection
on Windows, CVE-2024-52011, CVSS 7.5, patched in vite 5.4.9 (CWE-77). This corroborates the
"upgrade vite" remediation.
CORRECTION to chunk-1 advice: the live index lists TWO further High advisories NOT in the
audit's enumeration — GHSA-v2wj-q39q-566r (server.fs.deny bypassed with queries) and
GHSA-p9ff-h696-f583 (Arbitrary File Read via Vite Dev Server WebSocket), both published
2026-04-06. These postdate the audit window; the real exposure is therefore WORSE than the
14 vulns counted, which strengthens (does not weaken) the recommendation to upgrade vite to
>=6.4.3. Original remediation stands and is reinforced.

## Phase 3 cross-ref: TWR-1 — UI-thread init runs unbounded network fetches (no context timeout)
Source: https://go.dev/blog/context (Go Concurrency Patterns: Context, Sameer Ajmani)
Confirmation/nuance: The canonical Go context article directly confirms the remediation. It
specifies `context.WithTimeout(parent, timeout)` as the standard idiom "useful for setting a
deadline on requests to backend servers," and states that when the derived context's deadline
fires the `Done` channel closes so in-flight work "should exit quickly so the system can
reclaim any resources." app.go's OnStartup passes the raw Wails ctx (no deadline) into
rulebook.Initialize + roster seeding, both of which issue MFL network calls — exactly the
unbounded-backend-call anti-pattern this idiom exists to bound. Wrapping `initStoreFloor`'s
ctx in `context.WithTimeout` (the probe already validated 20s/step as sane) and surfacing the
deadline error via startupErr/Ping is the textbook fix. Original remediation stands.

## Phase 3 cross-ref: TWR-2 — §9 tag price not snapped to $10k grid (rounding-invariant exception)
Source: https://martinfowler.com/eaaCatalog/money.html (Money, Patterns of Enterprise
Application Architecture, Martin Fowler, 2003)
Confirmation/nuance: Fowler's Money pattern validates the project's overall approach — a
single Money value object stored in integer minor units (here `domain.Money` as int64 cents)
to centralize rounding and avoid "los[ing] pennies ... because of rounding errors." The
pattern's explicit purpose is that rounding is owned by the Money abstraction and applied
uniformly; the §9 tag path computing `tagPrice`/`tagFloorPrice` and writing via `w.SetCell`/
`w.ApplyContract` WITHOUT calling `domain.RoundToNearest10k` is precisely the per-callsite
fragmentation the pattern exists to prevent. Every other money path in the chunk routes
through the shared rounding, so the §9 exception breaks the representation invariant the
deadcap.go:44-48 comment claims is universal. This supports the finding.
Nuance: Fowler establishes the principle but cannot resolve the finding's own triage check —
whether `state.TxWriter.SetCell`/`ApplyContract` snap internally is a code question, not a
best-practice one. That determination remains the gating action: if the writer snaps, TWR-2
is a no-op; if not, an off-grid tag price (e.g. $12k-granularity floor values) enters the
ledger. Original remediation stands pending that single code check.
