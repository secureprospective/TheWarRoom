# Handoff

- Baton: Bee, `lane/flaky2`, 2026-11-19; PR #401 into `bundle/b9`.
- State: NativeAccount's flaky test asserted HostedLeague's mount before React.lazy resolved; it now awaits visible content and effect inside `act` (20/20 focused runs). Serverclient race is already fixed in this base via #379 (valid SSE fixture and mutex-protected snapshot; 50/50 runs). Roster-add failure was not reproduced (50/50 runs); cause remains unknown. CI exposed a shared `http.DefaultTransport` race in parallel MFL tests; `newTestClient` now clones a transport per fixture (50/50 host-routing runs).
- Verification: `make test`, frontend tsc, full Vitest (153 files/1026 tests), and pre-push passed. After the MFL fixture fix, `go test -race ./internal/mfl/...` passed. PR CI on `84db6d256` passed all four race shards, checks, release tools, and desktop package/install.
- Next: Await review and merge of PR #401; retain the note that the roster-add failure's cause is unknown.
- Blocked on: none.
- Tried and rejected: no sleeps, retries, skips, or production behavior changes.
