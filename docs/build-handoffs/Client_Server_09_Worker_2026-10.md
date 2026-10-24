# Server due worker

## Flow and ownership

- `cmd/twr-server` starts `leagueservice.DueWorker` in disposable fixture-change mode,
  cancels and joins it before closing the history store. It imports no desktop runtime.
- Startup, a coalesced post-commit wake, the nearest future deadline, or a one-minute
  reconciliation triggers a fresh read. Missing wakes and writes from another store
  are recovered by reconciliation. Errors are reported and retried after one second.
- Waivers process before bids; exhausted drafts close through the existing history
  processors. Their authorizers, expected revisions and append-time checks remain the
  authority. A stale scan cannot create a second award or accept a late bid/claim.
- `commissioner.NextDueDeadline` owns deadline projection beside the existing domain
  rules. The scan-start instant is retained when calculating the next timer so a
  deadline crossed during processing is not skipped until reconciliation. Already
  blocked/closed auctions wait for reconciliation rather than spinning.
- `DueWorker.Host` wraps hosted mutations without changing the shared mutation adapters.
  The store's existing credential admission calls the original credential and clock
  health inside the append transaction, before retry lookup. Successful writes wake
  scheduling only after the host returns its committed receipt; refusals do not.
- Worker admission accepts only each existing automatic principal's own processing
  command, in execute mode. Domain legality is still independently checked.

## Integration contract and limits

Construct the worker before concurrent requests; it installs the same clock in history.
Route hosted mutations through `worker.Host`. Football facts and clock health must be
thread-safe; missing clock health fails closed. Untrusted time suspends bid, waiver,
lineup and draft mutations and automatic processing, including queued writes/retries.
Recovery to trusted time resumes catch-up. Reads remain available.

Only fixture mode is executable today. Its explicit wall-clock trust is not production
clock-health evidence; deployable mode must supply that evidence. The local desktop
still needs its existing ticker: there was no hosted desktop ticker to remove.
No new public routes, credential authority, MFL calls or bindings were added.

## Evidence

`internal/leagueservice/due_*_test.go` exercises real SQLite append/reopen behavior:
nearest timers, startup/restart catch-up, lost/coalesced wakes, commit versus refusal
wakes, concurrent workers versus a late raise, late claims without worker progress,
blocked ties, narrow system authority and clock loss while a write waits for SQLite.
The short timer and blocked-tie tests also passed three consecutive race runs.
Final checks passed: targeted race suites for leagueservice, commissioner and the server
command; `go test ./...`; `make lint`; and the initial frozen-lockfile frontend build.
Lint uses a worktree-local cache to avoid stale paths from other worktrees.

Handwritten production footprint across the five affected production files: 242 → 536
physical lines (including new files); tests add 442 lines. This is added scheduling,
not a size-reduction claim. Placement and bloat gates require no metadata changes.
