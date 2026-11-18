# Handoff

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
