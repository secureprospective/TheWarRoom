# Handoff

- Baton: Bee, `lane/flaky2`, 2026-11-19; PR #401 into `bundle/b9`.
- State: NativeAccount's flaky test asserted HostedLeague's mount before React.lazy resolved; it now awaits visible content and effect inside `act` (20/20 focused runs). Serverclient race is already fixed in this base via #379 (valid SSE fixture and mutex-protected snapshot; 50/50 runs). Roster-add failure was not reproduced (50/50 runs); cause remains unknown. Initial CI exposed another test-only race: parallel MFL test servers close shared `http.DefaultTransport`; fixture now clones the transport per test. Fix passes `TestDo_HostRouting` 50/50 and `go test -race ./internal/mfl/...`.
- Verification: `make test`, frontend tsc, full Vitest (153 files/1026 tests), and pre-push passed before the MFL fixture fix. Initial PR CI failed only in `TestDo_HostRouting`; other checks and race shards passed.
- Next: Commit/push the MFL test-fixture fix, update PR #401's description, and wait for CI to pass on the new head.
- Blocked on: PR CI rerun.
- Tried and rejected: no sleeps, retries, skips, or production behavior changes.
