# Handoff

- Baton: Bee, 2026-11-17.
- State: Job 1 implemented and PR #324 is open: hosted SEND requires a blocker-free preview and confirmation, executes `trade.offer` with expected revision/request key, parses receipt, then verifies it via `trades.read`. Local SEND remains disabled; SAVE TO POCKET says unavailable. Partner standings use validated parser rows. No schema changes.
- Verification: frontend 887/887, TypeScript, production build, push hook, and PR checks pass. First CI attempt hit a race in unrelated `internal/serverclient.TestUnavailableKeychainLoginIsMemoryOnly`; rerun passed all shards and verify.
- Next: Christopher review; do not merge.
- Blocked on: tidy-check flags six dangling Chromium symlinks in external scratch directories; left untouched because outside authorized write scope.
- Tried/rejected: none.
