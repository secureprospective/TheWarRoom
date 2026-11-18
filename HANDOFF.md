# Handoff

- Baton: Bee, 2026-11-18 (Bundle 5 lane b5fix; PR #358 open against `bundle/b5`).
- State: Hosted Fix wires the nine actTable forms, reversible ledger entries and correction/calendar exceptions. Reversal uses `commish.correction.revert`; no new server read/command. 32-team and preview → review → execute behavior is tested.
- Scope: thin hosted Fix tab only; local mode unchanged.
- Verified after rebasing on #350: `pnpm exec tsc --noEmit`; frontend `pnpm test` (127 files, 923 tests); `make lint` (0 golangci-lint issues); `make bloat`; targeted Fix tests. Pre-push `verify` passed for changed packages. No Go files changed.
- Next: wait for PR #358 CI, then hand off for review into `bundle/b5`.
- Blocked: CI was queued at last check. `tidy-check.sh` reports other active launchers (b5league, b5people, b5picks, b5server, b2docket); they were not started by this lane and remain running. Their scratch is untouched under the requested write boundary.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Slop fixed: revert choices filter on server-provided undo eligibility/replacement state; calendar exceptions use the existing calendar read.
