# Handoff

- Baton: Bee, 2026-11-18 (Bundle 5 lane b5fix; PR #358 open against `bundle/b5`).
- State: Hosted Fix wires the nine actTable forms, reversible ledger entries and correction/calendar exceptions. Reversal uses `commish.correction.revert`; no new server read/command. 32-team and preview → review → execute behavior is tested.
- Scope: thin hosted Fix tab only; local mode unchanged.
- Verified locally: frontend typecheck; frontend suite (127 files, 923 tests); `make lint` (0 golangci-lint issues); `make bloat`; targeted Fix tests; pre-push verify.
- CI blocker: latest PR run 44241548732 failed shard 3's race detector in `internal/serverclient.TestUnavailableKeychainLoginIsMemoryOnly`: read at `credentials_test.go:119` races with `Desktop.clearCredential` at `credentials.go:64` in the invalidation watcher. That package is unchanged from `bundle/b5`; no Go files changed in this lane. Other CI shards and package/install passed; `verify` failed downstream.
- Next: ClaudeBox triages the existing `serverclient` race before PR #358 can be considered green, then review/merge into `bundle/b5`.
- Other work in flight: tidy-check reports launchers for b5league, b5people, b5picks, b5perm, b0load2 and b3home. This lane did not start them; their scratch and KEEP files are untouched under the requested write boundary.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Slop fixed: revert choices filter on server-provided undo eligibility/replacement state; calendar exceptions use the existing calendar read.
