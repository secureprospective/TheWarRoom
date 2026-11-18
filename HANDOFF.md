# Handoff

- Recent main handoff: PR #374 (player-search name matching) merged into `main` on 2026-11-18.
- Baton: Bee, 2026-11-18; lane `pulse-fixhome`, branch `session/pulse-fixhome`, based on `origin/main` `0c185549e`.
- State: Commissioner Home odds seeds cross JSON as exact decimal strings; frontend accepts decimal strings or legacy safe nonnegative integers. Current-week selection uses the date-covered schedule prefix, so absent later NFL dates no longer suppress an already-determined week; if current dates are missing, it still refuses to guess. An unpublished schedule gives the commissioner the action: Generate schedule, then Publish schedule, in Control Room > Season.
- Regressions failed before fixes and pass now: `TestNativeOddsSeedIsSerializedAsDecimalString`, `TestNativeWeekReadsCurrentBeforeMissingFutureDates`, `TestNativeWeekExplainsHowCommissionerPublishesMissingSchedule`; frontend covers exact-seed parsing and Pulse's action message.
- Verification: `make verify` passes on the final base (full Go race suite, TypeScript, 138 files/980 tests, frontend build, bundle budget, CSP); `make build` passes. Targeted Go race tests and Vitest pass. The initial 600s outer timeout ended before Go tests; rerun with sufficient duration passed.
- PR #376 is open into `main`; CI checks, all four test shards, package, install and verify passed. Body has two-line root cause and red/green test names.
- Lab inspection was read-only; `/var/log/twr-server.log` absent and checked `/var/log/messages` tail had no relevant diagnostics. No service/data writes or restarts; no permission checks changed.
- `git diff --check` passes. The latest `tidy-check.sh` found unrelated foreign-lane activity and scratch `~/scratch/resume-v15-rollout-1118` without `KEEP.md`; left them untouched because they are outside this lane. This lane created no scratch files.
- Next: ClaudeBox/Christopher review PR #376; do not merge without Christopher's go-ahead.
- Blocked on: none. Rejected: two initial Vitest commands omitted `cd frontend` and ran no tests; corrected. Fixed two long frontend lines found by the first full Vitest.
