# Handoff

- Baton: Bee, 2026-11-17.
- State: Job 1 implemented: hosted SEND requires a clear preview and confirmation, executes `trade.offer` with revision/request key, parses receipt, then confirms it via franchise-scoped `trades.read`. Local SEND remains disabled; SAVE TO POCKET says unavailable. Partner standings use parser-validated franchise rows. No schema changes.
- Tests: frontend 887/887 and TypeScript pass; production build passes. `make verify` ran all Go race packages successfully but its frontend suite caught a >110-char line; fixed, then all frontend tests passed. Need push-hook verify on final tree.
- Next: commit/push, open PR, verify CI; do not merge.
- Blocked on: tidy-check reports six dangling Chromium symlinks in external scratch directories; left untouched (outside authorized write scope). No other blocker.
- Tried/rejected: none.
