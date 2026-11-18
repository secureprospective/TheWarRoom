# Handoff

- Baton: Bee, 2026-11-18; lane `pulse-fixrace`, branch `session/pulse-fixrace`.
- State: stabilized `TestUnavailableKeychainLoginIsMemoryOnly`: the test server keeps the invalidation stream open instead of racing the test's credential assertions against the watcher's asynchronous 401 sign-out; test snapshots are taken under `Desktop.mu`. Production state transitions are serialized there.
- Evidence: historical failed-shard trace for PR #376 was unavailable; current PR #376 checks were green. `go test -race -count=200 -run TestUnavailableKeychainLoginIsMemoryOnly ./internal/serverclient/`, `go test -race ./internal/serverclient/...`, `make verify`, pre-push, and all PR #379 checks pass. PR body marks source locations explicitly as not recovered race-detector frames.
- PR #379 is open into `main`: https://github.com/secureprospective/TheWarRoom/pull/379. All test shards, checks, package/install and release-tools pass.
- Next: leave PR for review; do not merge.
- Tidy-check reports unrelated active lane launcher PID 1361666; left it untouched.
- Blocked on: original failed-shard trace was unavailable.

- Baton: Bee, 2026-11-18; lane `pulse-fixcr`, branch `session/pulse-fixcr`, based on `origin/main` at `0c185549e` (#374).
- State: added hosted workflow routes for Jobs, People, managed Schedule, Ledger/filtered Ledger, Notices and Calendar, reusing existing service authorization and Raw validators. Season now supplies a server-held franchise to Calendar.
- Root cause: the commands were absent from `commandcatalog.WorkflowCommands`; `RawWorkflow.Validate` rejected them before auth/admission (HTTP 400), hidden behind the UI's generic failure. Calendar also received an empty franchise. No permissions were weakened.
- Regression: `TestHostedCommissionerControlRoomReads` failed before fix (HTTP 400), now passes; exercises an unseated commissioner, owner denial for Jobs/Schedule, and ledger preservation.
- Verified: focused + package race tests; nativegen freshness; Season Vitest; `pnpm exec tsc --noEmit`; full Vitest (138 files/977 tests); `make lint`; final `make verify` pass (race suite, frontend tests/build, bundle and CSP checks).
- Lab logs: correct path `/var/lib/twr/server.log` per `deploy/server/twr-server.initd`; requested `/var/log/twr-server.log` absent. No matching Control Room log entries because validation failed before workflow logging.
- Tidy-check reports foreign `b7lab` launcher (PID 1140150) and `claudeos` SSH tunnel (PID 1170685); left untouched. Their lane owner should handle its scratch `KEEP.md`; this lane cannot write another lane's files.
- PR #377 is open into `main`: https://github.com/secureprospective/TheWarRoom/pull/377. CI is running; release-tools passed, package/tests/checks are pending.
- Next: wait for all PR checks green, then leave the branch for review; do not merge from this lane.
- Blocked on: CI completion.

- Baton: Bee, 2026-11-18; lane `pulse-perf`, branch `session/pulse-perf` at `28cd636a0`.
- State: local 128/256-client floods and CPU/mutex/block profiles complete. Lab5's ramp refusal spike did not persist in its 128-client hold; served throughput matches lab4's 256-client hold. The 64-request admission gate is unchanged; no safe code regression fix identified, no source changed.
- Evidence: local pprof hot path is session/league SQL; mutex wait is the cold reply-cache `sync.Once` fill. Lab4→lab5 diff leaves limits, reply cache and session query unchanged.
- Next: review the investigation result; rerun lab ramp only if a matched warmup/host comparison is needed.
- Blocked on: none.
