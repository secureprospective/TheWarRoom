# Block 0 truth pass (2026-10-19)

Block 0 measured the guesses the client/server plan (`Client_Server_Plan_2026-10.md`) rests on. Every row below was measured on 2026-10-19 unless it says otherwise. Logs are on the Beelink in `~/logs/TheWarRoom/block-0/`; Bee's code read is `~/.pi/agent/bee-warroom-block0-code-audit.out`.

## Results
| # | Guess | Result |
|---|---|---|
| 2 | Ledger replay is fast enough | **No: the plan's biggest risk.** See "Replay" below. |
| 3 | The engine can leave package main | **Yes.** The engine packages (commissioner, nativecheck, store/history, leagueclock) import no Wails, no MFL and nothing from package main. The server needs about 1,910 lines of adapters rebuilt across 16 root files. Couplings to watch: App/Wails lifecycle and identity (`app.go:39-70`), MFL-derived facts and seed data (`native_legality_app.go:22-55`, `native_seed_app.go:64-113`), and `TargetClock` merging the MFL calendar (`target_app.go:100-123`). |
| 4 | Cloudflare Tunnel keeps SSE open | **Not tested** (Christopher: no throwaway quick tunnel). Design for it: a heartbeat every 30 s (Cloudflare drops idle connections at about 100 s) plus automatic reconnect. Verify on the real tunnel. |
| 5 | The Proxmox host has room | **Yes.** 31 GB RAM with about 8 GB available, 12 threads, 597 GB free on `tank`. The server CT needs about 1 GB. |
| 6 | Windows/macOS builds | macOS **compiles** (`GOOS=darwin`, no cgo; a real Wails/macOS build is still untried). Windows **fails in one file only**: `instance_lock.go` (`syscall.Flock`). |
| 9 | CI blocks bad merges on main | **No.** Branch protection on `main` has no required status checks (enforce_admins and linear history are on). Christopher adds `verify` as a required check. |
| 11 | Argon2id at 64 MiB, t=3, p=1 on the CT | **Fine.** On the host's i7-8700: 157 ms alone, 187–208 ms for 2 at once, about 300 ms for 4. 128 MiB peak with the 2-at-once cap. |

## Replay
`TestReplayCost` (`internal/store/history/replay_bench_test.go`; opt-in with `TWR_REPLAY_N`) grows one league through the real append path and times a full `CommissionerLedger` read:

| Events | One append | Full ledger read |
|---|---|---|
| 1,001 | 0.39 s | 15.5 s |
| 2,001 | 1.59 s | 135 s |

Append cost grows about 4× when the ledger doubles (n²); reading grows about 9× (close to n³). A 600-event CPU profile puts 57% of all time in one chain: `CheckRulesAppend` → `RulesNow` → `SeasonAt`/`RulesInForce` → `rolloverInstants` → `activeRollover` → `Replay`. `rolloverInstants` (`internal/commissioner/rules_season.go:49-77`) replays the whole ledger prefix for **every Revert** and rollover entry (`rules_season.go:63`), and it runs at least twice per rules reading.

**Caveat:** the benchmark ledger is all reverts, the worst shape for this path. A real season (bids, lineups, results, with few reverts) will hurt less, but the cost is still the number of reverts times the ledger size, on every append and read. No production ledger is anywhere near 1,000 events today, which is why nobody has felt it.

**Fix (block B or C, before the server):** compute the active set once and update it incrementally (or memoize `activeRollover` by prefix length), so `rolloverInstants` stays linear. Then cache the decoded ledger and its projections by league revision (Bee, top efficiency fix 1). Rerun `TestReplayCost` to 10,000 events as the pass/fail gate. The 100k-event snapshot question from gate 1 waits for that rerun.

## Code efficiency (Bee's read, file:line in the .out)
1. Cache decoded ledger records and projections by league revision. Each `Events` read unmarshals the whole stream (`events.go:143-177`).
2. Drop redundant replays per owner action. Retry lookup, gate preflight and the transaction each load the ledger (`native_waivers_app.go:94-104`, `native_gate.go:79-103`, `commissioner.go:69-78`).
3. `NativeLeagues` scans the whole event table with no supporting index, and the waiver/bid clock calls it every second (`commissioner.go:363-380`, `native_waivers_app.go:60-91`). Add a `(stream,kind,league)` index; this also matches gate 1's sleep-until-deadline scheduler.
4. Cache parsed weekly stats by season and archive revision (`native_stats.go:20-64`).
5. Do the read-only work outside the single-writer lock (`commissioner.go:69-105`).

## Wall clock
The last green CI run took 34 min. Per package with -race, CI (GitHub runner) vs the Beelink: root 1,588 s vs 445 s; store/history 1,027 s vs 612 s; labcompile 296 s vs 240 s; facts 234 s vs 170 s; commissioner 186 s vs 151 s; lab 126 s vs 76 s.

1. **store/history has zero `t.Parallel()` calls.** Its 147 tests run one after another (610 s summed = the package's 612 s). The slowest single test is 49 s, so parallel tests take this package to about 1–2 min on the Beelink. **Biggest, cheapest win.**
2. **The replay fix above also speeds up the tests.** Tests that build ledgers pay the n² append.
3. The root package's slowest tests (50–147 s each) are archive and trade-builder tests that rebuild fixtures and DBs per test (`target_app_test.go:48-106`). Use a migrated template DB copied per test, share decoded fixtures (labcompile/facts/lab), and shard the root package in CI.
4. CI: `go install` builds wails and golangci-lint from source on every run; use prebuilt binaries or the lint action. The frontend builds twice (`verify.yml:40-43` and `Makefile:95`). `push` and `pull_request` both trigger for the same commit. Lint and tests run in series.
5. Leave the local push gate alone (172 s for Go): its time is the changed packages' tests, so fixes 1–3 shrink it too.

## What changes in the plan
- Insert **"engine replay fix + benchmark gate"** at the start of block C, before the server core. Without it the server cannot serve a season's ledger.
- Windows: the Flock fix (`LockFileEx`) is small. Move it into block B so the three-OS install check before Dec 18 can happen early.
- Christopher's clicks: make `verify` a required check on `main`.
- Still open (Christopher): the owners' OSes, whether a native league is live today, and the off-host backup destination.
