# Baton
Bee / lane L5 — 2026-11-14. Orchestrator reviews and merges; never merge this lane directly.

## Where it stands
Topic publisher, negotiated v2 SSE, managed desktop watcher, local long-poll and frontend subscription implemented.
Publisher API and source-traced ownership: `docs/architecture/hosted-changes.md`.

## Next move
L7 injects one `serverhttp.NewPublisher()` as `Hosted.Publisher` and publishes after scores are readable.
L11 uses authorized `ObserveView` + `Versions`, preserving snapshot coherence for reply tags.
L6/L12 consume `subscribeChanged`, reconcile on empty topics/connect/reconnect, then notify screens.
Reuse `Desktop.generation` with L1's transport generation; this base did not contain L1.

## Blocked / rejected
No guessed team scope: the existing recipient view gives only a revision; use permitted broad league invalidation.
No Wails event path or composition lifecycle edits. No event replay log; bounded hints fall back to resync.

## Evidence
47 distinct top-level Go tests passed with `-race` (23 serverclient; 16 stream/publisher; 8 hosted desktop).
Frontend subscription: 2 vitest tests; TypeScript and production build passed. Commit hooks passed.
Pre-push caught the old enrollment test's assumption of credential survival after a stream 403;
it now signs in again after seat assignment, retaining its enrollment/removal authorization assertions.
