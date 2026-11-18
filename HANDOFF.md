# Handoff

- Baton: Bee, 2026-11-18 (Bundle 5 lane b5fix; PR #358 open against `bundle/b5`).
- State: Hosted Fix wires the nine actTable forms, reversible ledger entries and correction/calendar exceptions. Reversal uses `commish.correction.revert`; no new server read/command. 32-team and preview → review → execute behavior is tested.
- Scope: thin hosted Fix tab only; local mode unchanged.
- Verified locally: frontend typecheck; frontend suite (127 files, 923 tests); `make lint` (0 golangci-lint issues); `make bloat`; targeted Fix tests; pre-push verify.
- CI: latest full run 44245483826 passed all four race-test shards and verify; package/install run 44245483883 passed. Earlier run 44241548732 caught an intermittent race in `internal/serverclient.TestUnavailableKeychainLoginIsMemoryOnly` (test read at `credentials_test.go:119`, concurrent `Desktop.clearCredential` write at `credentials.go:64`); a later full run passed unchanged. No Go files changed in this lane. Flagged for ClaudeBox; latest checks are green, but the earlier race should be reviewed before merge.
- Next: ClaudeBox reviews PR #358 and decides whether to triage the intermittent `serverclient` race before merge into `bundle/b5`.
- Other work in flight: tidy-check reports launchers for b5perm, b0load2, b3home and b5league. This lane did not start them; their scratch and KEEP files are untouched under the requested write boundary.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Slop fixed: revert choices filter on server-provided undo eligibility/replacement state; calendar exceptions use the existing calendar read.
